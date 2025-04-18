package scheduler

import (
	"github.com/clambin/ledswitcher/internal/client/scheduler/schedule"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/registry"
	"log/slog"
)

// Scheduler determines the LED state for each active host(s)
type Scheduler struct {
	schedule schedule.Schedule
	registry *registry.Registry
}

// New creates a Scheduler based on the provided pattern name
func New(cfg configuration.SchedulerConfiguration, registry *registry.Registry) (*Scheduler, error) {
	s, err := schedule.New(cfg.Mode)
	scheduler := Scheduler{
		schedule: s,
		registry: registry,
	}
	return &scheduler, err
}

// Next determines the required actions for the next state
func (s *Scheduler) Next() Actions {
	// only consider the active hosts
	hosts := s.registry.Hosts()
	count := len(hosts)
	if count == 0 {
		return nil
	}

	// get the next state and, for each host that is not in the desired state, create an action
	actions := make(Actions, 0, count)
	for index, state := range s.schedule.Next(count) {
		host := hosts[index]
		actions = append(actions, Action{
			Host:  host,
			State: state,
		})
	}
	return actions
}

// Action represents a state change for a host
type Action struct {
	Host  *registry.Host
	State bool
}

var _ slog.LogValuer = Actions{}

type Actions []Action

func (a Actions) LogValue() slog.Value {
	var output string
	for _, action := range a {
		if action.State {
			output += "1"
		} else {
			output += "0"
		}
	}
	return slog.StringValue(output)
}
