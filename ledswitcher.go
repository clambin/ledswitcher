package main

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/internal/cmd"
	"os"
	"os/signal"
	"syscall"
)

var version = "change-me"

func main() {
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()
	if err := cmd.Main(ctx, version); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to start: %s\n", err.Error())
		os.Exit(1)
	}
}
