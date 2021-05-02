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

	if _, err = a.Parse(os.Args[1:]); err != nil {
		a.Usage(os.Args[1:])
		os.Exit(1)
	}

	log.SetReportCaller(true)
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
		LEDSetter: &led.RealSetter{
			LEDPath: ledPath,
		},
	}

	// Run the server in the background
	go func() { s.Run() }()

	// leader election uses the Kubernetes API by writing to a
	// lock object, which can be a LeaseLock object (preferred),
	// a ConfigMap, or an Endpoints (deprecated) object.
	// Conflicting writes are detected and each client handles those actions
	// independently.
	var cfg *rest.Config
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
			Identity: s.Controller.MyURL,
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
				log.WithFields(log.Fields{
					"id":       s.Controller.MyURL,
					"rotation": rotation,
				}).Info("entering ticker loop")

				tickTimer := time.NewTimer(rotation)
			loop:
				for {
					select {
					case <-tickTimer.C:
						log.Debug("pre-tick")
						s.Controller.Tick <- struct{}{}
						log.Debug("post-tick")
					case <-ctx.Done():
						log.Debug("breaking loop")
						break loop
					}
				}
				tickTimer.Stop()
				log.Debug("exiting ticker loop")
			},
			OnStoppedLeading: func() {
				// we can do cleanup here
				log.WithField("id", hostname).Info("leader lost")
				// os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				log.WithField("id", identity).Info("new leader elected")
				// we're notified when new leader elected
				s.Controller.NewLeader <- identity
				// if identity == hostname {
				// I just got the lock
				//	return
				//}
			},
		},
	})

	log.Info("exiting")
}
