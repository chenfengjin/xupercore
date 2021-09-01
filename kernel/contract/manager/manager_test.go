package manager

import (
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	"github.com/xuperchain/xupercore/kernel/contract/mock"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
)

var contractConfig = &contract.ContractConfig{
	Xkernel: contract.XkernelConfig{
		Enable: true,
		Driver: "default",
	},
	LogDriver: mock.NewMockLogger(),
}

func TestCreate(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
}

func TestCreateSandbox(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()

	r := sandbox.NewMemXModel()
	state, err := m.NewStateSandbox(&contract.SandboxConfig{
		XMReader: r,
	})
	if err != nil {
		t.Fatal(err)
	}
	state.Put("test", []byte("key"), []byte("value"))
	if string(state.RWSet().WSet[0].Value) != "value" {
		t.Error("unexpected value")
	}
}

func TestInvoke(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()

	m.GetKernRegistry().RegisterKernMethod("$hello", "Hi", new(helloContract).Hi)

	resp, err := th.Invoke("xkernel", "$hello", "Hi", map[string][]byte{
		"name": []byte("xuper"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", resp.Body)
}

type helloContract struct {
}

func TestEmbededContractRegister(t *testing.T) {

	t.Run("TestRegisterOK", func(t *testing.T) {
		contract.RegisterEmbededContract("hi", helloContractCreator)
		contract.GetEmbededContractCretorFunc("hi")
	})
	t.Run("TestRegisterFail", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expect panic, but it did not")
			}
		}()
		contract.RegisterEmbededContract("failed", nullContractCretor)
	})
}

type nullContract struct {
}

func nullContractCretor() nullContract {
	return nullContract{}
}

type helloContractConfig struct {
}

func helloContractCreator(config helloContractConfig) contract.EmbededContract {
	return helloContract{}
}
func (h *helloContract) Hi(ctx contract.KContext) (*contract.Response, error) {
	name := ctx.Args()["name"]
	ctx.Put("test", []byte("k1"), []byte("v1"))
	return &contract.Response{
		Body: []byte("hello " + string(name)),
	}, nil
}

// type InstanceCreator interface {
// 	CreateInstance(...interface{})
// }
// type B struct {
// }
//
// func A(creator InstanceCreator) {
//
// }
//
// func (b *B) CreateInstance() {

// }
// func TestC(t *testing.T) {
// 	A(&B{})
// }

// type A struct {
// 	Name string
// }
//
// func TestReflect(t *testing.T) {
// 	reflectNew((*A)(nil))
// }
//
// //反射创建新对象。
// func reflectNew(target interface{}) {
// 	// orig := new(Employee)
// 	//
// 	// t := reflect.TypeOf(orig)
// 	// v := reflect.New(t.Elem())
// 	//
// 	// reflected pointer
// 	// newP := v.Interface()
// 	//
// 	// Unmarshal to reflected struct pointer
// 	// json.Unmarshal([]byte("{\"firstname\": \"bender\"}"), newP)
// 	//
// 	// fmt.Printf("%+v\n", newP)
//
// 	// if target == nil {
// 	// 	// fmt.Println("参数不能未空")
// 	// 	return
// 	// }
// 	//
// 	// t := reflect.TypeOf(target)
// 	// if t.Kind() == reflect.Ptr { //指针类型获取真正type需要调用Elem
// 	// 	t = t.Elem()
// 	// }
// 	//
// 	// newStruc := reflect.New(t)                            // 调用反射创建对象
// 	// newStruc.Elem().FieldByName("Name").SetString("Lily") //设置值
// 	//
// 	// newVal := newStruc.Elem().FieldByName("Name") //获取值
// 	//
// 	// fmt.Println(newVal.String())
// }
