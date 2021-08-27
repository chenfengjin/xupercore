package evm

import (
	"bytes"
	"math/big"

	_ "github.com/xuperchain/xupercore/bcs/contract/evm"
	_ "github.com/xuperchain/xupercore/bcs/contract/native"
	_ "github.com/xuperchain/xupercore/bcs/contract/xvm"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/protos"

	"encoding/hex"
	"io/ioutil"
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	_ "github.com/xuperchain/xupercore/kernel/contract/manager"
	"github.com/xuperchain/xupercore/kernel/contract/mock"
)

func TestEVMProxy(t *testing.T) {
	var contractConfig = &contract.ContractConfig{
		EnableUpgrade: true,
		Xkernel: contract.XkernelConfig{
			Enable: true,
			Driver: "default",
		},
		Native: contract.NativeConfig{
			Enable: true,
			Driver: "native",
		},
		EVM: contract.EVMConfig{
			Enable: true,
			Driver: "evm",
		},
		LogDriver: mock.NewMockLogger(),
	}
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()
	_, err := NewEVMProxy(m)
	if err != nil {
		t.Error(err)
		return
	}

	bin, err := ioutil.ReadFile("testdata/counter.bin")
	if err != nil {
		t.Error(err)
		return
	}
	abi, err := ioutil.ReadFile("testdata/counter.abi")
	if err != nil {
		t.Error(err)
		return
	}

	args := map[string][]byte{
		"contract_abi": abi,
	}
	data, err := hex.DecodeString(string((bin)))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := th.Deploy("evm", "counter", "counter", data, args)
	if err != nil {
		t.Fatal(err)
	}
	signedTx := []byte("0xf867808082520894f97798df751deb4b6e39d4cf998ee7cd4dcb9acc880de0b6b3a76400008025a0f0d2396973296cd6a71141c974d4a851f5eae8f08a8fba2dc36a0fef9bd6440ca0171995aa750d3f9f8e4d0eac93ff67634274f3c5acf422723f49ff09a6885422")
	t.Run("SendRawTransaction", func(t *testing.T) {
		th.SetUtxoReader(sandbox.NewUTXOReaderFromInput([]*protos.TxInput{
			{
				FromAddr: []byte("2C2D14A9A3F0D078AC8B38E3043D78CA8BC11029"),
				Amount:   big.NewInt(9999).Bytes(),
			},
		}))

		resp, err = th.Invoke("xkernel", "$evm", "SendRawTransaction", map[string][]byte{
			"signed_tx": signedTx,
		})
		if err != nil {
			t.Error(err)
			return
		}

		if hex.EncodeToString(resp.Body) != "6a59649d00a90b8333d7b188cb1fef3f940484eaac7844381da8c4cbc702a1d9" {
			t.Errorf("wrong tx hash:%s", hex.EncodeToString(resp.Body))
			return
		}

	})
	t.Run("GetTransactionReceipt", func(t *testing.T) {
		resp, err := th.Invoke("xkernel", "$evm", "GetTransactionReceipt", map[string][]byte{
			"tx_hash": []byte("tx_hash"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(resp.Body, signedTx) {
			t.Error(err)
			return
		}
	})

	t.Run("TxHash", func(t *testing.T) {

	})
	// t.Run("ContractCall", func(t *testing.T) {
	// 	resp, err = th.Invoke("xkernel", "$evm", "ContractCall", map[string][]byte{
	// 		"to":    []byte("313131312D2D2D2D2D2D2D2D2D636F756E746572"),
	// 		"from":  []byte("b60e8dd61c5d32be8058bb8eb970870f07233155"),
	// 		"input": []byte("ae896c870000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000678636861696e0000000000000000000000000000000000000000000000000000"),
	// 	})
	// 	if err != nil {
	// 		t.Error(err)
	// 		return
	// 	}
	// 	resp, err = th.Invoke("evm", "counter", "get", map[string][]byte{
	// 		"input":       []byte(`{"key":"xchain"}`),
	// 		"jsonEncoded": []byte("true"),
	// 	})
	// 	if err != nil {
	// 		t.Error(err)
	// 		return
	// 	}
	// 	if string(resp.Body) != `[{"0":"1"}]` {
	// 		t.Errorf("expect %s,get:%s", `[{"0":"1"}]`, string(resp.Body))
	// 		return
	// 	}
	//
	// })
	_ = resp
}

func TestVerifySignature(t *testing.T) {
	var nonce uint64 = 0
	var gasPrice uint64 = 0
	var gasLimit uint64 = 21000
	toString := "f97798df751deb4b6e39d4cf998ee7cd4dcb9acc"
	to, err := hex.DecodeString(toString)
	if err != nil {
		t.Error(err)
		return
	}
	valueStr := "0de0b6b3a7640000"
	value, err := hex.DecodeString(valueStr)
	if err != nil {
		t.Error(err)
		return
	}
	dataStr := ""
	data, err := hex.DecodeString(dataStr)
	if err != nil {
		t.Error(err)
		return
	}

	chainID := 1
	var V uint64 = 37
	net := uint64(chainID)
	RStr := "f0d2396973296cd6a71141c974d4a851f5eae8f08a8fba2dc36a0fef9bd6440c"
	R, err := hex.DecodeString(RStr)
	if err != nil {
		t.Error(err)
		return
	}
	SStr := "171995aa750d3f9f8e4d0eac93ff67634274f3c5acf422723f49ff09a6885422"
	S, err := hex.DecodeString(SStr)
	if err != nil {
		t.Error(err)
		return
	}
	p := &proxy{}
	if err := p.verifySignature(nonce, gasPrice, gasLimit, to, value, data, net, V, S, R); err != nil {
		t.Error(err)
		return
	}
}
