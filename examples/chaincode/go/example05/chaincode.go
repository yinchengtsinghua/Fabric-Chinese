
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


package example05

import (
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//这个chaincode是chaincode查询另一个chaincode的测试-调用chaincode example02并计算a和b的总和，并将其存储为状态

//simple chaincode示例simple chaincode实现
type SimpleChaincode struct{}

func toChaincodeArgs(args ...string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}

//init接受两个参数，一个字符串和int。该字符串将是具有
//作为一个值的int。
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
var sum string //跨账户资产持有总额。最初0
var sumVal int //持有金额
	var err error
	_, args := stub.GetFunctionAndParameters()
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

//初始化链码
	sum = args[0]
	sumVal, err = strconv.Atoi(args[1])
	if err != nil {
		return shim.Error("Expecting integer value for sum")
	}
	fmt.Printf("sumVal = %d\n", sumVal)

//将状态写入分类帐
	err = stub.PutState(sum, []byte(strconv.Itoa(sumVal)))
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

//调用查询另一个链代码并更新其自身状态
func (t *SimpleChaincode) invoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
var sum, channelName string //和实体
var Aval, Bval, sumVal int  //总和实体值-待计算
	var err error

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting atleast 2")
	}

chaincodeName := args[0] //期望您要调用的链码的名称，此名称将在链码安装时提供。
	sum = args[1]

	if len(args) > 2 {
		channelName = args[2]
	} else {
		channelName = ""
	}

//查询链代码示例02
	f := "query"
	queryArgs := toChaincodeArgs(f, "a")

//如果调用的链码在同一个通道上，
//然后通道默认为当前通道，参数[2]可以为“”。
//如果正在调用的链码位于不同的通道上，
//然后必须以args[2]指定通道名称。

	response := stub.InvokeChaincode(chaincodeName, queryArgs, channelName)
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to query chaincode. Got error: %s", response.Payload)
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}
	Aval, err = strconv.Atoi(string(response.Payload))
	if err != nil {
		errStr := fmt.Sprintf("Error retrieving state from ledger for queried chaincode: %s", err.Error())
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}

	queryArgs = toChaincodeArgs(f, "b")
	response = stub.InvokeChaincode(chaincodeName, queryArgs, channelName)
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to query chaincode. Got error: %s", response.Payload)
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}
	Bval, err = strconv.Atoi(string(response.Payload))
	if err != nil {
		errStr := fmt.Sprintf("Error retrieving state from ledger for queried chaincode: %s", err.Error())
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}

//计算和
	sumVal = Aval + Bval

//将Sumval写回分类帐
	err = stub.PutState(sum, []byte(strconv.Itoa(sumVal)))
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Printf("Invoke chaincode successful. Got sum %d\n", sumVal)
	return shim.Success([]byte(strconv.Itoa(sumVal)))
}

func (t *SimpleChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
var sum, channelName string //和实体
var Aval, Bval, sumVal int  //总和实体值-待计算
	var err error

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting atleast 2")
	}

chaincodeName := args[0] //期望您要调用的链码的名称，此名称将在链码安装时提供。
	sum = args[1]

	if len(args) > 2 {
		channelName = args[2]
	} else {
		channelName = ""
	}

//查询链代码示例02
	f := "query"
	queryArgs := toChaincodeArgs(f, "a")

//如果调用的链码在同一个通道上，
//然后通道默认为当前通道，参数[2]可以为“”。
//如果正在调用的链码位于不同的通道上，
//然后必须以args[2]指定通道名称。
	response := stub.InvokeChaincode(chaincodeName, queryArgs, channelName)
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to query chaincode. Got error: %s", response.Payload)
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}
	Aval, err = strconv.Atoi(string(response.Payload))
	if err != nil {
		errStr := fmt.Sprintf("Error retrieving state from ledger for queried chaincode: %s", err.Error())
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}

	queryArgs = toChaincodeArgs(f, "b")
	response = stub.InvokeChaincode(chaincodeName, queryArgs, channelName)
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to query chaincode. Got error: %s", response.Payload)
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}
	Bval, err = strconv.Atoi(string(response.Payload))
	if err != nil {
		errStr := fmt.Sprintf("Error retrieving state from ledger for queried chaincode: %s", err.Error())
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}

//计算和
	sumVal = Aval + Bval

	fmt.Printf("Query chaincode successful. Got sum %d\n", sumVal)
	jsonResp := "{\"Name\":\"" + sum + "\",\"Value\":\"" + strconv.Itoa(sumVal) + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)
	return shim.Success([]byte(strconv.Itoa(sumVal)))
}

func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "invoke" {
		return t.invoke(stub, args)
	} else if function == "query" {
		return t.query(stub, args)
	}

	return shim.Success([]byte("Invalid invoke function name. Expecting \"invoke\" \"query\""))
}
