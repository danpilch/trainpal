package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"

	"github.com/danpilch/trainpal/internal/api/rtt"
	"github.com/danpilch/trainpal/internal/api/tfl"
	"github.com/danpilch/trainpal/internal/config"
	"github.com/danpilch/trainpal/internal/monitor"
	"github.com/danpilch/trainpal/internal/notify"
	"github.com/danpilch/trainpal/internal/scheduler"
)

var CLI struct {
	Config string `help:"Path to config file" default:"config.yaml" type:"path"`
}

func main() {
	kong.Parse(&CLI)

	// Setup structured logging with logfmt
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	// Load configuration
	cfg, err := config.Load(CLI.Config)
	if err != nil {
		logger.WithField("error", err).Fatal("failed to load config")
	}

	// Get credentials from environment
	pushoverToken := os.Getenv("PUSHOVER_TOKEN")
	pushoverUser := os.Getenv("PUSHOVER_USER")
	if pushoverToken == "" || pushoverUser == "" {
		logger.Fatal("PUSHOVER_TOKEN and PUSHOVER_USER environment variables are required")
	}

	rttUsername := os.Getenv("RTT_USERNAME")
	rttPassword := os.Getenv("RTT_PASSWORD")
	if rttUsername == "" || rttPassword == "" {
		logger.Fatal("RTT_USERNAME and RTT_PASSWORD environment variables are required")
	}

	// Initialize clients
	rttClient := rtt.NewClient(rttUsername, rttPassword)
	tflClient := tfl.NewClient()
	notifier := notify.NewNotifier(pushoverToken, pushoverUser, logger)

	// Initialize monitors
	trainMonitor := monitor.NewTrainMonitor(rttClient, notifier, logger)
	tubeMonitor := monitor.NewTubeMonitor(tflClient, notifier, logger)

	// Initialize scheduler
	sched := scheduler.NewScheduler(cfg, trainMonitor, tubeMonitor, logger)

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.WithField("signal", sig).Info("received signal, shutting down")
		cancel()
	}()

	// Start scheduler
	logger.WithFields(logrus.Fields{
		"morning_train": cfg.MorningTrain.From + " -> " + cfg.MorningTrain.To + " @ " + cfg.MorningTrain.Departure,
		"evening_train": cfg.EveningTrain.From + " -> " + cfg.EveningTrain.To + " @ " + cfg.EveningTrain.Departure,
	}).Info("starting trainpal")

	sched.Start(ctx)

	// Wait for context cancellation
	<-ctx.Done()

	// Stop scheduler gracefully
	sched.Stop()
	logger.Info("trainpal stopped")
}
