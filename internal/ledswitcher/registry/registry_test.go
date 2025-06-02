package registry

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Register(t *testing.T) {
	r := New("localhost", slog.New(slog.DiscardHandler))
	r.Register("localhost", "http://localhost:8080")
	r.Hosts()[0].SetLEDState(true)
	for range maxFailures {
		r.UpdateHostState("localhost", true, false)
	}

	assert.Empty(t, r.Hosts())
	r.Register("localhost", "http://localhost:8080")
	require.Len(t, r.Hosts(), 1)
	assert.Equal(t, "localhost", r.Hosts()[0].Name)
	// re-registering should not affect the recorded LED state
	assert.True(t, r.Hosts()[0].LEDState())
}

func TestRegistry_HostState(t *testing.T) {
	r := New("localhost", slog.New(slog.DiscardHandler))
	r.Register("foo", "http://localhost:8080")
	up, found := r.HostState("foo")
	require.True(t, found)
	assert.False(t, up)

	r.Hosts()[0].SetLEDState(true)
	up, found = r.HostState("foo")
	require.True(t, found)
	assert.True(t, up)

	_, found = r.HostState("bar")
	assert.False(t, found)
}

func TestRegistry_Hosts(t *testing.T) {
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
			r := New("localhost", slog.New(slog.DiscardHandler))
			for _, host := range tt.hosts {
				r.Register(host, "http://"+host+":8080")
			}
			registeredHosts := make([]string, 0, len(tt.hosts))
			for _, host := range r.Hosts() {
				registeredHosts = append(registeredHosts, host.Name)
			}
			assert.Equal(t, tt.want, registeredHosts)
		})
	}
}

func TestRegistry_UpdateHostState(t *testing.T) {
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
			r := New("localhost", slog.New(slog.DiscardHandler))
			r.Register(tt.args.host, "http://"+tt.args.host+":8080")
			r.UpdateHostState(tt.args.host, tt.args.ledState, tt.args.reachable)
			hosts := r.Hosts()
			require.Len(t, hosts, 1)
			tt.wantLedState(t, hosts[0].LEDState())
			tt.wantReachable(t, hosts[0].IsAlive())
		})
	}
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
			r := New("localhost", slog.New(slog.DiscardHandler))
			r.hosts = map[string]*Host{h.Name: &h}
			r.Cleanup()
			assert.Len(t, r.Hosts(), tt.want)
		})
	}
}

func TestRegistry_Leader(t *testing.T) {
	r := New("localhost", slog.New(slog.DiscardHandler))
	for _, leader := range []string{"localhost", "other"} {
		r.SetLeader(leader)
		assert.Equal(t, leader == r.hostname, r.IsLeading())
	}
}

func TestRegistry_Collect(t *testing.T) {
	r := New("localhost", slog.New(slog.DiscardHandler))
	r.Register("localhost", "http://localhost:8080")

	assert.NoError(t, testutil.CollectAndCompare(r, bytes.NewBufferString(`
# HELP ledswitcher_registry_node_count Number of registered nodes
# TYPE ledswitcher_registry_node_count gauge
ledswitcher_registry_node_count 1
`)))

	r.Hosts()[0].failures.Store(10)
	assert.NoError(t, testutil.CollectAndCompare(r, bytes.NewBufferString(`
# HELP ledswitcher_registry_node_count Number of registered nodes
# TYPE ledswitcher_registry_node_count gauge
ledswitcher_registry_node_count 0
`)))
}
