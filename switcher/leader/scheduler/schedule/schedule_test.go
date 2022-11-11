package schedule_test

import (
	"github.com/clambin/ledswitcher/switcher/leader/scheduler/schedule"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew(t *testing.T) {
	testcases := []struct {
		name string
		pass bool
	}{
		{name: "linear", pass: true},
		{name: "alternating", pass: true},
		{name: "random", pass: true},
		{name: "binary", pass: true},
		{name: "", pass: false},
		{name: "invalid", pass: false},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := schedule.New(tt.name)
			if tt.pass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
