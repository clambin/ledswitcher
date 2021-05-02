package main

import (
	"context"
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/led"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/clambin/ledswitcher/internal/version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func main() {
	var (
		hostname           string
		err                error
		rotation           time.Duration
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
	a.Flag("debug", "Log debug messages").Default("false").BoolVar(&debug)
	a.Flag("rotation", "Delay of led switching to the next controller").Default("1s").DurationVar(&rotation)
	a.Flag("port", "Controller listener port").Default("8080").IntVar(&port)
	a.Flag("led-path", "path name to the sysfs directory for the LED").Default("/sys/class/leds/led1").StringVar(&ledPath)
	a.Flag("lock-name", "name of the election lock").Default("ledswitcher").StringVar(&leaseLockName)
	a.Flag("lock-namespace", "namespace of the election lock").Default("default").StringVar(&leaseLockNamespace)
	a.Flag("leader", "node to act as leader (if empty, k8s leader election will be used").Default("").StringVar(&leader)

	if _, err = a.Parse(os.Args[1:]); err != nil {
		a.Usage(os.Args[1:])
		os.Exit(1)
	}

	// log.SetReportCaller(true)
	log.WithField("version", version.BuildVersion).Info("starting")
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// Where are we?
	if hostname, err = os.Hostname(); err != nil {
		log.WithField("err", err).Fatal("unable to determine hostname")
	}

	// Set up the server
	s := server.Server{
		Port:       port,
		Controller: controller.New(hostname, port),
		LEDSetter:  &led.RealSetter{LEDPath: ledPath},
	}
	go s.Run()

	if leader == "" {
		runWithLeaderElection(leaseLockName, leaseLockNamespace, hostname, s.Controller, rotation)
	} else {
		runWithoutLeaderElection(s.Controller, rotation, hostname == leader)
	}

	log.Info("exiting")
}

func run(ctx context.Context, controllr *controller.Controller, rotation time.Duration, isLeader bool) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	if isLeader {
		log.WithField("rotation", rotation).Info("leader started")

		ticker := time.NewTicker(rotation)
	loop:
		for {
			select {
			case <-ticker.C:
				controllr.Tick <- struct{}{}
			case <-ctx.Done():
				break loop
			case <-interrupt:
				break loop
			}
		}
		log.Debug("leader stopped")
	} else {
		<-interrupt
	}
}

func runWithoutLeaderElection(controllr *controller.Controller, rotation time.Duration, isLeader bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	run(ctx, controllr, rotation, isLeader)
}

func runWithLeaderElection(leaseLockName, leaseLockNamespace, hostname string, controllr *controller.Controller, rotation time.Duration) {
	var (
		err error
		cfg *rest.Config
	)
	// leader election uses the Kubernetes API by writing to a
	// lock object, which can be a LeaseLock object (preferred),
	// a ConfigMap, or an Endpoints (deprecated) object.
	// Conflicting writes are detected and each client handles those actions
	// independently.
	if cfg, err = rest.InClusterConfig(); err != nil {
		log.WithField("err", err).Fatal("rest.InClusterConfig failed")
	}
	client := clientset.NewForConfigOrDie(cfg)

	// use a Go context so we can tell the leader election code when we
	// want to step down
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// we use the Lease lock type since edits to Leases are less common
	// and fewer objects in the cluster watch "all Leases".
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

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock: lock,
		// IMPORTANT: you MUST ensure that any code you have that
		// is protected by the lease must terminate **before**
		// you call cancel. Otherwise, you could have a background
		// loop still running and another process could
		// get elected before your background loop finished, violating
		// the stated goal of the lease.
		ReleaseOnCancel: true,
		LeaseDuration:   60 * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				// we are the leader
				run(ctx, controllr, rotation, true)
			},
			OnStoppedLeading: func() {
				log.WithField("id", hostname).Debug("leader lost")
				// os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				// we're notified when new leader elected
				log.WithField("id", identity).Info("new leader elected")
				controllr.NewLeader <- identity
			},
		},
	})
}
