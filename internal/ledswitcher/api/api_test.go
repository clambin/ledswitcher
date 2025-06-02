package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistrationRequest_LogValue(t *testing.T) {
	req := RegistrationRequest{
		Name: "localhost",
		URL:  "http://localhost:8080",
	}
	assert.Equal(t, "[name=localhost url=http://localhost:8080]", req.LogValue().String())
}
