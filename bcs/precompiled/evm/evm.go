package evm

import (
	"encoding/hex"
	"errors"
	"math/big"
	"strconv"

	"github.com/hyperledger/burrow/acm/balance"
	"github.com/hyperledger/burrow/txs/payload"

	"github.com/xuperchain/xupercore/bcs/contract/evm"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"

	"github.com/hyperledger/burrow/crypto"
	x "github.com/hyperledger/burrow/encoding/hex"
	"github.com/hyperledger/burrow/encoding/rlp"
	"github.com/hyperledger/burrow/rpc"
	"github.com/hyperledger/burrow/txs"

	"github.com/xuperchain/xupercore/kernel/contract"
)

const (
	DEFAULT_NET    = 1
	ETH_TX_PREFIX  = "ETH_TX_"
	BALANCE_PREFIX = "BALANCE_"
	// TODO  why add $
	CONTRACT_EVM      = "$evm"
	STATUS            = "STATUS"
	TRANSACTION_COUNT = "TRANSACTION_COUNT"
	PLEDGE_PREFIX     = "PLEDGE"
)

var (
	errNotImplemented = errors.New("not implemented")
)

// func NewEVMProxy(manager contract.Manager) (*EVMProxy, error) {
// 	registry := manager.GetKernRegistry()
// 	p := EVMProxy{}
// 	registry.RegisterKernMethod(CONTRACT_EVM, "SendRawTransaction", p.sendRawTransaction)
// 	registry.RegisterKernMethod(CONTRACT_EVM, "GetTransactionReceipt", p.getTransactionReceipt)
// 	registry.RegisterKernMethod(CONTRACT_EVM, "BalanceOf", p.balanceOf)
// 	registry.RegisterKernMethod(CONTRACT_EVM, "GetTransactionCount", p.transactionCount)
//
// 	// 质压赎回相关
// 	registry.RegisterKernMethod(CONTRACT_EVM, "Pledge", p.pledge)
// 	registry.RegisterKernMethod(CONTRACT_EVM, "Redeem", p.redeem)
// 	registry.RegisterKernMethod(CONTRACT_EVM, "Allowance", p.allowance)
// 	return &p, nil
// }

type EVMProxyInstaceCreator struct {
	config map[string]string
}

func (c *EVMProxy) Enabled() bool {
	// TODO
	return false
}

func (c *EVMProxyInstaceCreator) CreateInstance(configPah string) contract.PrecompiledContract {
	return &EVMProxy{}
}

type EVMProxy struct {
	config     map[string]string
	PledgeRate uint64
}

func (p *EVMProxy) sendRawTransaction(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()
	signedTx := args["signed_tx"]
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

	txHash, err := p.TxHash(from, strconv.Itoa(chainID), rawTx, amount)
	if err != nil {
		return nil, err
	}

	if err := ctx.Put(ETH_TX_PREFIX, txHash, signedTx); err != nil {
		return nil, err
	}

	resp, err := p.transactionCount(ctx)
	if err != nil {
		return nil, err
	}
	transactionCount, _ := new(big.Int).SetString(string(resp.Body), 10)
	transactionCount = transactionCount.Add(transactionCount, big.NewInt(1))

	if err := ctx.Put(STATUS, []byte(TRANSACTION_COUNT), []byte(transactionCount.String())); err != nil {
		return nil, err
	}

	if err := p.transfer(ctx, from.Bytes(), to.Bytes(), amount); err != nil {
		return nil, err
	}
	if len(rawTx.Data) == 0 {
		return &contract.Response{
			Status: 200,
			Body:   txHash,
		}, nil
	}

	contractName, err := evm.DetermineContractNameFromEVM(to)
	if err != nil {
		return nil, err
	}

	invokArgs := map[string][]byte{
		"input": rawTx.Data,
	}
	resp, err = ctx.Call("evm", contractName, "", invokArgs)
	return resp, err
}
func (p *EVMProxy) TxHash(from crypto.Address, chainId string, rawTx *rpc.RawTx, amount *big.Int) ([]byte, error) {
	to, err := crypto.AddressFromBytes(rawTx.To)
	if err != nil {
		return nil, err
	}

	tx := txs.Tx{
		ChainID: chainId,
		Payload: &payload.CallTx{
			Input: &payload.TxInput{
				Address: from,
				Amount:  amount.Uint64(),
				// first tx sequence should be 1,
				// but metamask starts at 0
				Sequence: rawTx.Nonce + 1,
			},
			Address:  &to,
			GasLimit: rawTx.GasLimit,
			GasPrice: rawTx.GasPrice,
			Data:     rawTx.Data,
		},
	}
	return tx.Hash(), nil
}

func (p *EVMProxy) verifySignature(
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
func (p *EVMProxy) getTransactionReceipt(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()
	txHash := args["tx_hash"]
	txHashByte, err := hex.DecodeString(string(txHash))
	if err != nil {
		return nil, err
	}
	tx, err := ctx.Get(ETH_TX_PREFIX, txHashByte)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: 200,
		Body:   tx,
	}, nil
}

func (p *EVMProxy) transfer(ctx contract.KContext, from, to []byte, amount *big.Int) error {
	if new(big.Int).SetBytes(from).Cmp(new(big.Int)) != 0 {
		fromBalanceByte, err := ctx.Get(BALANCE_PREFIX, from)
		if err != nil {
			return err
		}
		fromBalance, _ := new(big.Int).SetString(string(fromBalanceByte), 10)
		fromBalance = fromBalance.Sub(fromBalance, amount)
		if fromBalance.Cmp(amount) < 0 {
			return errors.New("balance not enough")
		}
		//  这里不能直接存 bytes, 当结果是0的时候会有问题
		if err := ctx.Put(BALANCE_PREFIX, from, []byte(fromBalance.String())); err != nil {
			return err
		}
	}

	toBalanceByte, err := ctx.Get(BALANCE_PREFIX, to)
	if err != nil {
		if err != sandbox.ErrNotFound {
			return err
		} else {
			toBalanceByte = []byte("0")
		}
	}
	toBalance, _ := new(big.Int).SetString(string(toBalanceByte), 10)

	toBalance = toBalance.Add(toBalance, amount)
	if err := ctx.Put(BALANCE_PREFIX, to, []byte(toBalance.String())); err != nil {
		return err
	}
	return nil
}

func (p *EVMProxy) balanceOf(ctx contract.KContext) (*contract.Response, error) {
	address := ctx.Args()["address"]
	addrss1, err := hex.DecodeString(string(address))
	if err != nil {
		return nil, err
	}
	balance, err := ctx.Get(BALANCE_PREFIX, addrss1)
	if err != nil {
		return nil, err
	}
	return &contract.Response{Body: balance}, nil
}

func (p *EVMProxy) transactionCount(ctx contract.KContext) (*contract.Response, error) {
	count, err := ctx.Get(STATUS, []byte(TRANSACTION_COUNT))
	if err != nil {
		if err != sandbox.ErrNotFound {
			return nil, err
		}
		count = []byte(big.NewInt(0).String())
	}
	return &contract.Response{
		Status: contract.StatusOK,
		Body:   count,
	}, nil
}

func (p *EVMProxy) pledge(ctx contract.KContext) (*contract.Response, error) {
	//  这里用十进制还是十六进制呢
	amount, err := ctx.TransferAmount()
	if err != nil {
		return nil, err
	}
	to := ctx.Args()["to"]
	toByte, err := hex.DecodeString(string(to))
	if err != nil {
		return nil, err
	}
	//  only 1:1 pedge is supported

	if err := ctx.Put(BALANCE_PREFIX, toByte, []byte(amount.String())); err != nil {
		return nil, err
	}
	if err := ctx.Put(PLEDGE_PREFIX, []byte(ctx.Initiator()), toByte); err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: contract.StatusOK,
	}, nil
}

//  赎回
func (p *EVMProxy) redeem(ctx contract.KContext) (*contract.Response, error) {
	initiator := ctx.Initiator()
	to, err := ctx.Get(PLEDGE_PREFIX, []byte(initiator))
	if err != nil {
		return nil, err
	}
	balance1, err := ctx.Get(BALANCE_PREFIX, to)
	if err != nil {
		return nil, err
	}
	amount, _ := new(big.Int).SetString(string(balance1), 10)
	if err := ctx.Transfer(ctx.ContractName(), initiator, amount); err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: contract.StatusOK,
	}, nil
}

//  质压额度
func (p *EVMProxy) allowance(ctx contract.KContext) (*contract.Response, error) {
	balance, err := ctx.Get(PLEDGE_PREFIX, []byte(ctx.Initiator()))
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: contract.StatusOK,
		Body:   balance,
	}, nil
}
func newEVMProxyInstaceCreator(config map[string]string) contract.EmbededContract {
	return EVMProxy{
		config: config,
	}
}

func init() {
	contract.RegisterEmbededContractCreatorFunc("ethproxy", newEVMProxyInstaceCreator)
}
