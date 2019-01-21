
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 *版权所有Greg Haskins保留所有权利
 *
 *SPDX许可证标识符：Apache-2.0
 *
 *本测试代码的目的是证明系统正确包装。
 *向上依赖。因此，我们综合了一个场景，其中链代码
 *直接和间接导入非标准依赖项，然后
 *希望进行单元测试，以验证包中是否包含所需的所有内容。
 *并最终正确构建。
 *
 **/


package main

import (
	"chaincodes/AutoVendor/directdep"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//simple chaincode示例simple chaincode实现
type SimpleChaincode struct {
}

func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Error("NOT IMPL")
}

func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Error("NOT IMPL")
}

func main() {
	directdep.PointlessFunction()

	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
