package native

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	log15 "github.com/xuperchain/log15"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pbrpc"
	"github.com/xuperchain/xupercore/protos"

	"google.golang.org/grpc"
)

type contractProcess struct {
	cfg *contract.NativeConfig

	name      string
	basedir   string
	binpath   string
	chainAddr string
	desc      *protos.WasmCodeDesc

	process       Process
	IPaddress     string
	monitorStopch chan struct{}
	monitorWaiter sync.WaitGroup
	logger        log15.Logger

	mutex     sync.Mutex
	rpcAddr   string
	rpcPort   int
	rpcConn   *grpc.ClientConn
	rpcClient pbrpc.NativeCodeClient
}

func newContractProcess(cfg *contract.NativeConfig, name, basedir, chainAddr string, desc *protos.WasmCodeDesc) (*contractProcess, error) {
	process := &contractProcess{
		cfg:           cfg,
		name:          name,
		basedir:       basedir,
		binpath:       filepath.Join(basedir, nativeCodeFileName(desc)),
		chainAddr:     chainAddr,
		desc:          desc,
		monitorStopch: make(chan struct{}),
		logger:        log15.New(),
		//logger:        log.DefaultLogger.New("contract", name),
	}
	return process, nil
}

func (c *contractProcess) makeNativeProcess() (Process, error) {
	envs := []string{
		"XCHAIN_CODE_PORT=" + strconv.Itoa(37103),
		"XCHAIN_CHAIN_ADDR=" + "tcp://10.12.200.56:37102",
	}
	startcmd, err := c.makeStartCommand()
	if err != nil {
		return nil, err
	}
	if !c.cfg.Docker.Enable {
		return &HostProcess{
			basedir:  c.basedir,
			startcmd: startcmd,
			envs:     envs,
			Logger:   c.logger,
		}, nil
	}
	mounts := []string{
		c.basedir,
	}
	return &DockerProcess{
		basedir:  c.basedir,
		startcmd: startcmd,
		envs:     envs,
		mounts:   mounts,
		// ports:    []string{strconv.Itoa(c.rpcPort)},
		cfg:    &c.cfg.Docker,
		Logger: c.logger,
	}, nil
}

// wait the subprocess to be ready
func (c *contractProcess) waitReply() error {
	const waitTimeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.TODO(), waitTimeout)
	defer cancel()
	for {
		_, err := c.rpcClient.Ping(ctx, new(pb.PingRequest))
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting native code start timeout. error:%s", err)
		default:
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func (c *contractProcess) heartBeat() error {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	_, err := c.rpcClient.Ping(ctx, new(pb.PingRequest))
	return err
}

func (c *contractProcess) monitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	c.monitorWaiter.Add(1)
	defer c.monitorWaiter.Done()
forloop:
	for {
		select {
		case <-c.monitorStopch:
			return
		case <-ticker.C:
			err := c.heartBeat()
			if err == nil {
				continue forloop
			}
			c.logger.Error("process heartbeat error", "error", err)
			err = c.restartProcess()
			if err != nil {
				c.logger.Error("restart process error", "error", err)
			}
		}
	}
}

func (c *contractProcess) resetRpcClient() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.rpcConn != nil {
		c.rpcConn.Close()
	}
	port, err := makeFreePort()
	if err != nil {
		return err
	}
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", c.IPaddress, 37103), grpc.WithInsecure())
	if err != nil {
		return err
	}
	c.rpcPort = port
	c.rpcConn = conn
	c.rpcClient = pbrpc.NewNativeCodeClient(c.rpcConn)
	return nil
}

func (c *contractProcess) RpcClient() pbrpc.NativeCodeClient {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.rpcClient
}

func (c *contractProcess) restartProcess() error {
	c.process.Stop(time.Second)
	return c.start(false)
}

func (c *contractProcess) start(startMonitor bool) error {
	var err error

	c.process, err = c.makeNativeProcess()
	if err != nil {
		return err
	}

	IPAddress, err := c.process.Start()
	if err != nil {
		return err
	}
	c.IPaddress = IPAddress
	err = c.resetRpcClient()
	if err != nil {
		return err
	}

	err = c.waitReply()
	if err != nil {
		// 避免启动失败后产生僵尸进程
		c.process.Stop(time.Second)
		return err
	}
	if startMonitor {
		go c.monitor()
	}

	return nil
}

func (c *contractProcess) Start() (string, error) {
	return "127.0.0.1", c.start(true)
}

func (c *contractProcess) Stop() {
	// close monitor and waiting monitor stoped
	close(c.monitorStopch)
	c.monitorWaiter.Wait()

	err := c.process.Stop(time.Second)
	if err != nil {
		c.logger.Error("process stoped error", "error", err)
	}
}

func (c *contractProcess) GetDesc() *protos.WasmCodeDesc {
	return c.desc
}

func (c *contractProcess) makeStartCommand() (*exec.Cmd, error) {
	switch c.desc.GetRuntime() {
	case "java":
		return exec.Command("java", "-jar", c.binpath), nil
	case "go":
		return exec.Command(c.binpath), nil
	default:
		return nil, fmt.Errorf("unsupported native contract runtime %s", c.desc.GetRuntime())
	}
}

func makeFreePort() (int, error) {
	// l, err := net.Listen("tcp", "127.0.0.1:0")
	// if err != nil {
	// 	return 0, err
	// }
	// addr := l.Addr().(*net.TCPAddr)
	// l.Close()
	// return addr.Port, nil
	return 37102, nil
}
