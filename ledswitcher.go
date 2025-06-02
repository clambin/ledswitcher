package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/ledswitcher"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	if err := run(ctx, configuration.GetConfiguration(), prometheus.DefaultRegisterer, version, os.Hostname); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to start: %s\n", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg configuration.Configuration, r prometheus.Registerer, version string, getHostname func() (string, error)) error {
	var opt slog.HandlerOptions
	if cfg.Debug {
		opt.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &opt))

	logger.Info("starting ledswitcher", "version", version)
	defer logger.Info("shutting down ledswitcher")

	if cfg.PProfAddr != "" {
		go func() {
			logger.Debug("starting pprof server", "addr", cfg.PProfAddr)
			if err := http.ListenAndServe(cfg.PProfAddr, nil); !errors.Is(err, http.ErrServerClosed) {
				logger.Error("failed to start pprof server", "err", err)
			}
		}()
	}
	go func() {
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	if getHostname == nil {
		getHostname = os.Hostname
	}
	server, err := ledswitcher.New(cfg, getHostname, r, logger)
	if err != nil {
		return err
	}

	if cfg.LeaderConfiguration.Leader != "" {
		server.SetLeader(cfg.LeaderConfiguration.Leader)
	} else {
		logger.Info("no leader specified. using k8s leader election")
		go runElection(ctx, cfg, server, logger.With(slog.String("component", "k8s")))
	}

	return server.Run(ctx)
}

func runElection(ctx context.Context, cfg configuration.Configuration, server *ledswitcher.LEDSwitcher, logger *slog.Logger) {
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
				server.SetLeader(identity)
			},
		},
	})
}
