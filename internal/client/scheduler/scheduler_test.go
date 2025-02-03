package scheduler_test

import (
	"github.com/clambin/ledswitcher/internal/client/scheduler"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
)

func TestScheduler_Next(t *testing.T) {
	r := registry.Registry{Logger: slog.Default()}

	s, err := scheduler.New(configuration.SchedulerConfiguration{Mode: "binary"}, &r)
	require.NoError(t, err)

	assert.Zero(t, "", s.Next())

	r.Register("host1")
	assert.Equal(t, "1", s.Next().LogValue().String())
	assert.Equal(t, "0", s.Next().LogValue().String())

	r.Register("host2")
	assert.Equal(t, "01", s.Next().LogValue().String())
	assert.Equal(t, "10", s.Next().LogValue().String())

	r.Register("host3")
	assert.Equal(t, "011", s.Next().LogValue().String())
	assert.Equal(t, "100", s.Next().LogValue().String())
}
