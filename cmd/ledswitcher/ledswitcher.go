package main

import (
	"context"
	"flag"
	"github.com/clambin/go-common/taskmanager"
	promserver "github.com/clambin/go-common/taskmanager/prometheus"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher"
	"github.com/clambin/ledswitcher/version"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := getConfiguration()

	var opts slog.HandlerOptions
	if cfg.Debug {
		opts.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &opts))

	slog.Info("ledswitcher starting", "version", version.BuildVersion)

	srv, err := switcher.New(cfg, logger)
	if err != nil {
		slog.Error("failed to create Switcher", "err", err)
		os.Exit(1)
	}
	prometheus.DefaultRegisterer.MustRegister(srv)

	tm := taskmanager.New(
		promserver.New(promserver.WithAddr(cfg.PrometheusAddr)),
		srv,
	)

	if cfg.LeaderConfiguration.Leader == "" {
		slog.Info("no leader provided. using k8s leader election instead")
		_ = tm.Add(taskmanager.TaskFunc(func(ctx context.Context) error {
			runWithLeaderElection(ctx, srv, cfg)
			return nil
		}))
	}

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()

	if err = tm.Run(ctx); err != nil {
		slog.Error("failed to start", "err", err)
		os.Exit(1)
	}

	slog.Info("exiting")
}

func runWithLeaderElection(ctx context.Context, srv *switcher.Switcher, cfg configuration.Configuration) {
	k8sCfg, err := rest.InClusterConfig()
	if err != nil {
		slog.Error("rest.InClusterConfig failed", "err", err)
		panic(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		slog.Error("unable to determine hostname", "err", err)
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

func getConfiguration() configuration.Configuration {
	var cfg configuration.Configuration
	flag.BoolVar(&cfg.Debug, "debug", false, "log debug messages")
	flag.DurationVar(&cfg.LeaderConfiguration.Rotation, "rotation", time.Second, "delay of LED switching to the next state")
	flag.StringVar(&cfg.LeaderConfiguration.Scheduler.Mode, "mode", "linear", "LED pattern mode")
	flag.StringVar(&cfg.Addr, "addr", ":8080", "controller address")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus", ":9090", "prometheus metrics address")
	flag.StringVar(&cfg.LedPath, "led-path", "/sys/class/leds/led1", "path name to the sysfs directory for the LED")
	flag.StringVar(&cfg.K8SConfiguration.LockName, "lock-name", "ledswitcher", "name of the k8s leader election lock")
	flag.StringVar(&cfg.K8SConfiguration.Namespace, "lock-namespace", "default", "namespace of the k8s leader election lock")
	flag.StringVar(&cfg.Leader, "leader", "", "node to act as leader (if empty, k8s leader election will be used")

	flag.Parse()
	return cfg
}
