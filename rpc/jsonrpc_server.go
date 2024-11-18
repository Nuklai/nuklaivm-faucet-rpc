// Copyright (C) 2024, Nuklai. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"errors"
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
		// Allow 1 request per 60 seconds with 5 as burst capacity
		limiter = rate.NewLimiter(rate.Every(60*time.Second), 5)
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
	addr, err := j.m.GetFaucetAddress(req.Context())
	if err != nil {
		return err
	}
	reply.Address = addr.String()
	return nil
}

type RequestTestFundsArgs struct {
	Address string `json:"address"`
}

type RequestTestFundsReply struct {
	TxID   ids.ID `json:"txID"`
	Amount uint64 `json:"amount"`
}

func (j *JSONRPCServer) RequestTestFunds(req *http.Request, args *RequestTestFundsArgs, reply *RequestTestFundsReply) error {
	addr, err := codec.StringToAddress(args.Address)
	if err != nil {
		return err
	}
	txID, amount, err := j.m.RequestTestFunds(req.Context(), addr)
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
