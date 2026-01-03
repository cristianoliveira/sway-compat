package main

import (
	"fmt"
	"os"

	"github.com/cristianoliveira/sway-compat/cmd"
	"github.com/cristianoliveira/sway-compat/internal/logger"
)

func main() {
	// Initialize logger
	log, err := logger.NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	logger.SetDefaultLogger(log)

	log.LogDebug("sway-compat starting", "config", log.GetConfig())

	cmd.Execute()
}
