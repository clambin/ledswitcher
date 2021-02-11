package client_test

import (
	"github.com/clambin/ledswitcher/internal/client"
	"github.com/clambin/ledswitcher/internal/server"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"testing"
	"time"
)

func TestIsActive(t *testing.T) {
	go func() {
		s := server.Server{
			Rotation: 250 * time.Millisecond,
			Expiry:   5 * time.Minute,
			Port:     8081,
		}
		s.Run()
	}()

	c := client.Client{
		Hostname:  "client1",
		MasterURL: "http://localhost:8081/",
	}

	active, err := c.IsActive()
	assert.Nil(t, err)
	assert.False(t, active)
	assert.Eventually(t, func() bool {
		active, err := c.IsActive()
		return active && err == nil
	}, 500*time.Millisecond, 100*time.Millisecond)
}

func TestRun(t *testing.T) {
	go func() {
		s := server.Server{
			Rotation: 250 * time.Millisecond,
			Expiry:   5 * time.Minute,
			Port:     8082,
		}
		s.Run()
	}()

	dir := initTestFiles()

	c := client.Client{
		Hostname:  "client1",
		MasterURL: "http://localhost:8082/",
		LEDPath:   dir,
	}

	err := c.Run()
	assert.Nil(t, err)
	assert.Equal(t, 0, getFileValue(dir))

	assert.Eventually(t, func() bool {
		_ = c.Run()
		return getFileValue(dir) == 255
	}, 500*time.Millisecond, 100*time.Millisecond)

	_ = os.RemoveAll(dir)
}

func initTestFiles() string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(dir, "brightness"), []byte("255"), 0640)
	if err != nil {
		log.Fatal(err)
	}

	return dir
}

func getFileValue(dirname string) int {
	value := -1
	data, err := ioutil.ReadFile(path.Join(dirname, "brightness"))
	if err == nil {
		if value, err = strconv.Atoi(string(data)); err != nil {
			value = -1
		}
	}

	return value
}
