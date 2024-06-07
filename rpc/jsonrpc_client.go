package rpc

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/hypersdk/requester"
)

type JSONRPCClient struct {
	requester *requester.EndpointRequester
}

func NewJSONRPCClient(uri string) *JSONRPCClient {
	req := requester.New(uri, "")
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

func (cli *JSONRPCClient) Challenge(ctx context.Context) ([]byte, uint16, error) {
	resp := new(ChallengeReply)
	err := cli.requester.SendRequest(
		ctx,
		"challenge",
		nil,
		resp,
	)
	return resp.Salt, resp.Difficulty, err
}

func (cli *JSONRPCClient) SolveChallenge(ctx context.Context, addr string, salt []byte, solution []byte) (ids.ID, uint64, error) {
	resp := new(SolveChallengeReply)
	err := cli.requester.SendRequest(
		ctx,
		"solveChallenge",
		&SolveChallengeArgs{
			Address:  addr,
			Salt:     salt,
			Solution: solution,
		},
		resp,
	)
	return resp.TxID, resp.Amount, err
}

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
