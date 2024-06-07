package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/hypersdk/crypto/ed25519"
	"github.com/ava-labs/hypersdk/utils"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/nuklai/nuklai-faucet/config"
	"github.com/nuklai/nuklai-faucet/manager"
	frpc "github.com/nuklai/nuklai-faucet/rpc"
)

var (
	httpConfig = &http.Server{
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
	err := godotenv.Overload()
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

	config, err := config.LoadConfigFromEnv()
	if err != nil {
		fatal(log, "cannot load config from environment variables", zap.Error(err))
	}
	log.Info("Config loaded from environment variables")

	if len(config.PrivateKeyBytes) == 0 {
		priv, err := ed25519.GeneratePrivateKey()
		if err != nil {
			fatal(log, "cannot generate private key", zap.Error(err))
		}
		config.PrivateKeyBytes = priv[:]
		fatal(log, "private key should be set in .env file after generation")
	}
	log.Info("Private key generated")

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

	mux.HandleFunc("/health", HealthHandler)
	log.Info("Health handler added")

	var db *sql.DB
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.PostgresHost, config.PostgresPort, config.PostgresUser, config.PostgresPassword, config.PostgresDBName, config.PostgresSSLMode))
		if err != nil {
			log.Warn("Error opening database", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}
		err = db.Ping()
		if err == nil {
			break
		}
		log.Warn("Database not ready, retrying...", zap.Error(err))
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		fatal(log, "could not connect to the database", zap.Error(err))
	}
	log.Info("Database connection established")

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

	faucetServer := frpc.NewJSONRPCServer(manager)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "could not read request body", http.StatusInternalServerError)
				return
			}

			var req frpc.JSONRPCRequest
			err = json.Unmarshal(body, &req)
			if err != nil {
				log.Error("Failed to unmarshal JSON-RPC request", zap.Error(err))
				http.Error(w, "invalid JSON-RPC request", http.StatusBadRequest)
				return
			}

			response := faucetServer.HandleRequest(req)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	})

	log.Info("Faucet handler added")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Info("Triggering server shutdown", zap.Any("signal", sig))
		cancel()
		_ = srv.Shutdown(ctx)
	}()
	log.Info("Server starting")

	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed", zap.Error(err))
	}
	log.Info("Server exited")
}
