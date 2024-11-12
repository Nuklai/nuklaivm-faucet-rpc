// Copyright (C) 2024, Nuklai. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/hypersdk/codec"
	"golang.org/x/time/rate"
)

var (
	ipLimiters = make(map[string]*rate.Limiter)
	mu         sync.Mutex
)

func getRateLimiter(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()
	limiter, exists := ipLimiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Every(15*time.Second), 10)
		ipLimiters[ip] = limiter
	}
	return limiter
}

type JSONRPCServer struct {
	m Manager
}

func NewJSONRPCServer(m Manager) *JSONRPCServer {
	return &JSONRPCServer{m}
}

type FaucetAddressReply struct {
	Address string `json:"address"`
}

func (j *JSONRPCServer) FaucetAddress(req *http.Request, _ *struct{}, reply *FaucetAddressReply) (err error) {
	// Retrieve client IP
	clientIP := req.RemoteAddr
	if !getRateLimiter(clientIP).Allow() {
		return fmt.Errorf("rate limit exceeded for IP %s", clientIP)
	}
	addr, err := j.m.GetFaucetAddress(req.Context())
	if err != nil {
		return err
	}
	reply.Address = addr.String()
	return nil
}

type ChallengeReply struct {
	Salt       []byte `json:"salt"`
	Difficulty uint16 `json:"difficulty"`
}

func (j *JSONRPCServer) Challenge(req *http.Request, _ *struct{}, reply *ChallengeReply) (err error) {
	salt, difficulty, err := j.m.GetChallenge(req.Context())
	if err != nil {
		return err
	}
	reply.Salt = salt
	reply.Difficulty = difficulty
	return nil
}

type SolveChallengeArgs struct {
	Address  string `json:"address"`
	Salt     []byte `json:"salt"`
	Solution []byte `json:"solution"`
}

type SolveChallengeReply struct {
	TxID   ids.ID `json:"txID"`
	Amount uint64 `json:"amount"`
}

func (j *JSONRPCServer) SolveChallenge(req *http.Request, args *SolveChallengeArgs, reply *SolveChallengeReply) error {
	addr, err := codec.StringToAddress(args.Address)
	if err != nil {
		return err
	}
	txID, amount, err := j.m.SolveChallenge(req.Context(), addr, args.Salt, args.Solution)
	if err != nil {
		return err
	}
	reply.TxID = txID
	reply.Amount = amount
	return nil
}

type UpdateNuklaiRPCArgs struct {
	AdminToken   string `json:"adminToken"`
	NuklaiRPCUrl string `json:"nuklaiRPCUrl"`
}

type UpdateNuklaiRPCReply struct {
	Success bool `json:"success"`
}

func (j *JSONRPCServer) UpdateNuklaiRPC(req *http.Request, args *UpdateNuklaiRPCArgs, reply *UpdateNuklaiRPCReply) error {
	// Validate the admin token
	if args.AdminToken != j.m.Config().AdminToken {
		return errors.New("unauthorized user")
	}
	err := j.m.UpdateNuklaiRPC(req.Context(), args.NuklaiRPCUrl)
	if err != nil {
		return err
	}
	reply.Success = true
	return nil
}
