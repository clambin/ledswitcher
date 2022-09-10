package main

import (
	"context"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/clambin/ledswitcher/endpoint"
	"github.com/clambin/ledswitcher/server"
	"github.com/clambin/ledswitcher/version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	var (
		hostname           string
		err                error
		rotation           time.Duration
		mode               string
		alternate          bool
		port               int
		ledPath            string
		debug              bool
		leaseLockName      string
		leaseLockNamespace string
		leader             string
	)

	// Parse args
	a := kingpin.New(filepath.Base(os.Args[0]), "ledswitcher")
	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("debug", "Log debug messages").Short('d').Default("false").BoolVar(&debug)
	a.Flag("rotation", "Delay of led switching to the next driver").Default("1s").DurationVar(&rotation)
	a.Flag("mode", "LED pattern mode (linear or alternating").Short('m').Default("linear").StringVar(&mode)
	a.Flag("alternate", "Alternate direction").Short('a').Default("false").BoolVar(&alternate)
	a.Flag("port", "Controller listener port").Default("8080").IntVar(&port)
	a.Flag("led-path", "path name to the sysfs directory for the LED").Default("/sys/class/leds/led1").StringVar(&ledPath)
	a.Flag("lock-name", "name of the election lock").Default("ledswitcher").StringVar(&leaseLockName)
	a.Flag("lock-namespace", "namespace of the election lock").Default("default").StringVar(&leaseLockNamespace)
	a.Flag("leader", "node to act as leader (if empty, k8s leader election will be used").Default("").StringVar(&leader)

	if _, err = a.Parse(os.Args[1:]); err != nil {
		a.Usage(os.Args[1:])
		os.Exit(1)
	}

	// fallback to previous options
	if alternate {
		mode = "alternating"
	}

	log.WithFields(log.Fields{
		"version":  version.BuildVersion,
		"mode":     mode,
		"interval": rotation,
	}).Info("starting")

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// Where are we?
	if hostname, err = os.Hostname(); err != nil {
		log.WithField("err", err).Fatal("unable to determine hostname")
	}

	var s *scheduler.Scheduler
	s, err = scheduler.New(mode)
	if err != nil {
		log.WithError(err).Fatalf("invalid mode: %s", mode)
	}

	srv := server.New(hostname, port, ledPath, rotation, s, leader)

	ctx, cancel := context.WithCancel(context.Background())

	srv.Start(ctx)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	if leader == "" {
		go runWithLeaderElection(ctx, srv.Endpoint, hostname, leaseLockName, leaseLockNamespace)
	}

	<-interrupt

	log.Info("shutting down")
	cancel()
	srv.Wait()
	log.Info("exiting")
}

func runWithLeaderElection(ctx context.Context, s *endpoint.Endpoint, hostname, leaseLockName, leaseLockNamespace string) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.WithField("err", err).Fatal("rest.InClusterConfig failed")
	}

	client := clientset.NewForConfigOrDie(cfg)
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseLockName,
			Namespace: leaseLockNamespace,
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
				<-ctx.Done()
			},
			OnStoppedLeading: func() {
				log.Fatal("leader lost")
			},
			OnNewLeader: func(identity string) {
				log.WithField("id", identity).Info("new leader elected")
				s.SetLeader(identity)
			},
		},
	})
}
