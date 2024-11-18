// Copyright (C) 2024, Nuklai. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"strings"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/hypersdk/requester"
)

const (
	JSONRPCEndpoint = "/faucet"
)

type JSONRPCClient struct {
	requester *requester.EndpointRequester
}

// New creates a new client object.
func NewJSONRPCClient(uri string) *JSONRPCClient {
	uri = strings.TrimSuffix(uri, "/")
	uri += JSONRPCEndpoint
	req := requester.New(uri, "faucet")
	return &JSONRPCClient{
		requester: req,
	}
}

func (cli *JSONRPCClient) FaucetAddress(ctx context.Context) (string, error) {
	resp := new(FaucetAddressReply)
	err := cli.requester.SendRequest(
		ctx,
		"faucetAddress",
		nil,
		resp,
	)
	return resp.Address, err
}

func (cli *JSONRPCClient) RequestTestFunds(ctx context.Context, addr string) (ids.ID, uint64, error) {
	resp := new(RequestTestFundsReply)
	err := cli.requester.SendRequest(
		ctx,
		"requestTestFunds",
		&RequestTestFundsArgs{
			Address: addr,
		},
		resp,
	)
	return resp.TxID, resp.Amount, err
}

// UpdateNuklaiRPC updates the RPC url for Nuklai, only if admin token is valid
func (cli *JSONRPCClient) UpdateNuklaiRPC(ctx context.Context, adminToken string, newNuklaiRPCUrl string) (bool, error) {
	resp := new(UpdateNuklaiRPCReply)
	err := cli.requester.SendRequest(
		ctx,
		"updateNuklaiRPC",
		&UpdateNuklaiRPCArgs{
			AdminToken:   adminToken,
			NuklaiRPCUrl: newNuklaiRPCUrl,
		},
		resp,
	)
	return resp.Success, err
}
