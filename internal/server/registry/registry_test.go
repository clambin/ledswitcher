package registry

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestRegistry_Leading(t *testing.T) {
	var r Registry
	for _, leading := range []bool{true, false} {
		r.Leading(leading)
		assert.Equal(t, leading, r.IsLeading())
	}
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
			for _, host := range r.GetHosts() {
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
		r.UpdateStatus("localhost", false, false)
	}
	assert.Empty(t, r.GetHosts())
	r.Register("localhost")
	require.Len(t, r.GetHosts(), 1)
	assert.Equal(t, "localhost", r.GetHosts()[0].Name)
}

func TestRegistry_UpdateStatus(t *testing.T) {
	type args struct {
		host      string
		state     bool
		reachable bool
	}
	tests := []struct {
		name        string
		args        args
		wantState   bool
		wantIsAlive bool
	}{
		{
			name: "on",
			args: args{
				host:      "localhost",
				state:     true,
				reachable: true,
			},
			wantState:   true,
			wantIsAlive: true,
		},
		{
			name: "off",
			args: args{
				host:      "localhost",
				state:     false,
				reachable: true,
			},
			wantState:   false,
			wantIsAlive: true,
		},
		{
			name: "down",
			args: args{
				host:      "localhost",
				reachable: false,
			},
			wantIsAlive: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &Registry{Logger: slog.Default()}
			r.Register(tt.args.host)
			r.UpdateStatus(tt.args.host, tt.args.state, tt.args.reachable)
			hosts := r.GetHosts()
			require.Len(t, hosts, 1)
			assert.Equal(t, tt.wantState, hosts[0].State)
			assert.Equal(t, tt.wantIsAlive, hosts[0].IsAlive())
		})
	}
}

func TestRegistry_Dead(t *testing.T) {
	r := &Registry{Logger: slog.Default()}
	r.Register("localhost")
	r.UpdateStatus("localhost", false, true)
	assert.True(t, r.GetHosts()[0].IsAlive())
	for range maxFailures {
		r.UpdateStatus("localhost", false, false)
	}
	assert.Empty(t, r.GetHosts())
}

func TestRegistry_Cleanup(t *testing.T) {
	var tests = []struct {
		name string
		host Host
		want int
	}{
		{
			name: "up",
			host: Host{Name: "host1", Failures: 0, LastUpdated: time.Now()},
			want: 1,
		},
		{
			name: "down, not yet exceeded threshold",
			host: Host{Name: "host1", Failures: maxFailures - 1},
			want: 1,
		},
		{
			name: "down, exceeded threshold",
			host: Host{Name: "host1", Failures: maxFailures},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &Registry{hosts: []*Host{&tt.host}}
			r.Cleanup()
			assert.Len(t, r.GetHosts(), tt.want)
		})
	}
}
