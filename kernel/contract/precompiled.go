package contract

import (
	"sync"
)

type EmbededContractRegistry interface {
	RegisterEmbededCreatorFunc(contract string, f EmbededContractCreatorFunc, configPath string)
}
type EmbededContractCreatorFunc func(interface{}) EmbededContract

type EmbededContract interface {
}

// type ObjectInstanceCreator interface {
// 	CreateInstance(string) EmbededContractCreatorFunc
// 	模块之间会有依赖，预编译合约允许用户声明自己的依赖哪些内核组件
// 	eg: evm proxy 依赖 evm 合约支持
// 	Dependencies()
// }

// type PrecompiledContract interface {
// 	Enabled() bool
// }
type Dependencies struct {
}

// precompiledRegistry 是上下文无关的，注册为全局对象
type precompiledRegistry struct {
	mutex   sync.Mutex
	objects map[string]EmbededContractCreatorFunc
}

func (r *precompiledRegistry) RegisterEmbededCreatorFunc(contract string, f EmbededContractCreatorFunc) {
	// creator := f()
	//  限制配置文件必须和 contract 一致，避免不同 Object 相同的config 配置
	// instance := creator.CreateInstance(contract + ".yaml")
	// r.mutex.Lock()
	// defer r.mutex.Unlock()
	// r.objects[contract] = instance
}

func (r *precompiledRegistry) GetEmbededContratCreatorFunc(name string) (EmbededContractCreatorFunc, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.objects[name], true
}

var (
	defaultRegistry = &precompiledRegistry{
		mutex:   sync.Mutex{},
		objects: map[string]EmbededContractCreatorFunc{},
	}
)

func RegisterEmbededContractCreatorFunc(contract string, creator EmbededContractCreatorFunc) {
	defaultRegistry.RegisterEmbededCreatorFunc(contract, creator)
}

func GetEmbededContractCretorFunc(name string) (EmbededContractCreatorFunc, bool) {
	return defaultRegistry.GetEmbededContratCreatorFunc(name)
}

type PreCompiledConf map[string]PreCompiledConfItem

type PreCompiledConfItem map[string]string
