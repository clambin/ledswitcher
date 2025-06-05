package configuration

import (
	"flag"
	"os"
	"time"
)

type Configuration struct {
	K8SConfiguration      K8SConfiguration
	PrometheusAddr        string
	PProfAddr             string
	NodeName              string
	EndpointConfiguration EndpointConfiguration
	LeaderConfiguration   LeaderConfiguration
	RedisConfiguration    RedisConfiguration
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

type RedisConfiguration struct {
	Addr     string
	Username string
	Password string
}

func GetConfiguration() Configuration {
	hostname := os.Getenv("NODE_NAME")
	if hostname == "" {
		var err error
		if hostname, err = os.Hostname(); err != nil {
			panic(err)
		}
	}
	var cfg Configuration
	flag.DurationVar(&cfg.LeaderConfiguration.Rotation, "rotation", time.Second, "delay of LED switching to the next state")
	flag.StringVar(&cfg.LeaderConfiguration.Scheduler.Mode, "mode", "linear", "LED pattern mode")
	flag.StringVar(&cfg.LeaderConfiguration.Leader, "leader", "", "leader node name (if empty, k8s leader election will be used")
	flag.StringVar(&cfg.EndpointConfiguration.LEDPath, "led-path", "/sys/class/leds/led1", "path name to the sysfs directory for the LED")
	flag.StringVar(&cfg.K8SConfiguration.LockName, "lock-name", "ledswitcher", "name of the k8s leader election lock")
	flag.StringVar(&cfg.K8SConfiguration.Namespace, "lock-namespace", "default", "namespace of the k8s leader election lock")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus", ":9090", "prometheus metrics address")
	flag.StringVar(&cfg.PProfAddr, "pprof", "", "pprof listener address (default: don't run pprof")
	flag.BoolVar(&cfg.Debug, "debug", false, "log debug messages")
	flag.StringVar(&cfg.RedisConfiguration.Addr, "redis.addr", "", "redis node address")
	flag.StringVar(&cfg.RedisConfiguration.Username, "redis.username", "", "redis node username")
	flag.StringVar(&cfg.RedisConfiguration.Password, "redis.password", "", "redis node password")
	flag.StringVar(&cfg.NodeName, "node-name", hostname, "node name")

	flag.Parse()
	return cfg
}
