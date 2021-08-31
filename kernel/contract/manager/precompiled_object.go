package manager

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"sync"
)

// PrecompiledRegistry 是上下文无关的，注册为全局对象
type PrecompiledRegistry struct {
	mutex   sync.Mutex
	objects map[string]contract.KernelObject
}

func (r *PrecompiledRegistry) RegisterKernelObject(contract string, creator contract.ObjectInstanceCreator, configPath string) {
	//  限制配置文件必须和 contract 一致，避免不同 Object 相同的config 配置
	instance := creator.CreateInstance(contract + ".yaml")
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.objects[contract] = instance
}

func (r *PrecompiledRegistry) ListObjects() map[string]contract.KernelObject {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.objects
}

var (
	defaultRegistry = &PrecompiledRegistry{
		mutex:   sync.Mutex{},
		objects: map[string]contract.KernelObject{},
	}
)

// use defaultRegistry to keep global clean
func RegisterKernelObject(contract string, creator contract.ObjectInstanceCreator, configPath string) {
	defaultRegistry.RegisterKernelObject(contract, creator, configPath)
}
