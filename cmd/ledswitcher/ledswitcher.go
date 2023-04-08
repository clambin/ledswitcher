package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher"
	"github.com/clambin/ledswitcher/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/slog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	cfg, err := configuration.GetConfigFromArgs(os.Args[1:])
	if err != nil {
		panic(err)
	}

	var opts slog.HandlerOptions
	if cfg.Debug {
		opts.Level = slog.LevelDebug
		opts.AddSource = true
	}
	slog.SetDefault(slog.New(opts.NewTextHandler(os.Stdout)))

	slog.Info("ledswitcher starting", "version", version.BuildVersion)

	go runPrometheusServer(cfg.PrometheusPort)

	srv, err := switcher.New(cfg)
	if err != nil {
		slog.Error("failed to create Switcher", err)
		panic(err)
	}
	prometheus.DefaultRegisterer.MustRegister(srv)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		srv.Run(ctx)
		wg.Done()
	}()

	if cfg.LeaderConfiguration.Leader == "" {
		slog.Info("no leader provided. using k8s leader election instead")
		wg.Add(1)
		go func() {
			runWithLeaderElection(ctx, srv, cfg)
			wg.Done()
		}()
	}

	ctx2, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer done()
	<-ctx2.Done()

	slog.Info("shutting down")
	cancel()
	wg.Wait()
	slog.Info("exiting")
}

func runWithLeaderElection(ctx context.Context, srv *switcher.Switcher, cfg configuration.Configuration) {
	k8sCfg, err := rest.InClusterConfig()
	if err != nil {
		slog.Error("rest.InClusterConfig failed", err)
		panic(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		slog.Error("unable to determine hostname", err)
		panic(err)
	}

	client := clientset.NewForConfigOrDie(k8sCfg)
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      cfg.K8SConfiguration.LockName,
			Namespace: cfg.K8SConfiguration.Namespace,
		},
		Client: client.CoordinationV1(),
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
				slog.Info("OnStartLeading called")
				//<-ctx.Done()
			},
			OnStoppedLeading: func() {
				slog.Info("leader lost")
				os.Exit(1)
			},
			OnNewLeader: func(identity string) {
				slog.Info("leader elected: " + identity)
				srv.SetLeader(identity)
			},
		},
	})
}

func runPrometheusServer(port int) {
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start Prometheus listener", err)
		panic(err)
	}
}
