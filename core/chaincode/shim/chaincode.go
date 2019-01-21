
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
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric/bccsp/factory"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	logging "github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

//垫片组记录器。
var chaincodeLogger = logging.MustGetLogger("shim")
var logOutput = os.Stderr

var key string
var cert string

const (
minUnicodeRuneValue   = 0            //U+ 0000
maxUnicodeRuneValue   = utf8.MaxRune //U+10ffff-最大（和未分配）码位
	compositeKeyNamespace = "\x00"
	emptyKeySubstitute    = "\x01"
)

//chaincodestub是传递给chaincode的对象，用于对
//API。
type ChaincodeStub struct {
	TxID                       string
	ChannelId                  string
	chaincodeEvent             *pb.ChaincodeEvent
	args                       [][]byte
	handler                    *Handler
	signedProposal             *pb.SignedProposal
	proposal                   *pb.Proposal
	validationParameterMetakey string

//从已签名的对象中提取的其他字段
	creator   []byte
	transient map[string][]byte
	binding   []byte

	decorations map[string][]byte
}

//从命令行或env var派生的对等地址
var peerAddress string

//
//所以我们可以用模拟对等流来替换它
type peerStreamGetter func(name string) (PeerChaincodeStream, error)

//设置模拟对等流getter的uts
var streamGetter peerStreamGetter

//非模拟用户CC流建立功能
func userChaincodeStreamGetter(name string) (PeerChaincodeStream, error) {
	flag.StringVar(&peerAddress, "peer.address", "", "peer address")
	if viper.GetBool("peer.tls.enabled") {
		keyPath := viper.GetString("tls.client.key.path")
		certPath := viper.GetString("tls.client.cert.path")

		data, err1 := ioutil.ReadFile(keyPath)
		if err1 != nil {
			err1 = errors.Wrap(err1, fmt.Sprintf("error trying to read file content %s", keyPath))
			chaincodeLogger.Errorf("%+v", err1)
			return nil, err1
		}
		key = string(data)

		data, err1 = ioutil.ReadFile(certPath)
		if err1 != nil {
			err1 = errors.Wrap(err1, fmt.Sprintf("error trying to read file content %s", certPath))
			chaincodeLogger.Errorf("%+v", err1)
			return nil, err1
		}
		cert = string(data)
	}

	flag.Parse()

	chaincodeLogger.Debugf("Peer address: %s", getPeerAddress())

//与验证对等端建立连接
	clientConn, err := newPeerClientConnection()
	if err != nil {
		err = errors.Wrap(err, "error trying to connect to local peer")
		chaincodeLogger.Errorf("%+v", err)
		return nil, err
	}

	chaincodeLogger.Debugf("os.Args returns: %s", os.Args)

	chaincodeSupportClient := pb.NewChaincodeSupportClient(clientConn)

//通过验证对等端建立流
	stream, err := chaincodeSupportClient.Register(context.Background())
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("error chatting with leader at address=%s", getPeerAddress()))
	}

	return stream, nil
}

//链码。
func Start(cc Chaincode) error {
//如果调用了start（），我们假设这是一个独立的链代码并设置
//格式化日志记录。
	SetupChaincodeLogging()

	chaincodename := viper.GetString("chaincode.id.name")
	if chaincodename == "" {
		return errors.New("error chaincode id not provided")
	}

	err := factory.InitFactories(factory.GetDefaultOpts())
	if err != nil {
		return errors.WithMessage(err, "internal error, BCCSP could not be initialized with default options")
	}

//模拟流未设置…获取真实流
	if streamGetter == nil {
		streamGetter = userChaincodeStreamGetter
	}

	stream, err := streamGetter(chaincodename)
	if err != nil {
		return err
	}

	err = chatWithPeer(chaincodename, stream, cc)

	return err
}

//IsEnabledForLogLevel检查是否为特定的日志记录级别启用了链码记录器。
//主要用于测试
func IsEnabledForLogLevel(logLevel string) bool {
	lvl, _ := logging.LogLevel(logLevel)
	return chaincodeLogger.IsEnabledFor(lvl)
}

var loggingSetup sync.Once

//设置链码日志记录设置链码日志记录格式和级别
//核心链码记录格式、核心链码记录级别的值
//和core\u chaincode\u logging\u垫片集，从core.yaml通过chaincode\u support.go
func SetupChaincodeLogging() {
	loggingSetup.Do(setupChaincodeLogging)
}

func setupChaincodeLogging() {
//这是1.2中的默认日志配置
	const defaultLogFormat = "%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}"
	const defaultLevel = logging.INFO

	viper.SetEnvPrefix("CORE")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

//设置进程范围日志后端
	logFormat := viper.GetString("chaincode.logging.format")
	if logFormat == "" {
		logFormat = defaultLogFormat
	}

	formatter := logging.MustStringFormatter(logFormat)
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, formatter)
	logging.SetBackend(backendFormatter).SetLevel(defaultLevel, "")

//为所有记录器设置默认日志级别
	chaincodeLogLevelString := viper.GetString("chaincode.logging.level")
	if chaincodeLogLevelString == "" {
		chaincodeLogger.Infof("Chaincode log level not provided; defaulting to: %s", defaultLevel.String())
		chaincodeLogLevelString = defaultLevel.String()
	}

	_, err := LogLevel(chaincodeLogLevelString)
	if err != nil {
		chaincodeLogger.Warningf("Error: '%s' for chaincode log level: %s; defaulting to %s", err, chaincodeLogLevelString, defaultLevel.String())
		chaincodeLogLevelString = defaultLevel.String()
	}

	initFromSpec(chaincodeLogLevelString, defaultLevel)

//覆盖垫片记录器的日志级别-注意：如果该值为
//日志级别为空或无效，然后调用上面的
//`initfromspec`已经设置了默认的日志级别，因此没有操作
//这里是必需的。
	shimLogLevelString := viper.GetString("chaincode.logging.shim")
	if shimLogLevelString != "" {
		shimLogLevel, err := LogLevel(shimLogLevelString)
		if err == nil {
			SetLoggingLevel(shimLogLevel)
		} else {
			chaincodeLogger.Warningf("Error: %s for shim log level: %s", err, shimLogLevelString)
		}
	}

//现在日志记录已经设置好了，打印构建级别。这有助于确保
//链码与对等机匹配。
	buildLevel := viper.GetString("chaincode.buildlevel")
	chaincodeLogger.Infof("Chaincode (build level: %s) starting up ...", buildLevel)
}

//这已从1.2日志记录实现中移除
func initFromSpec(spec string, defaultLevel logging.Level) {
	levelAll := defaultLevel
	var err error

	fields := strings.Split(spec, ":")
	for _, field := range fields {
		split := strings.Split(field, "=")
		switch len(split) {
		case 1:
			if levelAll, err = logging.LogLevel(field); err != nil {
				chaincodeLogger.Warningf("Logging level '%s' not recognized, defaulting to '%s': %s", field, defaultLevel, err)
levelAll = defaultLevel //需要重置，因为原始值被覆盖
			}
		case 2:
//<logger，<logger>…]=<level>
			levelSingle, err := logging.LogLevel(split[1])
			if err != nil {
				chaincodeLogger.Warningf("Invalid logging level in '%s' ignored", field)
				continue
			}

			if split[0] == "" {
				chaincodeLogger.Warningf("Invalid logging override specification '%s' ignored - no logger specified", field)
			} else {
				loggers := strings.Split(split[0], ",")
				for _, logger := range loggers {
					chaincodeLogger.Debugf("Setting logging level for logger '%s' to '%s'", logger, levelSingle)
					logging.SetLevel(levelSingle, logger)
				}
			}
		default:
			chaincodeLogger.Warningf("Invalid logging override '%s' ignored - missing ':'?", field)
		}
	}

logging.SetLevel(levelAll, "") //设置所有记录器的日志级别
}

//StartInProc是系统链码引导的入口点。它不是一个
//链码的API。
func StartInProc(env []string, args []string, cc Chaincode, recv <-chan *pb.ChaincodeMessage, send chan<- *pb.ChaincodeMessage) error {
	chaincodeLogger.Debugf("in proc %v", args)

	var chaincodename string
	for _, v := range env {
		if strings.Index(v, "CORE_CHAINCODE_ID_NAME=") == 0 {
			p := strings.SplitAfter(v, "CORE_CHAINCODE_ID_NAME=")
			chaincodename = p[1]
			break
		}
	}
	if chaincodename == "" {
		return errors.New("error chaincode id not provided")
	}

	stream := newInProcStream(recv, send)
	chaincodeLogger.Debugf("starting chat with peer using name=%s", chaincodename)
	err := chatWithPeer(chaincodename, stream, cc)
	return err
}

func getPeerAddress() string {
	if peerAddress != "" {
		return peerAddress
	}

	if peerAddress = viper.GetString("peer.address"); peerAddress == "" {
		chaincodeLogger.Fatalf("peer.address not configured, can't connect to peer")
	}

	return peerAddress
}

func newPeerClientConnection() (*grpc.ClientConn, error) {
	var peerAddress = getPeerAddress()
//设置keepalive选项以匹配链码服务器的静态设置
	kaOpts := &comm.KeepaliveOptions{
		ClientInterval: time.Duration(1) * time.Minute,
		ClientTimeout:  time.Duration(20) * time.Second,
	}
	if viper.GetBool("peer.tls.enabled") {
		return comm.NewClientConnectionWithAddress(peerAddress, true, true,
			comm.InitTLSForShim(key, cert), kaOpts)
	}
	return comm.NewClientConnectionWithAddress(peerAddress, true, false, nil, kaOpts)
}

func chatWithPeer(chaincodename string, stream PeerChaincodeStream, cc Chaincode) error {
//创建负责所有控制逻辑的填充处理程序
	handler := newChaincodeHandler(stream, cc)
	defer stream.CloseSend()

//在注册期间发送chaincodeid。
	chaincodeID := &pb.ChaincodeID{Name: chaincodename}
	payload, err := proto.Marshal(chaincodeID)
	if err != nil {
		return errors.Wrap(err, "error marshalling chaincodeID during chaincode registration")
	}

//在流上注册
	chaincodeLogger.Debugf("Registering.. sending %s", pb.ChaincodeMessage_REGISTER)
	if err = handler.serialSend(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTER, Payload: payload}); err != nil {
		return errors.WithMessage(err, "error sending chaincode REGISTER")
	}

//保留下面GRPC Recv的返回值
	type recvMsg struct {
		msg *pb.ChaincodeMessage
		err error
	}
	msgAvail := make(chan *recvMsg, 1)
	errc := make(chan error)

	receiveMessage := func() {
		in, err := stream.Recv()
		msgAvail <- &recvMsg{in, err}
	}

	go receiveMessage()
	for {
		select {
		case rmsg := <-msgAvail:
			switch {
			case rmsg.err == io.EOF:
				err = errors.Wrapf(rmsg.err, "received EOF, ending chaincode stream")
				chaincodeLogger.Debugf("%+v", err)
				return err
			case rmsg.err != nil:
				err := errors.Wrap(rmsg.err, "receive failed")
				chaincodeLogger.Errorf("Received error from server, ending chaincode stream: %+v", err)
				return err
			case rmsg.msg == nil:
				err := errors.New("received nil message, ending chaincode stream")
				chaincodeLogger.Debugf("%+v", err)
				return err
			default:
				chaincodeLogger.Debugf("[%s]Received message %s from peer", shorttxid(rmsg.msg.Txid), rmsg.msg.Type)
				err := handler.handleMessage(rmsg.msg, errc)
				if err != nil {
					err = errors.WithMessage(err, "error handling message")
					return err
				}

				go receiveMessage()
			}

		case sendErr := <-errc:
			if sendErr != nil {
				err := errors.Wrap(sendErr, "error sending")
				return err
			}
		}
	}
}

//--初始化存根---
//链码调用功能

func (stub *ChaincodeStub) init(handler *Handler, channelId string, txid string, input *pb.ChaincodeInput, signedProposal *pb.SignedProposal) error {
	stub.TxID = txid
	stub.ChannelId = channelId
	stub.args = input.Args
	stub.handler = handler
	stub.signedProposal = signedProposal
	stub.decorations = input.Decorations
	stub.validationParameterMetakey = pb.MetaDataKeys_VALIDATION_PARAMETER.String()

//TODO:健全性检查：用nil验证对init的每个调用
//SignedProposal是合法的，这意味着它是一个内部调用
//至系统链码。
	if signedProposal != nil {
		var err error

		stub.proposal, err = utils.GetProposal(signedProposal.ProposalBytes)
		if err != nil {
			return errors.WithMessage(err, "failed extracting signedProposal from signed signedProposal")
		}

//
		stub.creator, stub.transient, err = utils.GetChaincodeProposalContext(stub.proposal)
		if err != nil {
			return errors.WithMessage(err, "failed extracting signedProposal fields")
		}

		stub.binding, err = utils.ComputeProposalBinding(stub.proposal)
		if err != nil {
			return errors.WithMessage(err, "failed computing binding from signedProposal")
		}
	}

	return nil
}

//gettxid返回建议的事务ID
func (stub *ChaincodeStub) GetTxID() string {
	return stub.TxID
}

//GETCHANNEURID返回建议的信道
func (stub *ChaincodeStub) GetChannelID() string {
	return stub.ChannelId
}

func (stub *ChaincodeStub) GetDecorations() map[string][]byte {
	return stub.decorations
}

//------------调用链码函数-----------

//
func (stub *ChaincodeStub) InvokeChaincode(chaincodeName string, args [][]byte, channel string) pb.Response {
//在内部，我们将chaincode名称作为复合名称处理
	if channel != "" {
		chaincodeName = chaincodeName + "/" + channel
	}
	return stub.handler.handleInvokeChaincode(chaincodeName, args, stub.ChannelId, stub.TxID)
}

//-------状态函数------

//在interfaces.go中可以找到getstate文档
func (stub *ChaincodeStub) GetState(key string) ([]byte, error) {
//通过将集合设置为空字符串来访问公共数据
	collection := ""
	return stub.handler.handleGetState(collection, key, stub.ChannelId, stub.TxID)
}

//可在interfaces.go中找到setstatevalidationparameter文档
func (stub *ChaincodeStub) SetStateValidationParameter(key string, ep []byte) error {
	return stub.handler.handlePutStateMetadataEntry("", key, stub.validationParameterMetakey, ep, stub.ChannelId, stub.TxID)
}

//在interfaces.go中可以找到getStateValidationParameter文档
func (stub *ChaincodeStub) GetStateValidationParameter(key string) ([]byte, error) {
	md, err := stub.handler.handleGetStateMetadata("", key, stub.ChannelId, stub.TxID)
	if err != nil {
		return nil, err
	}
	if ep, ok := md[stub.validationParameterMetakey]; ok {
		return ep, nil
	}
	return nil, nil
}

//Putstate文档可以在interfaces.go中找到
func (stub *ChaincodeStub) PutState(key string, value []byte) error {
	if key == "" {
		return errors.New("key must not be an empty string")
	}
//通过将集合设置为空字符串来访问公共数据
	collection := ""
	return stub.handler.handlePutState(collection, key, value, stub.ChannelId, stub.TxID)
}

func (stub *ChaincodeStub) createStateQueryIterator(response *pb.QueryResponse) *StateQueryIterator {
	return &StateQueryIterator{CommonIterator: &CommonIterator{
		handler:    stub.handler,
		channelId:  stub.ChannelId,
		txid:       stub.TxID,
		response:   response,
		currentLoc: 0}}
}

//在interfaces.go中可以找到getqueryresult文档
func (stub *ChaincodeStub) GetQueryResult(query string) (StateQueryIteratorInterface, error) {
//通过将集合设置为空字符串来访问公共数据
	collection := ""
//忽略queryresponseMetadata，因为它不适用于没有分页的富查询
	iterator, _, err := stub.handleGetQueryResult(collection, query, nil)

	return iterator, err
}

//Delstate文档可以在interfaces.go中找到
func (stub *ChaincodeStub) DelState(key string) error {
//通过将集合设置为空字符串来访问公共数据
	collection := ""
	return stub.handler.handleDelState(collection, key, stub.ChannelId, stub.TxID)
}

//-------私有状态函数------

//getprivatedata文档可以在interfaces.go中找到
func (stub *ChaincodeStub) GetPrivateData(collection string, key string) ([]byte, error) {
	if collection == "" {
		return nil, fmt.Errorf("collection must not be an empty string")
	}
	return stub.handler.handleGetState(collection, key, stub.ChannelId, stub.TxID)
}

//PutPrivateData文档可以在interfaces.go中找到。
func (stub *ChaincodeStub) PutPrivateData(collection string, key string, value []byte) error {
	if collection == "" {
		return fmt.Errorf("collection must not be an empty string")
	}
	if key == "" {
		return fmt.Errorf("key must not be an empty string")
	}
	return stub.handler.handlePutState(collection, key, value, stub.ChannelId, stub.TxID)
}

//delprivatedata文档可以在interfaces.go中找到
func (stub *ChaincodeStub) DelPrivateData(collection string, key string) error {
	if collection == "" {
		return fmt.Errorf("collection must not be an empty string")
	}
	return stub.handler.handleDelState(collection, key, stub.ChannelId, stub.TxID)
}

//在interfaces.go中可以找到getprivatedatabyrange文档
func (stub *ChaincodeStub) GetPrivateDataByRange(collection, startKey, endKey string) (StateQueryIteratorInterface, error) {
	if collection == "" {
		return nil, fmt.Errorf("collection must not be an empty string")
	}
	if startKey == "" {
		startKey = emptyKeySubstitute
	}
	if err := validateSimpleKeys(startKey, endKey); err != nil {
		return nil, err
	}
//忽略queryresponseMetadata，因为它不适用于没有分页的范围查询
	iterator, _, err := stub.handleGetStateByRange(collection, startKey, endKey, nil)

	return iterator, err
}

func (stub *ChaincodeStub) createRangeKeysForPartialCompositeKey(objectType string, attributes []string) (string, string, error) {
	partialCompositeKey, err := stub.CreateCompositeKey(objectType, attributes)
	if err != nil {
		return "", "", err
	}
	startKey := partialCompositeKey
	endKey := partialCompositeKey + string(maxUnicodeRuneValue)

	return startKey, endKey, nil
}

//getprivatedatabypartalcompositekey文档可以在interfaces.go中找到。
func (stub *ChaincodeStub) GetPrivateDataByPartialCompositeKey(collection, objectType string, attributes []string) (StateQueryIteratorInterface, error) {
	if collection == "" {
		return nil, fmt.Errorf("collection must not be an empty string")
	}

	startKey, endKey, err := stub.createRangeKeysForPartialCompositeKey(objectType, attributes)
	if err != nil {
		return nil, err
	}
//忽略queryresponseMetadata，因为它不适用于不带分页的部分复合键查询
	iterator, _, err := stub.handleGetStateByRange(collection, startKey, endKey, nil)

	return iterator, err
}

//在interfaces.go中可以找到getprivatedataqueryresult文档
func (stub *ChaincodeStub) GetPrivateDataQueryResult(collection, query string) (StateQueryIteratorInterface, error) {
	if collection == "" {
		return nil, fmt.Errorf("collection must not be an empty string")
	}
//忽略queryresponseMetadata，因为它不适用于没有分页的范围查询
	iterator, _, err := stub.handleGetQueryResult(collection, query, nil)

	return iterator, err
}

//在interfaces.go中可以找到getprivatedatavalidationparameter文档
func (stub *ChaincodeStub) GetPrivateDataValidationParameter(collection, key string) ([]byte, error) {
	md, err := stub.handler.handleGetStateMetadata(collection, key, stub.ChannelId, stub.TxID)
	if err != nil {
		return nil, err
	}
	if ep, ok := md[stub.validationParameterMetakey]; ok {
		return ep, nil
	}
	return nil, nil
}

//setprivatedatavalidationparameter文档可以在interfaces.go中找到
func (stub *ChaincodeStub) SetPrivateDataValidationParameter(collection, key string, ep []byte) error {
	return stub.handler.handlePutStateMetadataEntry(collection, key, stub.validationParameterMetakey, ep, stub.ChannelId, stub.TxID)
}

//CommonIterator文档可以在interfaces.go中找到
type CommonIterator struct {
	handler    *Handler
	channelId  string
	txid       string
	response   *pb.QueryResponse
	currentLoc int
}

//StateQueryIterator文档可以在interfaces.go中找到
type StateQueryIterator struct {
	*CommonIterator
}

//HistoryQueryIterator文档可以在interfaces.go中找到
type HistoryQueryIterator struct {
	*CommonIterator
}

type resultType uint8

const (
	STATE_QUERY_RESULT resultType = iota + 1
	HISTORY_QUERY_RESULT
)

func createQueryResponseMetadata(metadataBytes []byte) (*pb.QueryResponseMetadata, error) {
	metadata := &pb.QueryResponseMetadata{}
	err := proto.Unmarshal(metadataBytes, metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

func (stub *ChaincodeStub) handleGetStateByRange(collection, startKey, endKey string,
	metadata []byte) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {

	response, err := stub.handler.handleGetStateByRange(collection, startKey, endKey, metadata, stub.ChannelId, stub.TxID)
	if err != nil {
		return nil, nil, err
	}

	iterator := stub.createStateQueryIterator(response)
	responseMetadata, err := createQueryResponseMetadata(response.Metadata)
	if err != nil {
		return nil, nil, err
	}

	return iterator, responseMetadata, nil
}

func (stub *ChaincodeStub) handleGetQueryResult(collection, query string,
	metadata []byte) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {

	response, err := stub.handler.handleGetQueryResult(collection, query, metadata, stub.ChannelId, stub.TxID)
	if err != nil {
		return nil, nil, err
	}

	iterator := stub.createStateQueryIterator(response)
	responseMetadata, err := createQueryResponseMetadata(response.Metadata)
	if err != nil {
		return nil, nil, err
	}

	return iterator, responseMetadata, nil
}

//在interfaces.go中可以找到GetStateByRange文档
func (stub *ChaincodeStub) GetStateByRange(startKey, endKey string) (StateQueryIteratorInterface, error) {
	if startKey == "" {
		startKey = emptyKeySubstitute
	}
	if err := validateSimpleKeys(startKey, endKey); err != nil {
		return nil, err
	}
	collection := ""

//忽略queryresponseMetadata，因为它不适用于没有分页的范围查询
	iterator, _, err := stub.handleGetStateByRange(collection, startKey, endKey, nil)

	return iterator, err
}

//在interfaces.go中可以找到getHistoryForkey文档
func (stub *ChaincodeStub) GetHistoryForKey(key string) (HistoryQueryIteratorInterface, error) {
	response, err := stub.handler.handleGetHistoryForKey(key, stub.ChannelId, stub.TxID)
	if err != nil {
		return nil, err
	}
	return &HistoryQueryIterator{CommonIterator: &CommonIterator{stub.handler, stub.ChannelId, stub.TxID, response, 0}}, nil
}

//可在interfaces.go中找到CreateCompositeKey文档
func (stub *ChaincodeStub) CreateCompositeKey(objectType string, attributes []string) (string, error) {
	return createCompositeKey(objectType, attributes)
}

//splitcompositekey文档可以在interfaces.go中找到
func (stub *ChaincodeStub) SplitCompositeKey(compositeKey string) (string, []string, error) {
	return splitCompositeKey(compositeKey)
}

func createCompositeKey(objectType string, attributes []string) (string, error) {
	if err := validateCompositeKeyAttribute(objectType); err != nil {
		return "", err
	}
	ck := compositeKeyNamespace + objectType + string(minUnicodeRuneValue)
	for _, att := range attributes {
		if err := validateCompositeKeyAttribute(att); err != nil {
			return "", err
		}
		ck += att + string(minUnicodeRuneValue)
	}
	return ck, nil
}

func splitCompositeKey(compositeKey string) (string, []string, error) {
	componentIndex := 1
	components := []string{}
	for i := 1; i < len(compositeKey); i++ {
		if compositeKey[i] == minUnicodeRuneValue {
			components = append(components, compositeKey[componentIndex:i])
			componentIndex = i + 1
		}
	}
	return components[0], components[1:], nil
}

func validateCompositeKeyAttribute(str string) error {
	if !utf8.ValidString(str) {
		return errors.Errorf("not a valid utf8 string: [%x]", str)
	}
	for index, runeValue := range str {
		if runeValue == minUnicodeRuneValue || runeValue == maxUnicodeRuneValue {
			return errors.Errorf(`input contain unicode %#U starting at position [%d]. %#U and %#U are not allowed in the input attribute of a composite key`,
				runeValue, index, minUnicodeRuneValue, maxUnicodeRuneValue)
		}
	}
	return nil
}

//
//我们验证simplekey以检查该键是否以0x00开头（哪个
//是compositekey的命名空间）。这有助于避免简单/复合
//关键碰撞。
func validateSimpleKeys(simpleKeys ...string) error {
	for _, key := range simpleKeys {
		if len(key) > 0 && key[0] == compositeKeyNamespace[0] {
			return errors.Errorf(`first character of the key [%s] contains a null character which is not allowed`, key)
		}
	}
	return nil
}

//可以通过chaincode调用getstatebypartialcompositekey函数来查询
//基于给定的部分复合键的状态。此函数返回
//迭代器，可用于对前缀为
//匹配给定的部分复合键。此函数只能用于
//部分复合键。对于完整的复合键，带有空响应的ITER
//将被退回。
func (stub *ChaincodeStub) GetStateByPartialCompositeKey(objectType string, attributes []string) (StateQueryIteratorInterface, error) {
	collection := ""
	startKey, endKey, err := stub.createRangeKeysForPartialCompositeKey(objectType, attributes)
	if err != nil {
		return nil, err
	}
//忽略queryresponseMetadata，因为它不适用于不带分页的部分复合键查询
	iterator, _, err := stub.handleGetStateByRange(collection, startKey, endKey, nil)

	return iterator, err
}

func createQueryMetadata(pageSize int32, bookmark string) ([]byte, error) {
//使用分页所需的页面大小和书签构造querymetadata
	metadata := &pb.QueryMetadata{PageSize: pageSize, Bookmark: bookmark}
	metadataBytes, err := proto.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	return metadataBytes, nil
}

func (stub *ChaincodeStub) GetStateByRangeWithPagination(startKey, endKey string, pageSize int32,
	bookmark string) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {

	if startKey == "" {
		startKey = emptyKeySubstitute
	}
	if err := validateSimpleKeys(startKey, endKey); err != nil {
		return nil, nil, err
	}

	collection := ""

	metadata, err := createQueryMetadata(pageSize, bookmark)
	if err != nil {
		return nil, nil, err
	}

	return stub.handleGetStateByRange(collection, startKey, endKey, metadata)
}

func (stub *ChaincodeStub) GetStateByPartialCompositeKeyWithPagination(objectType string, keys []string,
	pageSize int32, bookmark string) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {

	collection := ""

	metadata, err := createQueryMetadata(pageSize, bookmark)
	if err != nil {
		return nil, nil, err
	}

	startKey, endKey, err := stub.createRangeKeysForPartialCompositeKey(objectType, keys)
	if err != nil {
		return nil, nil, err
	}
	return stub.handleGetStateByRange(collection, startKey, endKey, metadata)
}

func (stub *ChaincodeStub) GetQueryResultWithPagination(query string, pageSize int32,
	bookmark string) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
//通过将集合设置为空字符串来访问公共数据
	collection := ""

	metadata, err := createQueryMetadata(pageSize, bookmark)
	if err != nil {
		return nil, nil, err
	}
	return stub.handleGetQueryResult(collection, query, metadata)
}

func (iter *StateQueryIterator) Next() (*queryresult.KV, error) {
	if result, err := iter.nextResult(STATE_QUERY_RESULT); err == nil {
		return result.(*queryresult.KV), err
	} else {
		return nil, err
	}
}

func (iter *HistoryQueryIterator) Next() (*queryresult.KeyModification, error) {
	if result, err := iter.nextResult(HISTORY_QUERY_RESULT); err == nil {
		return result.(*queryresult.KeyModification), err
	} else {
		return nil, err
	}
}

//HasNext文档可以在interfaces.go中找到
func (iter *CommonIterator) HasNext() bool {
	if iter.currentLoc < len(iter.response.Results) || iter.response.HasMore {
		return true
	}
	return false
}

//GetResultsFromBytes反序列化QueryResult并返回Kv结构
//或键修改，取决于结果类型（即状态（范围/执行）
//查询，历史查询）。注意commonledger.queryresult是一个空的golang
//可以保存任何类型的值的接口。
func (iter *CommonIterator) getResultFromBytes(queryResultBytes *pb.QueryResultBytes,
	rType resultType) (commonledger.QueryResult, error) {

	if rType == STATE_QUERY_RESULT {
		stateQueryResult := &queryresult.KV{}
		if err := proto.Unmarshal(queryResultBytes.ResultBytes, stateQueryResult); err != nil {
			return nil, errors.Wrap(err, "error unmarshaling result from bytes")
		}
		return stateQueryResult, nil

	} else if rType == HISTORY_QUERY_RESULT {
		historyQueryResult := &queryresult.KeyModification{}
		if err := proto.Unmarshal(queryResultBytes.ResultBytes, historyQueryResult); err != nil {
			return nil, err
		}
		return historyQueryResult, nil
	}
	return nil, errors.New("wrong result type")
}

func (iter *CommonIterator) fetchNextQueryResult() error {
	if response, err := iter.handler.handleQueryStateNext(iter.response.Id, iter.channelId, iter.txid); err == nil {
		iter.currentLoc = 0
		iter.response = response
		return nil
	} else {
		return err
	}
}

//NextResult返回下一个查询结果（即，kv结构或keyModification）
//从状态或历史查询迭代器。注意commonledger.queryresult是一个
//空的golang接口，它可以保存任何类型的值。
func (iter *CommonIterator) nextResult(rType resultType) (commonledger.QueryResult, error) {
	if iter.currentLoc < len(iter.response.Results) {
//从缓存结果有效访问元素时
		queryResult, err := iter.getResultFromBytes(iter.response.Results[iter.currentLoc], rType)
		if err != nil {
			chaincodeLogger.Errorf("Failed to decode query results: %+v", err)
			return nil, err
		}
		iter.currentLoc++

		if iter.currentLoc == len(iter.response.Results) && iter.response.HasMore {
//访问最后一项时，预取以更新hasmore标志
			if err = iter.fetchNextQueryResult(); err != nil {
				chaincodeLogger.Errorf("Failed to fetch next results: %+v", err)
				return nil, err
			}
		}

		return queryResult, err
	} else if !iter.response.HasMore {
//调用next（）时不检查hasmore
		return nil, errors.New("no such key")
	}

//不应该从这里掉下去
//案例：没有缓存结果，但hasmore为真。
	return nil, errors.New("invalid iterator state")
}

//关闭文档可以在interfaces.go中找到。
func (iter *CommonIterator) Close() error {
	_, err := iter.handler.handleQueryStateClose(iter.response.Id, iter.channelId, iter.txid)
	return err
}

//
func (stub *ChaincodeStub) GetArgs() [][]byte {
	return stub.args
}

//
func (stub *ChaincodeStub) GetStringArgs() []string {
	args := stub.GetArgs()
	strargs := make([]string, 0, len(args))
	for _, barg := range args {
		strargs = append(strargs, string(barg))
	}
	return strargs
}

//GetFunctionAndParameters documentation can be found in interfaces.go
func (stub *ChaincodeStub) GetFunctionAndParameters() (function string, params []string) {
	allargs := stub.GetStringArgs()
	function = ""
	params = []string{}
	if len(allargs) >= 1 {
		function = allargs[0]
		params = allargs[1:]
	}
	return
}

//getcreator文档可以在interfaces.go中找到
func (stub *ChaincodeStub) GetCreator() ([]byte, error) {
	return stub.creator, nil
}

//在interfaces.go中可以找到getTransient文档
func (stub *ChaincodeStub) GetTransient() (map[string][]byte, error) {
	return stub.transient, nil
}

//在interfaces.go中可以找到getbinding文档
func (stub *ChaincodeStub) GetBinding() ([]byte, error) {
	return stub.binding, nil
}

//在interfaces.go中可以找到getSignedProposal文档
func (stub *ChaincodeStub) GetSignedProposal() (*pb.SignedProposal, error) {
	return stub.signedProposal, nil
}

//在interfaces.go中可以找到getargsslice文档
func (stub *ChaincodeStub) GetArgsSlice() ([]byte, error) {
	args := stub.GetArgs()
	res := []byte{}
	for _, barg := range args {
		res = append(res, barg...)
	}
	return res, nil
}

//在interfaces.go中可以找到getxtimestamp文档
func (stub *ChaincodeStub) GetTxTimestamp() (*timestamp.Timestamp, error) {
	hdr, err := utils.GetHeader(stub.proposal.Header)
	if err != nil {
		return nil, err
	}
	chdr, err := utils.UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		return nil, err
	}

	return chdr.GetTimestamp(), nil
}

//

//
func (stub *ChaincodeStub) SetEvent(name string, payload []byte) error {
	if name == "" {
		return errors.New("event name can not be nil string")
	}
	stub.chaincodeEvent = &pb.ChaincodeEvent{EventName: name, Payload: payload}
	return nil
}

//-------------日志控制和链式记录器--------------------------------------------------------------------------------------------------------

//As independent programs, Go language chaincodes can use any logging
//methodology they choose, from simple fmt.Printf() to os.Stdout, to
//decorated logs created by the author's favorite logging package. 这个
//chaincode "shim" interface, however, is defined by the Hyperledger fabric
//and implements its own logging methodology. This methodology currently
//includes severity-based logging control and a standard way of decorating
//原木。
//
//The facilities defined here allow a Go language chaincode to control the
//记录其填充程序的级别，并创建自己的格式化日志
//与垫片日志一致，并与垫片日志临时交错
//了解垫片的底层实现，不了解
//其他包装要求。尤其是缺少包装要求
//重要的是，即使链代码恰好显式使用相同的
//将程序包作为垫片记录，除非链码实际包含为
//part of the hyperledger fabric source code tree it could actually end up
//使用日志包的不同二进制实例
//formats and severity levels than the binary package used by the shim.
//
//另一种可能已经采取并可能采取的方法
//在将来，将由chaincode为
//the shim to use, rather than the other way around as implemented
//在这里。There would be some complexities associated with that approach, so
//for the moment we have chosen the simpler implementation below. 垫片
//提供一个或多个抽象日志记录对象，用于通过
//newlogger（）API，并允许chaincode控制严重性级别
//使用setLoggingLevel（）API填充日志。

//LoggingLevel是控制
//链码日志记录。
type LoggingLevel logging.Level

//这些常量包含loggingLevel枚举
const (
	LogDebug    = LoggingLevel(logging.DEBUG)
	LogInfo     = LoggingLevel(logging.INFO)
	LogNotice   = LoggingLevel(logging.NOTICE)
	LogWarning  = LoggingLevel(logging.WARNING)
	LogError    = LoggingLevel(logging.ERROR)
	LogCritical = LoggingLevel(logging.CRITICAL)
)

var shimLoggingLevel = LogInfo //正确初始化所必需的；请参阅start（）。

//setLoggingLevel允许Go语言链码设置
//它的垫片。
func SetLoggingLevel(level LoggingLevel) {
	shimLoggingLevel = level
	logging.SetLevel(logging.Level(level), "shim")
}

//loglevel转换从critical、error、
//WARNING, NOTICE, INFO or DEBUG into an element of the LoggingLevel
//类型。如果出现错误，返回的级别为logerror。
func LogLevel(levelString string) (LoggingLevel, error) {
	l, err := logging.LogLevel(levelString)
	level := LoggingLevel(l)
	if err != nil {
		level = LogError
	}
	return level, err
}

//------------链码记录器-----------

//chaincodelogger是日志对象的抽象，供
//链码。这些对象是由newlogger API创建的。
type ChaincodeLogger struct {
	logger *logging.Logger
}

//NewLogger允许Go语言链码创建一个或多个日志记录
//其日志的格式将与和临时一致的对象
//与填充程序接口创建的日志交错。创建的日志
//通过提供的名称，可以将此对象与填充程序日志区分开来，
//会出现在日志中。
func NewLogger(name string) *ChaincodeLogger {
	return &ChaincodeLogger{logging.MustGetLogger(name)}
}

//setlevel设置链码记录器的日志记录级别。注意，目前
//当日志记录程序
//因此，日志记录程序应该被赋予除“shim”之外的唯一名称。
func (c *ChaincodeLogger) SetLevel(level LoggingLevel) {
	logging.SetLevel(logging.Level(level), c.logger.Module)
}

//如果记录器被启用在
//给定的日志记录级别。
func (c *ChaincodeLogger) IsEnabledFor(level LoggingLevel) bool {
	return c.logger.IsEnabledFor(logging.Level(level))
}

//只有当chaincodelogger loggingLevel设置为时，才会出现调试日志。
//LogDebug。
func (c *ChaincodeLogger) Debug(args ...interface{}) {
	c.logger.Debug(args...)
}

//如果chaincodelogger loggingLevel设置为
//
func (c *ChaincodeLogger) Info(args ...interface{}) {
	c.logger.Info(args...)
}

//如果chaincodelogger loggingLevel设置为
//lognotice、loginfo或logdebug。
func (c *ChaincodeLogger) Notice(args ...interface{}) {
	c.logger.Notice(args...)
}

//如果chaincodelogger loggingLevel设置为，将显示警告日志。
//logwarning、lognotice、loginfo或logdebug。
func (c *ChaincodeLogger) Warning(args ...interface{}) {
	c.logger.Warning(args...)
}

//如果chaincodelogger loggingLevel设置为，则将显示错误日志。
//logerror、logwarning、lognotice、loginfo或logdebug。
func (c *ChaincodeLogger) Error(args ...interface{}) {
	c.logger.Error(args...)
}

//关键日志总是出现；不能禁用。
func (c *ChaincodeLogger) Critical(args ...interface{}) {
	c.logger.Critical(args...)
}

//只有当chaincodelogger loggingLevel设置为时，才会出现debugf日志。
//LogDebug。
func (c *ChaincodeLogger) Debugf(format string, args ...interface{}) {
	c.logger.Debugf(format, args...)
}

//如果chaincodelogger loggingLevel设置为
//loginfo或logdebug。
func (c *ChaincodeLogger) Infof(format string, args ...interface{}) {
	c.logger.Infof(format, args...)
}

//如果设置ChaincodeLogger LoggingLevel，将出现通知日志。
//lognotice、loginfo或logdebug。
func (c *ChaincodeLogger) Noticef(format string, args ...interface{}) {
	c.logger.Noticef(format, args...)
}

//如果chaincodelogger loggingLevel设置为，将显示警告日志。
//logwarning、lognotice、loginfo或logdebug。
func (c *ChaincodeLogger) Warningf(format string, args ...interface{}) {
	c.logger.Warningf(format, args...)
}

//如果chaincodelogger loggingLevel设置为
//logerror、logwarning、lognotice、loginfo或logdebug。
func (c *ChaincodeLogger) Errorf(format string, args ...interface{}) {
	c.logger.Errorf(format, args...)
}

//关键日志总是出现；不能禁用。
func (c *ChaincodeLogger) Criticalf(format string, args ...interface{}) {
	c.logger.Criticalf(format, args...)
}
