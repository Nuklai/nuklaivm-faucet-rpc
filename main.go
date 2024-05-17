package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/hypersdk/crypto/ed25519"
	"github.com/ava-labs/hypersdk/server"
	"github.com/ava-labs/hypersdk/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/nuklai/nuklai-faucet/config"
	"github.com/nuklai/nuklai-faucet/manager"
	frpc "github.com/nuklai/nuklai-faucet/rpc"
)

var (
	allowedOrigins  = []string{"*"}
	allowedHosts    = []string{"*"}
	shutdownTimeout = 30 * time.Second
	httpConfig      = server.HTTPConfig{
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
)

func fatal(l logging.Logger, msg string, fields ...zap.Field) {
	l.Fatal(msg, fields...)
	os.Exit(1)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		utils.Outf("{{red}}Error loading .env file{{/}}: %v\n", err)
		os.Exit(1)
	}

	logFactory := logging.NewFactory(logging.Config{
		DisplayLevel: logging.Info,
	})
	l, err := logFactory.Make("main")
	if err != nil {
		utils.Outf("{{red}}unable to initialize logger{{/}}: %v\n", err)
		os.Exit(1)
	}
	log := l

	// Load config from environment variables
	config, err := config.LoadConfigFromEnv()
	if err != nil {
		fatal(log, "cannot load config from environment variables", zap.Error(err))
	}

	// Create private key
	if len(config.PrivateKeyBytes) == 0 {
		priv, err := ed25519.GeneratePrivateKey()
		if err != nil {
			fatal(log, "cannot generate private key", zap.Error(err))
		}
		config.PrivateKeyBytes = priv[:]
		fatal(log, "private key should be set in .env file after generation")
	}

	// Create server
	listenAddress := net.JoinHostPort(config.HTTPHost, fmt.Sprintf("%d", config.HTTPPort))
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		fatal(log, "cannot create listener", zap.Error(err))
	}
	srv, err := server.New("", log, listener, httpConfig, allowedOrigins, allowedHosts, shutdownTimeout)
	if err != nil {
		fatal(log, "cannot create server", zap.Error(err))
	}

	// Start manager with context handling
	manager, err := manager.New(log, config)
	if err != nil {
		fatal(log, "cannot create manager", zap.Error(err))
	}
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := manager.Run(ctx); err != nil {
			log.Error("manager error", zap.Error(err))
		}
	}()

	// Add faucet handler
	faucetServer := frpc.NewJSONRPCServer(manager)
	handler, err := server.NewHandler(faucetServer, "faucet")
	if err != nil {
		fatal(log, "cannot create handler", zap.Error(err))
	}
	if err := srv.AddRoute(handler, "faucet", ""); err != nil {
		fatal(log, "cannot add faucet route", zap.Error(err))
	}

	// Start server
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Info("triggering server shutdown", zap.Any("signal", sig))
		cancel() // this will signal the manager's run function to stop
		_ = srv.Shutdown()
	}()
	log.Info("server exited", zap.Error(srv.Dispatch()))
}
