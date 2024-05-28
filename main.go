package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
	httpConfig = server.HTTPConfig{
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

// HealthHandler responds with a simple health check status
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	err := godotenv.Overload() // Overload the environment variables with those from the .env file
	if err != nil {
		utils.Outf("{{red}}Error loading .env file{{/}}: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Loaded environment variables from .env file")

	logFactory := logging.NewFactory(logging.Config{
		DisplayLevel: logging.Info,
	})
	l, err := logFactory.Make("main")
	if err != nil {
		utils.Outf("{{red}}unable to initialize logger{{/}}: %v\n", err)
		os.Exit(1)
	}
	log := l
	log.Info("Logger initialized")

	// Load config from environment variables
	config, err := config.LoadConfigFromEnv()
	if err != nil {
		fatal(log, "cannot load config from environment variables", zap.Error(err))
	}
	log.Info("Config loaded from environment variables")

	// Create private key
	if len(config.PrivateKeyBytes) == 0 {
		priv, err := ed25519.GeneratePrivateKey()
		if err != nil {
			fatal(log, "cannot generate private key", zap.Error(err))
		}
		config.PrivateKeyBytes = priv[:]
		fatal(log, "private key should be set in .env file after generation")
	}
	log.Info("Private key generated")

	// Create server
	listenAddress := net.JoinHostPort(config.HTTPHost, fmt.Sprintf("%d", config.HTTPPort))
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		fatal(log, "cannot create listener", zap.Error(err))
	}
	log.Info("Listener created", zap.String("address", listenAddress))

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:         listenAddress,
		Handler:      mux,
		ReadTimeout:  httpConfig.ReadTimeout,
		WriteTimeout: httpConfig.WriteTimeout,
		IdleTimeout:  httpConfig.IdleTimeout,
	}

	// Add health check handler
	mux.HandleFunc("/health", HealthHandler)
	log.Info("Health handler added")

	// Start manager with context handling
	manager, err := manager.New(log, config)
	if err != nil {
		fatal(log, "cannot create manager", zap.Error(err))
	}
	log.Info("Manager created")
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		log.Info("Starting manager")
		if err := manager.Run(ctx); err != nil {
			log.Error("Manager error", zap.Error(err))
		}
	}()

	// Add faucet handler
	faucetServer := frpc.NewJSONRPCServer(manager)
	handler, err := server.NewHandler(faucetServer, "faucet")
	if err != nil {
		fatal(log, "cannot create handler", zap.Error(err))
	}
	mux.Handle("/", handler)
	log.Info("Faucet handler added")

	// Start server
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Info("Triggering server shutdown", zap.Any("signal", sig))
		cancel() // this will signal the manager's run function to stop
		_ = srv.Shutdown(ctx)
	}()
	log.Info("Server starting")

	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed", zap.Error(err))
	}
	log.Info("Server exited")
}
