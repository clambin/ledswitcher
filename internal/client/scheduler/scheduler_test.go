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

	s, err := scheduler.New(configuration.SchedulerConfiguration{Mode: "linear"}, &r)
	require.NoError(t, err)

	assert.Zero(t, "", s.Next())

	r.Register("host1")
	assert.Equal(t, scheduler.Actions{{Host: "host1", State: true}}, s.Next())
	assert.Equal(t, scheduler.Actions{
		{Host: "host1", State: true},
	}, s.Next())

	r.Register("host2")
	assert.Equal(t, scheduler.Actions{
		{Host: "host1", State: false},
		{Host: "host2", State: true},
	}, s.Next())
	assert.Equal(t, scheduler.Actions{
		{Host: "host1", State: true},
		{Host: "host2", State: false},
	}, s.Next())

	r.Register("host3")
	assert.Equal(t, scheduler.Actions{
		{Host: "host1", State: false},
		{Host: "host2", State: true},
		{Host: "host3", State: false},
	}, s.Next())
	assert.Equal(t, scheduler.Actions{
		{Host: "host1", State: false},
		{Host: "host2", State: false},
		{Host: "host3", State: true},
	}, s.Next())
}

func TestActions_LogValue(t *testing.T) {
	tests := []struct {
		name string
		a    scheduler.Actions
		want string
	}{
		{
			name: "1",
			a:    scheduler.Actions{{State: true}},
			want: "1",
		},
		{
			name: "10",
			a:    scheduler.Actions{{State: true}, {State: false}},
			want: "10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.LogValue().String())
		})
	}
}
