package main

import (
	"context"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher"
	"github.com/clambin/ledswitcher/version"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	log.WithFields(log.Fields{
		"version": version.BuildVersion,
	}).Info("starting")

	cfg, err := configuration.GetConfigFromArgs(os.Args[1:])
	if err != nil {
		log.WithError(err).Fatal("invalid argument(s)")
	}

	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	srv, err := switcher.New(cfg)
	if err != nil {
		log.WithError(err).Fatal("failed to create Switcher")
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		srv.Run(ctx)
		wg.Done()
	}()

	if cfg.LeaderConfiguration.Leader == "" {
		log.Info("no leader provided. using k8s leader election instead")
		wg.Add(1)
		go func() {
			runWithLeaderElection(ctx, srv, cfg)
			wg.Done()
		}()
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interrupt

	log.Info("shutting down")
	cancel()
	wg.Wait()
	log.Info("exiting")
}

func runWithLeaderElection(ctx context.Context, srv *switcher.Switcher, cfg configuration.Configuration) {
	k8sCfg, err := rest.InClusterConfig()
	if err != nil {
		log.WithError(err).Fatal("rest.InClusterConfig failed")
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.WithError(err).Fatal("unable to determine hostname")
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
				log.Info("OnStartLeading called")
				//<-ctx.Done()
			},
			OnStoppedLeading: func() {
				log.Fatal("leader lost")
			},
			OnNewLeader: func(identity string) {
				log.Infof("leader elected: %s", identity)
				srv.SetLeader(identity)
			},
		},
	})
}
