package contract

import (
	"sync"
)

type ObjectInstanceCreator interface {
	CreateInstance(string) PrecompiledContract
	// 模块之间会有依赖，预编译合约允许用户声明自己的依赖哪些内核组件
	// eg: evm proxy 依赖 evm 合约支持
	// Dependencies()
}
type Dependencies struct {
}
type PrecompiledContract interface {
	Enabled() bool
}

// PrecompiledRegistry 是上下文无关的，注册为全局对象
type PrecompiledRegistry struct {
	mutex   sync.Mutex
	objects map[string]PrecompiledContract
}

func (r *PrecompiledRegistry) RegisterKernelObject(contract string, f ObjectInstanceCreatorFunc, configPath string) {
	creator := f()
	//  限制配置文件必须和 contract 一致，避免不同 Object 相同的config 配置
	instance := creator.CreateInstance(contract + ".yaml")
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.objects[contract] = instance
}

func (r *PrecompiledRegistry) ListObjects() map[string]PrecompiledContract {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.objects
}

var (
	defaultRegistry = &PrecompiledRegistry{
		mutex:   sync.Mutex{},
		objects: map[string]PrecompiledContract{},
	}
)

func RegisterKernelObject(contract string, creator ObjectInstanceCreatorFunc, configPath string) {
	defaultRegistry.RegisterKernelObject(contract, creator, configPath)
}

type ObjectInstanceCreatorFunc func() ObjectInstanceCreator
