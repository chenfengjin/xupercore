package contract

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	typeEmbededContract = reflect.TypeOf(new(EmbededContract)).Elem()
)

type EmbededContractFunc func(KContext) (Response, error)
type EmbededContractRegistry interface {
	RegisterEmbededCreatorFunc(contract string, f EmbededContractCreator, configPath string)
	RegisterEmbededContractFunc(contract string, contractFunc EmbededContractFunc)
}

// type EmbededContractCreator func(...interface{}) EmbededContract
type EmbededContractCreator interface{}

// type EmbededContractCreator interface{}
type EmbededContract interface {
}

// precompiledRegistry 是上下文无关的，注册为全局对象
type precompiledRegistry struct {
	mutex sync.Mutex
	funcs map[string]EmbededContractCreator
}

func (r *precompiledRegistry) registerEmbededContract(name string, f EmbededContractCreator) {
	fv := reflect.TypeOf(f)
	numOut := fv.NumOut()
	if numOut > 1 {
		//  just panic as it happens at startup
		panic("EmbededContractCreator must return and only return EmbededContract")
	}
	out := fv.Out(0)
	if out != typeEmbededContract {
		panic("EmbededContractCreator must return EmbededContract")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.funcs[name]; ok {
		//  just panic as it happens at startup
		panic(fmt.Sprint("contract %s alread exist", name))
	}
	r.funcs[name] = f
}

func (r *precompiledRegistry) getEmbededContratCreatorFunc(name string) EmbededContractCreator {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.funcs[name]
}

var (
	defaultRegistry = &precompiledRegistry{
		mutex: sync.Mutex{},
		funcs: map[string]EmbededContractCreator{},
	}
)

func RegisterEmbededContract(contract string, creator EmbededContractCreator) {
	defaultRegistry.registerEmbededContract(contract, creator)
}

func GetEmbededContractCretorFunc(name string) EmbededContractCreator {
	return defaultRegistry.getEmbededContratCreatorFunc(name)
}
