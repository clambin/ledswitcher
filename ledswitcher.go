package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/clambin/go-common/httputils"
	"github.com/clambin/ledswitcher/internal/client"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/metrics"
	"github.com/clambin/ledswitcher/internal/registry"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/clambin/ledswitcher/pkg/ledberry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

var version = "change-me"

func main() {
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()
	if err := run(ctx, configuration.GetConfiguration(), prometheus.DefaultRegisterer, version); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to start: %s\n", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg configuration.Configuration, promReg prometheus.Registerer, version string) error {
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

	if cfg.PProfAddr != "" {
		go func() { _ = http.ListenAndServe(cfg.PProfAddr, nil) }()
	}

	var g errgroup.Group
	g.Go(func() error {
		return httputils.RunServer(ctx, &http.Server{Addr: cfg.PrometheusAddr, Handler: promhttp.Handler()})
	})
	g.Go(func() error {
		return httputils.RunServer(ctx, &http.Server{Addr: cfg.Addr, Handler: h})
	})
	g.Go(func() error {
		return c.Run(ctx)
	})

	if cfg.LeaderConfiguration.Leader != "" {
		c.Leader <- cfg.LeaderConfiguration.Leader
	} else {
		logger.Info("no leader specified. using k8s leader election")
		go runElection(ctx, cfg, c.Leader, logger.With(slog.String("component", "k8s")))
	}

	return g.Wait()
}

func build(cfg configuration.Configuration, promReg prometheus.Registerer, logger *slog.Logger) (http.Handler, *client.Client, error) {
	led, err := initLED(cfg)
	if err != nil {
		return nil, nil, err
	}
	m := metrics.New()
	r := registry.Registry{Logger: logger.With(slog.String("component", "registry"))}
	httpClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: m.ClientMiddleware(http.DefaultTransport),
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, fmt.Errorf("hostname: %w", err)
	}
	c, err := client.NewWithHTTPClient(cfg, hostname, &r, httpClient, logger.With(slog.String("component", "client")))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid client configuration: %w", err)
	}
	h := m.ServerMiddleware(server.New(led, c, &r, logger.With(slog.String("component", "server"))))

	promReg.MustRegister(m, &r)
	return h, c, err
}

func initLED(cfg configuration.Configuration) (*ledberry.LED, error) {
	led, err := ledberry.New(cfg.LedPath)
	if err == nil {
		err = led.SetActiveMode("none")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to access led: %w", err)
	}
	return led, nil
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
			},
			OnNewLeader: func(identity string) {
				logger.Info("leader elected", "leader", identity)
				ch <- identity
			},
		},
	})
}
