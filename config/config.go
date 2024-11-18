// Copyright (C) 2024, Nuklai. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ava-labs/hypersdk/auth"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/crypto/ed25519"
	"github.com/nuklai/nuklaivm/consts"
)

type Config struct {
	HTTPHost string
	HTTPPort int

	PrivateKeyBytes []byte

	NuklaiRPC string
	Amount    uint64

	AdminToken       string
	BalanceThreshold uint64

	// PostgreSQL configuration
	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDBName   string
	PostgresSSLMode  string
}

func (c *Config) PrivateKey() auth.PrivateKey {
	pk := ed25519.PrivateKey(c.PrivateKeyBytes)
	return auth.PrivateKey{
		Address: auth.NewED25519Address(pk.PublicKey()),
		Bytes:   c.PrivateKeyBytes,
	}
}

func (c *Config) PrivateKeyEd25519() ed25519.PrivateKey {
	return ed25519.PrivateKey(c.PrivateKeyBytes)
}

func (c *Config) Address() codec.Address {
	return auth.NewED25519Address(c.PrivateKeyEd25519().PublicKey())
}

func (c *Config) AddressBech32() string {
	return c.Address().String()
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

	amount, err := strconv.ParseUint(GetEnv("AMOUNT", "1000000000"), 10, 64)
	if err != nil {
		return nil, err
	}

	balanceThreshold, err := strconv.ParseUint(GetEnv("BALANCE_THRESHOLD", "25"), 10, 64)
	if err != nil {
		return nil, err
	}

	// Attempt to decode as base64 first
	privateKeyStr := GetEnv("PRIVATE_KEY_BYTES", "Mjsdj07tXw2p2pMHGwNPLc6dLSJpLBcvPLJSpk3fr9AbBX3jICl8Ka0MH1ieohaGnPGTjYjJ+9cNZ0gyPb8vpw==")
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyStr)
	if err == nil && len(privateKeyBytes) == ed25519.PrivateKeyLen {
		// Successfully decoded as base64 and the length is correct
		fmt.Println("Decoded private key as base64 successfully.")
	} else {
		// If base64 decoding fails, try hex decoding
		privateKeyBytes, err = codec.LoadHex(strings.TrimSpace(privateKeyStr), ed25519.PrivateKeyLen)
		if err != nil {
			return nil, fmt.Errorf("failed to decode private key string: input is not valid hex or length mismatch")
		}
		fmt.Println("Decoded key as hex successfully.")
	}

	postgresPort, err := strconv.Atoi(GetEnv("POSTGRES_PORT", "5432"))
	if err != nil {
		return nil, err
	}

	postgresEnableSSL := GetEnv("POSTGRES_ENABLESSL", "false")
	postgresSSLMode := "disable"
	if parsed, err := strconv.ParseBool(postgresEnableSSL); err == nil && parsed {
		postgresSSLMode = "require"
	}

	nuklaiRPC := GetEnv("NUKLAI_RPC", "http://127.0.0.1:9650")

	return &Config{
		HTTPHost: GetEnv("HOST", ""),
		HTTPPort: port,

		PrivateKeyBytes: privateKeyBytes,

		NuklaiRPC: fmt.Sprintf("%s/ext/bc/%s", nuklaiRPC, consts.Name),
		Amount:    amount,

		AdminToken:       GetEnv("ADMIN_TOKEN", "ADMIN_TOKEN"),
		BalanceThreshold: balanceThreshold,

		PostgresHost:     GetEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     postgresPort,
		PostgresUser:     GetEnv("POSTGRES_USER", "user"),
		PostgresPassword: GetEnv("POSTGRES_PASSWORD", "password"),
		PostgresDBName:   GetEnv("POSTGRES_DBNAME", "dbname"),
		PostgresSSLMode:  postgresSSLMode,
	}, nil
}
