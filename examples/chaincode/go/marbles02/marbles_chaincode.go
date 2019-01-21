
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 SPDX许可证标识符：Apache-2.0
**/


//==chaincode执行示例（cli）==

//====调用大理石===
//对等链代码调用-c myc1-n marbles-c'“args”：[“initmarble”，“marble1”，“blue”，“35”，“tom”]
//对等链代码调用-c myc1-n marbles-c'“args”：[“initmarble”，“marble2”，“red”，“50”，“tom”]
//对等链代码调用-c myc1-n marbles-c'“args”：[“initmarble”，“marble3”，“blue”，“70”，“tom”]
//对等链代码调用-c myc1-n marbles-c'“args”：[“transfermable”，“marble2”，“jerry”]
//对等链代码调用-c myc1-n大理石-c'“args”：[“TransferMarbleBasedOnColor”，“Blue”，“Jerry”]
//对等链代码调用-c myc1-n marbles-c'“args”：[“delete”，“marble1”]

//====查询大理石===
//对等链码查询-c myc1-n marbles-c'“args”：[“readmarble”，“marble1”]
//对等链码查询-c myc1-n marbles-c'“args”：[“getmarblesbrange”，“marble1”，“marble3”]
//对等链码查询-c myc1-n marbles-c'“args”：[“getHistoryFormable”，“marble1”]

//富查询（仅当couchdb用作状态数据库时支持）：
//对等链码查询-c myc1-n marbles-c'“args”：[“querymarblesbyowner”，“tom”]
//对等链码查询-c myc1-n marbles-c'“args”：[“query marbles”，“\”选择器\“：\”所有者\“：\”汤姆\“”]”

//带分页的富查询（仅当couchdb用作状态数据库时支持）：
//对等链码查询-c myc1-n大理石-c'“args”：[“QueryMarblesWithPagination”，“\”选择器\“：\”所有者\“：\”汤姆\“”，“3”，“]”

//支持couchdb丰富查询的索引
//
//为了使JSON查询高效，需要使用couchdb中的索引，并且
//任何带有排序的JSON查询。从Hyperledger Fabric 1.1开始，索引可以与
//META-INF/statedb/couchdb/indexes目录中的链码。每个索引都必须定义在自己的索引中
//扩展名为*.json的文本文件，索引定义格式为json，并遵循
//couchdb index json语法，如文档所示：
//http://docs.couchdb.org/en/2.1.1/api/database/find.html#db-index
//
//这个marbles02示例chaincode演示了
//可以在META-INF/statedb/couchdb/indexes/indexowner.json中找到的索引。
//对于将链代码部署到生产环境中，建议
//定义链码旁边的任何索引，以便链码和支持索引
//一旦在对等机上安装了链码，就会自动作为一个单元部署
//在通道上实例化。有关详细信息，请参阅Hyperledger Fabric文档。
//
//如果您可以在开发环境中访问对等的CouchDB状态数据库，
//您可能需要迭代地测试各种索引，以支持链式代码查询。你
//可以使用couchdb fauxton接口或命令行curl实用程序来创建和更新
//索引。然后，在完成索引后，将索引定义与
//meta-inf/statedb/couchdb/indexes目录中的链代码，用于打包和部署
//管理环境。
//
//在下面的示例中，您可以找到支持Marbles02的索引定义。
//链码查询，以及在开发环境中可以使用的语法
//在couchdb fauxton接口或curl命令行实用程序中创建索引。
//

//例如主机名：访问couchdb的端口配置。
//
//要从另一个Docker容器或从Vagrant环境访问CouchDB Docker容器，请执行以下操作：
//http://couchdb:5984/
//
//在CouchDB Docker容器内
//http://127.0.0.1:5984/

//doctype、owner的索引。
//
//在couchdb channel_chaincode数据库中定义索引的curl命令行示例
//curl-i-x post-h“内容类型：application/json”-d“\”index\“：\”fields\“：[\”doctype\“，\”owner\“]，\”name\“：\”index owner\“，\”ddoc\“：\”indexownerdoc\“，\”type\“：\”json\“”http://hostname:port/myc1\ u marbles/\u index
//

//doctype、owner、size（降序）的索引。
//
//在couchdb channel_chaincode数据库中定义索引的curl命令行示例
//

//指定了索引设计文档和索引名称的富查询（仅当CouchDB用作状态数据库时支持）：
//对等链码查询-c myc1-n marbles-c'“args”：[“query marbles”，“\”选择器\“：\”doctype\“：\”marble\“，\”owner\“：\”tom\“，\”use\ index\“：[\”\u design/indexownerdoc\“，\”index owner\“]”

//仅指定索引设计文档的富查询（仅当CouchDB用作状态数据库时支持）：
//peer chaincode query -C myc1 -n marbles -c '{"Args":["queryMarbles","{\"selector\":{\"docType\":{\"$eq\":\"marble\"},\"owner\":{\"$eq\":\"tom\"},\"size\":{\"$gt\":0}},\"fields\":[\"docType\",\"owner\",\"size\"],\"sort\":[{\"size\":\"desc\"}],\"use_index\":\"_design/indexSizeSortDoc\"}"]}'

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//simple chaincode示例simple chaincode实现
type SimpleChaincode struct {
}

type marble struct {
ObjectType string `json:"docType"` //doctype用于区分状态数据库中的各种对象类型
Name       string `json:"name"`    //需要使用fieldtags来防止case四处跳跃。
	Color      string `json:"color"`
	Size       int    `json:"size"`
	Owner      string `json:"owner"`
}

//==============================================================
//主要的
//==============================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

//初始化链码
//=================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

//调用-调用的入口点
//======================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

//处理不同的功能
if function == "initMarble" { //创建新大理石
		return t.initMarble(stub, args)
} else if function == "transferMarble" { //更改特定大理石的所有者
		return t.transferMarble(stub, args)
} else if function == "transferMarblesBasedOnColor" { //转移某一颜色的所有大理石
		return t.transferMarblesBasedOnColor(stub, args)
} else if function == "delete" { //删除大理石
		return t.delete(stub, args)
} else if function == "readMarble" { //读大理石
		return t.readMarble(stub, args)
} else if function == "queryMarblesByOwner" { //使用富查询查找所有者x的大理石
		return t.queryMarblesByOwner(stub, args)
} else if function == "queryMarbles" { //基于即席丰富查询查找大理石
		return t.queryMarbles(stub, args)
} else if function == "getHistoryForMarble" { //获取大理石的值历史记录
		return t.getHistoryForMarble(stub, args)
} else if function == "getMarblesByRange" { //基于范围查询获取大理石
		return t.getMarblesByRange(stub, args)
	} else if function == "getMarblesByRangeWithPagination" {
		return t.getMarblesByRangeWithPagination(stub, args)
	} else if function == "queryMarblesWithPagination" {
		return t.queryMarblesWithPagination(stub, args)
	}

fmt.Println("invoke did not find func: " + function) //错误
	return shim.Error("Received unknown function invocation")
}

//==========================================
//initmarble-创建一个新的大理石，存储到chaincode状态
//==========================================
func (t *SimpleChaincode) initMarble(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

//0 1 2 3
//“asdf”，“blue”，“35”，“bob”
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}

//====输入环境卫生===
	fmt.Println("- start init marble")
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("3rd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return shim.Error("4th argument must be a non-empty string")
	}
	marbleName := args[0]
	color := strings.ToLower(args[1])
	owner := strings.ToLower(args[3])
	size, err := strconv.Atoi(args[2])
	if err != nil {
		return shim.Error("3rd argument must be a numeric string")
	}

//====检查大理石是否已存在===
	marbleAsBytes, err := stub.GetState(marbleName)
	if err != nil {
		return shim.Error("Failed to get marble: " + err.Error())
	} else if marbleAsBytes != nil {
		fmt.Println("This marble already exists: " + marbleName)
		return shim.Error("This marble already exists: " + marbleName)
	}

//==== Create marble object and marshal to JSON ====
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
	err = stub.PutState(marbleName, marbleJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

//====索引大理石以启用基于颜色的范围查询，例如返回所有蓝色大理石===
//“index”是处于状态的普通键/值项。
//该键是一个复合键，首先列出要对其进行范围查询的元素。
//在我们的例子中，复合键基于indexname~color~name。
//这将启用基于匹配indexname~color~*的复合键的非常有效的状态范围查询。
	indexName := "color~name"
	colorNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{marble.Color, marble.Name})
	if err != nil {
		return shim.Error(err.Error())
	}
//将索引项保存到状态。只需要密钥名，不需要存储大理石的副本。
//注意-传递“nil”值将有效地从状态中删除键，因此我们将空字符作为值传递
	value := []byte{0x00}
	stub.PutState(colorNameIndexKey, value)

//===已保存和索引大理石。返回成功===
	fmt.Println("- end init marble")
	return shim.Success(nil)
}

//================================
//read marble-从chaincode状态读取大理石
//================================
func (t *SimpleChaincode) readMarble(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the marble to query")
	}

	name = args[0]
valAsbytes, err := stub.GetState(name) //从chaincode状态获取大理石
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Marble does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

//=====================================
//删除-从状态中删除大理石键/值对
//=====================================
func (t *SimpleChaincode) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var marbleJSON marble
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	marbleName := args[0]

//为了保持颜色~名称索引，我们需要先读取大理石并获取其颜色。
valAsbytes, err := stub.GetState(marbleName) //从chaincode状态获取大理石
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

err = stub.DelState(marbleName) //从链码状态移除大理石
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
	err = stub.DelState(colorNameIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}
	return shim.Success(nil)
}

//==========================================
//transfer a marble by setting a new owner name on the marble
//==========================================
func (t *SimpleChaincode) transferMarble(stub shim.ChaincodeStubInterface, args []string) pb.Response {

//0 1
//“名字”，“鲍伯”
	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	marbleName := args[0]
	newOwner := strings.ToLower(args[1])
	fmt.Println("- start transferMarble ", marbleName, newOwner)

	marbleAsBytes, err := stub.GetState(marbleName)
	if err != nil {
		return shim.Error("Failed to get marble:" + err.Error())
	} else if marbleAsBytes == nil {
		return shim.Error("Marble does not exist")
	}

	marbleToTransfer := marble{}
err = json.Unmarshal(marbleAsBytes, &marbleToTransfer) //将其取消标记为json.parse（）。
	if err != nil {
		return shim.Error(err.Error())
	}
marbleToTransfer.Owner = newOwner //更改所有者

	marbleJSONasBytes, _ := json.Marshal(marbleToTransfer)
err = stub.PutState(marbleName, marbleJSONasBytes) //重写大理石
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end transferMarble (success)")
	return shim.Success(nil)
}

//===================================================================
//ConstructQueryResponseForMiterator构造一个包含以下查询结果的JSON数组
//给定结果迭代器
//===================================================================
func constructQueryResponseFromIterator(resultsIterator shim.StateQueryIteratorInterface) (*bytes.Buffer, error) {
//buffer是包含queryresults的JSON数组
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
//在数组成员之前添加逗号，对第一个数组成员取消逗号
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
//记录是一个JSON对象，所以我们按原样编写
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return &buffer, nil
}

//===================================================================
//addpaginationmetadatatoqueryresults添加queryresponseMetadata，其中包含pagination
//信息，到构造的查询结果
//===================================================================
func addPaginationMetadataToQueryResults(buffer *bytes.Buffer, responseMetadata *pb.QueryResponseMetadata) *bytes.Buffer {

	buffer.WriteString("[{\"ResponseMetadata\":{\"RecordsCount\":")
	buffer.WriteString("\"")
	buffer.WriteString(fmt.Sprintf("%v", responseMetadata.FetchedRecordsCount))
	buffer.WriteString("\"")
	buffer.WriteString(", \"Bookmark\":")
	buffer.WriteString("\"")
	buffer.WriteString(responseMetadata.Bookmark)
	buffer.WriteString("\"}}]")

	return buffer
}

//===================================================================
//GetMarblesByRange根据提供的开始键和结束键执行范围查询。

//只读函数结果通常不会提交给排序。如果是只读的
//结果将提交到订单，或者如果查询在更新事务中使用
//然后提交给订购方，提交方将重新执行以保证
//结果集在认可时间和提交时间之间是稳定的。交易是
//如果结果集在批核之间发生更改，则由提交对等方失效
//时间和承诺时间。
//因此，范围查询是基于查询结果执行更新事务的安全选项。
//===================================================================
func (t *SimpleChaincode) getMarblesByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	startKey := args[0]
	endKey := args[1]

	resultsIterator, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Printf("- getMarblesByRange queryResult:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

//===示例：GetStateByPartialCompositeKey/RangeQuery=============================
//TransferMarbleBasedOnColor将把给定颜色的大理石转移给某个新所有者。
//对color~name'index'使用getStateByPartialCompositeKey（范围查询）。
//提交对等方将重新执行范围查询，以确保结果集稳定。
//在认可时间和承诺时间之间。交易被
//如果结果集在认可时间和认可时间之间发生了更改，则提交对等方。
//因此，范围查询是基于查询结果执行更新事务的安全选项。
//===================================================================
func (t *SimpleChaincode) transferMarblesBasedOnColor(stub shim.ChaincodeStubInterface, args []string) pb.Response {

//0 1
//“颜色”，“鲍伯”
	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	color := args[0]
	newOwner := strings.ToLower(args[1])
	fmt.Println("- start transferMarblesBasedOnColor ", color, newOwner)

//按颜色查询颜色~name索引
//这将对以“color”开头的所有键执行键范围查询。
	coloredMarbleResultsIterator, err := stub.GetStateByPartialCompositeKey("color~name", []string{color})
	if err != nil {
		return shim.Error(err.Error())
	}
	defer coloredMarbleResultsIterator.Close()

//遍历结果集，对于找到的每个大理石，传输到newowner
	var i int
	for i = 0; coloredMarbleResultsIterator.HasNext(); i++ {
//注意，我们没有得到值（第二个返回变量），我们只是从复合键中得到大理石名称。
		responseRange, err := coloredMarbleResultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

//从color~name复合键获取颜色和名称
		objectType, compositeKeyParts, err := stub.SplitCompositeKey(responseRange.Key)
		if err != nil {
			return shim.Error(err.Error())
		}
		returnedColor := compositeKeyParts[0]
		returnedMarbleName := compositeKeyParts[1]
		fmt.Printf("- found a marble from index:%s color:%s name:%s\n", objectType, returnedColor, returnedMarbleName)

//现在调用找到的大理石的传递函数。
//重复使用用于传送单个弹珠的相同功能
		response := t.transferMarble(stub, []string{returnedMarbleName, newOwner})
//如果传输失败，则中断循环并返回错误
		if response.Status != shim.OK {
			return shim.Error("Transfer failed: " + response.Message)
		}
	}

	responsePayload := fmt.Sprintf("Transferred %d %s marbles to %s", i, color, newOwner)
	fmt.Println("- end transferMarblesBasedOnColor: " + responsePayload)
	return shim.Success([]byte(responsePayload))
}

//=====富格查询========================================================================
//下面提供了两个富查询示例（参数化查询和即席查询）。
//富查询将查询字符串传递给状态数据库。
//富查询仅受状态数据库实现支持
//它支持丰富的查询（例如couchdb）。
//查询字符串采用基础状态数据库的语法。
//对于丰富的查询，不能保证结果集在
//认可时间和承诺时间，又称“幻影阅读”。
//因此，在更新事务中不应使用富查询，除非
//应用程序处理认可和提交时间之间结果集更改的可能性。
//富查询可用于针对对等机的时间点查询。
//===================================================================

//=====示例：参数化的Rich查询==================================
//QueryMarblesByOwner基于传入的所有者查询大理石。
//这是一个参数化查询的示例，其中查询逻辑被烘焙到链代码中，
//接受单个查询参数（所有者）。
//仅在支持丰富查询的状态数据库（如CouchDB）上可用
//===================================================================
func (t *SimpleChaincode) queryMarblesByOwner(stub shim.ChaincodeStubInterface, args []string) pb.Response {

//零
//“鲍勃”
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	owner := strings.ToLower(args[0])

	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"marble\",\"owner\":\"%s\"}}", owner)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

//=====示例：即席富查询=========================================================
//查询大理石使用查询字符串来执行大理石的查询。
//查询字符串匹配状态数据库语法按原样传入和执行。
//支持可由客户端在运行时定义的即席查询。
//如果不需要这样做，请按照QueryMarbleForOwner示例进行参数化查询。
//仅在支持丰富查询的状态数据库（如CouchDB）上可用
//===================================================================
func (t *SimpleChaincode) queryMarbles(stub shim.ChaincodeStubInterface, args []string) pb.Response {

//零
//“查询字符串”
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	queryString := args[0]

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

//===================================================================
//GetQueryResultForQueryString执行传入的查询字符串。
//结果集被构建并作为包含JSON结果的字节数组返回。
//===================================================================
func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", buffer.String())

	return buffer.Bytes(), nil
}

//=====分页================================================================
//分页提供了一种方法来检索具有定义的pagesize和
//起点（书签）。空字符串书签定义查询的第一个“页面”
//结果。分页查询返回可在
//用于检索下一页结果的下一个查询。分页查询扩展
//丰富的查询和范围查询，包括页面大小和书签。
//
//本例中提供了两个示例。第一种方法是通过分页来实现MarbleByRangeWithPagination
//执行分页范围查询。
//第二个例子是富即席查询的分页查询。
//===================================================================

//=====示例：带范围查询的分页=======================================
//GetMarbleByRangeWithPagination根据开始键和结束键执行范围查询，
//页面大小和书签。

//提取的记录数将等于或小于页面大小。
//分页范围查询仅对只读事务有效。
//===================================================================
func (t *SimpleChaincode) getMarblesByRangeWithPagination(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}

	startKey := args[0]
	endKey := args[1]
//ParseInt的返回类型为Int64
	pageSize, err := strconv.ParseInt(args[2], 10, 32)
	if err != nil {
		return shim.Error(err.Error())
	}
	bookmark := args[3]

	resultsIterator, responseMetadata, err := stub.GetStateByRangeWithPagination(startKey, endKey, int32(pageSize), bookmark)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return shim.Error(err.Error())
	}

	bufferWithPaginationInfo := addPaginationMetadataToQueryResults(buffer, responseMetadata)

	fmt.Printf("- getMarblesByRange queryResult:\n%s\n", bufferWithPaginationInfo.String())

	return shim.Success(buffer.Bytes())
}

//=====示例：使用即席富查询分页====================================================
//QueryMarblesWithPagination使用查询字符串、页面大小和书签执行查询
//大理石。查询字符串匹配状态数据库语法按原样传入和执行。
//提取的记录数将等于或小于指定的页面大小。
//支持可由客户端在运行时定义的即席查询。
//如果不需要这样做，请按照QueryMarbleForOwner示例进行参数化查询。
//仅在支持丰富查询的状态数据库（如CouchDB）上可用
//分页查询仅对只读事务有效。
//===================================================================
func (t *SimpleChaincode) queryMarblesWithPagination(stub shim.ChaincodeStubInterface, args []string) pb.Response {

//零
//“查询字符串”
	if len(args) < 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	queryString := args[0]
//ParseInt的返回类型为Int64
	pageSize, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return shim.Error(err.Error())
	}
	bookmark := args[2]

	queryResults, err := getQueryResultForQueryStringWithPagination(stub, queryString, int32(pageSize), bookmark)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

//===================================================================
//GetQueryResultForQueryStringWithPagination执行传入的查询字符串
//分页信息。结果集被构建并作为包含JSON结果的字节数组返回。
//===================================================================
func getQueryResultForQueryStringWithPagination(stub shim.ChaincodeStubInterface, queryString string, pageSize int32, bookmark string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, responseMetadata, err := stub.GetQueryResultWithPagination(queryString, pageSize, bookmark)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	bufferWithPaginationInfo := addPaginationMetadataToQueryResults(buffer, responseMetadata)

	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", bufferWithPaginationInfo.String())

	return buffer.Bytes(), nil
}

func (t *SimpleChaincode) getHistoryForMarble(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	marbleName := args[0]

	fmt.Printf("- start getHistoryForMarble: %s\n", marbleName)

	resultsIterator, err := stub.GetHistoryForKey(marbleName)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

//Buffer是一个包含大理石历史值的JSON数组
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
//在数组成员之前添加逗号，对第一个数组成员取消逗号
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(response.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Value\":")
//如果是对给定键执行的删除操作，则需要设置
//对应值为空。否则，我们将写入响应。值
//原样（值本身是JSON大理石）
		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(", \"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getHistoryForMarble returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}
