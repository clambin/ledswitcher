package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/ledswitcher/internal/client"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/registry"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/clambin/ledswitcher/internal/server/ledsetter"
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
	buckets = []float64{.0001, .0005, .001, .005, .01, .05}

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
			Buckets: buckets,
		},
		[]string{"code", "method"},
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
			Buckets: buckets,
		},
		[]string{"code", "method"},
	)
)

func Main(ctx context.Context, version string) error {
	return runWithConfiguration(ctx, configuration.GetConfiguration(), prometheus.DefaultRegisterer, version)
}

func runWithConfiguration(ctx context.Context, cfg configuration.Configuration, promReg prometheus.Registerer, version string) error {
	var opt slog.HandlerOptions
	if cfg.Debug {
		opt.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &opt))

	h, c, err := build(cfg, promReg, logger)
	if err != nil {
		return err
	}

	logger.Info("starting ledswitcher", "version", version)
	defer logger.Info("shutting down ledswitcher")

	var g errgroup.Group
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	runHTTPServer(ctx, cfg.PrometheusAddr, mux, &g, logger)
	runHTTPServer(ctx, cfg.Addr, h, &g, logger)
	g.Go(func() error { return c.Run(ctx) })

	if cfg.LeaderConfiguration.Leader != "" {
		c.Leader <- cfg.LeaderConfiguration.Leader
	} else {
		logger.Info("no leader specified. using k8s leader election")
		go runElection(ctx, cfg, c.Leader, logger.With(slog.String("component", "k8s")))
	}

	return g.Wait()
}

func build(cfg configuration.Configuration, promReg prometheus.Registerer, logger *slog.Logger) (http.Handler, *client.Client, error) {
	ledSetter := ledsetter.Setter{LEDPath: cfg.LedPath}
	r := registry.Registry{Logger: logger.With(slog.String("component", "registry"))}
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: promhttp.InstrumentRoundTripperCounter(clientCounter,
			promhttp.InstrumentRoundTripperDuration(clientDuration,
				http.DefaultTransport,
			),
		),
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("hostname: %w", err)
	}
	c, err := client.NewWithHTTPClient(cfg, hostname, &r, httpClient, logger.With(slog.String("component", "client")))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid client configuration: %w", err)
	}
	h := promhttp.InstrumentHandlerCounter(serverCounter,
		promhttp.InstrumentHandlerDuration(serverDuration,
			server.New(&ledSetter, c, &r, logger.With(slog.String("component", "server"))),
		),
	)

	promReg.MustRegister(serverCounter, serverDuration, clientCounter, clientDuration, &r)
	return h, c, err
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
				logger.Info("OnStartLeading called")
			},
			OnStoppedLeading: func() {
				logger.Info("leader lost")
			},
			OnNewLeader: func(identity string) {
				logger.Info("leader elected", "leader", identity)
				ch <- identity
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
