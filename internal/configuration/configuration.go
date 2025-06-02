package configuration

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"time"
)

type Configuration struct {
	K8SConfiguration      K8SConfiguration
	Addr                  string
	PrometheusAddr        string
	PProfAddr             string
	LeaderConfiguration   LeaderConfiguration
	EndpointConfiguration EndpointConfiguration
	Debug                 bool
}

type LeaderConfiguration struct {
	Leader    string
	Scheduler SchedulerConfiguration
	Rotation  time.Duration
}

type EndpointConfiguration struct {
	LEDPath string
}

type SchedulerConfiguration struct {
	Mode string
}

type K8SConfiguration struct {
	LockName  string
	Namespace string
}

// URLFromHost converts a host to a URL, using Addr to determine the latter.  If host is blank, the system's hostname is used.
// No scheme (eg http://) is added.
func (c Configuration) URLFromHost(host string) (string, error) {
	if host == "" {
		return "", errors.New("host is empty")
	}
	_, port, err := net.SplitHostPort(c.Addr)
	if err != nil {
		return "", fmt.Errorf("failed to determine port: %w", err)
	}
	return host + ":" + port, nil
}

func (c Configuration) MustURLFromHost(host string) string {
	url, err := c.URLFromHost(host)
	if err != nil {
		panic(err)
	}
	return url
}

func GetConfiguration() Configuration {
	var cfg Configuration
	flag.DurationVar(&cfg.LeaderConfiguration.Rotation, "rotation", time.Second, "delay of LED switching to the next state")
	flag.StringVar(&cfg.LeaderConfiguration.Scheduler.Mode, "mode", "linear", "LED pattern mode")
	flag.StringVar(&cfg.LeaderConfiguration.Leader, "leader", "", "leader node name (if empty, k8s leader election will be used")
	flag.StringVar(&cfg.EndpointConfiguration.LEDPath, "led-path", "/sys/class/leds/led1", "path name to the sysfs directory for the LED")
	flag.StringVar(&cfg.K8SConfiguration.LockName, "lock-name", "ledswitcher", "name of the k8s leader election lock")
	flag.StringVar(&cfg.K8SConfiguration.Namespace, "lock-namespace", "default", "namespace of the k8s leader election lock")
	flag.StringVar(&cfg.Addr, "addr", ":8080", "controller address")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus", ":9090", "prometheus metrics address")
	flag.StringVar(&cfg.PProfAddr, "pprof", "", "pprof listener address (default: don't run pprof")
	flag.BoolVar(&cfg.Debug, "debug", false, "log debug messages")

	flag.Parse()
	return cfg
}
