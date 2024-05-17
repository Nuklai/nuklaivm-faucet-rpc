package config

import (
	"encoding/base64"
	"os"
	"strconv"

	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/crypto/ed25519"
	"github.com/nuklai/nuklaivm/auth"
	"github.com/nuklai/nuklaivm/consts"
)

type Config struct {
	HTTPHost string
	HTTPPort int

	PrivateKeyBytes []byte

	NuklaiRPC             string
	Amount                uint64
	StartDifficulty       uint16
	SolutionsPerSalt      int
	TargetDurationPerSalt int64 // seconds

	AdminToken string
}

func (c *Config) PrivateKey() ed25519.PrivateKey {
	return ed25519.PrivateKey(c.PrivateKeyBytes)
}

func (c *Config) Address() codec.Address {
	return auth.NewED25519Address(c.PrivateKey().PublicKey())
}

func (c *Config) AddressBech32() string {
	return codec.MustAddressBech32(consts.HRP, c.Address())
}

func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func LoadConfigFromEnv() (*Config, error) {
	port, err := strconv.Atoi(GetEnv("PORT", "10591"))
	if err != nil {
		return nil, err
	}

	amount, err := strconv.ParseUint(GetEnv("AMOUNT", "100000000"), 10, 64)
	if err != nil {
		return nil, err
	}

	startDifficulty, err := strconv.ParseUint(GetEnv("START_DIFFICULTY", "25"), 10, 16)
	if err != nil {
		return nil, err
	}

	solutionsPerSalt, err := strconv.Atoi(GetEnv("SOLUTIONS_PER_SALT", "10"))
	if err != nil {
		return nil, err
	}

	targetDurationPerSalt, err := strconv.ParseInt(GetEnv("TARGET_DURATION_PER_SALT", "300"), 10, 64)
	if err != nil {
		return nil, err
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(GetEnv("PRIVATE_KEY_BYTES", "Mjsdj07tXw2p2pMHGwNPLc6dLSJpLBcvPLJSpk3fr9AbBX3jICl8Ka0MH1ieohaGnPGTjYjJ+9cNZ0gyPb8vpw=="))
	if err != nil {
		return nil, err
	}

	return &Config{
		HTTPHost: GetEnv("HOST", ""),
		HTTPPort: port,

		PrivateKeyBytes: privateKeyBytes,

		NuklaiRPC:             os.Getenv("NUKLAI_RPC"),
		Amount:                amount,
		StartDifficulty:       uint16(startDifficulty),
		SolutionsPerSalt:      solutionsPerSalt,
		TargetDurationPerSalt: targetDurationPerSalt,

		AdminToken: GetEnv("ADMIN_TOKEN", "ADMIN_TOKEN"),
	}, nil
}
