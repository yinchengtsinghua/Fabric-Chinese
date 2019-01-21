
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


package container

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/container/ccintf"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type VMProvider interface {
	NewVM() VM
}

type Builder interface {
	Build() (io.Reader, error)
}

//虚拟机是支持任意虚拟机的抽象虚拟映像
type VM interface {
	Start(ccid ccintf.CCID, args []string, env []string, filesToUpload map[string][]byte, builder Builder) error
	Stop(ccid ccintf.CCID, timeout uint, dontkill bool, dontremove bool) error
	HealthCheck(context.Context) error
}

type refCountedLock struct {
	refCount int
	lock     *sync.RWMutex
}

//vmcontroller-管理vms
//. abstract construction of different types of VMs (we only care about Docker for now)
//. 管理虚拟机的生命周期（从build开始，start，stop…
//最终可能需要细粒度管理）
type VMController struct {
	sync.RWMutex
	containerLocks map[string]*refCountedLock
	vmProviders    map[string]VMProvider
}

var vmLogger = flogging.MustGetLogger("container")

//new vmcontroller创建vmcontroller的新实例
func NewVMController(vmProviders map[string]VMProvider) *VMController {
	return &VMController{
		containerLocks: make(map[string]*refCountedLock),
		vmProviders:    vmProviders,
	}
}

func (vmc *VMController) newVM(typ string) VM {
	v, ok := vmc.vmProviders[typ]
	if !ok {
		vmLogger.Panicf("Programming error: unsupported VM type: %s", typ)
	}
	return v.NewVM()
}

func (vmc *VMController) lockContainer(id string) {
//将容器锁置于全局锁下
	vmc.Lock()
	var refLck *refCountedLock
	var ok bool
	if refLck, ok = vmc.containerLocks[id]; !ok {
		refLck = &refCountedLock{refCount: 1, lock: &sync.RWMutex{}}
		vmc.containerLocks[id] = refLck
	} else {
		refLck.refCount++
		vmLogger.Debugf("refcount %d (%s)", refLck.refCount, id)
	}
	vmc.Unlock()
	vmLogger.Debugf("waiting for container(%s) lock", id)
	refLck.lock.Lock()
	vmLogger.Debugf("got container (%s) lock", id)
}

func (vmc *VMController) unlockContainer(id string) {
	vmc.Lock()
	if refLck, ok := vmc.containerLocks[id]; ok {
		if refLck.refCount <= 0 {
			panic("refcnt <= 0")
		}
		refLck.lock.Unlock()
		if refLck.refCount--; refLck.refCount == 0 {
			vmLogger.Debugf("container lock deleted(%s)", id)
			delete(vmc.containerLocks, id)
		}
	} else {
		vmLogger.Debugf("no lock to unlock(%s)!!", id)
	}
	vmc.Unlock()
}

//vmcreq-所有请求都应该实现这个接口。
//上下文应该在每一层通过并测试，直到我们停止
//请注意，我们将在堆栈中第一个不
//取上下文
type VMCReq interface {
	Do(v VM) error
	GetCCID() ccintf.CCID
}

//StartContainerReq-用于启动容器的属性。
type StartContainerReq struct {
	ccintf.CCID
	Builder       Builder
	Args          []string
	Env           []string
	FilesToUpload map[string][]byte
}

//PlatformBuilder使用
//平台包生成了ckerbuild函数。
//这对建筑商来说是个相当尴尬的地方，应该是
//很可能会被推到码头控制中心，因为它只是
//创建Docker图像，但这样做需要污染
//带CD的DockerController软件包，也就是
//不受欢迎的
type PlatformBuilder struct {
	Type             string
	Path             string
	Name             string
	Version          string
	CodePackage      []byte
	PlatformRegistry *platforms.Registry
}

//基于CD构建tar流
func (b *PlatformBuilder) Build() (io.Reader, error) {
	return b.PlatformRegistry.GenerateDockerBuild(
		b.Type,
		b.Path,
		b.Name,
		b.Version,
		b.CodePackage,
	)
}

func (si StartContainerReq) Do(v VM) error {
	return v.Start(si.CCID, si.Args, si.Env, si.FilesToUpload, si.Builder)
}

func (si StartContainerReq) GetCCID() ccintf.CCID {
	return si.CCID
}

//StopContainerReq - properties for stopping a container.
type StopContainerReq struct {
	ccintf.CCID
	Timeout uint
//默认情况下，我们会在停止后杀死容器
	Dontkill bool
//默认情况下，我们会在杀死后移除容器
	Dontremove bool
}

func (si StopContainerReq) Do(v VM) error {
	return v.Stop(si.CCID, si.Timeout, si.Dontkill, si.Dontremove)
}

func (si StopContainerReq) GetCCID() ccintf.CCID {
	return si.CCID
}

func (vmc *VMController) Process(vmtype string, req VMCReq) error {
	v := vmc.newVM(vmtype)
	ccid := req.GetCCID()
	id := ccid.GetName()

	vmc.lockContainer(id)
	defer vmc.unlockContainer(id)
	return req.Do(v)
}

//GetChaincodePackageBytes creates bytes for docker container generation using the supplied chaincode specification
func GetChaincodePackageBytes(pr *platforms.Registry, spec *pb.ChaincodeSpec) ([]byte, error) {
	if spec == nil || spec.ChaincodeId == nil {
		return nil, fmt.Errorf("invalid chaincode spec")
	}

	return pr.GetDeploymentPayload(spec.CCType(), spec.Path())
}
