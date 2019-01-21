
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


package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//这个链代码实现了一个存储在状态中的简单映射。
//以下操作可用。

//调用操作
//Put-需要两个参数，一个键和一个值
//删除-需要密钥
//get-需要一个参数、一个键，并返回一个值
//keys-不需要参数，返回所有键

//simple chaincode示例simple chaincode实现
type SimpleChaincode struct {
}

type PageResponse struct {
	Bookmark string   `json:"bookmark"`
	Keys     []string `json:"keys"`
}

//init是一个NOP
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

//invoke有两个函数
//put-接受两个参数，一个键和一个值，并将它们存储在状态中
//移除-接受一个参数、一个键，并从状态中移除if
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	switch function {

	case "putPrivate":
		return t.putPrivate(stub, args)

	case "removePrivate":
		return t.removePrivate(stub, args)

	case "getPrivate":
		return t.getPrivate(stub, args)

	case "keysPrivate":
		return t.keysPrivate(stub, args)

	case "queryPrivate":
		return t.queryPrivate(stub, args)

	case "put":
		return t.put(stub, args)

	case "remove":
		return t.remove(stub, args)

	case "get":
		return t.get(stub, args)

	case "keys":
		return t.keys(stub, args)

	case "keysByPage":
		return t.keysByPage(stub, args)

	case "query":
		return t.query(stub, args)

	case "queryByPage":
		return t.queryByPage(stub, args)

	case "history":
		return t.history(stub, args)

	case "getPut":
		return t.getPut(stub, args)

	case "getPutPrivate":
		return t.getPutPrivate(stub, args)

	default:
		return shim.Error("Unsupported operation")
	}
}

func (t *SimpleChaincode) putPrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("put operation on private data must include three arguments: [collection, key, value]")
	}
	collection := args[0]
	key := args[1]
	value := args[2]

	if err := stub.PutPrivateData(collection, key, []byte(value)); err != nil {
		fmt.Printf("Error putting private data%s", err)
		return shim.Error(fmt.Sprintf("put operation failed. Error updating state: %s", err))
	}

	return shim.Success(nil)
}
func (t *SimpleChaincode) removePrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("remove operation on private data must include two arguments: [collection, key]")
	}
	collection := args[0]
	key := args[1]

	err := stub.DelPrivateData(collection, key)
	if err != nil {
		return shim.Error(fmt.Sprintf("remove operation on private data failed. Error updating state: %s", err))
	}
	return shim.Success(nil)
}
func (t *SimpleChaincode) getPrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("get operation on private data must include two arguments: [collection, key]")
	}
	collection := args[0]
	key := args[1]
	value, err := stub.GetPrivateData(collection, key)
	if err != nil {
		return shim.Error(fmt.Sprintf("get operation on private data failed. Error accessing state: %s", err))
	}
	jsonVal, err := json.Marshal(string(value))
	return shim.Success(jsonVal)

}
func (t *SimpleChaincode) keysPrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("range query operation on private data must include three arguments, a collection, key and value")
	}
	collection := args[0]
	startKey := args[1]
	endKey := args[2]

//使用迭代器时需要睡眠来测试对等机的超时行为
	stime := 0
	if len(args) > 3 {
		stime, _ = strconv.Atoi(args[3])
	}

	keysIter, err := stub.GetPrivateDataByRange(collection, startKey, endKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("keys operation failed on private data. Error accessing state: %s", err))
	}
	defer keysIter.Close()

	var keys []string
	for keysIter.HasNext() {
//如果规定了睡眠时间，小睡一会。
		if stime > 0 {
			time.Sleep(time.Duration(stime) * time.Millisecond)
		}

		response, iterErr := keysIter.Next()
		if iterErr != nil {
			return shim.Error(fmt.Sprintf("keys operation on private data failed. Error accessing state: %s", err))
		}
		keys = append(keys, response.Key)
	}

	for key, value := range keys {
		fmt.Printf("key %d contains %s\n", key, value)
	}

	jsonKeys, err := json.Marshal(keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("keys operation on private data failed. Error marshaling JSON: %s", err))
	}

	return shim.Success(jsonKeys)
}

func (t *SimpleChaincode) queryPrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	collection := args[0]
	query := args[1]
	keysIter, err := stub.GetPrivateDataQueryResult(collection, query)
	if err != nil {
		return shim.Error(fmt.Sprintf("query operation on private data failed. Error accessing state: %s", err))
	}
	defer keysIter.Close()

	var keys []string
	for keysIter.HasNext() {
		response, iterErr := keysIter.Next()
		if iterErr != nil {
			return shim.Error(fmt.Sprintf("query operation on private data failed. Error accessing state: %s", err))
		}
		keys = append(keys, response.Key)
	}

	jsonKeys, err := json.Marshal(keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query operation on private data failed. Error marshaling JSON: %s", err))
	}

	return shim.Success(jsonKeys)
}
func (t *SimpleChaincode) put(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("put operation must include two arguments: [key, value]")
	}
	key := args[0]
	value := args[1]

	if err := stub.PutState(key, []byte(value)); err != nil {
		fmt.Printf("Error putting state %s", err)
		return shim.Error(fmt.Sprintf("put operation failed. Error updating state: %s", err))
	}

	indexName := "compositeKeyTest"
	compositeKeyTestIndex, err := stub.CreateCompositeKey(indexName, []string{key})
	if err != nil {
		return shim.Error(err.Error())
	}

	valueByte := []byte{0x00}
	if err := stub.PutState(compositeKeyTestIndex, valueByte); err != nil {
		fmt.Printf("Error putting state with compositeKey %s", err)
		return shim.Error(fmt.Sprintf("put operation failed. Error updating state with compositeKey: %s", err))
	}

	return shim.Success(nil)
}
func (t *SimpleChaincode) remove(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("remove operation must include one argument: [key]")
	}
	key := args[0]

	err := stub.DelState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("remove operation failed. Error updating state: %s", err))
	}
	return shim.Success(nil)
}
func (t *SimpleChaincode) get(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("get operation must include one argument, a key")
	}
	key := args[0]
	value, err := stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("get operation failed. Error accessing state: %s", err))
	}
	jsonVal, err := json.Marshal(string(value))
	return shim.Success(jsonVal)
}
func (t *SimpleChaincode) keys(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("keys operation must include two arguments, a key and value")
	}
	startKey := args[0]
	endKey := args[1]

//使用迭代器时需要睡眠来测试对等机的超时行为
	stime := 0
	if len(args) > 2 {
		stime, _ = strconv.Atoi(args[2])
	}

	keysIter, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("keys operation failed. Error accessing state: %s", err))
	}
	defer keysIter.Close()

	var keys []string
	for keysIter.HasNext() {
//如果规定了睡眠时间，小睡一会。
		if stime > 0 {
			time.Sleep(time.Duration(stime) * time.Millisecond)
		}

		response, iterErr := keysIter.Next()
		if iterErr != nil {
			return shim.Error(fmt.Sprintf("keys operation failed. Error accessing state: %s", err))
		}
		keys = append(keys, response.Key)
	}

	for key, value := range keys {
		fmt.Printf("key %d contains %s\n", key, value)
	}

	jsonKeys, err := json.Marshal(keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("keys operation failed. Error marshaling JSON: %s", err))
	}

	return shim.Success(jsonKeys)
}

func (t *SimpleChaincode) keysByPage(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 4 {
		return shim.Error("paginated range query operation must include four arguments, a key, value, pageSize and a bookmark")
	}
	startKey := args[0]
	endKey := args[1]
	pageSize, parserr := strconv.ParseInt(args[2], 10, 32)
	if parserr != nil {
		return shim.Error(fmt.Sprintf("error parsing range pagesize: %s", parserr))
	}
	bookmark := args[3]

//使用迭代器时需要睡眠来测试对等机的超时行为
	stime := 0
	if len(args) > 4 {
		stime, _ = strconv.Atoi(args[4])
	}

	keysIter, resp, err := stub.GetStateByRangeWithPagination(startKey, endKey, int32(pageSize), bookmark)
	if err != nil {
		return shim.Error(fmt.Sprintf("keysByPage operation failed. Error accessing state: %s", err))
	}
	defer keysIter.Close()

	var keys []string
	for keysIter.HasNext() {
//如果规定了睡眠时间，小睡一会。
		if stime > 0 {
			time.Sleep(time.Duration(stime) * time.Millisecond)
		}

		response, iterErr := keysIter.Next()
		if iterErr != nil {
			return shim.Error(fmt.Sprintf("keysByPage operation failed. Error accessing state: %s", err))
		}
		keys = append(keys, response.Key)
	}

	for index, value := range keys {
		fmt.Printf("key %d contains %s\n", index, value)
	}

	jsonResp := PageResponse{
		Bookmark: resp.Bookmark,
		Keys:     keys,
	}

	queryResp, err := json.Marshal(jsonResp)
	if err != nil {
		return shim.Error(fmt.Sprintf("keysByPage operation failed. Error marshaling JSON: %s", err))
	}

	return shim.Success(queryResp)
}
func (t *SimpleChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	query := args[0]
	keysIter, err := stub.GetQueryResult(query)
	if err != nil {
		return shim.Error(fmt.Sprintf("query operation failed. Error accessing state: %s", err))
	}
	defer keysIter.Close()

	var keys []string
	for keysIter.HasNext() {
		response, iterErr := keysIter.Next()
		if iterErr != nil {
			return shim.Error(fmt.Sprintf("query operation failed. Error accessing state: %s", err))
		}
		keys = append(keys, response.Key)
	}

	jsonKeys, err := json.Marshal(keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query operation failed. Error marshaling JSON: %s", err))
	}

	return shim.Success(jsonKeys)
}
func (t *SimpleChaincode) queryByPage(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	query := args[0]
	pageSize, parserr := strconv.ParseInt(args[1], 10, 32)
	if parserr != nil {
		return shim.Error(fmt.Sprintf("error parsing query pagesize: %s", parserr))
	}
	bookmark := args[2]

	keysIter, resp, err := stub.GetQueryResultWithPagination(query, int32(pageSize), bookmark)
	if err != nil {
		return shim.Error(fmt.Sprintf("queryByPage operation failed. Error accessing state: %s", err))
	}
	defer keysIter.Close()

	var keys []string
	for keysIter.HasNext() {
		response, iterErr := keysIter.Next()
		if iterErr != nil {
			return shim.Error(fmt.Sprintf("queryByPage operation failed. Error accessing state: %s", err))
		}
		keys = append(keys, response.Key)
	}

	for key, value := range keys {
		fmt.Printf("key %d contains %s\n", key, value)
	}

	jsonResp := PageResponse{
		Bookmark: resp.Bookmark,
		Keys:     keys,
	}

	queryResp, err := json.Marshal(jsonResp)
	if err != nil {
		return shim.Error(fmt.Sprintf("queryByPage operation failed. Error marshaling JSON: %s", err))
	}

	return shim.Success(queryResp)
}
func (t *SimpleChaincode) history(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	key := args[0]
	keysIter, err := stub.GetHistoryForKey(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("get history operation failed. Error accessing state: %s", err))
	}
	defer keysIter.Close()

	var keys []string
	for keysIter.HasNext() {
		response, iterErr := keysIter.Next()
		if iterErr != nil {
			return shim.Error(fmt.Sprintf("get history operation failed. Error accessing state: %s", err))
		}
		keys = append(keys, response.TxId)
	}

	for key, txID := range keys {
		fmt.Printf("key %d contains %s\n", key, txID)
	}

	jsonKeys, err := json.Marshal(keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("get history operation failed. Error marshaling JSON: %s", err))
	}

	return shim.Success(jsonKeys)
}
func (t *SimpleChaincode) getPut(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	_ = t.get(stub, args)
	return t.put(stub, args)
}
func (t *SimpleChaincode) getPutPrivate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	_ = t.getPrivate(stub, args)
	return t.putPrivate(stub, args)
}
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
