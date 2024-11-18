// Copyright (C) 2024, Nuklai. All rights reserved.
// See the file LICENSE for licensing terms.

package manager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/hypersdk/api/indexer"
	"github.com/ava-labs/hypersdk/api/jsonrpc"
	"github.com/ava-labs/hypersdk/auth"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"

	fconfig "github.com/nuklai/nuklai-faucet/config"
	"github.com/nuklai/nuklai-faucet/database"
	"github.com/nuklai/nuklaivm/actions"
	"github.com/nuklai/nuklaivm/consts"
	"github.com/nuklai/nuklaivm/storage"
	nutils "github.com/nuklai/nuklaivm/utils"
	"github.com/nuklai/nuklaivm/vm"

	"go.uber.org/zap"
)

type Manager struct {
	log    logging.Logger
	config *fconfig.Config

	hyperSDKRPC     *jsonrpc.JSONRPCClient
	hyperVMRPC      *vm.JSONRPCClient
	hyperIndexerRPC *indexer.Client

	factory chain.AuthFactory

	healthMu   sync.RWMutex
	cancelFunc context.CancelFunc

	db *database.DB
}

func New(logger logging.Logger, config *fconfig.Config, db *sql.DB) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	hyperVMRPC := vm.NewJSONRPCClient(config.NuklaiRPC)
	hyperSDKRPC := jsonrpc.NewJSONRPCClient(config.NuklaiRPC)
	hyperIndexerRPC := indexer.NewClient(config.NuklaiRPC)

	dbInstance, err := database.NewDB(db)
	if err != nil {
		cancel()
		return nil, err
	}
	m := &Manager{log: logger, config: config, hyperSDKRPC: hyperSDKRPC, hyperVMRPC: hyperVMRPC, hyperIndexerRPC: hyperIndexerRPC, factory: auth.NewED25519Factory(config.PrivateKeyEd25519()), cancelFunc: cancel, db: dbInstance}
	bal, err := hyperVMRPC.Balance(ctx, m.config.AddressBech32(), consts.Symbol)
	if err != nil {
		return nil, err
	}
	m.log.Info("faucet initialized",
		zap.String("address", m.config.AddressBech32()),
		zap.String("balance", nutils.FormatBalance(bal, consts.Decimals)),
	)
	return m, nil
}

func (m *Manager) Run(ctx context.Context) error {
	m.log.Info("Manager run started")
	<-ctx.Done()
	m.db.Close()
	m.log.Info("Manager run completed", zap.Error(ctx.Err()))
	return ctx.Err()
}

func (m *Manager) SendFundsRetry(ctx context.Context, destination codec.Address, amount uint64) (ids.ID, uint64, error) {
	var lastErr error

	for retries := 0; retries < 5; retries++ {
		m.log.Info("SendFundsRetry attempt", zap.Int("retry", retries+1))

		// Attempt to send funds
		txID, maxFee, err := m.sendFunds(ctx, destination, amount)
		if err == nil {
			// Wait for transaction to complete
			success, waitErr := m.waitForTransactionWithIndexer(ctx, txID, 55*time.Second)
			if waitErr != nil {
				m.log.Error("Transaction wait failed", zap.Error(waitErr))
				return ids.Empty, 0, waitErr
			}
			if success {
				return txID, maxFee, nil
			}
		}

		// Handle errors and retry logic
		m.log.Error("Error sending funds", zap.Error(err))
		lastErr = err
		time.Sleep(time.Second * time.Duration(retries+1))
	}

	return ids.Empty, 0, fmt.Errorf("failed after retries: %w", lastErr)
}

func (m *Manager) GetFaucetAddress(_ context.Context) (codec.Address, error) {
	return m.config.Address(), nil
}

func (m *Manager) sendFunds(_ctx context.Context, destination codec.Address, amount uint64) (ids.ID, uint64, error) {
	ctx, cancel := context.WithTimeout(_ctx, 30*time.Second)
	defer cancel()

	m.log.Info("Attempting to send funds", zap.String("destination", destination.String()), zap.Uint64("amount", amount))

	// Check user balance
	bal, err := m.hyperVMRPC.Balance(ctx, destination.String(), consts.Symbol)
	if err != nil {
		m.log.Error("Failed to fetch balance", zap.Error(err))
		return ids.Empty, 0, err
	}
	if bal > m.config.BalanceThreshold {
		m.log.Warn("User balance is above threshold", zap.String("balance", nutils.FormatBalance(bal, consts.Decimals)))
		return ids.Empty, 0, errors.New("user balance above threshold")
	}

	// Check faucet balanace
	bal, err = m.hyperVMRPC.Balance(ctx, m.config.AddressBech32(), consts.Symbol)
	if err != nil {
		m.log.Error("Failed to fetch balance", zap.Error(err))
		return ids.Empty, 0, err
	}
	if bal < amount*2 {
		m.log.Warn("Faucet has insufficient funds", zap.String("balance", nutils.FormatBalance(bal, consts.Decimals)))
		return ids.Empty, 0, errors.New("insufficient balance")
	}

	parser, err := m.hyperVMRPC.Parser(ctx)
	if err != nil {
		m.log.Error("Failed to create parser", zap.Error(err))
		return ids.Empty, 0, err
	}
	submitTxFunc, tx, _, err := m.hyperSDKRPC.GenerateTransaction(ctx, parser, []chain.Action{&actions.Transfer{
		To:           destination,
		AssetAddress: storage.NAIAddress,
		Value:        amount,
	}}, m.factory)
	if err != nil {
		m.log.Error("Failed to generate transaction", zap.Error(err))
		return ids.Empty, 0, err
	}

	m.log.Info("Generated transaction", zap.String("txID", tx.ID().String()))

	// Submit the transaction
	err = submitTxFunc(ctx)
	if err != nil {
		m.log.Error("Failed to submit transaction", zap.Error(err))
		return ids.Empty, 0, err
	}

	// Log success
	m.log.Info("Transaction submitted successfully",
		zap.String("txID", tx.ID().String()),
		zap.String("destination", destination.String()),
		zap.Uint64("amount", amount),
	)

	// Save transaction details to database
	_ = m.db.SaveTransaction(tx.ID().String(), destination.String(), amount)

	return tx.ID(), amount, nil
}

func (m *Manager) waitForTransactionWithIndexer(ctx context.Context, txID ids.ID, timeout time.Duration) (bool, error) {
	startTime := time.Now()

	for retries := 0; retries < 10; retries++ {
		// Check if we've exceeded the timeout
		if time.Since(startTime) > timeout {
			return false, fmt.Errorf("transaction wait timed out")
		}

		// Call the GetTx method of hyperIndexerRPC
		resp, found, err := m.hyperIndexerRPC.GetTx(ctx, txID)
		if err != nil {
			m.log.Error("Error calling hyperIndexerRPC.GetTx", zap.Error(err))
			return false, err
		}

		if found {
			if resp.Success {
				m.log.Info("Transaction successful", zap.String("txID", txID.String()))
				return true, nil
			}
			m.log.Warn("Transaction failed", zap.String("txID", txID.String()))
			return false, fmt.Errorf("transaction failed")
		}

		// Log and retry if the transaction is not found
		m.log.Info("Transaction not found yet, retrying...", zap.String("txID", txID.String()), zap.Int("retry", retries+1))

		// Exponential backoff for retries
		time.Sleep(time.Duration(100*(retries+1)) * time.Millisecond)
	}

	return false, fmt.Errorf("transaction status check failed after retries")
}

func (m *Manager) RequestTestFunds(ctx context.Context, solver codec.Address) (ids.ID, uint64, error) {
	m.healthMu.Lock()
	defer m.healthMu.Unlock()

	txID, maxFee, err := m.SendFundsRetry(ctx, solver, m.config.Amount)
	if err != nil {
		m.log.Error("Failed to send funds", zap.Error(err))
		return ids.Empty, 0, err
	}
	m.log.Info("Fauceted funds",
		zap.Stringer("txID", txID),
		zap.String("max fee", nutils.FormatBalance(maxFee, consts.Decimals)),
		zap.String("destination", solver.String()),
		zap.String("amount", nutils.FormatBalance(m.config.Amount, consts.Decimals)),
	)
	return txID, m.config.Amount, nil
}

func (m *Manager) UpdateNuklaiRPC(ctx context.Context, newNuklaiRPCUrl string) error {
	m.healthMu.Lock()
	defer m.healthMu.Unlock()

	m.log.Info("Updating nuklaiRPC URL", zap.String("old URL", m.config.NuklaiRPC), zap.String("new URL", newNuklaiRPCUrl))

	m.config.NuklaiRPC = fmt.Sprintf("%s/ext/bc/%s", newNuklaiRPCUrl, consts.Name)

	hyperSDKRPC := jsonrpc.NewJSONRPCClient(newNuklaiRPCUrl)
	networkID, subnetID, chainID, err := hyperSDKRPC.Network(ctx)
	if err != nil {
		m.log.Error("Failed to fetch network details", zap.Error(err))
		return fmt.Errorf("failed to fetch network details: %w", err)
	}
	m.log.Info("Fetched network details", zap.Uint32("network ID", networkID), zap.String("subnet ID", subnetID.String()), zap.String("chain ID", chainID.String()))

	m.hyperSDKRPC = hyperSDKRPC
	m.hyperVMRPC = vm.NewJSONRPCClient(newNuklaiRPCUrl)

	bal, err := m.hyperVMRPC.Balance(ctx, m.config.AddressBech32(), consts.Symbol)
	if err != nil {
		return err
	}

	m.log.Info("RPC client has been updated and manager reinitialized",
		zap.String("new RPC URL", newNuklaiRPCUrl),
		zap.Uint32("network ID", networkID),
		zap.String("chain ID", chainID.String()),
		zap.String("address", m.config.AddressBech32()),
		zap.String("balance", nutils.FormatBalance(bal, consts.Decimals)),
	)

	return nil
}

// Config returns the configuration of the manager
func (m *Manager) Config() *fconfig.Config {
	return m.config
}
