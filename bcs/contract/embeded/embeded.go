package embeded

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
)

type embededVM struct {
	registry contract.KernRegistry
	config   *bridge.InstanceCreatorConfig
}

func newEmbededVM(config *bridge.InstanceCreatorConfig) (bridge.InstanceCreator, error) {
	return &embededVM{
		registry: config.VMConfig.(*contract.XkernelConfig).Registry,
		config:   config,
	}, nil
}

// CreateInstance instances a wasm virtual machine instance which can run a single contract call
func (k *embededVM) CreateInstance(ctx *bridge.Context, cp bridge.ContractCodeProvider) (bridge.Instance, error) {
	return newEmbededInstance(ctx, k.config.SyscallService, k.registry), nil
}

func (k *embededVM) RemoveCache(name string) {
}

type embededInstance struct {
	ctx      *bridge.Context
	kctx     *Context
	registry contract.KernRegistry
}

func newEmbededInstance(ctx *bridge.Context, syscall *bridge.SyscallService, registry contract.KernRegistry) *embededInstance {
	return &embededInstance{
		ctx:      ctx,
		kctx:     newKContext(ctx, syscall),
		registry: registry,
	}
}

func (k *embededInstance) Exec() error {
	method, err := k.registry.GetKernMethod(k.ctx.ContractName, k.ctx.Method)
	if err != nil {
		return err
	}

	resp, err := method(k.kctx)
	if err != nil {
		return err
	}
	k.ctx.Output = &pb.Response{
		Status:  int32(resp.Status),
		Message: resp.Message,
		Body:    resp.Body,
	}
	return nil
}

func (k *embededInstance) ResourceUsed() contract.Limits {
	return k.kctx.used
}

func (k *embededInstance) Release() {
}

func (k *embededInstance) Abort(msg string) {
}

func init() {
	bridge.Register("embeded", "default", newEmbededVM)
}
