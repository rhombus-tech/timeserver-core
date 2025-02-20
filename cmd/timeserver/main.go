package main

import (
	"context"
	"crypto/ed25519"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rhombus-tech/timeserver-core/core/config"
	"github.com/rhombus-tech/timeserver-core/core/server"
	"github.com/sirupsen/logrus"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	// Generate or load private key
	var privKey ed25519.PrivateKey
	if cfg.Security.KeyPath != "" {
		keyData, err := os.ReadFile(cfg.Security.KeyPath)
		if err != nil {
			logrus.WithError(err).WithField("keyfile", cfg.Security.KeyPath).Fatal("Failed to read private key")
		}
		privKey = ed25519.PrivateKey(keyData)
	} else {
		_, privKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to generate private key")
		}
	}

	// Create server
	srv, err := server.NewServer(cfg.Server.ID, cfg.Regions[0].ID, privKey, cfg.Network.BootstrapPeers)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create server")
	}

	// Create context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	if err := srv.Start(ctx); err != nil {
		logrus.WithError(err).Fatal("Failed to start server")
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Stop server gracefully
	if err := srv.Stop(); err != nil {
		logrus.WithError(err).Error("Error stopping server")
	}
}
