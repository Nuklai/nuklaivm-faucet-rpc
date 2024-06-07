package rpc

import (
	"context"
	"encoding/json"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/nuklai/nuklai-faucet/manager"
	"github.com/nuklai/nuklaivm/consts"
)

type JSONRPCServer struct {
	m *manager.Manager
}

func NewJSONRPCServer(m *manager.Manager) *JSONRPCServer {
	return &JSONRPCServer{m}
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *jsonrpcError `json:"error,omitempty"`
	ID      interface{}   `json:"id"`
}

type jsonrpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (j *JSONRPCServer) HandleRequest(req JSONRPCRequest) JSONRPCResponse {
	var result interface{}
	var jsonErr *jsonrpcError

	switch req.Method {
	case "faucetAddress":
		var params struct{}
		err := json.Unmarshal(req.Params, &params)
		if err != nil {
			jsonErr = &jsonrpcError{Code: -32602, Message: "Invalid params"}
			break
		}
		var reply FaucetAddressReply
		jsonErr = j.FaucetAddress(params, &reply)
		result = reply

	case "challenge":
		var params struct{}
		err := json.Unmarshal(req.Params, &params)
		if err != nil {
			jsonErr = &jsonrpcError{Code: -32602, Message: "Invalid params"}
			break
		}
		var reply ChallengeReply
		jsonErr = j.Challenge(params, &reply)
		result = reply

	case "solveChallenge":
		var params SolveChallengeArgs
		err := json.Unmarshal(req.Params, &params)
		if err != nil {
			jsonErr = &jsonrpcError{Code: -32602, Message: "Invalid params"}
			break
		}
		var reply SolveChallengeReply
		jsonErr = j.SolveChallenge(params, &reply)
		result = reply

	case "updateNuklaiRPC":
		var params UpdateNuklaiRPCArgs
		err := json.Unmarshal(req.Params, &params)
		if err != nil {
			jsonErr = &jsonrpcError{Code: -32602, Message: "Invalid params"}
			break
		}
		var reply UpdateNuklaiRPCReply
		jsonErr = j.UpdateNuklaiRPC(params, &reply)
		result = reply

	default:
		jsonErr = &jsonrpcError{Code: -32601, Message: "Method not found"}
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		Error:   jsonErr,
		ID:      req.ID,
	}
}

type FaucetAddressReply struct {
	Address string `json:"address"`
}

func (j *JSONRPCServer) FaucetAddress(_ struct{}, reply *FaucetAddressReply) *jsonrpcError {
	addr, err := j.m.GetFaucetAddress(context.Background())
	if err != nil {
		return &jsonrpcError{Code: -32000, Message: err.Error()}
	}
	reply.Address = codec.MustAddressBech32(consts.HRP, addr)
	return nil
}

type ChallengeReply struct {
	Salt       []byte `json:"salt"`
	Difficulty uint16 `json:"difficulty"`
}

func (j *JSONRPCServer) Challenge(_ struct{}, reply *ChallengeReply) *jsonrpcError {
	salt, difficulty, err := j.m.GetChallenge(context.Background())
	if err != nil {
		return &jsonrpcError{Code: -32000, Message: err.Error()}
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

func (j *JSONRPCServer) SolveChallenge(args SolveChallengeArgs, reply *SolveChallengeReply) *jsonrpcError {
	addr, err := codec.ParseAddressBech32(consts.HRP, args.Address)
	if err != nil {
		return &jsonrpcError{Code: -32602, Message: "Invalid address"}
	}
	txID, amount, err := j.m.SolveChallenge(context.Background(), addr, args.Salt, args.Solution)
	if err != nil {
		return &jsonrpcError{Code: -32000, Message: err.Error()}
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

func (j *JSONRPCServer) UpdateNuklaiRPC(args UpdateNuklaiRPCArgs, reply *UpdateNuklaiRPCReply) *jsonrpcError {
	if args.AdminToken != j.m.Config().AdminToken {
		return &jsonrpcError{Code: -32000, Message: "unauthorized user"}
	}
	err := j.m.UpdateNuklaiRPC(context.Background(), args.NuklaiRPCUrl)
	if err != nil {
		return &jsonrpcError{Code: -32000, Message: err.Error()}
	}
	reply.Success = true
	return nil
}
