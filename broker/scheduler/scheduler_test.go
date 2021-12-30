package scheduler_test

import (
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNew(t *testing.T) {
	for _, mode := range scheduler.Modes {
		_, err := scheduler.New(mode)
		assert.NoError(t, err, mode)
	}
	_, err := scheduler.New("invalid")
	assert.Error(t, err)
}

func TestScheduler_Next(t *testing.T) {
	s, err := scheduler.New("linear")
	require.NoError(t, err)

	assert.Zero(t, "", s.Next())

	s.Register("host1")
	assert.Equal(t, "host1", s.Next())
	assert.Equal(t, "host1", s.Next())

	s.Register("host2")
	assert.Equal(t, "host2", s.Next())
	assert.Equal(t, "host1", s.Next())

	s.Register("host3")
	assert.Equal(t, "host2", s.Next())
	assert.Equal(t, "host3", s.Next())
	assert.Equal(t, "host1", s.Next())
}

func TestScheduler_UpdateStatus(t *testing.T) {
	s, err := scheduler.New("linear")
	require.NoError(t, err)

	s.Register("host1")
	assert.Equal(t, "host1", s.Next())
	assert.Equal(t, "host1", s.Next())

	s.Register("host2")
	assert.Equal(t, "host2", s.Next())
	assert.Equal(t, "host1", s.Next())

	for i := 0; i < 5; i++ {
		s.UpdateStatus("host2", false)
	}

	s.Register("host3")
	assert.Equal(t, "host3", s.Next())
	assert.Equal(t, "host1", s.Next())

	s.UpdateStatus("host2", true)
	assert.Equal(t, "host2", s.Next())
	assert.Equal(t, "host3", s.Next())
	assert.Equal(t, "host1", s.Next())

	for _, h := range []string{"host2", "host3"} {
		for i := 0; i < 5; i++ {
			s.UpdateStatus(h, false)
		}
	}

	assert.Equal(t, "host1", s.Next())

	for _, h := range []string{"host2", "host3"} {
		s.UpdateStatus(h, true)
	}

	assert.Equal(t, "host2", s.Next())
	assert.Equal(t, "host3", s.Next())

	for _, h := range []string{"host1", "host2", "host3"} {
		for i := 0; i < 5; i++ {
			s.UpdateStatus(h, false)
		}
	}

	assert.Equal(t, "", s.Next())
}

func TestScheduler_UpdateStatus_NoRegister(t *testing.T) {
	// normally will never happen: broker will never call a host that didn't register first, so UpdateStatus for
	// an unregistered host is impossible. Just implementing this for code coverage
	s, err := scheduler.New("linear")
	require.NoError(t, err)

	s.UpdateStatus("host1", true)
	assert.Equal(t, "host1", s.Next())
}

func TestScheduler_GetHosts(t *testing.T) {
	s, err := scheduler.New("linear")
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

func TestScheduler_GetCurrentHost(t *testing.T) {
	s, err := scheduler.New("linear")
	require.NoError(t, err)

	assert.Empty(t, s.GetCurrentHost())

	s.Register("host1")
	assert.Equal(t, "host1", s.Next())
	assert.Equal(t, "host1", s.GetCurrentHost())

	s.Register("host2")
	assert.Equal(t, "host1", s.GetCurrentHost())
	assert.Equal(t, "host2", s.Next())
	assert.Equal(t, "host2", s.GetCurrentHost())
	assert.Equal(t, "host1", s.Next())
	assert.Equal(t, "host1", s.GetCurrentHost())
}

func TestScheduler_Register(t *testing.T) {
	s, err := scheduler.New("linear")
	require.NoError(t, err)

	s.Register("host1")
	s.Register("host2")
	s.Register("host1")

	health := s.GetHosts()
	assert.Len(t, health, 2)
}
