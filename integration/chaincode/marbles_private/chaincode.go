
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


package marbles_private

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//MarblePrivateChaincode示例Chaincode实现
type MarblesPrivateChaincode struct {
}

type marble struct {
ObjectType string `json:"docType"` //doctype用于区分状态数据库中的各种对象类型
Name       string `json:"name"`    //需要使用fieldtags来防止case四处跳跃。
	Color      string `json:"color"`
	Size       int    `json:"size"`
	Owner      string `json:"owner"`
}

type marblePrivateDetails struct {
ObjectType string `json:"docType"` //doctype用于区分状态数据库中的各种对象类型
Name       string `json:"name"`    //需要使用fieldtags来防止case四处跳跃。
	Price      int    `json:"price"`
}

//初始化链码
//=================
func (t *MarblesPrivateChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

//调用-调用的入口点
//======================
func (t *MarblesPrivateChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

//处理不同的功能
	switch function {
	case "initMarble":
//创建新大理石
		return t.initMarble(stub, args)
	case "readMarble":
//读大理石
		return t.readMarble(stub, args)
	case "readMarblePrivateDetails":
//阅读大理石私人细节
		return t.readMarblePrivateDetails(stub, args)
	case "delete":
//删除大理石
		return t.delete(stub, args)
	default:
//错误
		fmt.Println("invoke did not find func: " + function)
		return shim.Error("Received unknown function invocation")
	}
}

//==========================================
//initmarble-创建一个新的大理石，存储到chaincode状态
//==========================================
func (t *MarblesPrivateChaincode) initMarble(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

//0-名称1-颜色2-尺寸3-所有者4-价格
//“asdf”，“blue”，“35”，“bob”，“99”
	if len(args) != 5 {
		return shim.Error("Incorrect number of arguments. Expecting 5")
	}

//====输入环境卫生===
	fmt.Println("- start init marble")
	if len(args[0]) == 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) == 0 {
		return shim.Error("2nd argument must be a non-empty string")
	}
	if len(args[2]) == 0 {
		return shim.Error("3rd argument must be a non-empty string")
	}
	if len(args[3]) == 0 {
		return shim.Error("4th argument must be a non-empty string")
	}
	if len(args[4]) == 0 {
		return shim.Error("5th argument must be a non-empty string")
	}
	marbleName := args[0]
	color := strings.ToLower(args[1])
	owner := strings.ToLower(args[3])
	size, err := strconv.Atoi(args[2])
	if err != nil {
		return shim.Error("3rd argument must be a numeric string")
	}
	price, err := strconv.Atoi(args[4])
	if err != nil {
		return shim.Error("5th argument must be a numeric string")
	}

//====检查大理石是否已存在===
	marbleAsBytes, err := stub.GetPrivateData("collectionMarbles", marbleName)
	if err != nil {
		return shim.Error("Failed to get marble: " + err.Error())
	} else if marbleAsBytes != nil {
		fmt.Println("This marble already exists: " + marbleName)
		return shim.Error("This marble already exists: " + marbleName)
	}

//====创建大理石对象并封送到JSON===
	objectType := "marble"
	marble := &marble{objectType, marbleName, color, size, owner}
	marbleJSONasBytes, err := json.Marshal(marble)
	if err != nil {
		return shim.Error(err.Error())
	}
//或者，如果不想使用结构编组，则手动构建大理石JSON字符串
//marblejsonasstring：=`“doctype”：“大理石”，“name”：“`+marble name+`”，“color”：“`+color+`”，“size”：`+strconv.itoa（size）+`，“owner”：“`+owner+`”`
//MarblejSonasBytes：=[]字节（str）

//===将大理石保存到状态===
	err = stub.PutPrivateData("collectionMarbles", marbleName, marbleJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

//====保存大理石私人详细信息===
	objectType = "marblePrivateDetails"
	marblePrivateDetails := &marblePrivateDetails{objectType, marbleName, price}
	marblePrivateDetailsBytes, err := json.Marshal(marblePrivateDetails)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutPrivateData("collectionMarblePrivateDetails", marbleName, marblePrivateDetailsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

//===已保存大理石。返回成功===
	fmt.Println("- end init marble")
	return shim.Success(nil)
}

//================================
//read marble-从chaincode状态读取大理石
//================================
func (t *MarblesPrivateChaincode) readMarble(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the marble to query")
	}

	name = args[0]
valAsbytes, err := stub.GetPrivateData("collectionMarbles", name) //从chaincode状态获取大理石
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Marble does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

//================================
//read marble read marble private details-从chaincode状态读取大理石私有详细信息
//================================
func (t *MarblesPrivateChaincode) readMarblePrivateDetails(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the marble to query")
	}

	name = args[0]
valAsbytes, err := stub.GetPrivateData("collectionMarblePrivateDetails", name) //从chaincode状态获取大理石私有详细信息
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get private details for " + name + ": " + err.Error() + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Marble private details does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

//=====================================
//删除-从状态中删除大理石键/值对
//=====================================
func (t *MarblesPrivateChaincode) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var marbleJSON marble
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	marbleName := args[0]

//为了保持颜色~名称索引，我们需要先读取大理石并获取其颜色。
valAsbytes, err := stub.GetPrivateData("collectionMarbles", marbleName) //从chaincode状态获取大理石
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + marbleName + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Marble does not exist: " + marbleName + "\"}"
		return shim.Error(jsonResp)
	}

	err = json.Unmarshal([]byte(valAsbytes), &marbleJSON)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to decode JSON of: " + marbleName + "\"}"
		return shim.Error(jsonResp)
	}

err = stub.DelPrivateData("collectionMarbles", marbleName) //从链码状态移除大理石
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

//维护指标
	indexName := "color~name"
	colorNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{marbleJSON.Color, marbleJSON.Name})
	if err != nil {
		return shim.Error(err.Error())
	}

//删除状态的索引项。
	err = stub.DelPrivateData("collectionMarbles", colorNameIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

//删除大理石的私人详细信息
	err = stub.DelPrivateData("collectionMarblePrivateDetails", marbleName)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}
