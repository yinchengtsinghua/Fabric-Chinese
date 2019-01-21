
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司。保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package inproccontroller

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/container"
	"github.com/hyperledger/fabric/core/container/ccintf"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//ContainerType是InProc容器类型的字符串
//已在container.vmcontroller中注册
const ContainerType = "SYSTEM"

type inprocContainer struct {
	ChaincodeSupport ccintf.CCSupport
	chaincode        shim.Chaincode
	running          bool
	args             []string
	env              []string
	stopChan         chan struct{}
}

var (
	inprocLogger = flogging.MustGetLogger("inproccontroller")

//TODO这是一种非常简单的测试方法，我们应该找到其他方法
//测试或不静态注入这些依赖项。
	_shimStartInProc    = shim.StartInProc
	_inprocLoggerErrorf = inprocLogger.Errorf
)

//错误

//SyscRegistereder注册错误
type SysCCRegisteredErr string

func (s SysCCRegisteredErr) Error() string {
	return fmt.Sprintf("%s already registered", string(s))
}

//注册表存储注册的系统链代码。
//它实现了container.vmprovider和scc.registrar
type Registry struct {
	typeRegistry     map[string]*inprocContainer
	instRegistry     map[string]*inprocContainer
	ChaincodeSupport ccintf.CCSupport
}

//NewRegistry创建一个初始化的注册表，准备注册系统链代码。
//返回的*注册表没有准备好按原样使用。必须设置chaincode支持
//在任何链码调用发生之前，只要有一个可用的。这是因为
//链码支持以前是一种潜在的依赖关系，悄悄地出现在上下文中，但现在
//它正成为启动的一个明确部分。
func NewRegistry() *Registry {
	return &Registry{
		typeRegistry: make(map[string]*inprocContainer),
		instRegistry: make(map[string]*inprocContainer),
	}
}

//newvm创建一个inproc vm实例
func (r *Registry) NewVM() container.VM {
	return NewInprocVM(r)
}

//寄存器用给定的路径注册系统链码。应调用部署以初始化
func (r *Registry) Register(ccid *ccintf.CCID, cc shim.Chaincode) error {
	name := ccid.GetName()
	inprocLogger.Debugf("Registering chaincode instance: %s", name)
	tmp := r.typeRegistry[name]
	if tmp != nil {
		return SysCCRegisteredErr(name)
	}

	r.typeRegistry[name] = &inprocContainer{chaincode: cc}
	return nil
}

//inprocvm是一个vm。它由可执行文件名标识
type InprocVM struct {
	id       string
	registry *Registry
}

//NeWiNeCuPvm创建新的InPoCVM
func NewInprocVM(r *Registry) *InprocVM {
	return &InprocVM{
		registry: r,
	}
}

func (vm *InprocVM) getInstance(ipctemplate *inprocContainer, instName string, args []string, env []string) (*inprocContainer, error) {
	ipc := vm.registry.instRegistry[instName]
	if ipc != nil {
		inprocLogger.Warningf("chaincode instance exists for %s", instName)
		return ipc, nil
	}
	ipc = &inprocContainer{
		ChaincodeSupport: vm.registry.ChaincodeSupport,
		args:             args,
		env:              env,
		chaincode:        ipctemplate.chaincode,
		stopChan:         make(chan struct{}),
	}
	vm.registry.instRegistry[instName] = ipc
	inprocLogger.Debugf("chaincode instance created for %s", instName)
	return ipc, nil
}

func (ipc *inprocContainer) launchInProc(id string, args []string, env []string) error {
	if ipc.ChaincodeSupport == nil {
		inprocLogger.Panicf("Chaincode support is nil, most likely you forgot to set it immediately after calling inproccontroller.NewRegsitry()")
	}

	peerRcvCCSend := make(chan *pb.ChaincodeMessage)
	ccRcvPeerSend := make(chan *pb.ChaincodeMessage)
	var err error
	ccchan := make(chan struct{}, 1)
	ccsupportchan := make(chan struct{}, 1)
shimStartInProc := _shimStartInProc //在测试中避免比赛的阴影
	go func() {
		defer close(ccchan)
		inprocLogger.Debugf("chaincode started for %s", id)
		if args == nil {
			args = ipc.args
		}
		if env == nil {
			env = ipc.env
		}
		err := shimStartInProc(env, args, ipc.chaincode, ccRcvPeerSend, peerRcvCCSend)
		if err != nil {
			err = fmt.Errorf("chaincode-support ended with err: %s", err)
			_inprocLoggerErrorf("%s", err)
		}
		inprocLogger.Debugf("chaincode ended for %s with err: %s", id, err)
	}()

//避免数据争用的阴影函数
	inprocLoggerErrorf := _inprocLoggerErrorf
	go func() {
		defer close(ccsupportchan)
		inprocStream := newInProcStream(peerRcvCCSend, ccRcvPeerSend)
		inprocLogger.Debugf("chaincode-support started for  %s", id)
		err := ipc.ChaincodeSupport.HandleChaincodeStream(inprocStream)
		if err != nil {
			err = fmt.Errorf("chaincode ended with err: %s", err)
			inprocLoggerErrorf("%s", err)
		}
		inprocLogger.Debugf("chaincode-support ended for %s with err: %s", id, err)
	}()

	select {
	case <-ccchan:
		close(peerRcvCCSend)
		inprocLogger.Debugf("chaincode %s quit", id)
	case <-ccsupportchan:
		close(ccRcvPeerSend)
		inprocLogger.Debugf("chaincode support %s quit", id)
	case <-ipc.stopChan:
		close(ccRcvPeerSend)
		close(peerRcvCCSend)
		inprocLogger.Debugf("chaincode %s stopped", id)
	}
	return err
}

//开始启动以前注册的系统编解码器链接
func (vm *InprocVM) Start(ccid ccintf.CCID, args []string, env []string, filesToUpload map[string][]byte, builder container.Builder) error {
	path := ccid.GetName()

	ipctemplate := vm.registry.typeRegistry[path]

	if ipctemplate == nil {
		return fmt.Errorf(fmt.Sprintf("%s not registered", path))
	}

	instName := vm.GetVMName(ccid)

	ipc, err := vm.getInstance(ipctemplate, instName, args, env)

	if err != nil {
		return fmt.Errorf(fmt.Sprintf("could not create instance for %s", instName))
	}

	if ipc.running {
		return fmt.Errorf(fmt.Sprintf("chaincode running %s", path))
	}

	ipc.running = true

	go func() {
		defer func() {
			if r := recover(); r != nil {
				inprocLogger.Criticalf("caught panic from chaincode  %s", instName)
			}
		}()
		ipc.launchInProc(instName, args, env)
	}()

	return nil
}

//停止停止系统编解码器链接
func (vm *InprocVM) Stop(ccid ccintf.CCID, timeout uint, dontkill bool, dontremove bool) error {
	path := ccid.GetName()

	ipctemplate := vm.registry.typeRegistry[path]
	if ipctemplate == nil {
		return fmt.Errorf("%s not registered", path)
	}

	instName := vm.GetVMName(ccid)

	ipc := vm.registry.instRegistry[instName]

	if ipc == nil {
		return fmt.Errorf("%s not found", instName)
	}

	if !ipc.running {
		return fmt.Errorf("%s not running", instName)
	}

	ipc.stopChan <- struct{}{}

	delete(vm.registry.instRegistry, instName)
//停站
	return nil
}

//为了实现vmprovider接口，提供了healthcheck。
//它总是返回零……
func (vm *InprocVM) HealthCheck(ctx context.Context) error {
	return nil
}

//getvmname忽略对等名和网络名，因为它只需要在
//过程。它接受格式函数参数以允许
//基于所需名称使用的格式。
func (vm *InprocVM) GetVMName(ccid ccintf.CCID) string {
	return ccid.GetName()
}
