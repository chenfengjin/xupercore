package contract

type KernRegistry interface {
	RegisterKernMethod(contract, method string, handler KernMethod)
	// RegisterShortcut 用于contractName缺失的时候选择哪个合约名字和合约方法来执行对应的kernel合约
	RegisterShortcut(oldmethod, contract, method string)
	// 注册一个对象上的所有kern方法，修改此对象即可
	RegisterKernelObject(contract string, object ObjectInstanceCreator, configPath string)
	GetKernMethod(contract, method string) (KernMethod, error)
}

type ObjectInstanceCreator interface {
	CreateInstance(configPaukomah string) KernelObject
	// 模块之间会有依赖
	// Dependencies()
}
type Dependencies struct {
}
type KernelObject interface {
	Enabled() bool
}

type KernMethod func(ctx KContext) (*Response, error)

type KContext interface {
	// 交易相关数据
	Args() map[string][]byte
	Initiator() string
	Caller() string
	AuthRequire() []string
	TransferAmount() string
	ContractName() string
	// 状态修改接口
	StateSandbox

	AddResourceUsed(delta Limits)
	ResourceLimit() Limits

	Call(module, contract, method string, args map[string][]byte) (*Response, error)

	// 合约异步事件调用
	EmitAsyncTask(event string, args interface{}) error
}
