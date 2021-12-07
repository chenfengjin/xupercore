package evm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	"github.com/xuperchain/xupercore/lib/crypto/hash"
)

const (
	evmAddressFiller = "-"

	contractNamePrefixs    = "1111"
	contractAccountPrefixs = "1112"

	typeXchainAddr = "xchain"

	typeContractName    = "contract-name"
	typeContractAccount = "contract-account"
)

// transfer xchain address to evm address
func XchainToEVMAddress(addr string) (crypto.Address, error) {
	rawAddr := base58.Decode(addr)
	ripemd160Hash := rawAddr[1:21]
	return crypto.AddressFromBytes(ripemd160Hash)
}

// transfer evm address to xchain address
func EVMAddressToXchain(evmAddress crypto.Address) (string, error) {
	addrType := 1
	nVersion := uint8(addrType)
	bufVersion := []byte{byte(nVersion)}

	outputRipemd160 := evmAddress.Bytes()

	strSlice := make([]byte, len(bufVersion)+len(outputRipemd160))
	copy(strSlice, bufVersion)
	copy(strSlice[len(bufVersion):], outputRipemd160)

	checkCode := hash.DoubleSha256(strSlice)
	simpleCheckCode := checkCode[:4]
	slice := make([]byte, len(strSlice)+len(simpleCheckCode))
	copy(slice, strSlice)
	copy(slice[len(strSlice):], simpleCheckCode)

	return base58.Encode(slice), nil
}

// transfer contract name to evm address
func ContractNameToEVMAddress(contractName string) (crypto.Address, error) {
	contractNameLength := len(contractName)
	var prefixStr string
	for i := 0; i < binary.Word160Length-contractNameLength-utils.GetContractNameMinSize(); i++ {
		prefixStr += evmAddressFiller
	}
	contractName = prefixStr + contractName
	contractName = contractNamePrefixs + contractName
	return crypto.AddressFromBytes([]byte(contractName))
}

// transfer evm address to contract name
func EVMAddressToContractName(evmAddr crypto.Address) (string, error) {
	contractNameWithPrefix := evmAddr.Bytes()
	contractNameStrWithPrefix := string(contractNameWithPrefix)
	prefixIndex := strings.LastIndex(contractNameStrWithPrefix, evmAddressFiller)
	if prefixIndex == -1 {
		return contractNameStrWithPrefix[4:], nil
	}
	return contractNameStrWithPrefix[prefixIndex+1:], nil
}

// transfer contract account to evm address
func ContractAccountToEVMAddress(contractAccount string) (crypto.Address, error) {
	contractAccountValid := contractAccount[2:18]
	contractAccountValid = contractAccountPrefixs + contractAccountValid
	return crypto.AddressFromBytes([]byte(contractAccountValid))
}

// transfer evm address to contract account
func EVMAddressToContractAccount(evmAddr crypto.Address) (string, error) {
	contractNameWithPrefix := evmAddr.Bytes()
	contractNameStrWithPrefix := string(contractNameWithPrefix)
	return "XC" + contractNameStrWithPrefix[4:], nil
}

// determine whether it is a contract account
func DetermineContractAccount(account string) bool {
	reg, err := regexp.Compile("XC\\d{16}")
	if err != nil {
		return false
	}
	matched := reg.MatchString(account)
	return matched
}

// // determine whether it is a contract name
func determineContractName(contractName string) error {
	return contract.ValidContractName(contractName)
}

// determine whether it is a contract name
func DetermineContractNameFromEVM(evmAddr crypto.Address) (string, error) {
	var addr string
	var err error

	evmAddrWithPrefix := evmAddr.Bytes()
	evmAddrStrWithPrefix := string(evmAddrWithPrefix)
	if evmAddrStrWithPrefix[0:4] != contractNamePrefixs {
		return "", fmt.Errorf("not a valid contract name from evm")
	} else {
		addr, err = EVMAddressToContractName(evmAddr)
	}

	if err != nil {
		return "", err
	}

	return addr, nil
}

// determine an EVM address
func DetermineEVMAddress(evmAddr crypto.Address) (string, string, error) {
	evmAddrWithPrefix := evmAddr.Bytes()
	evmAddrStrWithPrefix := string(evmAddrWithPrefix)

	var addr, addrType string
	var err error
	if evmAddrStrWithPrefix[0:4] == contractAccountPrefixs {
		addr, err = EVMAddressToContractAccount(evmAddr)
		addrType = typeContractAccount
	} else if evmAddrStrWithPrefix[0:4] == contractNamePrefixs {
		addr, err = EVMAddressToContractName(evmAddr)
		addrType = typeContractName
	} else {
		addr, err = EVMAddressToXchain(evmAddr)
		addrType = typeXchainAddr
	}
	if err != nil {
		return "", "", err
	}

	return addr, addrType, nil
}

func xchainAddressType(addr string) (string, error) {
	if DetermineContractAccount(addr) {
		return typeContractAccount, nil
	}
	if err := determineContractName(addr); err == nil {
		return typeContractName, nil
	}
	if err := DetermineXchainKeyAddress(addr); err == nil {
		return typeXchainAddr, nil
	} else {
		return "", fmt.Errorf("bad address %w", err)
	}
}

func DetermineXchainKeyAddress(addr string) error {
	rawAddr := base58.Decode(addr)
	if len(rawAddr) < 21 {
		return fmt.Errorf("%s is not a valid address", addr)
	}
	return nil
}

// determine an xchain address
func DetermineXchainAddress(xAddr string) (string, string, error) {
	var addr crypto.Address
	var err error
	addrType, err := xchainAddressType(xAddr)
	if err != nil {
		return "", "", err
	}

	switch addrType {
	case typeContractAccount:
		addr, err = ContractAccountToEVMAddress(xAddr)
	case typeContractName:
		addr, err = ContractNameToEVMAddress(xAddr)
	case typeXchainAddr:
		addr, err = XchainToEVMAddress(xAddr)
	}
	if err != nil {
		return "", "", err
	}

	return addr.String(), addrType, nil
}
