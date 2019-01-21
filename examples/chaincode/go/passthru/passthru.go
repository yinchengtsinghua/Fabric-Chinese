
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package main

import (
	"fmt"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//passthroughchaincode通过invoke和query传递到另一个chaincode，其中
//called chaincodeid=函数
//调用了chaincode的函数=args[0]
//调用了chaincode的args=args[1:]
type PassthruChaincode struct {
}

func toChaincodeArgs(args ...string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}

//如果函数在任何地方都有字符串“error”，init func将返回错误。
func (p *PassthruChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	function, _ := stub.GetFunctionAndParameters()
	if strings.Index(function, "error") >= 0 {
		return shim.Error(function)
	}
	return shim.Success([]byte(function))
}

//帮手
func (p *PassthruChaincode) iq(stub shim.ChaincodeStubInterface, function string, args []string) pb.Response {
	if function == "" {
		return shim.Error("Chaincode ID not provided")
	}
	chaincodeID := function

	return stub.InvokeChaincode(chaincodeID, toChaincodeArgs(args...), "")
}

//invoke通过invoke调用传递
func (p *PassthruChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	return p.iq(stub, function, args)
}

func main() {
	err := shim.Start(new(PassthruChaincode))
	if err != nil {
		fmt.Printf("Error starting Passthru chaincode: %s", err)
	}
}
