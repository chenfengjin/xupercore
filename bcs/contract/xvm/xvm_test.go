package xvm

import (
	"io/ioutil"
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	_ "github.com/xuperchain/xupercore/kernel/contract/manager"

	"github.com/xuperchain/xupercore/kernel/contract/mock"
)

var contractConfig = &contract.ContractConfig{
	EnableUpgrade: true,
	Xkernel: contract.XkernelConfig{
		Enable: true,
		Driver: "default",
	},
	Wasm: contract.WasmConfig{
		Enable: true,
		Driver: "xvm",
		XVM: contract.XVMConfig{
			OptLevel: 0,
		},
	},
	LogDriver: mock.NewMockLogger(),
}

func BenchmarkXVMDeploy(b *testing.B) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()

	bin, err := ioutil.ReadFile("/Users/chenfengjin/baidu/contract-sdk-cpp/build/counter.wasm")
	if err != nil {
		b.Fatal(err)
	}
	_, err = th.Deploy("wasm", "c", "counter", bin, map[string][]byte{
		"creator": []byte("icexin"),
	})
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := th.Invoke("wasm", "counter", "increase", map[string][]byte{
			"key": []byte("xchain"),
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
