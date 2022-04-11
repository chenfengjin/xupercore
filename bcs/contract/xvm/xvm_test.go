package xvm

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/mock"

	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	_ "github.com/xuperchain/xupercore/kernel/contract/manager"
)

// aaaaa

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
			OptLevel: 2,
		},
	},
	LogDriver: mock.NewMockLogger(),
}

func BenchmarkWasmDeploy(t *testing.B) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
	bin, err := ioutil.ReadFile("/Users/chenfengjin/baidu/xuperchain/auto/counter.wasm")
	if err != nil {
		panic(err)
	}

	_, err = th.Deploy("wasm", "c", fmt.Sprintf("counter"), bin, map[string][]byte{
		"creator": []byte("icexin"),
	})

	if err != nil {
		t.Fatal(err)
	}

	t.ResetTimer()

	for i := 0; i < t.N; i++ {
		_, err = th.Invoke("wasm", "counter", "increase", map[string][]byte{
			"key": []byte("xchain"),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// 4305988964
	// t.Log(24808886 / time.Millisecond)
	// invoke
	// xvm opt0 222781    -> 0.22ms
	// xvm opt3 138032    -> 0.13 ms
	// ixvm 18102673 -> 18ms
}

// func TestNativeInvoke(t *testing.T) {
// th := mock.NewTestHelper(contractConfig)
// defer th.Close()
//
// // bin, err := compile(th)
// // if err != nil {
// // 	t.Fatal(err)
// // }
//
// _, err := th.Deploy("native", "go", "counter", bin, map[string][]byte{
// 	"creator": []byte("icexin"),
// })
// if err != nil {
// 	t.Fatal(err)
// }
//
// resp, err := th.Invoke("native", "counter", "increase", map[string][]byte{
// 	"key": []byte("k1"),
// })
// if err != nil {
// 	t.Fatal(err)
// }
//
// t.Logf("body:%s", resp.Body)
// }
