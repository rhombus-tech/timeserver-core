package main

import (
	"context"
	"crypto/ed25519"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rhombus-tech/timeserver-core/core/config"
	"github.com/rhombus-tech/timeserver-core/core/p2p"
	"github.com/rhombus-tech/timeserver-core/core/server"
)

func main() {
	configFile := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Generate or load keys
	var privKey ed25519.PrivateKey
	if cfg.Security.KeyPath != "" {
		keyData, err := os.ReadFile(cfg.Security.KeyPath)
		if err != nil {
			log.Fatalf("Failed to read key file: %v", err)
		}
		privKey = ed25519.PrivateKey(keyData)
	} else {
		_, privKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			log.Fatalf("Failed to generate key: %v", err)
		}
	}

	// Convert ed25519 key to libp2p key
	p2pKey, err := crypto.UnmarshalEd25519PrivateKey(privKey)
	if err != nil {
		log.Fatalf("Failed to convert key: %v", err)
	}

	// Create server
	nodeID := fmt.Sprintf("%x", privKey.Public().(ed25519.PublicKey))
	srv, err := server.NewServer(nodeID, privKey, cfg.Server.ValidatorIDs)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create P2P network
	network, err := p2p.NewP2PNetwork(nodeID, p2pKey, []string{cfg.Server.ListenAddr})
	if err != nil {
		log.Fatalf("Failed to create P2P network: %v", err)
	}

	// Start server
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Start P2P network
	if err := network.Start(ctx); err != nil {
		log.Fatalf("Failed to start P2P network: %v", err)
	}

	// Connect to bootstrap nodes
	for _, addr := range cfg.Network.BootstrapNodes {
		if err := network.Connect(ctx, addr); err != nil {
			log.Printf("Failed to connect to peer %s: %v", addr, err)
		}
	}

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")

	// Stop server and network
	if err := srv.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	if err := network.Stop(); err != nil {
		log.Printf("Error stopping network: %v", err)
	}
}
