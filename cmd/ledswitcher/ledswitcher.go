package main

import (
	"context"
	"flag"
	"github.com/clambin/go-common/http/metrics"
	"github.com/clambin/go-common/http/middleware"
	"github.com/clambin/go-common/http/roundtripper"
	"github.com/clambin/go-common/taskmanager"
	"github.com/clambin/go-common/taskmanager/httpserver"
	promserver "github.com/clambin/go-common/taskmanager/prometheus"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/endpoint"
	"github.com/clambin/ledswitcher/internal/leader"
	"github.com/clambin/ledswitcher/internal/led"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version = "change-me"

func main() {
	cfg := getConfiguration()

	var opts slog.HandlerOptions
	if cfg.Debug {
		opts.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &opts))

	logger.Info("ledswitcher starting", "version", version)
	defer logger.Info("ledswitcher exiting")

	l := makeLeader(cfg.LeaderConfiguration, logger.With("component", "leader"))
	ep := makeEndpoint(cfg, logger.With("component", "endpoint"))

	serverMetrics := metrics.NewRequestSummaryMetrics("ledswitcher", "server", nil)
	prometheus.MustRegister(serverMetrics)
	mw := middleware.WithRequestMetrics(serverMetrics)

	m := http.NewServeMux()
	m.Handle("POST /leader/register", l.RegisterHandler)
	m.Handle("GET /leader/stats", l.StatsHandler)
	m.Handle("GET /endpoint/health", ep.HealthHandler)
	m.Handle("/endpoint/led", ep.LEDHandler)

	tm := taskmanager.New(
		promserver.New(promserver.WithAddr(cfg.PrometheusAddr)),
		httpserver.New(cfg.Addr, mw(m)),
		ep,
		l,
	)

	if cfg.LeaderConfiguration.Leader == "" {
		logger.Info("no leader provided. using k8s leader election instead")
		_ = tm.Add(taskmanager.TaskFunc(func(ctx context.Context) error {
			runWithLeaderElection(ctx, ep, l, cfg, logger.With("component", "k8s"))
			return nil
		}))
	}

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()

	if err := tm.Run(ctx); err != nil {
		slog.Error("failed to start", "err", err)
		os.Exit(1)
	}
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
	flag.StringVar(&cfg.Leader, "leader", "", "leader node name (if empty, k8s leader election will be used")

	flag.Parse()
	return cfg
}

func getEndpointURL(cfg configuration.Configuration) string {
	hostname, err := os.Hostname()
	if err != nil {
		panic("unable to determine hostname: " + err.Error())
	}
	return cfg.LeaderURL(hostname)
}

func makeLeader(cfg configuration.LeaderConfiguration, logger *slog.Logger) *leader.Leader {
	leaderClientMetrics := metrics.NewRequestSummaryMetrics("ledswitcher", "leader", nil)
	prometheus.MustRegister(leaderClientMetrics)
	leaderClient := http.Client{Transport: roundtripper.New(roundtripper.WithRequestMetrics(leaderClientMetrics))}

	l, err := leader.New(cfg, &leaderClient, logger)
	if err != nil {
		panic(err)
	}
	return l
}

func makeEndpoint(cfg configuration.Configuration, logger *slog.Logger) *endpoint.Endpoint {
	endpointClientMetrics := metrics.NewRequestSummaryMetrics("ledswitcher", "endpoint", nil)
	prometheus.MustRegister(endpointClientMetrics)
	endpointClient := http.Client{Transport: roundtripper.New(roundtripper.WithRequestMetrics(endpointClientMetrics))}

	setter := led.Setter{LEDPath: cfg.LedPath}
	return endpoint.New(getEndpointURL(cfg), 0, &endpointClient, &setter, logger)
}

func runWithLeaderElection(ctx context.Context, ep *endpoint.Endpoint, l *leader.Leader, cfg configuration.Configuration, logger *slog.Logger) {
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
				logger.Info("OnStartLeading called")
				//<-ctx.Done()
			},
			OnStoppedLeading: func() {
				logger.Info("leader lost")
				os.Exit(1)
			},
			OnNewLeader: func(identity string) {
				logger.Info("leader elected: " + identity)
				ep.SetLeaderURL(cfg.LeaderURL(identity))
				l.SetLeading(identity == hostname)
			},
		},
	})
}
