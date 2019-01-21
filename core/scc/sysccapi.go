
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


package scc

import (
	"errors"
	"fmt"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/container/ccintf"
	"github.com/hyperledger/fabric/core/container/inproccontroller"
	"github.com/hyperledger/fabric/core/peer"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
)

var sysccLogger = flogging.MustGetLogger("sccapi")

//注册器提供了一种注册系统链码的方法
type Registrar interface {
//寄存器注册系统链码
	Register(ccid *ccintf.CCID, cc shim.Chaincode) error
}

//系统链代码定义初始化系统链代码所需的元数据
//当织物出现时。通过添加
//importSyscs.go中的条目
type SystemChaincode struct {
//系统链码的唯一名称
	Name string

//系统链码的路径；当前未使用
	Path string

//initargs用于启动系统链代码的初始化参数
	InitArgs [][]byte

//链代码保存实际的链表实例。
	Chaincode shim.Chaincode

//InvokableExternal跟踪
//可以调用此系统链代码
//通过发送给此对等方的建议
	InvokableExternal bool

//invokablecc2cc跟踪
//可以调用此系统链代码
//通过链码到链码的方式
//调用
	InvokableCC2CC bool

//启用了一个方便的开关来启用/禁用系统链码
//必须从importSyscs.go中删除条目
	Enabled bool
}

type SysCCWrapper struct {
	SCC *SystemChaincode
}

func (sccw *SysCCWrapper) Name() string              { return sccw.SCC.Name }
func (sccw *SysCCWrapper) Path() string              { return sccw.SCC.Path }
func (sccw *SysCCWrapper) InitArgs() [][]byte        { return sccw.SCC.InitArgs }
func (sccw *SysCCWrapper) Chaincode() shim.Chaincode { return sccw.SCC.Chaincode }
func (sccw *SysCCWrapper) InvokableExternal() bool   { return sccw.SCC.InvokableExternal }
func (sccw *SysCCWrapper) InvokableCC2CC() bool      { return sccw.SCC.InvokableCC2CC }
func (sccw *SysCCWrapper) Enabled() bool             { return sccw.SCC.Enabled }

type SelfDescribingSysCC interface {
//系统链码的唯一名称
	Name() string

//系统链码的路径；当前未使用
	Path() string

//initargs用于启动系统链代码的初始化参数
	InitArgs() [][]byte

//chaincode返回基础chaincode
	Chaincode() shim.Chaincode

//InvokableExternal跟踪
//可以调用此系统链代码
//通过发送给此对等方的建议
	InvokableExternal() bool

//invokablecc2cc跟踪
//可以调用此系统链代码
//通过链码到链码的方式
//调用
	InvokableCC2CC() bool

//启用了一个方便的开关来启用/禁用系统链码
//必须从importSyscs.go中删除条目
	Enabled() bool
}

//registeryscc向对等端注册给定的系统链码
func (p *Provider) registerSysCC(syscc SelfDescribingSysCC) (bool, error) {
	if !syscc.Enabled() || !isWhitelisted(syscc) {
		sysccLogger.Info(fmt.Sprintf("system chaincode (%s,%s,%t) disabled", syscc.Name(), syscc.Path(), syscc.Enabled()))
		return false, nil
	}

//这是一个丑陋的黑客，版本应该绑定到链码实例，而不是对等二进制。
	version := util.GetSysCCVersion()

	ccid := &ccintf.CCID{
		Name:    syscc.Name(),
		Version: version,
	}
	err := p.Registrar.Register(ccid, syscc.Chaincode())
	if err != nil {
//如果该类型已注册，则实例可能不是…继续前进
		if _, ok := err.(inproccontroller.SysCCRegisteredErr); !ok {
			errStr := fmt.Sprintf("could not register (%s,%v): %s", syscc.Path(), syscc, err)
			sysccLogger.Error(errStr)
			return false, fmt.Errorf(errStr)
		}
	}

	sysccLogger.Infof("system chaincode %s(%s) registered", syscc.Name(), syscc.Path())
	return true, err
}

//deployyscc在链上部署给定的系统链代码
func deploySysCC(chainID string, ccprov ccprovider.ChaincodeProvider, syscc SelfDescribingSysCC) error {
	if !syscc.Enabled() || !isWhitelisted(syscc) {
		sysccLogger.Info(fmt.Sprintf("system chaincode (%s,%s) disabled", syscc.Name(), syscc.Path()))
		return nil
	}

	txid := util.GenerateUUID()

//注意，这个结构几乎没有初始化，
//我们省略了历史查询执行者，提案
//以及签署的提案
	txParams := &ccprovider.TransactionParams{
		TxID:      txid,
		ChannelID: chainID,
	}

	if chainID != "" {
		lgr := peer.GetLedger(chainID)
		if lgr == nil {
			panic(fmt.Sprintf("syschain %s start up failure - unexpected nil ledger for channel %s", syscc.Name(), chainID))
		}

		txsim, err := lgr.NewTxSimulator(txid)
		if err != nil {
			return err
		}

		txParams.TXSimulator = txsim
		defer txsim.Done()
	}

	chaincodeID := &pb.ChaincodeID{Path: syscc.Path(), Name: syscc.Name()}
	spec := &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: &pb.ChaincodeInput{Args: syscc.InitArgs()}}

	chaincodeDeploymentSpec := &pb.ChaincodeDeploymentSpec{ExecEnv: pb.ChaincodeDeploymentSpec_SYSTEM, ChaincodeSpec: spec}

//这是一个丑陋的黑客，版本应该绑定到链码实例，而不是对等二进制。
	version := util.GetSysCCVersion()

	cccid := &ccprovider.CCContext{
		Name:    chaincodeDeploymentSpec.ChaincodeSpec.ChaincodeId.Name,
		Version: version,
	}

	resp, _, err := ccprov.ExecuteLegacyInit(txParams, cccid, chaincodeDeploymentSpec)
	if err == nil && resp.Status != shim.OK {
		err = errors.New(resp.Message)
	}

	sysccLogger.Infof("system chaincode %s/%s(%s) deployed", syscc.Name(), chainID, syscc.Path())

	return err
}

//DeDeploySyscc停止系统链码并从InProcController注销它
func deDeploySysCC(chainID string, ccprov ccprovider.ChaincodeProvider, syscc SelfDescribingSysCC) error {
//这是一个丑陋的黑客，版本应该绑定到链码实例，而不是对等二进制。
	version := util.GetSysCCVersion()

	ccci := &ccprovider.ChaincodeContainerInfo{
		Type:          "GOLANG",
		Name:          syscc.Name(),
		Path:          syscc.Path(),
		Version:       version,
		ContainerType: inproccontroller.ContainerType,
	}

	err := ccprov.Stop(ccci)

	return err
}

func isWhitelisted(syscc SelfDescribingSysCC) bool {
	chaincodes := viper.GetStringMapString("chaincode.system")
	val, ok := chaincodes[syscc.Name()]
	enabled := val == "enable" || val == "true" || val == "yes"
	return ok && enabled
}
