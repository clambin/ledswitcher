package leader

import (
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher/leader/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestLeader_Advance(t *testing.T) {
	l, _ := New(configuration.LeaderConfiguration{
		Scheduler: configuration.SchedulerConfiguration{Mode: "linear"},
	})

	var eps endpoints
	var servers []*httptest.Server
	for i := 0; i < 4; i++ {
		var e endpoint
		eps = append(eps, &e)
		servers = append(servers, httptest.NewServer(http.HandlerFunc(e.handle)))
	}

	tests := []struct {
		name   string
		input  []bool
		states []bool
	}{
		{
			name:   "all on",
			input:  []bool{true, true, true, true},
			states: []bool{true, true, true, true},
		},
		{
			name:   "all off",
			input:  []bool{false, false, false, false},
			states: []bool{false, false, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var next []scheduler.Action
			for i, state := range tt.input {
				next = append(next, scheduler.Action{
					Host:  servers[i].URL,
					State: state,
				})
			}
			l.advance(next)

			assert.Equal(t, tt.states, eps.getState())
		})
	}

	hosts := l.scheduler.GetHosts()
	require.Len(t, hosts, len(eps))
	for _, host := range hosts {
		assert.True(t, host.IsAlive(), host.Name)
	}

	for _, s := range servers {
		s.Client()
	}
}

type endpoints []*endpoint

func (e endpoints) getState() []bool {
	var states []bool
	for _, ep := range e {
		states = append(states, ep.getState())
	}
	return states
}

type endpoint struct {
	state bool
	lock  sync.RWMutex
}

func (e *endpoint) handle(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/led" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	switch req.Method {
	case http.MethodPost:
		e.state = true
	case http.MethodDelete:
		e.state = false
	default:
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}
}

func (e *endpoint) getState() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.state
}