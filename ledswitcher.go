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

	"github.com/clambin/ledswitcher/elect"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/ledswitcher"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var version = "change-me"

func main() {
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	if err = run(ctx, configuration.GetConfiguration(), prometheus.DefaultRegisterer, version, hostname); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to start: %s\n", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg configuration.Configuration, r prometheus.Registerer, version string, hostname string) error {
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
		logger.Debug("starting prometheus server", "addr", cfg.PrometheusAddr)
		if err := http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler()); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start prometheus server", "err", err)
		}
	}()

	server, err := ledswitcher.New(cfg, hostname, r, logger)
	if err != nil {
		return err
	}

	if cfg.LeaderConfiguration.Leader != "" {
		server.SetLeader(cfg.LeaderConfiguration.Leader)
	} else {
		logger.Info("no leader specified. using k8s leader election")
		go elect.RunOrDie(
			ctx,
			cfg.K8SConfiguration.Namespace,
			cfg.K8SConfiguration.LockName,
			hostname,
			func(identity string) { server.SetLeader(identity) },
			logger.With(slog.String("component", "k8s")),
		)
	}

	return server.Run(ctx)
}
