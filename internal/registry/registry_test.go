package registry

import (
	"bytes"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
)

func TestRegistry_Leading(t *testing.T) {
	var r Registry
	for _, leading := range []bool{true, false} {
		r.Leading(leading)
		assert.Equal(t, leading, r.IsLeading())
	}
}

func TestRegistry_HostState(t *testing.T) {
	r := Registry{Logger: slog.Default()}
	r.Register("foo")

	up, found := r.HostState("foo")
	assert.True(t, found)
	assert.False(t, up)

	r.Hosts()[0].SetLEDState(true)
	up, found = r.HostState("foo")
	assert.True(t, found)
	assert.True(t, up)

	_, found = r.HostState("bar")
	assert.False(t, found)
}

func TestRegistry_GetHosts(t *testing.T) {
	tests := []struct {
		name  string
		hosts []string
		want  []string
	}{
		{
			name:  "empty",
			hosts: []string{},
			want:  []string{},
		},
		{
			name:  "single host",
			hosts: []string{"localhost"},
			want:  []string{"localhost"},
		},
		{
			name:  "multiple hosts appear in order",
			hosts: []string{"host1", "host2", "host3"},
			want:  []string{"host1", "host2", "host3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &Registry{Logger: slog.Default()}
			for _, host := range tt.hosts {
				r.Register(host)
			}
			registeredHosts := make([]string, 0, len(tt.hosts))
			for _, host := range r.Hosts() {
				registeredHosts = append(registeredHosts, host.Name)
			}
			assert.Equal(t, tt.want, registeredHosts)
		})
	}
}

func TestRegistry_ReRegister(t *testing.T) {
	r := &Registry{Logger: slog.Default()}
	r.Register("localhost")
	for range maxFailures {
		r.UpdateHostState("localhost", false, false)
	}
	assert.Empty(t, r.Hosts())
	r.Register("localhost")
	require.Len(t, r.Hosts(), 1)
	assert.Equal(t, "localhost", r.Hosts()[0].Name)
}

func TestRegistry_UpdateStatus(t *testing.T) {
	type args struct {
		host      string
		ledState  bool
		reachable bool
	}
	tests := []struct {
		name          string
		args          args
		wantLedState  assert.BoolAssertionFunc
		wantReachable assert.BoolAssertionFunc
	}{
		{
			name: "on",
			args: args{
				host:      "localhost",
				ledState:  true,
				reachable: true,
			},
			wantLedState:  assert.True,
			wantReachable: assert.True,
		},
		{
			name: "off",
			args: args{
				host:      "localhost",
				ledState:  false,
				reachable: true,
			},
			wantLedState:  assert.False,
			wantReachable: assert.True,
		},
		{
			name: "down",
			args: args{
				host:      "localhost",
				reachable: false,
			},
			wantLedState:  assert.False,
			wantReachable: assert.True,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &Registry{Logger: slog.Default()}
			r.Register(tt.args.host)
			r.UpdateHostState(tt.args.host, tt.args.ledState, tt.args.reachable)
			hosts := r.Hosts()
			require.Len(t, hosts, 1)
			tt.wantLedState(t, hosts[0].LEDState())
			tt.wantReachable(t, hosts[0].IsAlive())
		})
	}
}

func TestRegistry_Dead(t *testing.T) {
	r := &Registry{Logger: slog.Default()}
	r.Register("localhost")
	r.UpdateHostState("localhost", false, true)
	assert.True(t, r.Hosts()[0].IsAlive())
	for range maxFailures {
		r.UpdateHostState("localhost", false, false)
	}
	assert.Empty(t, r.Hosts())
}

func TestRegistry_Cleanup(t *testing.T) {
	var tests = []struct {
		name     string
		failures int32
		want     int
	}{
		{
			name:     "up",
			failures: 0,
			want:     1,
		},
		{
			name:     "down, not yet exceeded threshold",
			failures: maxFailures - 1,
			want:     1,
		},
		{
			name:     "down, exceeded threshold",
			failures: maxFailures,
			want:     0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := Host{Name: "host1"}
			h.failures.Store(tt.failures)
			r := &Registry{hosts: map[string]*Host{h.Name: &h}, Logger: slog.Default()}
			r.Cleanup()
			assert.Len(t, r.Hosts(), tt.want)
		})
	}
}

func TestRegistry_Collect(t *testing.T) {
	r := Registry{Logger: slog.Default()}
	r.Register("localhost")

	assert.NoError(t, testutil.CollectAndCompare(&r, bytes.NewBufferString(`
# HELP ledswitcher_registry_node_count Number of registered nodes
# TYPE ledswitcher_registry_node_count gauge
ledswitcher_registry_node_count 1
`)))

	r.Hosts()[0].failures.Store(10)
	assert.NoError(t, testutil.CollectAndCompare(&r, bytes.NewBufferString(`
# HELP ledswitcher_registry_node_count Number of registered nodes
# TYPE ledswitcher_registry_node_count gauge
ledswitcher_registry_node_count 0
`)))
}
