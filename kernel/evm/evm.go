package evm

import (
	"encoding/hex"
	"strconv"

	"github.com/hyperledger/burrow/acm/balance"
	"github.com/xuperchain/xupercore/bcs/contract/evm"

	"github.com/hyperledger/burrow/crypto"
	x "github.com/hyperledger/burrow/encoding/hex"
	"github.com/hyperledger/burrow/encoding/rlp"
	"github.com/hyperledger/burrow/rpc"
	"github.com/hyperledger/burrow/txs"
	"github.com/xuperchain/xupercore/kernel/contract"
)

const (
	DEFAULT_NET   = 1
	ETH_TX_PREFIX = "ETH_TX_"
)

type EVMProxy interface {
}

func NewEVMProxy(manager contract.Manager) (EVMProxy, error) {
	registry := manager.GetKernRegistry()
	p := proxy{}
	// SendTransaction is not used currently
	// registry.RegisterKernMethod("$evm", "SendTransaction", p.sendTransaction)
	registry.RegisterKernMethod("$evm", "SendRawTransaction", p.sendRawTransaction)
	registry.RegisterKernMethod("$evm", "GetTransactionReceipt", p.getTransactionReceipt)

	// registry.RegisterKernMethod("$evm", "ContractCall", p.ContractCall)
	return &p, nil
}

type proxy struct {
}

// not used currently
func (p *proxy) sendTransaction(ctx contract.KContext) (*contract.Response, error) {
	// 数据类型转换
	//byte:
	// byte --> hex string --> byte -->
	// byte  <-- hex string <--
	// string
	//string --> byte -->
	//string <--
	var nonce, gasPrice, gasLimit int
	var to, value, data []byte
	var net, V uint64
	var S, R []byte
	var err error
	args := ctx.Args()
	nonceStr := args["nonce"]
	gasPriceStr := args["gas_price"]
	gasLimitStr := args["gas_limit"]
	nonce, err = strconv.Atoi(string(nonceStr))
	if err != nil {
		return nil, err
	}
	gasPrice, err = strconv.Atoi(string(gasPriceStr))
	if err != nil {
		return nil, err
	}

	gasLimit, err = strconv.Atoi(string(gasLimitStr))
	if err != nil {
		return nil, err
	}
	toStr := args["to"]
	to, err = hex.DecodeString(string(toStr))
	if err != nil {
		return nil, err
	}
	valueStr := args["value"]
	value, err = hex.DecodeString(string((valueStr)))
	if err != nil {
		return nil, err
	}
	dataStr := args["data"]
	data, err = hex.DecodeString(string(dataStr))
	if err != nil {
		return nil, err
	}
	net = DEFAULT_NET
	VStr := args["v"]
	V, err = strconv.ParseUint(string(VStr), 10, 64)
	if err != nil {
		return nil, err
	}

	SStr := args["s"]
	S, err = hex.DecodeString(string(SStr))
	if err != nil {
		return nil, err
	}
	RStr := args["r"]
	R, err = hex.DecodeString(string(RStr))
	if err != nil {
		return nil, err
	}

	if err := p.verifySignature(uint64(nonce), uint64(gasPrice), uint64(gasLimit), to, value, data, net, V, S, R); err != nil {
		return nil, err
	}
	return p.ContractCall(ctx)
}

func (p *proxy) sendRawTransaction(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()
	signedTx := args["signed_tx"]
	txHash := args["tx_hash"]
	data, err := x.DecodeToBytes(string(signedTx))
	if err != nil {
		return nil, err
	}

	rawTx := new(rpc.RawTx)
	err = rlp.Decode(data, rawTx)
	if err != nil {
		return nil, err
	}
	chainID := DEFAULT_NET

	net := uint64(chainID)

	if err := p.verifySignature(
		rawTx.Nonce, rawTx.GasPrice, rawTx.GasLimit,
		rawTx.To, rawTx.Value, rawTx.Data,
		net, rawTx.V, rawTx.S, rawTx.R,
	); err != nil {
		return nil, err
	}
	to, err := crypto.AddressFromBytes(rawTx.To)
	if err != nil {
		return nil, err
	}

	enc, err := txs.RLPEncode(rawTx.Nonce, rawTx.GasPrice, rawTx.GasLimit, rawTx.To, rawTx.Value, rawTx.Data)
	if err != nil {
		return nil, err
	}

	sig := crypto.CompressedSignatureFromParams(rawTx.V-net-8-1, rawTx.R, rawTx.S)
	pub, err := crypto.PublicKeyFromSignature(sig, crypto.Keccak256(enc))
	if err != nil {
		return nil, err
	}

	from := pub.GetAddress()
	amount := balance.WeiToNative(rawTx.Value)

	if err := ctx.Put(ETH_TX_PREFIX, txHash, signedTx); err != nil {
		return nil, err
	}

	if err := ctx.Transfer(from.String(), to.String(), amount); err != nil {
		return nil, err
	}
	if len(rawTx.Data) == 0 {
		return &contract.Response{
			Status: 200,
			Body:   []byte("ok"),
		}, nil
	}
	contractName, err := evm.DetermineContractNameFromEVM(to)
	if err != nil {
		return nil, err
	}

	invokArgs := map[string][]byte{
		"input": rawTx.Data,
	}
	resp, err := ctx.Call("evm", contractName, "", invokArgs)
	return resp, err
}

func (p *proxy) ContractCall(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()
	input, err := hex.DecodeString(string(args["input"]))
	if err != nil {
		return nil, err
	}

	invokArgs := map[string][]byte{
		"input": input,
	}
	address, err := crypto.AddressFromHexString(string(args["to"]))
	if err != nil {
		return nil, err
	}
	contractName, err := evm.DetermineContractNameFromEVM(address)
	if err != nil {
		return nil, err
	}

	resp, err := ctx.Call("evm", contractName, "", invokArgs)
	return resp, err

}

func (p *proxy) verifySignature(
	nonce, gasPrice, gasLimit uint64,
	to, value, data []byte,
	net, V uint64,
	S, R []byte) error {
	enc, err := txs.RLPEncode(nonce, gasPrice, gasLimit, to, value, data)
	if err != nil {
		return err
	}

	sig := crypto.CompressedSignatureFromParams(V-net-8-1, R, S)
	pub, err := crypto.PublicKeyFromSignature(sig, crypto.Keccak256(enc))
	if err != nil {
		return err
	}
	unc := crypto.UncompressedSignatureFromParams(R, S)
	signature, err := crypto.SignatureFromBytes(unc, crypto.CurveTypeSecp256k1)
	if err != nil {
		return err
	}
	if err := pub.Verify(enc, signature); err != nil {
		return err
	}
	return nil
}
func (p *proxy) getTransactionReceipt(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()
	txHash := args["tx_hash"]
	tx, err := ctx.Get(ETH_TX_PREFIX, txHash)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: 200,
		Body:   tx,
	}, nil
}
