
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


package shim

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
	mockpeer "github.com/hyperledger/fabric/common/mocks/peer"
	"github.com/hyperledger/fabric/common/util"
	lproto "github.com/hyperledger/fabric/protos/ledger/queryresult"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

//Shimtestcc示例简单的链代码实现
type shimTestCC struct {
}

func (t *shimTestCC) Init(stub ChaincodeStubInterface) pb.Response {
	_, args := stub.GetFunctionAndParameters()
var A, B string    //实体
var Aval, Bval int //资产持有量
	var err error

	if len(args) != 4 {
		return Error("Incorrect number of arguments. Expecting 4")
	}

//初始化链码
	A = args[0]
	Aval, err = strconv.Atoi(args[1])
	if err != nil {
		return Error("Expecting integer value for asset holding")
	}
	B = args[2]
	Bval, err = strconv.Atoi(args[3])
	if err != nil {
		return Error("Expecting integer value for asset holding")
	}

//将状态写入分类帐
	err = stub.PutState(A, []byte(strconv.Itoa(Aval)))
	if err != nil {
		return Error(err.Error())
	}

	err = stub.PutState(B, []byte(strconv.Itoa(Bval)))
	if err != nil {
		return Error(err.Error())
	}

	return Success(nil)
}

func (t *shimTestCC) Invoke(stub ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "invoke" {
//从A到B支付X单位
		return t.invoke(stub, args)
	} else if function == "delete" {
//从实体的状态中删除实体
		return t.delete(stub, args)
	} else if function == "query" {
//旧的“查询”现在在invoke中实现
		return t.query(stub, args)
	} else if function == "cc2cc" {
		return t.cc2cc(stub, args)
	} else if function == "rangeq" {
		return t.rangeq(stub, args)
	} else if function == "historyq" {
		return t.historyq(stub, args)
	} else if function == "richq" {
		return t.richq(stub, args)
	} else if function == "putep" {
		return t.putEP(stub)
	} else if function == "getep" {
		return t.getEP(stub)
	}

	return Error("Invalid invoke function name. Expecting \"invoke\" \"delete\" \"query\"")
}

//交易从A到B支付X单位
func (t *shimTestCC) invoke(stub ChaincodeStubInterface, args []string) pb.Response {
var A, B string    //实体
var Aval, Bval int //资产持有量
var X int          //交易价值
	var err error

	if len(args) != 3 {
		return Error("Incorrect number of arguments. Expecting 3")
	}

	A = args[0]
	B = args[1]

//从分类帐中获取状态
//托多：能打个GetAllState电话到Ledger很好
	Avalbytes, err := stub.GetState(A)
	if err != nil {
		return Error("Failed to get state")
	}
	if Avalbytes == nil {
		return Error("Entity not found")
	}
	Aval, _ = strconv.Atoi(string(Avalbytes))

	Bvalbytes, err := stub.GetState(B)
	if err != nil {
		return Error("Failed to get state")
	}
	if Bvalbytes == nil {
		return Error("Entity not found")
	}
	Bval, _ = strconv.Atoi(string(Bvalbytes))

//执行
	X, err = strconv.Atoi(args[2])
	if err != nil {
		return Error("Invalid transaction amount, expecting a integer value")
	}
	Aval = Aval - X
	Bval = Bval + X

//将状态写回分类帐
	err = stub.PutState(A, []byte(strconv.Itoa(Aval)))
	if err != nil {
		return Error(err.Error())
	}

	err = stub.PutState(B, []byte(strconv.Itoa(Bval)))
	if err != nil {
		return Error(err.Error())
	}

	return Success(nil)
}

//从状态中删除实体
func (t *shimTestCC) delete(stub ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return Error("Incorrect number of arguments. Expecting 1")
	}

	A := args[0]

//从分类帐中的状态中删除键
	err := stub.DelState(A)
	if err != nil {
		return Error("Failed to delete state")
	}

	return Success(nil)
}

//
func (t *shimTestCC) query(stub ChaincodeStubInterface, args []string) pb.Response {
var A string //实体
	var err error

	if len(args) != 1 {
		return Error("Incorrect number of arguments. Expecting name of the person to query")
	}

	A = args[0]

//从分类帐中获取状态
	Avalbytes, err := stub.GetState(A)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + A + "\"}"
		return Error(jsonResp)
	}

	if Avalbytes == nil {
		jsonResp := "{\"Error\":\"Nil amount for " + A + "\"}"
		return Error(jsonResp)
	}

	return Success(Avalbytes)
}

//CC2CC呼叫
func (t *shimTestCC) cc2cc(stub ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return Error("Invalid number of args for cc2cc. expecting at least 1")
	}
	return stub.InvokeChaincode(args[0], util.ToChaincodeArgs(args...), "")
}

//rangeq调用范围查询
func (t *shimTestCC) rangeq(stub ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return Error("Incorrect number of arguments. Expecting keys for range query")
	}

	A := args[0]
	B := args[0]

//从分类帐中获取状态
	resultsIterator, err := stub.GetStateByRange(A, B)
	if err != nil {
		return Error(err.Error())
	}
	defer resultsIterator.Close()

//buffer是包含queryresults的JSON数组
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return Error(err.Error())
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

	return Success(buffer.Bytes())
}

//Richq致电Tichq查询
func (t *shimTestCC) richq(stub ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return Error("Incorrect number of arguments. Expecting keys for range query")
	}

	query := args[0]

//从分类帐中获取状态
	resultsIterator, err := stub.GetQueryResult(query)
	if err != nil {
		return Error(err.Error())
	}
	defer resultsIterator.Close()

//buffer是包含queryresults的JSON数组
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return Error(err.Error())
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

	return Success(buffer.Bytes())
}

//rangeq调用范围查询
func (t *shimTestCC) historyq(stub ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return Error("Incorrect number of arguments. Expecting 1")
	}

	key := args[0]

	resultsIterator, err := stub.GetHistoryForKey(key)
	if err != nil {
		return Error(err.Error())
	}
	defer resultsIterator.Close()

	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return Error(err.Error())
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
		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return Success(buffer.Bytes())
}

func (t *shimTestCC) putEP(stub ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	err := stub.SetStateValidationParameter(string(args[1]), args[2])
	if err != nil {
		return Error(err.Error())
	}
	return Success(nil)
}

func (t *shimTestCC) getEP(stub ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	ep, err := stub.GetStateValidationParameter(string(args[1]))
	if err != nil {
		return Error(err.Error())
	}
	return Success(ep)
}

//可以在真正的链代码之外测试的测试go填充程序功能
//语境。

//testshimlogging只是测试API是否工作。这些测试测试
//以正确控制填充程序的日志对象和日志级别
//功能。
func TestShimLogging(t *testing.T) {
	SetLoggingLevel(LogCritical)
	if shimLoggingLevel != LogCritical {
		t.Errorf("shimLoggingLevel is not LogCritical as expected")
	}
	if chaincodeLogger.IsEnabledFor(logging.DEBUG) {
		t.Errorf("The chaincodeLogger should not be enabled for DEBUG")
	}
	if !chaincodeLogger.IsEnabledFor(logging.CRITICAL) {
		t.Errorf("The chaincodeLogger should be enabled for CRITICAL")
	}
	var level LoggingLevel
	var err error
	level, err = LogLevel("debug")
	if err != nil {
		t.Errorf("LogLevel(debug) failed")
	}
	if level != LogDebug {
		t.Errorf("LogLevel(debug) did not return LogDebug")
	}
	level, err = LogLevel("INFO")
	if err != nil {
		t.Errorf("LogLevel(INFO) failed")
	}
	if level != LogInfo {
		t.Errorf("LogLevel(INFO) did not return LogInfo")
	}
	level, err = LogLevel("Notice")
	if err != nil {
		t.Errorf("LogLevel(Notice) failed")
	}
	if level != LogNotice {
		t.Errorf("LogLevel(Notice) did not return LogNotice")
	}
	level, err = LogLevel("WaRnInG")
	if err != nil {
		t.Errorf("LogLevel(WaRnInG) failed")
	}
	if level != LogWarning {
		t.Errorf("LogLevel(WaRnInG) did not return LogWarning")
	}
	level, err = LogLevel("ERRor")
	if err != nil {
		t.Errorf("LogLevel(ERRor) failed")
	}
	if level != LogError {
		t.Errorf("LogLevel(ERRor) did not return LogError")
	}
	level, err = LogLevel("critiCAL")
	if err != nil {
		t.Errorf("LogLevel(critiCAL) failed")
	}
	if level != LogCritical {
		t.Errorf("LogLevel(critiCAL) did not return LogCritical")
	}
	level, err = LogLevel("foo")
	if err == nil {
		t.Errorf("LogLevel(foo) did not fail")
	}
	if level != LogError {
		t.Errorf("LogLevel(foo) did not return LogError")
	}
}

//testchaincodelogging测试chaincode的日志API。
func TestChaincodeLogging(t *testing.T) {

//from start（）-我们不能从此测试调用start（）。
	format := logging.MustStringFormatter("%{time:15:04:05.000} [%{module}] %{level:.4s} : %{message}")
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	logging.SetBackend(backendFormatter).SetLevel(logging.Level(shimLoggingLevel), "shim")

	foo := NewLogger("foo")
	bar := NewLogger("bar")

	foo.Debugf("Foo is debugging: %d", 10)
	bar.Infof("Bar is informational? %s.", "Yes")
	foo.Noticef("NOTE NOTE NOTE")
	bar.Warningf("Danger, Danger %s %s", "Will", "Robinson!")
	foo.Errorf("I'm sorry Dave, I'm afraid I can't do that.")
	bar.Criticalf("PI is not equal to 3.14, we computed it as %.2f", 4.13)

	bar.Debug("Foo is debugging:", 10)
	foo.Info("Bar is informational?", "Yes.")
	bar.Notice("NOTE NOTE NOTE")
	foo.Warning("Danger, Danger", "Will", "Robinson!")
	bar.Error("I'm sorry Dave, I'm afraid I can't do that.")
	foo.Critical("PI is not equal to", 3.14, ", we computed it as", 4.13)

	foo.SetLevel(LogWarning)
	if foo.IsEnabledFor(LogDebug) {
		t.Errorf("'foo' should not be enabled for LogDebug")
	}
	if !foo.IsEnabledFor(LogCritical) {
		t.Errorf("'foo' should be enabled for LogCritical")
	}
	bar.SetLevel(LogCritical)
	if bar.IsEnabledFor(LogDebug) {
		t.Errorf("'bar' should not be enabled for LogDebug")
	}
	if !bar.IsEnabledFor(LogCritical) {
		t.Errorf("'bar' should be enabled for LogCritical")
	}
}

func TestNilEventName(t *testing.T) {
	stub := ChaincodeStub{}
	if err := stub.SetEvent("", []byte("event payload")); err == nil {
		t.Error("Event name can not be nil string.")
	}

}

type testCase struct {
	name         string
	ccLogLevel   string
	shimLogLevel string
}

func TestSetupChaincodeLogging_shim(t *testing.T) {
	var tests = []struct {
		name         string
		ccLogLevel   string
		shimLogLevel string
	}{
		{name: "ValidLevels", ccLogLevel: "debug", shimLogLevel: "warning"},
		{name: "EmptyLevels", ccLogLevel: "", shimLogLevel: ""},
		{name: "BadShimLevel", ccLogLevel: "debug", shimLogLevel: "war"},
		{name: "BadCCLevel", ccLogLevel: "deb", shimLogLevel: "notice"},
		{name: "EmptyShimLevel", ccLogLevel: "error", shimLogLevel: ""},
		{name: "EmptyCCLevel", ccLogLevel: "", shimLogLevel: "critical"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			viper.Set("chaincode.logging.level", tc.ccLogLevel)
			viper.Set("chaincode.logging.shim", tc.shimLogLevel)

			setupChaincodeLogging()

			_, ccErr := logging.LogLevel(tc.ccLogLevel)
			_, shimErr := logging.LogLevel(tc.shimLogLevel)
			if ccErr == nil {
				assert.Equal(t, strings.ToUpper(tc.ccLogLevel), logging.GetLevel("ccLogger").String())
				if shimErr == nil {
					assert.Equal(t, strings.ToUpper(tc.shimLogLevel), logging.GetLevel("shim").String())
				} else {
					assert.Equal(t, strings.ToUpper(tc.ccLogLevel), logging.GetLevel("shim").String())
				}
			} else {
				assert.Equal(t, flogging.DefaultLevel(), logging.GetLevel("ccLogger").String())
				if shimErr == nil {
					assert.Equal(t, strings.ToUpper(tc.shimLogLevel), logging.GetLevel("shim").String())
				} else {
					assert.Equal(t, flogging.DefaultLevel(), logging.GetLevel("shim").String())
				}
			}
		})
	}
}

//在此处存储流CC映射
var mockPeerCCSupport = mockpeer.NewMockPeerSupport()

func mockChaincodeStreamGetter(name string) (PeerChaincodeStream, error) {
	return mockPeerCCSupport.GetCC(name)
}

func setupcc(name string) *mockpeer.MockCCComm {
	viper.Set("chaincode.id.name", name)
	send := make(chan *pb.ChaincodeMessage)
	recv := make(chan *pb.ChaincodeMessage)
	ccSide, _ := mockPeerCCSupport.AddCC(name, recv, send)
	ccSide.SetPong(true)
	return mockPeerCCSupport.GetCCMirror(name)
}

//将此项分配给Done和FailNow并继续使用它们
func setuperror() chan error {
	return make(chan error)
}

func processDone(t *testing.T, done chan error, expecterr bool) {
	err := <-done
	if expecterr != (err != nil) {
		if err == nil {
			t.Fatalf("Expected error but got success")
		} else {
			t.Fatalf("Expected success but got error %s", err)
		}
	}
}

//testinvoke测试init和invoke以及许多存根函数
//例如get/put/del/range…
func TestInvoke(t *testing.T) {
	streamGetter = mockChaincodeStreamGetter
	cc := &shimTestCC{}
//viper.set（“chaincode.logging.shim”，“调试”）。
	ccname := "shimTestCC"
	peerSide := setupcc(ccname)
	defer mockPeerCCSupport.RemoveCC(ccname)
//启动垫片+链条代码
	go Start(cc)

	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	peerDone := make(chan struct{})
	defer close(peerDone)

//启动模拟对等机
	go func() {
		respSet := &mockpeer.MockResponseSet{
			DoneFunc:  errorFunc,
			ErrorFunc: nil,
			Responses: []*mockpeer.MockResponse{
				{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTER}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTERED}},
			},
		}
		peerSide.SetResponses(respSet)
		peerSide.SetKeepAlive(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_KEEPALIVE})
		err := peerSide.Run(peerDone)
		assert.NoError(t, err, "peer side run failed")
	}()

//等待初始化
	processDone(t, done, false)

	channelId := "testchannel"

	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_READY, Txid: "1", ChannelId: channelId})

	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("init"), []byte("A"), []byte("100"), []byte("B"), []byte("200")}, Decorations: nil}
	payload := utils.MarshalOrPanic(ci)
	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Txid: "2"}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "2", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Txid: "2"}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "2", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "2", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

//
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INIT, Payload: payload, Txid: "2", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//好调用
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Txid: "3", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: []byte("100"), Txid: "3", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Txid: "3", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: []byte("200"), Txid: "3", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Txid: "3", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "3", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Txid: "3", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "3", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "3", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("invoke"), []byte("A"), []byte("B"), []byte("10")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "3", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//坏话
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Txid: "3a", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: []byte("100"), Txid: "3a", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Txid: "3a", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: []byte("200"), Txid: "3a", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Txid: "3a", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Txid: "3a", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "3a", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("invoke"), []byte("A"), []byte("B"), []byte("10")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "3a", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//坏得
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Txid: "3b", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Txid: "3b", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "3b", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("invoke"), []byte("A"), []byte("B"), []byte("10")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "3b", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//坏删除
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_DEL_STATE, Txid: "4", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Txid: "4", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "4", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("delete"), []byte("A")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "4", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//好删除
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_DEL_STATE, Txid: "4a", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "4a", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "4a", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("delete"), []byte("A")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "4a", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//不良调用
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "5", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("badinvoke")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "5", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//范围查询

//创建响应
	rangeQueryResponse := &pb.QueryResponse{Results: []*pb.QueryResultBytes{
		{ResultBytes: utils.MarshalOrPanic(&lproto.KV{Namespace: "getputcc", Key: "A", Value: []byte("100")})},
		{ResultBytes: utils.MarshalOrPanic(&lproto.KV{Namespace: "getputcc", Key: "B", Value: []byte("200")})}},
		HasMore: true}
	rangeQPayload := utils.MarshalOrPanic(rangeQueryResponse)

//创建下一个响应
	rangeQueryNext := &pb.QueryResponse{Results: nil, HasMore: false}

	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE_BY_RANGE, Txid: "6", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: rangeQPayload, Txid: "6", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_NEXT, Txid: "6", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: utils.MarshalOrPanic(rangeQueryNext), Txid: "6", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_CLOSE, Txid: "6", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "6", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "6", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("rangeq"), []byte("A"), []byte("B")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "6", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//错误范围查询

//创建响应
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE_BY_RANGE, Txid: "6a", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Payload: payload, Txid: "6a", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "6a", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("rangeq"), []byte("A"), []byte("B")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "6a", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//

//创建响应
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE_BY_RANGE, Txid: "6b", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: rangeQPayload, Txid: "6b", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_NEXT, Txid: "6b", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Txid: "6b", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_CLOSE, Txid: "6b", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "6b", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "6b", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("rangeq"), []byte("A"), []byte("B")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "6b", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//error range query close

//创建响应
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE_BY_RANGE, Txid: "6c", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: rangeQPayload, Txid: "6c", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_NEXT, Txid: "6c", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Txid: "6c", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_CLOSE, Txid: "6c", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Txid: "6c", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "6c", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("rangeq"), []byte("A"), []byte("B")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "6c", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//历史查询

//创建响应
	historyQueryResponse := &pb.QueryResponse{Results: []*pb.QueryResultBytes{
		{ResultBytes: utils.MarshalOrPanic(&lproto.KeyModification{TxId: "6", Value: []byte("100")})}},
		HasMore: true}
	payload = utils.MarshalOrPanic(historyQueryResponse)

	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_HISTORY_FOR_KEY, Txid: "7", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: payload, Txid: "7", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_NEXT, Txid: "7", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: utils.MarshalOrPanic(rangeQueryNext), Txid: "7", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_CLOSE, Txid: "7", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "7", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "7", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("historyq"), []byte("A")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "7", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//error history query

//创建响应
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_HISTORY_FOR_KEY, Txid: "7a", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Payload: payload, Txid: "7a", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "7a", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("historyq"), []byte("A")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "7a", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//查询结果

//创建响应
	getQRResp := &pb.QueryResponse{Results: []*pb.QueryResultBytes{
		{ResultBytes: utils.MarshalOrPanic(&lproto.KV{Namespace: "getputcc", Key: "A", Value: []byte("100")})},
		{ResultBytes: utils.MarshalOrPanic(&lproto.KV{Namespace: "getputcc", Key: "B", Value: []byte("200")})}},
		HasMore: true}
	getQRRespPayload := utils.MarshalOrPanic(getQRResp)

//创建下一个响应
	rangeQueryNext = &pb.QueryResponse{Results: nil, HasMore: false}

	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_QUERY_RESULT, Txid: "8", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: getQRRespPayload, Txid: "8", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_NEXT, Txid: "8", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: utils.MarshalOrPanic(rangeQueryNext), Txid: "8", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_CLOSE, Txid: "8", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "8", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "8", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("richq"), []byte("A")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "8", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//查询结果错误

	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_QUERY_RESULT, Txid: "8a", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Payload: nil, Txid: "8a", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "8a", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("richq"), []byte("A")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "8a", ChannelId: channelId})

//等待完成
	processDone(t, done, false)
}

func TestSetKeyEP(t *testing.T) {
	streamGetter = mockChaincodeStreamGetter
	cc := &shimTestCC{}
	ccname := "shimTestCC"
	peerSide := setupcc(ccname)
	defer mockPeerCCSupport.RemoveCC(ccname)
//启动垫片+链条代码
	go Start(cc)

	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	peerDone := make(chan struct{})
	defer close(peerDone)

//启动模拟对等机
	go func() {
		respSet := &mockpeer.MockResponseSet{
			DoneFunc:  errorFunc,
			ErrorFunc: nil,
			Responses: []*mockpeer.MockResponse{
				{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTER}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTERED}},
			},
		}
		peerSide.SetResponses(respSet)
		peerSide.SetKeepAlive(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_KEEPALIVE})
		err := peerSide.Run(peerDone)
		assert.NoError(t, err, "peer side run failed")
	}()

//等待初始化
	processDone(t, done, false)

	channelID := "testchannel"

	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_READY, Txid: "1", ChannelId: channelID})

	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("init"), []byte("A"), []byte("100"), []byte("B"), []byte("200")}, Decorations: nil}
	payload := utils.MarshalOrPanic(ci)
	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Txid: "2"}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "2", ChannelId: channelID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Txid: "2"}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "2", ChannelId: channelID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "2", ChannelId: channelID}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

//使用从prev init计算的有效负载
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INIT, Payload: payload, Txid: "2", ChannelId: channelID})

	processDone(t, done, false)

//设置一个EP
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE_METADATA, Txid: "4", ChannelId: channelID}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: nil, Txid: "4", ChannelId: channelID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "4", ChannelId: channelID}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("putep"), []byte("A"), []byte("epA")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)

	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "4", ChannelId: channelID})

//等待完成
	processDone(t, done, false)

//设置一个EP
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE_METADATA, Txid: "5", ChannelId: channelID}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: []byte("epA"), Txid: "5", ChannelId: channelID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "5", ChannelId: channelID}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("getep"), []byte("A")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)

	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "5", ChannelId: channelID})

//等待完成
	processDone(t, done, false)

}

func TestStartInProc(t *testing.T) {
	streamGetter = mockChaincodeStreamGetter
	cc := &shimTestCC{}
	ccname := "shimTestCC"
	peerSide := setupcc(ccname)
	defer mockPeerCCSupport.RemoveCC(ccname)

	done := setuperror()

	doneFunc := func(ind int, err error) {
		done <- err
	}

	peerDone := make(chan struct{})
	defer close(peerDone)

//启动模拟对等机
	go func() {
		respSet := &mockpeer.MockResponseSet{
			DoneFunc:  doneFunc,
			ErrorFunc: nil,
			Responses: []*mockpeer.MockResponse{
				{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTER}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTERED}},
			},
		}
		peerSide.SetResponses(respSet)
		peerSide.SetKeepAlive(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_KEEPALIVE})
		err := peerSide.Run(peerDone)
		assert.NoError(t, err, "peer side run failed")
	}()

//启动垫片+链条代码
	go StartInProc([]string{"CORE_CHAINCODE_ID_NAME=shimTestCC", "CORE_CHAINCODE_LOGGING_SHIM=debug"}, nil, cc, peerSide.GetSendStream(), peerSide.GetRecvStream())

//等待初始化
	processDone(t, done, false)

	channelId := "testchannel"
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_READY, Txid: "1", ChannelId: channelId})
}

func TestCC2CC(t *testing.T) {
	streamGetter = mockChaincodeStreamGetter
	cc := &shimTestCC{}
//viper.set（“chaincode.logging.shim”，“调试”）。
	ccname := "shimTestCC"
	peerSide := setupcc(ccname)
	defer mockPeerCCSupport.RemoveCC(ccname)
//启动垫片+链条代码
	go Start(cc)

	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	peerDone := make(chan struct{})
	defer close(peerDone)

//启动模拟对等机
	go func() {
		respSet := &mockpeer.MockResponseSet{
			DoneFunc:  errorFunc,
			ErrorFunc: nil,
			Responses: []*mockpeer.MockResponse{
				{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTER}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTERED}},
			},
		}
		peerSide.SetResponses(respSet)
		peerSide.SetKeepAlive(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_KEEPALIVE})
		err := peerSide.Run(peerDone)
		assert.NoError(t, err, "peer side run failed")
	}()

//等待初始化
	processDone(t, done, false)

	channelId := "testchannel"

	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_READY, Txid: "1", ChannelId: channelId})

	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("init"), []byte("A"), []byte("100"), []byte("B"), []byte("200")}, Decorations: nil}
	payload := utils.MarshalOrPanic(ci)
	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Txid: "2", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "2", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Txid: "2", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: "2", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "2", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

//使用从prev init计算的有效负载
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INIT, Payload: payload, Txid: "2", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//C2CC
	innerResp := utils.MarshalOrPanic(&pb.Response{Status: OK, Payload: []byte("CC2CC rocks")})
	cc2ccresp := utils.MarshalOrPanic(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: innerResp})
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INVOKE_CHAINCODE, Txid: "3", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: cc2ccresp, Txid: "3", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "3", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("cc2cc"), []byte("othercc"), []byte("arg1"), []byte("arg2")}, Decorations: nil}
	payload = utils.MarshalOrPanic(ci)
	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "3", ChannelId: channelId})

//等待完成
	processDone(t, done, false)

//错误响应cc2cc
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: errorFunc,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INVOKE_CHAINCODE, Txid: "4", ChannelId: channelId}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Payload: cc2ccresp, Txid: "4", ChannelId: channelId}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Txid: "4", ChannelId: channelId}, RespMsg: nil},
		},
	}
	peerSide.SetResponses(respSet)

	peerSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION, Payload: payload, Txid: "4", ChannelId: channelId})

//等待完成
	processDone(t, done, false)
}

func TestRealPeerStream(t *testing.T) {
	viper.Set("peer.address", "127.0.0.1:12345")
	_, err := userChaincodeStreamGetter("fake")
	assert.Error(t, err)
}

func TestSend(t *testing.T) {
	ch := make(chan *pb.ChaincodeMessage)

	stream := newInProcStream(ch, ch)

//良好发送（非阻塞发送和接收）
	msg := &pb.ChaincodeMessage{}
	go stream.Send(msg)
	msg2, _ := stream.Recv()
	assert.Equal(t, msg, msg2, "send != recv")

//关闭频道
	close(ch)

//发送错误，应死机、解锁并返回错误
	err := stream.Send(msg)
	assert.NotNil(t, err, "should have errored on panic")
}
