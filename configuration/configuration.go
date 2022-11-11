package configuration

import (
	"fmt"
	"github.com/clambin/ledswitcher/version"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"path/filepath"
	"time"
)

type Configuration struct {
	Debug          bool
	ServerPort     int
	PrometheusPort int
	LedPath        string
	LeaderConfiguration
	K8SConfiguration
}

type LeaderConfiguration struct {
	Leader    string
	Rotation  time.Duration
	Scheduler SchedulerConfiguration
}

type SchedulerConfiguration struct {
	Mode string
}

type K8SConfiguration struct {
	LockName  string
	Namespace string
}

func GetConfigFromArgs(args []string) (Configuration, error) {
	var cfg Configuration
	var alternate bool

	a := kingpin.New(filepath.Base(os.Args[0]), "ledswitcher")
	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("debug", "Log debug messages").Short('d').Default("false").BoolVar(&cfg.Debug)
	a.Flag("rotation", "Delay of led switching to the next driver").Default("1s").DurationVar(&cfg.LeaderConfiguration.Rotation)
	a.Flag("mode", "LED pattern mode (linear or alternating").Short('m').Default("linear").StringVar(&cfg.LeaderConfiguration.Scheduler.Mode)
	a.Flag("alternate", "Alternate direction").Short('a').Default("false").BoolVar(&alternate)
	a.Flag("port", "Controller listener port").Default("8080").IntVar(&cfg.ServerPort)
	a.Flag("prometheus", "Prometheus metrics listener port").Default("9090").IntVar(&cfg.PrometheusPort)
	a.Flag("led-path", "path name to the sysfs directory for the LED").Default("/sys/class/leds/led1").StringVar(&cfg.LedPath)
	a.Flag("lock-name", "name of the election lock").Default("ledswitcher").StringVar(&cfg.K8SConfiguration.LockName)
	a.Flag("lock-namespace", "namespace of the election lock").Default("default").StringVar(&cfg.K8SConfiguration.Namespace)
	a.Flag("leader", "node to act as leader (if empty, k8s leader election will be used").Default("").StringVar(&cfg.Leader)

	if _, err := a.Parse(args); err != nil {
		return cfg, fmt.Errorf("invalid command line arguments: %w", err)
	}

	// fallback to previous options
	if alternate {
		cfg.LeaderConfiguration.Scheduler.Mode = "alternating"
	}
	return cfg, nil
}
