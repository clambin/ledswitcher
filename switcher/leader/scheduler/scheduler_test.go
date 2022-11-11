package scheduler_test

import (
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher/leader/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScheduler_UpdateStatus_NoRegister(t *testing.T) {
	// normally will never happen: broker will never call a host that didn't register first, so UpdateStatus for
	// an unregistered host is impossible.
	s, err := scheduler.New(configuration.SchedulerConfiguration{Mode: "linear"})
	require.NoError(t, err)

	s.UpdateStatus("host1", true)
	assert.Equal(t, []scheduler.Action{
		{Host: "host1", State: true},
	}, s.Next())
}

func TestScheduler_GetHosts(t *testing.T) {
	s, err := scheduler.New(configuration.SchedulerConfiguration{Mode: "linear"})
	require.NoError(t, err)

	s.Register("host1")
	hosts := s.GetHosts()
	require.Len(t, hosts, 1)
	assert.Equal(t, "host1", hosts[0].Name)

	s.Register("host2")
	hosts = s.GetHosts()
	require.Len(t, hosts, 2)
	assert.Equal(t, "host1", hosts[0].Name)
	assert.Equal(t, "host2", hosts[1].Name)
}

func TestScheduler_Register(t *testing.T) {
	s, err := scheduler.New(configuration.SchedulerConfiguration{Mode: "linear"})
	require.NoError(t, err)

	s.Register("host1")
	s.Register("host2")
	s.Register("host1")

	health := s.GetHosts()
	assert.Len(t, health, 2)
}
