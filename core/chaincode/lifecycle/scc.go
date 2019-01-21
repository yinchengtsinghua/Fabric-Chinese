
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


package lifecycle

import (
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	lb "github.com/hyperledger/fabric/protos/peer/lifecycle"
	"github.com/pkg/errors"
)

const (
//InstalledChaincoDefuncName是用于安装链码的链码函数名
	InstallChaincodeFuncName = "InstallChaincode"

//queryinstalledchaincodefuncname是用于查询已安装链码的链码函数名
	QueryInstalledChaincodeFuncName = "QueryInstalledChaincode"
)

//sccfunctions提供带有具体参数的支持实现
//对于每个SCC功能
type SCCFunctions interface {
//InstallChainCode将链码定义保存到磁盘
	InstallChaincode(name, version string, chaincodePackage []byte) (hash []byte, err error)

//queryinstalledchaincode返回已安装链码的给定名称和版本的哈希
	QueryInstalledChaincode(name, version string) (hash []byte, err error)
}

//SCC实现了满足chaincode接口所需的方法。
//它将调用调用路由到支持实现。
type SCC struct {
	Protobuf  Protobuf
	Functions SCCFunctions
}

//name返回“+生命周期”
func (scc *SCC) Name() string {
	return "+lifecycle"
}

//路径返回“github.com/hyperledger/fabric/core/chaincode/lifecycle”
func (scc *SCC) Path() string {
	return "github.com/hyperledger/fabric/core/chaincode/lifecycle"
}

//
func (scc *SCC) InitArgs() [][]byte {
	return nil
}

//chaincode返回对自身的引用
func (scc *SCC) Chaincode() shim.Chaincode {
	return scc
}

//InvokableExternal返回true
func (scc *SCC) InvokableExternal() bool {
	return true
}

//InvokableCC2CC返回true
func (scc *SCC) InvokableCC2CC() bool {
	return true
}

//启用返回真
func (scc *SCC) Enabled() bool {
	return true
}

//init对于系统链码基本上是无用的，并且总是返回success
func (scc *SCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

//invoke接受chaincode调用参数并将其路由到正确的
//底层生命周期操作。所有函数只接受一个参数
//键入marshaled lb.<functionname>args并返回marshaled lb.<functionname>result
func (scc *SCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) == 0 {
		return shim.Error("lifecycle scc must be invoked with arguments")
	}

	if len(args) != 2 {
		return shim.Error(fmt.Sprintf("lifecycle scc operations require exactly two arguments but received %d", len(args)))
	}

	funcName := args[0]
	inputBytes := args[1]

//添加ACL

	switch string(funcName) {
//每个生命周期SCC函数在这里都有一个例子
	case InstallChaincodeFuncName:
		input := &lb.InstallChaincodeArgs{}
		err := scc.Protobuf.Unmarshal(inputBytes, input)
		if err != nil {
			err = errors.WithMessage(err, "failed to decode input arg to InstallChaincode")
			return shim.Error(err.Error())
		}

		hash, err := scc.Functions.InstallChaincode(input.Name, input.Version, input.ChaincodeInstallPackage)
		if err != nil {
			err = errors.WithMessage(err, "failed to invoke backing InstallChaincode")
			return shim.Error(err.Error())
		}

		resultBytes, err := scc.Protobuf.Marshal(&lb.InstallChaincodeResult{
			Hash: hash,
		})
		if err != nil {
			err = errors.WithMessage(err, "failed to marshal result")
			return shim.Error(err.Error())
		}

		return shim.Success(resultBytes)
	case QueryInstalledChaincodeFuncName:
		input := &lb.QueryInstalledChaincodeArgs{}
		err := scc.Protobuf.Unmarshal(inputBytes, input)
		if err != nil {
			err = errors.WithMessage(err, "failed to decode input arg to QueryInstalledChaincode")
			return shim.Error(err.Error())
		}

		hash, err := scc.Functions.QueryInstalledChaincode(input.Name, input.Version)
		if err != nil {
			err = errors.WithMessage(err, "failed to invoke backing QueryInstalledChaincode")
			return shim.Error(err.Error())
		}

		resultBytes, err := scc.Protobuf.Marshal(&lb.QueryInstalledChaincodeResult{
			Hash: hash,
		})
		if err != nil {
			err = errors.WithMessage(err, "failed to marshal result")
			return shim.Error(err.Error())
		}

		return shim.Success(resultBytes)
	default:
		return shim.Error(fmt.Sprintf("unknown lifecycle function: %s", funcName))
	}
}
