
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


//包填充程序为链代码提供API以访问其状态
//变量、事务上下文和调用其他链码。
package shim

import (
	"container/list"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
)

//垫片组记录器。
var mockLogger = logging.MustGetLogger("mock")

//mockstub是用于单元测试链码的chaincodestub接口的一种实现。
//在对init或invoke的单元测试调用中，使用这个函数而不是chaincodestub。
type MockStub struct {
//用存根调用的参数
	args [][]byte

//返回将调用此项的链代码的指针，由构造函数设置。
//如果一个对等机调用这个存根，那么将从这里调用链码。
	cc Chaincode

//一个好名字，可以用来记录
	Name string

//状态保留名称-值对
	State map[string][]byte

//键按词汇顺序存储映射值列表
	Keys *list.List

//可从此mockstub调用的其他mockstub链码的注册列表
	Invokables map[string]*MockStub

//在被调用/部署时存储事务UUID
//TODO如果链码使用递归，这可能需要一个TxID堆栈，或者可能是一个引用计数映射
	TxID string

	TxTimestamp *timestamp.Timestamp

//模仿签名的目的
	signedProposal *pb.SignedProposal

//存储建议的频道ID
	ChannelID string

	PvtState map[string]map[string][]byte

//存储按密钥认可策略，第一个映射索引是集合，第二个映射索引是密钥
	EndorsementPolicies map[string]map[string][]byte

//存储链码事件的通道
	ChaincodeEventsChannel chan *pb.ChaincodeEvent

	Decorations map[string][]byte
}

func (stub *MockStub) GetTxID() string {
	return stub.TxID
}

func (stub *MockStub) GetChannelID() string {
	return stub.ChannelID
}

func (stub *MockStub) GetArgs() [][]byte {
	return stub.args
}

func (stub *MockStub) GetStringArgs() []string {
	args := stub.GetArgs()
	strargs := make([]string, 0, len(args))
	for _, barg := range args {
		strargs = append(strargs, string(barg))
	}
	return strargs
}

func (stub *MockStub) GetFunctionAndParameters() (function string, params []string) {
	allargs := stub.GetStringArgs()
	function = ""
	params = []string{}
	if len(allargs) >= 1 {
		function = allargs[0]
		params = allargs[1:]
	}
	return
}

//用于向链码指示它是事务的一部分。
//当链码互相调用时，这一点很重要。
//mockstub目前不支持并发事务。
func (stub *MockStub) MockTransactionStart(txid string) {
	stub.TxID = txid
	stub.setSignedProposal(&pb.SignedProposal{})
	stub.setTxTimestamp(util.CreateUtcTimestamp())
}

//结束模拟事务，清除UUID。
func (stub *MockStub) MockTransactionEnd(uuid string) {
	stub.signedProposal = nil
	stub.TxID = ""
}

//Register a peer chaincode with this MockStub
//
//OtherStub是对等机的模拟存根，已初始化
func (stub *MockStub) MockPeerChaincode(invokableChaincodeName string, otherStub *MockStub) {
	stub.Invokables[invokableChaincodeName] = otherStub
}

//初始化此链码，同时启动和结束事务。
func (stub *MockStub) MockInit(uuid string, args [][]byte) pb.Response {
	stub.args = args
	stub.MockTransactionStart(uuid)
	res := stub.cc.Init(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

//调用这个链码，也可以启动和结束一个事务。
func (stub *MockStub) MockInvoke(uuid string, args [][]byte) pb.Response {
	stub.args = args
	stub.MockTransactionStart(uuid)
	res := stub.cc.Invoke(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

func (stub *MockStub) GetDecorations() map[string][]byte {
	return stub.Decorations
}

//调用这个链码，也可以启动和结束一个事务。
func (stub *MockStub) MockInvokeWithSignedProposal(uuid string, args [][]byte, sp *pb.SignedProposal) pb.Response {
	stub.args = args
	stub.MockTransactionStart(uuid)
	stub.signedProposal = sp
	res := stub.cc.Invoke(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

func (stub *MockStub) GetPrivateData(collection string, key string) ([]byte, error) {
	m, in := stub.PvtState[collection]

	if !in {
		return nil, nil
	}

	return m[key], nil
}

func (stub *MockStub) PutPrivateData(collection string, key string, value []byte) error {
	m, in := stub.PvtState[collection]
	if !in {
		stub.PvtState[collection] = make(map[string][]byte)
		m, in = stub.PvtState[collection]
	}

	m[key] = value

	return nil
}

func (stub *MockStub) DelPrivateData(collection string, key string) error {
	return errors.New("Not Implemented")
}

func (stub *MockStub) GetPrivateDataByRange(collection, startKey, endKey string) (StateQueryIteratorInterface, error) {
	return nil, errors.New("Not Implemented")
}

func (stub *MockStub) GetPrivateDataByPartialCompositeKey(collection, objectType string, attributes []string) (StateQueryIteratorInterface, error) {
	return nil, errors.New("Not Implemented")
}

func (stub *MockStub) GetPrivateDataQueryResult(collection, query string) (StateQueryIteratorInterface, error) {
//由于模拟引擎没有查询引擎，因此未实现。
//但是，支持字符串匹配的非常简单的查询引擎
//可以实现以测试框架是否支持查询
	return nil, errors.New("Not Implemented")
}

//GetState从分类帐中检索给定键的值
func (stub *MockStub) GetState(key string) ([]byte, error) {
	value := stub.State[key]
	mockLogger.Debug("MockStub", stub.Name, "Getting", key, value)
	return value, nil
}

//putstate将指定的“value”和“key”写入分类帐。
func (stub *MockStub) PutState(key string, value []byte) error {
	if stub.TxID == "" {
		err := errors.New("cannot PutState without a transactions - call stub.MockTransactionStart()?")
		mockLogger.Errorf("%+v", err)
		return err
	}

//如果该值为零或为空，请删除该键
	if len(value) == 0 {
		mockLogger.Debug("MockStub", stub.Name, "PutState called, but value is nil or empty. Delete ", key)
		return stub.DelState(key)
	}

	mockLogger.Debug("MockStub", stub.Name, "Putting", key, value)
	stub.State[key] = value

//将密钥插入已排序的密钥列表
	for elem := stub.Keys.Front(); elem != nil; elem = elem.Next() {
		elemValue := elem.Value.(string)
		comp := strings.Compare(key, elemValue)
		mockLogger.Debug("MockStub", stub.Name, "Compared", key, elemValue, " and got ", comp)
		if comp < 0 {
//键<elem，插入elem之前
			stub.Keys.InsertBefore(key, elem)
			mockLogger.Debug("MockStub", stub.Name, "Key", key, " inserted before", elem.Value)
			break
		} else if comp == 0 {
//密钥存在，无需更改
			mockLogger.Debug("MockStub", stub.Name, "Key", key, "already in State")
			break
} else { //COMP＞0
//键>Elem，继续查找，除非这是列表的结尾
			if elem.Next() == nil {
				stub.Keys.PushBack(key)
				mockLogger.Debug("MockStub", stub.Name, "Key", key, "appended")
				break
			}
		}
	}

//空键列表的特殊情况
	if stub.Keys.Len() == 0 {
		stub.Keys.PushFront(key)
		mockLogger.Debug("MockStub", stub.Name, "Key", key, "is first element in list")
	}

	return nil
}

//Delstate从分类帐中删除指定的“key”及其值。
func (stub *MockStub) DelState(key string) error {
	mockLogger.Debug("MockStub", stub.Name, "Deleting", key, stub.State[key])
	delete(stub.State, key)

	for elem := stub.Keys.Front(); elem != nil; elem = elem.Next() {
		if strings.Compare(key, elem.Value.(string)) == 0 {
			stub.Keys.Remove(elem)
		}
	}

	return nil
}

func (stub *MockStub) GetStateByRange(startKey, endKey string) (StateQueryIteratorInterface, error) {
	if err := validateSimpleKeys(startKey, endKey); err != nil {
		return nil, err
	}
	return NewMockStateRangeQueryIterator(stub, startKey, endKey), nil
}

//可以通过chaincode调用getqueryresult函数来执行
//对状态数据库的富查询。仅受状态数据库实现支持
//支持丰富的查询。查询字符串的语法为基础
//状态数据库。返回一个迭代器，它可以用来迭代（下一步）
//查询结果集
func (stub *MockStub) GetQueryResult(query string) (StateQueryIteratorInterface, error) {
//由于模拟引擎没有查询引擎，因此未实现。
//但是，支持字符串匹配的非常简单的查询引擎
//可以实现以测试框架是否支持查询
	return nil, errors.New("not implemented")
}

//可以通过chaincode调用getHistoryForkey函数以返回
//关键值跨越时间。GetHistoryForkey用于只读查询。
func (stub *MockStub) GetHistoryForKey(key string) (HistoryQueryIteratorInterface, error) {
	return nil, errors.New("not implemented")
}

//可以通过chaincode调用getstatebypartialcompositekey函数来查询
//基于给定的部分复合键的状态。此函数返回
//迭代器，可用于对前缀为
//匹配给定的部分复合键。此函数只能用于
//部分复合键。对于完整的复合键，带有空响应的ITER
//将被退回。
func (stub *MockStub) GetStateByPartialCompositeKey(objectType string, attributes []string) (StateQueryIteratorInterface, error) {
	partialCompositeKey, err := stub.CreateCompositeKey(objectType, attributes)
	if err != nil {
		return nil, err
	}
	return NewMockStateRangeQueryIterator(stub, partialCompositeKey, partialCompositeKey+string(maxUnicodeRuneValue)), nil
}

//CreateCompositeKey组合属性列表
//形成复合键。
func (stub *MockStub) CreateCompositeKey(objectType string, attributes []string) (string, error) {
	return createCompositeKey(objectType, attributes)
}

//SplitCompositeKey将组合键拆分为属性
//在其上形成复合键。
func (stub *MockStub) SplitCompositeKey(compositeKey string) (string, []string, error) {
	return splitCompositeKey(compositeKey)
}

func (stub *MockStub) GetStateByRangeWithPagination(startKey, endKey string, pageSize int32,
	bookmark string) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	return nil, nil, nil
}

func (stub *MockStub) GetStateByPartialCompositeKeyWithPagination(objectType string, keys []string,
	pageSize int32, bookmark string) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	return nil, nil, nil
}

func (stub *MockStub) GetQueryResultWithPagination(query string, pageSize int32,
	bookmark string) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	return nil, nil, nil
}

//
//例如stub1.invokeChaincode（“stub2hash”，funcargs，channel）
//在调用此函数之前，请确保创建另一个mockstub stub2，调用stub2.mockinit（uuid、func、args）
//并通过调用stub1.mockpeerchaincode（“stub2hash”，stub2）将其注册到stub1。
func (stub *MockStub) InvokeChaincode(chaincodeName string, args [][]byte, channel string) pb.Response {
//在内部，我们使用chaincode名称作为复合名称
	if channel != "" {
		chaincodeName = chaincodeName + "/" + channel
	}
//
	otherStub := stub.Invokables[chaincodeName]
	mockLogger.Debug("MockStub", stub.Name, "Invoking peer chaincode", otherStub.Name, args)
//函数，字符串：=getFuncargs（args）
	res := otherStub.MockInvoke(stub.TxID, args)
	mockLogger.Debug("MockStub", stub.Name, "Invoked peer chaincode", otherStub.Name, "got", fmt.Sprintf("%+v", res))
	return res
}

//未实施
func (stub *MockStub) GetCreator() ([]byte, error) {
	return nil, nil
}

//未实施
func (stub *MockStub) GetTransient() (map[string][]byte, error) {
	return nil, nil
}

//未实施
func (stub *MockStub) GetBinding() ([]byte, error) {
	return nil, nil
}

//未实施
func (stub *MockStub) GetSignedProposal() (*pb.SignedProposal, error) {
	return stub.signedProposal, nil
}

func (stub *MockStub) setSignedProposal(sp *pb.SignedProposal) {
	stub.signedProposal = sp
}

//未实施
func (stub *MockStub) GetArgsSlice() ([]byte, error) {
	return nil, nil
}

func (stub *MockStub) setTxTimestamp(time *timestamp.Timestamp) {
	stub.TxTimestamp = time
}

func (stub *MockStub) GetTxTimestamp() (*timestamp.Timestamp, error) {
	if stub.TxTimestamp == nil {
		return nil, errors.New("TxTimestamp not set.")
	}
	return stub.TxTimestamp, nil
}

func (stub *MockStub) SetEvent(name string, payload []byte) error {
	stub.ChaincodeEventsChannel <- &pb.ChaincodeEvent{EventName: name, Payload: payload}
	return nil
}

func (stub *MockStub) SetStateValidationParameter(key string, ep []byte) error {
	return stub.SetPrivateDataValidationParameter("", key, ep)
}

func (stub *MockStub) GetStateValidationParameter(key string) ([]byte, error) {
	return stub.GetPrivateDataValidationParameter("", key)
}

func (stub *MockStub) SetPrivateDataValidationParameter(collection, key string, ep []byte) error {
	m, in := stub.EndorsementPolicies[collection]
	if !in {
		stub.EndorsementPolicies[collection] = make(map[string][]byte)
		m, in = stub.EndorsementPolicies[collection]
	}

	m[key] = ep
	return nil
}

func (stub *MockStub) GetPrivateDataValidationParameter(collection, key string) ([]byte, error) {
	m, in := stub.EndorsementPolicies[collection]

	if !in {
		return nil, nil
	}

	return m[key], nil
}

//初始化内部状态映射的构造函数
func NewMockStub(name string, cc Chaincode) *MockStub {
	mockLogger.Debug("MockStub(", name, cc, ")")
	s := new(MockStub)
	s.Name = name
	s.cc = cc
	s.State = make(map[string][]byte)
	s.PvtState = make(map[string]map[string][]byte)
	s.EndorsementPolicies = make(map[string]map[string][]byte)
	s.Invokables = make(map[string]*MockStub)
	s.Keys = list.New()
s.ChaincodeEventsChannel = make(chan *pb.ChaincodeEvent, 100) //为非阻塞的setevent调用定义大容量。
	s.Decorations = make(map[string][]byte)

	return s
}

/***********************
 范围查询迭代器
***********************/


type MockStateRangeQueryIterator struct {
	Closed   bool
	Stub     *MockStub
	StartKey string
	EndKey   string
	Current  *list.Element
}

//如果范围查询迭代器包含其他键，则hasNext返回true
//和价值观。
func (iter *MockStateRangeQueryIterator) HasNext() bool {
	if iter.Closed {
//以前称为close（）。
		mockLogger.Debug("HasNext() but already closed")
		return false
	}

	if iter.Current == nil {
		mockLogger.Error("HasNext() couldn't get Current")
		return false
	}

	current := iter.Current
	for current != nil {
//如果这是对所有键的开放式查询，则返回true
		if iter.StartKey == "" && iter.EndKey == "" {
			return true
		}
		comp1 := strings.Compare(current.Value.(string), iter.StartKey)
		comp2 := strings.Compare(current.Value.(string), iter.EndKey)
		if comp1 >= 0 {
			if comp2 < 0 {
				mockLogger.Debug("HasNext() got next")
				return true
			} else {
				mockLogger.Debug("HasNext() but no next")
				return false

			}
		}
		current = current.Next()
	}

//我们已经达到了基本价值的末端
	mockLogger.Debug("HasNext() but no next")
	return false
}

//Next返回范围查询迭代器中的下一个键和值。
func (iter *MockStateRangeQueryIterator) Next() (*queryresult.KV, error) {
	if iter.Closed == true {
		err := errors.New("MockStateRangeQueryIterator.Next() called after Close()")
		mockLogger.Errorf("%+v", err)
		return nil, err
	}

	if iter.HasNext() == false {
		err := errors.New("MockStateRangeQueryIterator.Next() called when it does not HaveNext()")
		mockLogger.Errorf("%+v", err)
		return nil, err
	}

	for iter.Current != nil {
		comp1 := strings.Compare(iter.Current.Value.(string), iter.StartKey)
		comp2 := strings.Compare(iter.Current.Value.(string), iter.EndKey)
//与开始键和结束键进行比较。或者，如果这是
//所有键，它应始终返回键和值
		if (comp1 >= 0 && comp2 < 0) || (iter.StartKey == "" && iter.EndKey == "") {
			key := iter.Current.Value.(string)
			value, err := iter.Stub.GetState(key)
			iter.Current = iter.Current.Next()
			return &queryresult.KV{Key: key, Value: value}, err
		}
		iter.Current = iter.Current.Next()
	}
	err := errors.New("MockStateRangeQueryIterator.Next() went past end of range")
	mockLogger.Errorf("%+v", err)
	return nil, err
}

//关闭关闭范围查询迭代器。完成后应调用此函数
//从迭代器读取以释放资源。
func (iter *MockStateRangeQueryIterator) Close() error {
	if iter.Closed == true {
		err := errors.New("MockStateRangeQueryIterator.Close() called after Close()")
		mockLogger.Errorf("%+v", err)
		return err
	}

	iter.Closed = true
	return nil
}

func (iter *MockStateRangeQueryIterator) Print() {
	mockLogger.Debug("MockStateRangeQueryIterator {")
	mockLogger.Debug("Closed?", iter.Closed)
	mockLogger.Debug("Stub", iter.Stub)
	mockLogger.Debug("StartKey", iter.StartKey)
	mockLogger.Debug("EndKey", iter.EndKey)
	mockLogger.Debug("Current", iter.Current)
	mockLogger.Debug("HasNext?", iter.HasNext())
	mockLogger.Debug("}")
}

func NewMockStateRangeQueryIterator(stub *MockStub, startKey string, endKey string) *MockStateRangeQueryIterator {
	mockLogger.Debug("NewMockStateRangeQueryIterator(", stub, startKey, endKey, ")")
	iter := new(MockStateRangeQueryIterator)
	iter.Closed = false
	iter.Stub = stub
	iter.StartKey = startKey
	iter.EndKey = endKey
	iter.Current = stub.Keys.Front()

	iter.Print()

	return iter
}

func getBytes(function string, args []string) [][]byte {
	bytes := make([][]byte, 0, len(args)+1)
	bytes = append(bytes, []byte(function))
	for _, s := range args {
		bytes = append(bytes, []byte(s))
	}
	return bytes
}

func getFuncArgs(bytes [][]byte) (string, []string) {
	mockLogger.Debugf("getFuncArgs(%x)", bytes)
	function := string(bytes[0])
	args := make([]string, len(bytes)-1)
	for i := 1; i < len(bytes); i++ {
		mockLogger.Debugf("getFuncArgs - i:%x, len(bytes):%x", i, len(bytes))
		args[i-1] = string(bytes[i])
	}
	return function, args
}
