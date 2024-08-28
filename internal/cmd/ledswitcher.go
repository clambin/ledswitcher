package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/ledswitcher/internal/client"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/clambin/ledswitcher/internal/server/ledsetter"
	"github.com/clambin/ledswitcher/internal/server/registry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var (
	serverCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ledswitcher_server_api_requests_total",
			Help: "A serverCounter for requests to the wrapped handler.",
		},
		[]string{"code", "method"},
	)

	serverDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ledswitcher_server_api_request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method"},
	)

	clientCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ledswitcher_client_api_requests_total",
			Help: "A counter for requests from the wrapped client.",
		},
		[]string{"code", "method"},
	)

	clientDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ledswitcher_client_api_request_duration_seconds",
			Help:    "A histogram of request latencies.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"code", "method"},
	)
)

func Main(ctx context.Context, version string) error {
	return runWithConfiguration(ctx, configuration.GetConfiguration(), prometheus.DefaultRegisterer, version)
}

func runWithConfiguration(ctx context.Context, cfg configuration.Configuration, promReg prometheus.Registerer, version string) error {
	h, c, r, logger, err := build(cfg, promReg)
	if err != nil {
		return err
	}

	logger.Info("starting ledswitcher", "version", version)
	defer logger.Info("shutting down ledswitcher")

	if cfg.LeaderConfiguration.Leader == "" {
		cfg.LeaderConfiguration.Leader = electLeader(ctx, cfg, logger)
	}
	r.Leading(weAreLeading(cfg))

	var g errgroup.Group
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	runHTTPServer(ctx, cfg.PrometheusAddr, mux, &g, logger)
	runHTTPServer(ctx, cfg.Addr, h, &g, logger)
	g.Go(func() error { return c.Run(ctx) })
	return g.Wait()
}

func build(cfg configuration.Configuration, promReg prometheus.Registerer) (http.Handler, *client.Client, *registry.Registry, *slog.Logger, error) {
	var opt slog.HandlerOptions
	if cfg.Debug {
		opt.Level = slog.LevelDebug
	}
	l := slog.New(slog.NewTextHandler(os.Stderr, &opt))

	ledSetter := ledsetter.Setter{LEDPath: cfg.LedPath}
	r := registry.Registry{Logger: l.With(slog.String("component", "registry"))}
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: promhttp.InstrumentRoundTripperDuration(clientDuration,
			promhttp.InstrumentRoundTripperCounter(clientCounter, http.DefaultTransport),
		),
	}
	c, err := client.NewWithHTTPClient(cfg, &r, httpClient, l.With(slog.String("component", "client")))
	if err != nil {
		err = fmt.Errorf("invalid client configuration: %w", err)
	}
	h := promhttp.InstrumentHandlerDuration(serverDuration,
		promhttp.InstrumentHandlerCounter(serverCounter,
			server.New(&ledSetter, c, &r, l.With(slog.String("component", "server"))),
		),
	)

	promReg.MustRegister(serverCounter, serverDuration, clientCounter, clientDuration)
	return h, c, &r, l, err
}

func electLeader(ctx context.Context, cfg configuration.Configuration, logger *slog.Logger) string {
	ch := make(chan string)
	go runElection(ctx, cfg, ch, logger.With(slog.String("component", "k8s")))
	return <-ch
}

func runElection(ctx context.Context, cfg configuration.Configuration, ch chan<- string, logger *slog.Logger) {
	k8sCfg, err := rest.InClusterConfig()
	if err != nil {
		logger.Error("rest.InClusterConfig failed", "err", err)
		panic(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		logger.Error("unable to determine hostname", "err", err)
		panic(err)
	}

	c := clientset.NewForConfigOrDie(k8sCfg)
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      cfg.K8SConfiguration.LockName,
			Namespace: cfg.K8SConfiguration.Namespace,
		},
		Client: c.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: hostname,
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   60 * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logger.Debug("OnStartLeading called")
			},
			OnStoppedLeading: func() {
				logger.Info("leader lost")
				os.Exit(1)
			},
			OnNewLeader: func(identity string) {
				logger.Info("leader elected", "leader", identity)
				ch <- identity
				<-ctx.Done()
			},
		},
	})
	panic("unreachable")
}

func runHTTPServer(ctx context.Context, addr string, h http.Handler, g *errgroup.Group, logger *slog.Logger) {
	s := &http.Server{Addr: addr, Handler: h}
	g.Go(func() error {
		err := s.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		if err != nil {
			logger.Error("server failed to start", "err", err)
		}
		return err
	})
	g.Go(func() error {
		<-ctx.Done()
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := s.Shutdown(stopCtx)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		if err != nil {
			logger.Error("server failed to stop", "err", err)
		}
		return err
	})
}

func weAreLeading(cfg configuration.Configuration) bool {
	if cfg.LeaderConfiguration.Leader == "localhost" {
		return true
	}
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return cfg.LeaderConfiguration.Leader == hostname
}
