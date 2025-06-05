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

	"github.com/clambin/ledswitcher/elect"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/schedule"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/clambin/ledswitcher/ledberry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
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

func run(ctx context.Context, cfg configuration.Configuration, r prometheus.Registerer, version string) error {
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

	s, err := schedule.New(cfg.LeaderConfiguration.Scheduler.Mode)
	if err != nil {
		return fmt.Errorf("schedule: %w", err)
	}
	led, err := ledberry.New(cfg.EndpointConfiguration.LEDPath)
	if err != nil {
		return fmt.Errorf("led: %w", err)
	}

	srv := server.NewServer(
		cfg.NodeName,
		s,
		redis.NewClient(&redis.Options{
			Addr:     cfg.RedisConfiguration.Addr,
			Username: cfg.RedisConfiguration.Username,
			Password: cfg.RedisConfiguration.Password,
		}),
		led,
		cfg.LeaderConfiguration.Rotation,
		10*time.Second,
		time.Minute,
		r,
		logger,
	)

	if cfg.LeaderConfiguration.Leader != "" {
		srv.SetLeader(cfg.LeaderConfiguration.Leader)
	} else {
		logger.Info("no leader specified. using k8s leader election")
		go elect.RunOrDie(
			ctx,
			cfg.K8SConfiguration.Namespace,
			cfg.K8SConfiguration.LockName,
			cfg.NodeName,
			func(identity string) { srv.SetLeader(identity) },
			logger.With(slog.String("component", "k8s")),
		)
	}

	return srv.Run(ctx)
}
