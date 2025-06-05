package configuration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConfiguration(t *testing.T) {
	want := Configuration{
		Debug:          false,
		PrometheusAddr: ":9090",
		LeaderConfiguration: LeaderConfiguration{
			Leader:   "",
			Rotation: 1000000000,
			Scheduler: SchedulerConfiguration{
				Mode: "linear",
			},
		},
		EndpointConfiguration: EndpointConfiguration{
			LEDPath: "/sys/class/leds/led1",
		},
		K8SConfiguration: K8SConfiguration{
			LockName:  "ledswitcher",
			Namespace: "default",
		},
	}
	got := GetConfiguration()
	got.NodeName = ""
	assert.Equal(t, want, got)
}
