
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

//sleeper chaincode休眠并与一个状态变量一起工作
//init-1参数，以毫秒为单位的睡眠时间
//调用-4或3个参数，“输入”或“获取”，设置值和睡眠时间（毫秒）
//
//Sleeper可用于测试“chaincode.ExecuteTimeout”属性

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//SleeperChaincode示例简单链代码实现
type SleeperChaincode struct {
}

func (t *SleeperChaincode) sleep(sleepTime string) {
	st, _ := strconv.Atoi(sleepTime)
	if st >= 0 {
		time.Sleep(time.Duration(st) * time.Millisecond)
	}
}

//init初始化chaincode…它只需要休眠一个bi
func (t *SleeperChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetStringArgs()

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	sleepTime := args[0]

	t.sleep(sleepTime)

	return shim.Success(nil)
}

//invoke设置键/值并休眠一点
func (t *SleeperChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "put" {
		if len(args) != 3 {
			return shim.Error("Incorrect number of arguments. Expecting 3")
		}

//从A到B支付X单位
		return t.invoke(stub, args)
	} else if function == "get" {
		if len(args) != 2 {
			return shim.Error("Incorrect number of arguments. Expecting 2")
		}

//旧的“查询”现在在invoke中实现
		return t.query(stub, args)
	}

	return shim.Error("Invalid invoke function name. Expecting \"put\" or \"get\"")
}

//交易从A到B支付X单位
func (t *SleeperChaincode) invoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
//设置状态
	key := args[0]
	val := args[1]

	err := stub.PutState(key, []byte(val))
	if err != nil {
		return shim.Error(err.Error())
	}

	sleepTime := args[2]

//睡一会儿
	t.sleep(sleepTime)

	return shim.Success([]byte("OK"))
}

//表示链码查询的查询回调
func (t *SleeperChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	key := args[0]

//从分类帐中获取状态
	val, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	sleepTime := args[1]

//睡一会儿
	t.sleep(sleepTime)

	return shim.Success(val)
}

func main() {
	err := shim.Start(new(SleeperChaincode))
	if err != nil {
		fmt.Printf("Error starting Sleeper chaincode: %s", err)
	}
}
