
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
	"crypto/tls"
	"fmt"
	"math"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

//go：生成仿冒者-o模拟/交付\过滤。go-伪造名称交付过滤。交付过滤

//deliver filtered定义抽象传递筛选后的grpc调用以提交对等方的接口
type DeliverFiltered interface {
	Send(*common.Envelope) error
	Recv() (*pb.DeliverResponse, error)
	CloseSend() error
}

//go：生成伪造者-o mock/delivery-client.go-forke-name deliver client。交付客户

//DeliverClient定义用于创建DeliverFiltered客户端的接口
type DeliverClient interface {
//newdeliverfilterd返回deliverfiltered
	NewDeliverFiltered(ctx context.Context, opts ...grpc.CallOption) (DeliverFiltered, error)

//证书返回传递客户端提交对等端的TLS证书
	Certificate() *tls.Certificate
}

//DeliverClient实现DeliverClient接口
type deliverClient struct {
	peerAddr           string
	serverNameOverride string
	grpcClient         *comm.GRPCClient
	conn               *grpc.ClientConn
}

func NewDeliverClient(config *ClientConfig) (DeliverClient, error) {
	grpcClient, err := createGrpcClient(&config.CommitPeerCfg, config.TlsEnabled)
	if err != nil {
		err = errors.WithMessage(err, fmt.Sprintf("failed to create a GRPCClient to peer %s", config.CommitPeerCfg.Address))
		logger.Errorf("%s", err)
		return nil, err
	}
	conn, err := grpcClient.NewConnection(config.CommitPeerCfg.Address, config.CommitPeerCfg.ServerNameOverride)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed to connect to commit peer %s", config.CommitPeerCfg.Address))
	}

	return &deliverClient{
		peerAddr:           config.CommitPeerCfg.Address,
		serverNameOverride: config.CommitPeerCfg.ServerNameOverride,
		grpcClient:         grpcClient,
		conn:               conn,
	}, nil
}

//newdeliverfilterd创建deliverfiltered客户端
func (d *deliverClient) NewDeliverFiltered(ctx context.Context, opts ...grpc.CallOption) (DeliverFiltered, error) {
	if d.conn != nil {
//关闭旧连接，因为新连接将重新启动其超时
		d.conn.Close()
	}

//创建到对等端的新连接
	var err error
	d.conn, err = d.grpcClient.NewConnection(d.peerAddr, d.serverNameOverride)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed to connect to commit peer %s", d.peerAddr))
	}

//创建新的传递筛选
	df, err := pb.NewDeliverClient(d.conn).DeliverFiltered(ctx)
	if err != nil {
		rpcStatus, _ := status.FromError(err)
		return nil, errors.Wrapf(err, "failed to new a deliver filtered, rpcStatus=%+v", rpcStatus)
	}
	return df, nil
}

func (d *deliverClient) Certificate() *tls.Certificate {
	cert := d.grpcClient.Certificate()
	return &cert
}

//用seekposition_new for block创建签名信封
func CreateDeliverEnvelope(channelId string, creator []byte, signer SignerIdentity, cert *tls.Certificate) (*common.Envelope, error) {
	var tlsCertHash []byte
	var err error
//检查客户端证书并在证书上计算sha2-256（如果存在）
	if cert != nil && len(cert.Certificate) > 0 {
		tlsCertHash, err = factory.GetDefault().Hash(cert.Certificate[0], &bccsp.SHA256Opts{})
		if err != nil {
			err = errors.New("failed to compute SHA256 on client certificate")
			logger.Errorf("%s", err)
			return nil, err
		}
	}

	_, header, err := CreateHeader(common.HeaderType_DELIVER_SEEK_INFO, channelId, creator, tlsCertHash)
	if err != nil {
		return nil, err
	}

	start := &ab.SeekPosition{
		Type: &ab.SeekPosition_Newest{
			Newest: &ab.SeekNewest{},
		},
	}

	stop := &ab.SeekPosition{
		Type: &ab.SeekPosition_Specified{
			Specified: &ab.SeekSpecified{
				Number: math.MaxUint64,
			},
		},
	}

	seekInfo := &ab.SeekInfo{
		Start:    start,
		Stop:     stop,
		Behavior: ab.SeekInfo_BLOCK_UNTIL_READY,
	}

	raw, err := proto.Marshal(seekInfo)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling SeekInfo")
	}

	envelope, err := CreateEnvelope(raw, header, signer)
	if err != nil {
		return nil, err
	}

	return envelope, nil
}

func DeliverSend(df DeliverFiltered, address string, envelope *common.Envelope) error {
	err := df.Send(envelope)
	df.CloseSend()
	if err != nil {
		return errors.Wrapf(err, "failed to send deliver envelope to peer %s", address)
	}
	return nil
}

func DeliverReceive(df DeliverFiltered, address string, txid string, eventCh chan TxEvent) error {
	event := TxEvent{
		Txid:       txid,
		Committed:  false,
		CommitPeer: address,
	}

read:
	for {
		resp, err := df.Recv()
		if err != nil {
			event.Err = errors.WithMessage(err, fmt.Sprintf("error receiving deliver response from peer %s", address))
			break read
		}
		switch r := resp.Type.(type) {
		case *pb.DeliverResponse_FilteredBlock:
			filteredTransactions := r.FilteredBlock.FilteredTransactions
			for _, tx := range filteredTransactions {
				logger.Debugf("deliverReceive got filteredTransaction for transaction [%s], status [%s]", tx.Txid, tx.TxValidationCode)
				if tx.Txid == txid {
					if tx.TxValidationCode == pb.TxValidationCode_VALID {
						event.Committed = true
					} else {
						event.Err = errors.Errorf("transaction [%s] status is not valid: %s", tx.Txid, tx.TxValidationCode)
					}
					break read
				}
			}
		case *pb.DeliverResponse_Status:
			event.Err = errors.Errorf("deliver completed with status (%s) before txid %s received from peer %s", r.Status, txid, address)
			break read
		default:
			event.Err = errors.Errorf("received unexpected response type (%T) from peer %s", r, address)
			break read
		}
	}

	if event.Err != nil {
		logger.Errorf("Error: %s", event.Err)
	}
	select {
	case eventCh <- event:
		logger.Debugf("Received transaction deliver event %+v", event)
	default:
		logger.Errorf("Event channel full. Discarding event %+v", event)
	}

	return event.Err
}

//deliverwaitforresponse等待eventchan有值（即已收到响应）或ctx超时
//此函数假定eventch仅用于指定的txid
//如果一个事件被多个事务共享，那么应该使用一个循环来监听来自多个事务的事件。
func DeliverWaitForResponse(ctx context.Context, eventCh chan TxEvent, txid string) (bool, error) {
	select {
	case event, _ := <-eventCh:
		if txid == event.Txid {
			return event.Committed, event.Err
		} else {
//不应该到这里
			return false, errors.Errorf("no event received for txid %s", txid)
		}
	case <-ctx.Done():
		err := errors.Errorf("timed out waiting for committing txid %s", txid)
		logger.Errorf("%s", err)
		return false, err
	}
}
