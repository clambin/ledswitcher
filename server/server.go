package server

import (
	"context"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/clambin/ledswitcher/driver"
	"github.com/clambin/ledswitcher/endpoint"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Server implements the ledswitcher logic.  It contains:
// 1. a Broker that will determine the next LED to switch on, based on the registered endpoints
// 2. a Driver that will take the next Endpoint from the Broker and issue an HTTP request to that endpoint
// 3. an Endpoint that will listen for incoming requests the leading driver and switch the LED on/off. In parallel,
// it will also attempt to registered with the leader's broker.
//
// Each ledswitcher contains all three components. The main programme (ledswitcher.go) implements the logic to determine
// which instance is the leader, which is either fixed by configuration, or through k8s leader election.
type Server struct {
	Broker   broker.Broker
	Driver   *driver.Driver
	Endpoint *endpoint.Endpoint
	wg       sync.WaitGroup
}

// New creates a new Server
func New(hostname string, port int, ledPath string, rotation time.Duration, scheduler *scheduler.Scheduler, leader string) (server *Server) {
	b := broker.New(rotation, scheduler)
	server = &Server{
		Broker:   b,
		Driver:   driver.New(b),
		Endpoint: endpoint.New(hostname, port, ledPath, b),
	}
	if leader != "" {
		server.Endpoint.SetLeader(leader)
	}
	return
}

// Start starts a Server
func (s *Server) Start(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		s.Broker.Run(ctx)
		s.wg.Done()
	}()

	s.wg.Add(1)
	go func() {
		s.Driver.Run(ctx)
		s.wg.Done()
	}()

	s.wg.Add(1)
	go func() {
		if err := s.Endpoint.Run(ctx); err != nil {
			log.WithError(err).Fatal("failed to start endpoint")
		}
		s.wg.Done()
	}()
}

// Wait waits for a server to shut down
func (s *Server) Wait() {
	s.wg.Wait()
}
