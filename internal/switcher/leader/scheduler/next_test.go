package scheduler_test

import (
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/switcher/leader/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScheduler_Next(t *testing.T) {
	s, err := scheduler.New(configuration.SchedulerConfiguration{Mode: "linear"})
	require.NoError(t, err)

	assert.Zero(t, "", s.Next())

	s.Register("host1")
	assert.Equal(t, []scheduler.Action{{Host: "host1", State: true}}, s.Next())
	assert.Empty(t, s.Next())

	s.Register("host2")
	assert.Equal(t, []scheduler.Action{
		{Host: "host1", State: false},
		{Host: "host2", State: true},
	}, s.Next())
	assert.Equal(t, []scheduler.Action{
		{Host: "host1", State: true},
		{Host: "host2", State: false},
	}, s.Next())

	s.Register("host3")
	assert.Equal(t, []scheduler.Action{
		{Host: "host1", State: false},
		{Host: "host2", State: true},
	}, s.Next())
	assert.Equal(t, []scheduler.Action{
		{Host: "host2", State: false},
		{Host: "host3", State: true},
	}, s.Next())
}

func TestScheduler_Next_StatusChanges(t *testing.T) {
	s, err := scheduler.New(configuration.SchedulerConfiguration{Mode: "linear"})
	require.NoError(t, err)

	s.Register("host1")
	_ = s.Next() // host1
	_ = s.Next() // host1

	s.Register("host2")
	_ = s.Next() // host2
	_ = s.Next() // host1

	// host2 down
	hostDown(s, "host2")

	// new host
	s.Register("host3")

	assert.Equal(t, []scheduler.Action{
		{Host: "host1", State: false},
		{Host: "host3", State: true},
	}, s.Next())
	assert.Equal(t, []scheduler.Action{
		{Host: "host1", State: true},
		{Host: "host3", State: false},
	}, s.Next())

	// host2 back up
	s.UpdateStatus("host2", true)

	assert.Equal(t, []scheduler.Action{
		{Host: "host1", State: false},
		{Host: "host2", State: true},
	}, s.Next())
	assert.Equal(t, []scheduler.Action{
		{Host: "host2", State: false},
		{Host: "host3", State: true},
	}, s.Next())

	// hosts 2&3 down
	hostDown(s, "host2", "host3")

	assert.Equal(t, []scheduler.Action{
		{Host: "host1", State: true},
	}, s.Next())

	// all hosts down
	hostDown(s, "host1")

	assert.Empty(t, s.Next())
}

func hostDown(s *scheduler.Scheduler, hosts ...string) {
	for _, host := range hosts {
		for i := 0; i < 5; i++ {
			s.UpdateStatus(host, false)
		}
	}
}
