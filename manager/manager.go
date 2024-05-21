package manager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/avalanchego/utils/timer"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/rpc"
	"github.com/ava-labs/hypersdk/utils"
	fconfig "github.com/nuklai/nuklai-faucet/config"
	"github.com/nuklai/nuklai-faucet/database"
	"github.com/nuklai/nuklaivm/actions"
	"github.com/nuklai/nuklaivm/auth"
	"github.com/nuklai/nuklaivm/challenge"
	nconsts "github.com/nuklai/nuklaivm/consts"
	nrpc "github.com/nuklai/nuklaivm/rpc"
	"go.uber.org/zap"
)

type Manager struct {
	log    logging.Logger
	config *fconfig.Config

	cli  *rpc.JSONRPCClient
	ncli *nrpc.JSONRPCClient

	factory *auth.ED25519Factory

	l            sync.RWMutex
	t            *timer.Timer
	lastRotation int64
	salt         []byte
	difficulty   uint16
	solutions    set.Set[ids.ID]
	cancelFunc   context.CancelFunc

	db *database.DB
}

func New(logger logging.Logger, config *fconfig.Config) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cli := rpc.NewJSONRPCClient(config.NuklaiRPC)
	networkID, _, chainID, err := cli.Network(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	ncli := nrpc.NewJSONRPCClient(config.NuklaiRPC, networkID, chainID)

	// Set default values to the current directory
	defaultDir, err := os.Getwd()
	if err != nil {
		panic("Failed to get current working directory: " + err.Error())
	}
	databaseFolder := fconfig.GetEnv("NUKLAI_FAUCET_DB_PATH", filepath.Join(defaultDir, ".nuklai-faucet/db"))
	if err := os.MkdirAll(databaseFolder, os.ModePerm); err != nil {
		panic("failed to create database directory: " + err.Error())
	}
	dbPath := filepath.Join(databaseFolder, "faucet.db")

	db, err := database.NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	m := &Manager{log: logger, config: config, cli: cli, ncli: ncli, factory: auth.NewED25519Factory(config.PrivateKey()), cancelFunc: cancel, db: db}
	m.lastRotation = time.Now().Unix()
	m.difficulty = m.config.StartDifficulty
	m.solutions = set.NewSet[ids.ID](m.config.SolutionsPerSalt)
	m.salt, err = challenge.New()
	if err != nil {
		return nil, err
	}
	bal, err := ncli.Balance(ctx, m.config.AddressBech32(), ids.Empty)
	if err != nil {
		return nil, err
	}
	m.log.Info("faucet initialized",
		zap.String("address", m.config.AddressBech32()),
		zap.Uint16("difficulty", m.difficulty),
		zap.String("balance", utils.FormatBalance(bal, nconsts.Decimals)),
	)
	m.t = timer.NewTimer(m.updateDifficulty)
	return m, nil
}

func (m *Manager) Run(ctx context.Context) error {
	m.t.SetTimeoutIn(time.Duration(m.config.TargetDurationPerSalt) * time.Second)
	go m.t.Dispatch()
	<-ctx.Done()
	m.t.Stop()
	m.db.Close()
	return ctx.Err()
}

func (m *Manager) updateDifficulty() {
	m.l.Lock()
	defer m.l.Unlock()

	// If time since [lastRotation] is within half of the target duration,
	// we attempted to update difficulty when we just reset during block processing.
	now := time.Now().Unix()
	if now-m.lastRotation < m.config.TargetDurationPerSalt/2 {
		return
	}

	// Decrease difficulty if there are no solutions in this period
	if m.difficulty > m.config.StartDifficulty && m.solutions.Len() == 0 {
		m.difficulty--
		m.log.Info("decreasing faucet difficulty", zap.Uint16("new difficulty", m.difficulty))
	}
	m.lastRotation = time.Now().Unix()
	salt, err := challenge.New()
	if err != nil {
		panic(err)
	}
	m.salt = salt
	m.solutions.Clear()
	m.t.SetTimeoutIn(time.Duration(m.config.TargetDurationPerSalt) * time.Second)
}

func (m *Manager) GetFaucetAddress(_ context.Context) (codec.Address, error) {
	return m.config.Address(), nil
}

func (m *Manager) GetChallenge(_ context.Context) ([]byte, uint16, error) {
	m.l.RLock()
	defer m.l.RUnlock()

	return m.salt, m.difficulty, nil
}

func (m *Manager) sendFunds(ctx context.Context, destination codec.Address, amount uint64) (ids.ID, uint64, error) {
	parser, err := m.ncli.Parser(ctx)
	if err != nil {
		return ids.Empty, 0, err
	}
	submit, tx, maxFee, err := m.cli.GenerateTransaction(ctx, parser, nil, &actions.Transfer{
		To:    destination,
		Asset: ids.Empty,
		Value: amount,
	}, m.factory)
	if err != nil {
		return ids.Empty, 0, err
	}
	if amount < maxFee {
		m.log.Warn("abandoning airdrop because network fee is greater than amount", zap.String("maxFee", utils.FormatBalance(maxFee, nconsts.Decimals)))
		return ids.Empty, 0, errors.New("network fee too high")
	}
	bal, err := m.ncli.Balance(ctx, m.config.AddressBech32(), ids.Empty)
	if err != nil {
		return ids.Empty, 0, err
	}
	if bal < maxFee+amount {
		// This is a "best guess" heuristic for balance as there may be txs in-flight.
		m.log.Warn("faucet has insufficient funds", zap.String("balance", utils.FormatBalance(bal, nconsts.Decimals)))
		return ids.Empty, 0, errors.New("insufficient balance")
	}

	err = submit(ctx)
	if err == nil {
		destinationAddr, err := codec.AddressBech32(nconsts.HRP, destination)
		if err != nil {
			return ids.Empty, 0, err
		}
		_ = m.db.SaveTransaction(tx.ID().String(), destinationAddr, amount)
	}
	return tx.ID(), maxFee, err
}

func (m *Manager) SolveChallenge(ctx context.Context, solver codec.Address, salt []byte, solution []byte) (ids.ID, uint64, error) {
	m.l.Lock()
	defer m.l.Unlock()

	// Ensure solution is valid
	if !bytes.Equal(m.salt, salt) {
		return ids.Empty, 0, errors.New("salt expired")
	}
	if !challenge.Verify(salt, solution, m.difficulty) {
		return ids.Empty, 0, errors.New("invalid solution")
	}
	solutionID := utils.ToID(solution)
	if m.solutions.Contains(solutionID) {
		return ids.Empty, 0, errors.New("duplicate solution")
	}

	// Issue transaction
	txID, maxFee, err := m.sendFunds(ctx, solver, m.config.Amount)
	if err != nil {
		return ids.Empty, 0, err
	}
	m.log.Info("fauceted funds",
		zap.Stringer("txID", txID),
		zap.String("max fee", utils.FormatBalance(maxFee, nconsts.Decimals)),
		zap.String("destination", codec.MustAddressBech32(nconsts.HRP, solver)),
		zap.String("amount", utils.FormatBalance(m.config.Amount, nconsts.Decimals)),
	)
	m.solutions.Add(solutionID)

	// Roll salt if hit expected solutions
	if m.solutions.Len() >= m.config.SolutionsPerSalt {
		m.difficulty++
		m.log.Info("increasing faucet difficulty", zap.Uint16("new difficulty", m.difficulty))
		m.lastRotation = time.Now().Unix()
		m.salt, err = challenge.New()
		if err != nil {
			// Should never happen
			return ids.Empty, 0, err
		}
		m.solutions.Clear()
		m.t.Cancel()
		m.t.SetTimeoutIn(time.Duration(m.config.TargetDurationPerSalt) * time.Second)
	}
	return txID, m.config.Amount, nil
}

func (m *Manager) UpdateNuklaiRPC(ctx context.Context, newNuklaiRPCUrl string) error {
	m.l.Lock()
	defer m.l.Unlock()

	m.log.Info("Updating nuklaiRPC URL", zap.String("old URL", m.config.NuklaiRPC), zap.String("new URL", newNuklaiRPCUrl))

	// Updating the configuration
	m.config.NuklaiRPC = newNuklaiRPCUrl

	// Re-initializing the RPC clients
	cli := rpc.NewJSONRPCClient(newNuklaiRPCUrl)
	networkID, _, chainID, err := cli.Network(ctx)
	m.log.Info("Fetching network details", zap.Uint32("network ID", networkID), zap.String("chain ID", chainID.String()))
	if err != nil {
		m.log.Error("Failed to fetch network details", zap.Error(err))
		return fmt.Errorf("failed to fetch network details: %w", err)
	}

	// Reassign the newly created clients
	m.cli = cli
	m.ncli = nrpc.NewJSONRPCClient(newNuklaiRPCUrl, networkID, chainID)

	// Reinitialize dependent properties
	m.salt, err = challenge.New()
	if err != nil {
		m.log.Error("Failed to generate new salt", zap.Error(err))
		return fmt.Errorf("failed to generate new salt: %w", err)
	}
	m.solutions = set.NewSet[ids.ID](m.config.SolutionsPerSalt)
	m.difficulty = m.config.StartDifficulty
	m.lastRotation = time.Now().Unix()

	bal, err := m.ncli.Balance(ctx, m.config.AddressBech32(), ids.Empty)
	if err != nil {
		return err
	}
	m.t = timer.NewTimer(m.updateDifficulty)

	m.log.Info("RPC client has been updated and manager reinitialized",
		zap.String("new RPC URL", newNuklaiRPCUrl),
		zap.Uint32("network ID", networkID),
		zap.String("chain ID", chainID.String()),
		zap.String("address", m.config.AddressBech32()),
		zap.Uint16("difficulty", m.difficulty),
		zap.String("balance", utils.FormatBalance(bal, nconsts.Decimals)),
	)

	return nil
}

// Config returns the configuration of the manager
func (m *Manager) Config() *fconfig.Config {
	return m.config
}