package main

import (
	"fmt"
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/endpoint"
	"github.com/clambin/ledswitcher/internal/led"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/clambin/ledswitcher/internal/version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func main() {
	var (
		hostname   string
		err        error
		masterHost string
		masterURL  string
		rotation   time.Duration
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
	a.Flag("port", "Controller listener port").Default("8080").IntVar(&port)
	a.Flag("master", "Hostname of instance that acts as controller").Required().StringVar(&masterHost)
	a.Flag("led-path", "path name to the sysfs directory for the LED").Default("/sys/class/leds/led1").StringVar(&ledPath)

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

	// Set up master URL
	masterURL = fmt.Sprintf("http://%s:%d", masterHost, port)

	// Set up the server
	s := server.Server{
		Port:      port,
		IsMaster:  hostname == masterHost,
		MasterURL: masterURL,
		Controller: controller.Controller{
			Rotation: rotation,
		},
		Endpoint: endpoint.Endpoint{
			Name:      hostname,
			Hostname:  hostname,
			Port:      port,
			MasterURL: masterURL,
			LEDSetter: &led.RealSetter{
				LEDPath: ledPath,
			},
		},
	}

	// If we are the designated master, run the controller
	if s.IsMaster {
		log.Infof("server running on %s", hostname)
		go func() {
			s.Controller.Run()
		}()
	}

	// Run the API server in the background
	go func() { s.Run() }()

	// Register the endpoint
	s.Endpoint.Register()

	// Re-register periodically in case we lose connection
	refresh := time.NewTicker(30 * time.Second)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

loop:
	for {
		select {
		case <-refresh.C:
			s.Endpoint.Register()
		case <-interrupt:
			break loop
		}
	}

	_ = s.Endpoint.LEDSetter.SetLED(true)

	log.Info("exiting")
}
