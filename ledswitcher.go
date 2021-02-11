package main

import (
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/endpoint"
	"github.com/clambin/ledswitcher/internal/led"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/clambin/ledswitcher/internal/version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var (
		hostname   string
		err        error
		interval   time.Duration
		masterHost string
		masterURL  string
		rotation   time.Duration
		expiry     time.Duration
		port       int
		ledPath    string
		debug      bool
	)
	// Parse args
	a := kingpin.New(filepath.Base(os.Args[0]), "ledswitcher")
	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("debug", "Log debug messages").Default("false").BoolVar(&debug)
	a.Flag("rotation", "Delay of led switching to the next controller").Default("1s").DurationVar(&rotation)
	a.Flag("expiry", "Remove clients from the list if we have not seen them").Default("1m").DurationVar(&expiry)
	a.Flag("port", "Controller listener port").Default("8080").IntVar(&port)
	a.Flag("master", "Hostname of instance that acts as controller").Required().StringVar(&masterHost)
	a.Flag("master-url", "URL used to reach the master").Default("http://ledswitcher:8080").StringVar(&masterURL)
	a.Flag("led-path", "path name to the sysfs directory for the LED").Default("/host/sys/class/leds/led1").StringVar(&ledPath)

	if _, err = a.Parse(os.Args[1:]); err != nil {
		a.Usage(os.Args[1:])
		os.Exit(1)
	}

	log.WithField("version", version.BuildVersion).Info("starting")
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// Set refresh interval
	// TODO: tweak this. accuracy vs. load
	interval = 100 * time.Millisecond

	log.WithFields(log.Fields{
		"rotation": rotation,
		"interval": interval,
	}).Debug("check intervals")

	// Where are we?
	if hostname, err = os.Hostname(); err != nil {
		log.WithField("err", err).Fatal("unable to determine hostname")
	}

	// Set up the server
	s := server.Server{
		Port:      port,
		IsMaster:  hostname == masterHost,
		MasterURL: masterURL,
		Controller: controller.Controller{
			Rotation: rotation,
			Expiry:   expiry,
		},
		Endpoint: endpoint.Endpoint{
			Name:     hostname,
			Hostname: hostname,
			Port:     port,
			LEDSetter: &led.RealSetter{
				LEDPath: ledPath,
			},
		},
	}

	// If we are the designated master, run the controller
	if hostname == masterHost {
		go func() {
			s.Controller.Run()
		}()
	}

	// Register the endpoint
	s.Endpoint.Register(masterURL)

	// Run the API server
	s.Run()

	log.Info("exiting")
}
