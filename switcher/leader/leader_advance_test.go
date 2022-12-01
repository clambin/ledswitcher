package leader

import (
	"errors"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher/caller/mocks"
	"github.com/clambin/ledswitcher/switcher/leader/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLeader_Advance(t *testing.T) {
	c := mocks.NewCaller(t)
	l, _ := New(configuration.LeaderConfiguration{
		Scheduler: configuration.SchedulerConfiguration{Mode: "linear"},
	}, c)

	type action struct {
		a   scheduler.Action
		err error
	}

	type testCase struct {
		name   string
		action []action
	}

	testCases := []testCase{
		{
			name: "no actions",
		},
		{
			name: "all pass",
			action: []action{
				{a: scheduler.Action{Host: "http://foo:1234", State: false}, err: nil},
				{a: scheduler.Action{Host: "http://bar:1234", State: true}, err: nil},
			},
		},
		{
			name: "one error",
			action: []action{
				{a: scheduler.Action{Host: "http://foo:1234", State: false}, err: nil},
				{a: scheduler.Action{Host: "http://bar:1234", State: true}, err: errors.New("fail")},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var next []scheduler.Action
			for _, a := range tt.action {
				switch a.a.State {
				case true:
					c.On("SetLEDOn", a.a.Host).Return(a.err).Once()
				case false:
					c.On("SetLEDOff", a.a.Host).Return(a.err).Once()
				}
				next = append(next, a.a)
			}
			l.advance(next)
		})
	}

	hosts := l.scheduler.GetHosts()
	require.Len(t, hosts, 2)
	for _, host := range hosts {
		assert.True(t, host.IsAlive(), host.Name)
	}
}
