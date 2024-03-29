package configuration

import (
	"time"
)

type Configuration struct {
	Debug          bool
	Addr           string
	PrometheusAddr string
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
