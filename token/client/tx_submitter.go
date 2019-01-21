
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

package client

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/comm"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	peercommon "github.com/hyperledger/fabric/peer/common"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token.client")

type TxSubmitter struct {
	Config        *ClientConfig
	Signer        SignerIdentity
	Creator       []byte
	OrdererClient OrdererClient
	DeliverClient DeliverClient
}

//TxEvent包含令牌事务提交的信息
//如果应用程序希望在提交令牌事务时得到通知，
//执行以下操作：
//-创建大小为1或更大的事件chan，例如txchan:=make（chan txevent，1）
//-调用client.submitTransactionWithChan（txBytes，txChan）
//-实现从txchan读取txevent的函数，以便在事务提交或失败时通知它。
type TxEvent struct {
	Txid       string
	Committed  bool
	CommitPeer string
	Err        error
}

//NewTransactionSubmitter从令牌客户端配置创建新的TxSubmitter
func NewTxSubmitter(config *ClientConfig) (*TxSubmitter, error) {
	err := ValidateClientConfig(config)
	if err != nil {
		return nil, err
	}

//TODO:使msptype可配置
	mspType := "bccsp"
	peercommon.InitCrypto(config.MspDir, config.MspId, mspType)

	Signer, err := mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		return nil, err
	}
	creator, err := Signer.Serialize()
	if err != nil {
		return nil, err
	}

	ordererClient, err := NewOrdererClient(config)
	if err != nil {
		return nil, err
	}

	deliverClient, err := NewDeliverClient(config)
	if err != nil {
		return nil, err
	}

	return &TxSubmitter{
		Config:        config,
		Signer:        Signer,
		Creator:       creator,
		OrdererClient: ordererClient,
		DeliverClient: deliverClient,
	}, nil
}

//SubmitTransaction向Fabric提交令牌事务。
//它将tokenTransaction字节和waitTimeInSeconds作为输入参数。
//“waitTimeInSeconds”表示等待事务提交事件的时间。
//如果为0，函数将不等待提交事务。
//如果它大于0，函数将等待直到提交超时或事务，以较早者为准。
func (s *TxSubmitter) SubmitTransaction(txEnvelope *common.Envelope, waitTimeInSeconds int) (committed bool, txId string, err error) {
	if waitTimeInSeconds > 0 {
		waitTime := time.Second * time.Duration(waitTimeInSeconds)
		ctx, cancelFunc := context.WithTimeout(context.Background(), waitTime)
		defer cancelFunc()
		localCh := make(chan TxEvent, 1)
		committed, txId, err = s.sendTransactionInternal(txEnvelope, ctx, localCh, true)
		close(localCh)
		return
	} else {
		committed, txId, err = s.sendTransactionInternal(txEnvelope, context.Background(), nil, false)
		return
	}
}

//SubmitTransactionWithChan使用事件通道向Fabric提交令牌事务。
//此函数不等待事务提交，并在订购方客户机收到响应后立即返回。
//在事务完成时，将通过从事件ch中读取事件来通知应用程序。
//当事务提交或失败时，将向Eventch添加一个事件，以便通知应用程序。
//如果eventch的缓冲区大小为0或其缓冲区已满，将返回一个错误。
func (s *TxSubmitter) SubmitTransactionWithChan(txEnvelope *common.Envelope, eventCh chan TxEvent) (committed bool, txId string, err error) {
	committed, txId, err = s.sendTransactionInternal(txEnvelope, context.Background(), eventCh, false)
	return
}

func (s *TxSubmitter) sendTransactionInternal(txEnvelope *common.Envelope, ctx context.Context, eventCh chan TxEvent, waitForCommit bool) (bool, string, error) {
	if eventCh != nil && cap(eventCh) == 0 {
		return false, "", errors.New("eventCh buffer size must be greater than 0")
	}
	if eventCh != nil && len(eventCh) == cap(eventCh) {
		return false, "", errors.New("eventCh buffer is full. Read events and try again")
	}

	txid, err := getTransactionId(txEnvelope)
	if err != nil {
		return false, "", err
	}

	broadcast, err := s.OrdererClient.NewBroadcast(context.Background())
	if err != nil {
		return false, "", err
	}

	committed := false
	if eventCh != nil {
		deliverFiltered, err := s.DeliverClient.NewDeliverFiltered(ctx)
		if err != nil {
			return false, "", err
		}
		blockEnvelope, err := CreateDeliverEnvelope(s.Config.ChannelId, s.Creator, s.Signer, s.DeliverClient.Certificate())
		if err != nil {
			return false, "", err
		}
		err = DeliverSend(deliverFiltered, s.Config.CommitPeerCfg.Address, blockEnvelope)
		if err != nil {
			return false, "", err
		}
		go DeliverReceive(deliverFiltered, s.Config.CommitPeerCfg.Address, txid, eventCh)
	}

	err = BroadcastSend(broadcast, s.Config.OrdererCfg.Address, txEnvelope)
	if err != nil {
		return false, txid, err
	}

//等待订购方广播的响应-它不等待提交对等响应
	responses := make(chan common.Status)
	errs := make(chan error, 1)
	go BroadcastReceive(broadcast, s.Config.OrdererCfg.Address, responses, errs)
	_, err = BroadcastWaitForResponse(responses, errs)

//在这种情况下，等待来自传递服务的提交事件
	if eventCh != nil && waitForCommit {
		committed, err = DeliverWaitForResponse(ctx, eventCh, txid)
	}

	return committed, txid, err
}

func (s *TxSubmitter) CreateTxEnvelope(txBytes []byte) (string, *common.Envelope, error) {
//channelid string，creator[]byte，signer signeridentity，cert*tls.certificate
//，s.config.channelid，s.creator，s.signer，s.orderclient.certificate（）。
	var tlsCertHash []byte
	var err error
//检查客户端证书并在证书上计算sha2-256（如果存在）
	cert := s.OrdererClient.Certificate()
	if cert != nil && len(cert.Certificate) > 0 {
		tlsCertHash, err = factory.GetDefault().Hash(cert.Certificate[0], &bccsp.SHA256Opts{})
		if err != nil {
			err = errors.New("failed to compute SHA256 on client certificate")
			logger.Errorf("%s", err)
			return "", nil, err
		}
	}

	txid, header, err := CreateHeader(common.HeaderType_TOKEN_TRANSACTION, s.Config.ChannelId, s.Creator, tlsCertHash)
	if err != nil {
		return txid, nil, err
	}

	txEnvelope, err := CreateEnvelope(txBytes, header, s.Signer)
	return txid, txEnvelope, err
}

//CreateHeader为令牌事务创建common.Header
//tlscerthash用于客户端tls证书，仅在clientauthRequired为true时适用。
func CreateHeader(txType common.HeaderType, channelId string, creator []byte, tlsCertHash []byte) (string, *common.Header, error) {
	ts, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return "", nil, err
	}

	nonce, err := crypto.GetRandomNonce()
	if err != nil {
		return "", nil, err
	}

	txId, err := utils.ComputeTxID(nonce, creator)
	if err != nil {
		return "", nil, err
	}

	chdr := &common.ChannelHeader{
		Type:        int32(txType),
		ChannelId:   channelId,
		TxId:        txId,
		Epoch:       uint64(0),
		Timestamp:   ts,
		TlsCertHash: tlsCertHash,
	}
	chdrBytes, err := proto.Marshal(chdr)
	if err != nil {
		return "", nil, err
	}

	shdr := &common.SignatureHeader{
		Creator: creator,
		Nonce:   nonce,
	}
	shdrBytes, err := proto.Marshal(shdr)
	if err != nil {
		return "", nil, err
	}

	header := &common.Header{
		ChannelHeader:   chdrBytes,
		SignatureHeader: shdrBytes,
	}

	return txId, header, nil
}

//CreateEnvelope使用给定的Tx字节、头和签名者创建一个common.Envelope
func CreateEnvelope(data []byte, header *common.Header, signer SignerIdentity) (*common.Envelope, error) {
	payload := &common.Payload{
		Header: header,
		Data:   data,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal common.Payload")
	}

	signature, err := signer.Sign(payloadBytes)
	if err != nil {
		return nil, err
	}

	txEnvelope := &common.Envelope{
		Payload:   payloadBytes,
		Signature: signature,
	}

	return txEnvelope, nil
}

func getTransactionId(txEnvelope *common.Envelope) (string, error) {
	payload := common.Payload{}
	err := proto.Unmarshal(txEnvelope.Payload, &payload)
	if err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal envelope payload")
	}

	channelHeader := common.ChannelHeader{}
	err = proto.Unmarshal(payload.Header.ChannelHeader, &channelHeader)
	if err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal channel header")
	}

	return channelHeader.TxId, nil
}

//creategrpclient基于toke客户端配置返回comm.grpcclient
func createGrpcClient(cfg *ConnectionConfig, tlsEnabled bool) (*comm.GRPCClient, error) {
	clientConfig := comm.ClientConfig{Timeout: time.Second}

	if tlsEnabled {
		if cfg.TlsRootCertFile == "" {
			return nil, errors.New("missing TlsRootCertFile in client config")
		}
		caPEM, err := ioutil.ReadFile(cfg.TlsRootCertFile)
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("unable to load TLS cert from %s", cfg.TlsRootCertFile))
		}
		secOpts := &comm.SecureOptions{
			UseTLS:            true,
			ServerRootCAs:     [][]byte{caPEM},
			RequireClientCert: false,
		}
		clientConfig.SecOpts = secOpts
	}

	return comm.NewGRPCClient(clientConfig)
}
