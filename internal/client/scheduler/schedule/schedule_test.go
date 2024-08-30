package schedule

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew(t *testing.T) {
	testcases := []struct {
		name string
		want assert.ErrorAssertionFunc
	}{
		{name: "linear", want: assert.NoError},
		{name: "alternating", want: assert.NoError},
		{name: "random", want: assert.NoError},
		{name: "binary", want: assert.NoError},
		{name: "reverse-binary", want: assert.NoError},
		{name: "", want: assert.Error},
		{name: "invalid", want: assert.Error},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.name)
			tt.want(t, err)
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
