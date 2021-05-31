package main

import (
	"context"
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/led"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/clambin/ledswitcher/internal/version"
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
	"time"
)

func main() {
	var (
		hostname           string
		err                error
		rotation           time.Duration
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
	a.Flag("rotation", "Delay of led switching to the next controller").Default("1s").DurationVar(&rotation)
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

	log.WithField("version", version.BuildVersion).Info("starting")
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// Where are we?
	if hostname, err = os.Hostname(); err != nil {
		log.WithField("err", err).Fatal("unable to determine hostname")
	}

	// Create the controller
	c := controller.New(hostname, port, rotation, alternate)
	go c.Run()

	// Set up the REST server
	s := &server.Server{
		Port:       port,
		Controller: c,
		LEDSetter:  &led.RealSetter{LEDPath: ledPath},
	}
	go s.Run()

	if leader == "" {
		runWithLeaderElection(leaseLockName, leaseLockNamespace, c)
	} else {
		runWithoutLeaderElection(c, controller.MakeURL(leader, port), hostname == leader)
	}

	log.Info("exiting")
}

func runWithoutLeaderElection(controllr *controller.Controller, leaderURL string, leading bool) {
	controllr.NewLeader <- leaderURL

	if leading {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		controllr.Lead(ctx)
	} else {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
	}
}

func runWithLeaderElection(leaseLockName, leaseLockNamespace string, controllr *controller.Controller) {
	var (
		err error
		cfg *rest.Config
	)

	if cfg, err = rest.InClusterConfig(); err != nil {
		log.WithField("err", err).Fatal("rest.InClusterConfig failed")
	}
	client := clientset.NewForConfigOrDie(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseLockName,
			Namespace: leaseLockNamespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: controllr.MyURL,
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
				controllr.Lead(ctx)
			},
			OnStoppedLeading: func() {
				log.Info("leader lost")
			},
			OnNewLeader: func(identity string) {
				log.WithField("id", identity).Info("new leader elected")
				controllr.NewLeader <- identity
			},
		},
	})
}
