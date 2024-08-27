package schedule

import (
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
			_, err := New(tt.name)
			if tt.pass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func Test_intToBits(t *testing.T) {
	tests := []struct {
		val  int
		len  int
		want []bool
	}{
		{val: 1, len: 1, want: []bool{true}},
		{val: 1, len: 2, want: []bool{false, true}},
		{val: 1, len: 4, want: []bool{false, false, false, true}},
		{val: 2, len: 4, want: []bool{false, false, true, false}},
		{val: 16, len: 4, want: []bool{false, false, false, false}},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, intToBits(tt.val, tt.len))
	}
}
