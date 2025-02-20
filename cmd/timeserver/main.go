package main

import (
	"context"
	"crypto/ed25519"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rhombus-tech/timeserver-core/core/config"
	"github.com/rhombus-tech/timeserver-core/core/server"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Generate or load private key
	var privKey ed25519.PrivateKey
	if cfg.Security.KeyPath != "" {
		keyData, err := os.ReadFile(cfg.Security.KeyPath)
		if err != nil {
			log.Fatalf("Failed to read private key: %v", err)
		}
		privKey = ed25519.PrivateKey(keyData)
	} else {
		_, privKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			log.Fatalf("Failed to generate private key: %v", err)
		}
	}

	// Create server
	srv, err := server.NewServer(cfg.Server.ID, cfg.Regions[0].ID, privKey, cfg.Network.BootstrapPeers)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Create context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Stop server gracefully
	if err := srv.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}
}
