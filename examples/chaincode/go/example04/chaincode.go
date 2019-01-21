
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


package example04

import (
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//此链码是对调用另一个链码的链码的测试-调用链码示例02

//simple chaincode示例simple chaincode实现
type SimpleChaincode struct{}

func toChaincodeArgs(args ...string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}

//init接受两个参数，一个字符串和int。这些参数存储在状态下的key/value对中。
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
var event string //指示事件是否发生。最初0
var eventVal int //事件状态
	var err error
	_, args := stub.GetFunctionAndParameters()
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

//初始化链码
	event = args[0]
	eventVal, err = strconv.Atoi(args[1])
	if err != nil {
		return shim.Error("Expecting integer value for event status")
	}
	fmt.Printf("eventVal = %d\n", eventVal)

	err = stub.PutState(event, []byte(strconv.Itoa(eventVal)))
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

//invoke在收到事件并更改事件状态时调用另一个chaincode-chaincode示例02
func (t *SimpleChaincode) invoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
var event string //事件实体
var eventVal int //事件状态
	var err error

	if len(args) != 3 && len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 3 or 4")
	}

	chainCodeToCall := args[0]
	event = args[1]
	eventVal, err = strconv.Atoi(args[2])
	if err != nil {
		return shim.Error("Expected integer value for event state change")
	}
	channelID := ""
	if len(args) == 4 {
		channelID = args[3]
	}

	if eventVal != 1 {
		fmt.Printf("Unexpected event. Doing nothing\n")
		return shim.Success(nil)
	}

	f := "invoke"
	invokeArgs := toChaincodeArgs(f, "a", "b", "10")
	response := stub.InvokeChaincode(chainCodeToCall, invokeArgs, channelID)
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to invoke chaincode. Got error: %s", string(response.Payload))
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}

	fmt.Printf("Invoke chaincode successful. Got response %s", string(response.Payload))

//将事件状态写回分类帐
	err = stub.PutState(event, []byte(strconv.Itoa(eventVal)))
	if err != nil {
		return shim.Error(err.Error())
	}

	return response
}

func (t *SimpleChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
var event string //事件实体
	var err error

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting entity to query")
	}

	event = args[0]
	var jsonResp string

//从分类帐中获取状态
	eventValbytes, err := stub.GetState(event)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + event + "\"}"
		return shim.Error(jsonResp)
	}

	if eventValbytes == nil {
		jsonResp = "{\"Error\":\"Nil value for " + event + "\"}"
		return shim.Error(jsonResp)
	}

	if len(args) > 3 {
		chainCodeToCall := args[1]
		queryKey := args[2]
		channel := args[3]
		f := "query"
		invokeArgs := toChaincodeArgs(f, queryKey)
		response := stub.InvokeChaincode(chainCodeToCall, invokeArgs, channel)
		if response.Status != shim.OK {
			errStr := fmt.Sprintf("Failed to invoke chaincode. Got error: %s", err.Error())
			fmt.Printf(errStr)
			return shim.Error(errStr)
		}
		jsonResp = string(response.Payload)
	} else {
		jsonResp = "{\"Name\":\"" + event + "\",\"Amount\":\"" + string(eventValbytes) + "\"}"
	}
	fmt.Printf("Query Response: %s\n", jsonResp)

	return shim.Success([]byte(jsonResp))
}

//结构调用invoke以执行事务
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "invoke" {
		return t.invoke(stub, args)
	} else if function == "query" {
		return t.query(stub, args)
	}

	return shim.Error("Invalid invoke function name. Expecting \"invoke\" \"query\"")
}
