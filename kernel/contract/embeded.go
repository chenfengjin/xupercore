package contract

import (
	"fmt"
	"sync"
)

type EmbededContractRegistry interface {
	RegisterEmbededCreatorFunc(contract string, f EmbededContractCreatorFunc, configPath string)
}
type EmbededContractCreatorFunc func(interface{}) EmbededContract

type EmbededContract interface {
}

// precompiledRegistry 是上下文无关的，注册为全局对象
type precompiledRegistry struct {
	mutex sync.Mutex
	funcs map[string]EmbededContractCreatorFunc
}

func (r *precompiledRegistry) registerEmbededContract(name string, f EmbededContractCreatorFunc) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.funcs[name]; ok {
		//  just panic as it happens at startup
		panic(fmt.Sprint("contract %s alread exist", name))
	}
	r.funcs[name] = f
}

func (r *precompiledRegistry) getEmbededContratCreatorFunc(name string) (EmbededContractCreatorFunc, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.funcs[name], true
}

var (
	defaultRegistry = &precompiledRegistry{
		mutex: sync.Mutex{},
		funcs: map[string]EmbededContractCreatorFunc{},
	}
)

func RegisterEmbededContractCreatorFunc(contract string, creator EmbededContractCreatorFunc) {
	defaultRegistry.registerEmbededContract(contract, creator)
}

func GetEmbededContractCretorFunc(name string) (EmbededContractCreatorFunc, bool) {
	return defaultRegistry.getEmbededContratCreatorFunc(name)
}
