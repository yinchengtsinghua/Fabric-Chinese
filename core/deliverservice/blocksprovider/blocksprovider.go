
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


package blocksprovider

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/gossip/api"
	gossipcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/common"
	gossip_proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/orderer"
)

//提供查询接口的适配器
//当前分类帐高度的分类帐提交者
type LedgerInfo interface {
//LedgerHeight返回当前本地分类帐高度
	LedgerHeight() (uint64, error)
}

//GossipsServiceAdapter提供基本功能
//送货服务需要八卦服务
type GossipServiceAdapter interface {
//PeerSofChannel返回具有指定通道成员的切片
	PeersOfChannel(gossipcommon.ChainID) []discovery.NetworkMember

//AddPayload将负载添加到本地状态同步缓冲区
	AddPayload(chainID string, payload *gossip_proto.Payload) error

//在同龄人之间闲聊信息
	Gossip(msg *gossip_proto.GossipMessage)
}

//blocksProvider用于从订购服务读取块
//对于它订阅的指定链
type BlocksProvider interface {
//传递块开始传递和传播块
	DeliverBlocks()

//UpdateClientEndpoints更新终结点
	UpdateOrderingEndpoints(endpoints []string)

//停止关闭块提供程序并停止传递新块
	Stop()
}

//BlocksDeliverer定义了实际上有助于
//只使用
//块提供程序所需的方法。
//这也使GRPC流的生产实现脱钩
//为了使代码更加模块化和可测试性。
type BlocksDeliverer interface {
//recv从订购服务检索响应
	Recv() (*orderer.DeliverResponse, error)

//发送将信封发送到订购服务
	Send(*common.Envelope) error
}

type streamClient interface {
	BlocksDeliverer

//更新端点更新对服务端点排序
	UpdateEndpoints(endpoints []string)

//获取端点
	GetEndpoints() []string

//关闭关闭流及其基础连接
	Close()

//断开与远程节点的连接，并在预定义的时间段内禁用重新连接到当前端点
	Disconnect(disableEndpoint bool)
}

//BlocksProviderImpl BlocksProvider接口的实际实现
type blocksProviderImpl struct {
	chainID string

	client streamClient

	gossip GossipServiceAdapter

	mcs api.MessageCryptoService

	done int32

	wrongStatusThreshold int
}

const wrongStatusThreshold = 10

var maxRetryDelay = time.Second * 10
var logger = flogging.MustGetLogger("blocksProvider")

//NewblocksProvider构造函数用于创建块交付器实例的函数
func NewBlocksProvider(chainID string, client streamClient, gossip GossipServiceAdapter, mcs api.MessageCryptoService) BlocksProvider {
	return &blocksProviderImpl{
		chainID:              chainID,
		client:               client,
		gossip:               gossip,
		mcs:                  mcs,
		wrongStatusThreshold: wrongStatusThreshold,
	}
}

//用于将块从订购服务中拉出到
//将它们分布在各个对等点上
func (b *blocksProviderImpl) DeliverBlocks() {
	errorStatusCounter := 0
	statusCounter := 0
	defer b.client.Close()
	for !b.isDone() {
		msg, err := b.client.Recv()
		if err != nil {
			logger.Warningf("[%s] Receive error: %s", b.chainID, err.Error())
			return
		}
		switch t := msg.Type.(type) {
		case *orderer.DeliverResponse_Status:
			if t.Status == common.Status_SUCCESS {
				logger.Warningf("[%s] ERROR! Received success for a seek that should never complete", b.chainID)
				return
			}
			if t.Status == common.Status_BAD_REQUEST || t.Status == common.Status_FORBIDDEN {
				logger.Errorf("[%s] Got error %v", b.chainID, t)
				errorStatusCounter++
				if errorStatusCounter > b.wrongStatusThreshold {
					logger.Criticalf("[%s] Wrong statuses threshold passed, stopping block provider", b.chainID)
					return
				}
			} else {
				errorStatusCounter = 0
				logger.Warningf("[%s] Got error %v", b.chainID, t)
			}
			maxDelay := float64(maxRetryDelay)
			currDelay := float64(time.Duration(math.Pow(2, float64(statusCounter))) * 100 * time.Millisecond)
			time.Sleep(time.Duration(math.Min(maxDelay, currDelay)))
			if currDelay < maxDelay {
				statusCounter++
			}
			if t.Status == common.Status_BAD_REQUEST {
				b.client.Disconnect(false)
			} else {
				b.client.Disconnect(true)
			}
			continue
		case *orderer.DeliverResponse_Block:
			errorStatusCounter = 0
			statusCounter = 0
			blockNum := t.Block.Header.Number

			marshaledBlock, err := proto.Marshal(t.Block)
			if err != nil {
				logger.Errorf("[%s] Error serializing block with sequence number %d, due to %s", b.chainID, blockNum, err)
				continue
			}
			if err := b.mcs.VerifyBlock(gossipcommon.ChainID(b.chainID), blockNum, marshaledBlock); err != nil {
				logger.Errorf("[%s] Error verifying block with sequnce number %d, due to %s", b.chainID, blockNum, err)
				continue
			}

			numberOfPeers := len(b.gossip.PeersOfChannel(gossipcommon.ChainID(b.chainID)))
//创建接收到块的有效负载
			payload := createPayload(blockNum, marshaledBlock)
//使用有效负载创建八卦消息
			gossipMsg := createGossipMsg(b.chainID, payload)

			logger.Debugf("[%s] Adding payload to local buffer, blockNum = [%d]", b.chainID, blockNum)
//将有效负载添加到本地状态有效负载缓冲区
			if err := b.gossip.AddPayload(b.chainID, payload); err != nil {
				logger.Warningf("Block [%d] received from ordering service wasn't added to payload buffer: %v", blockNum, err)
			}

//与其他节点的八卦消息
			logger.Debugf("[%s] Gossiping block [%d], peers number [%d]", b.chainID, blockNum, numberOfPeers)
			if !b.isDone() {
				b.gossip.Gossip(gossipMsg)
			}
		default:
			logger.Warningf("[%s] Received unknown: %v", b.chainID, t)
			return
		}
	}
}

//停止停止阻止传递提供程序
func (b *blocksProviderImpl) Stop() {
	atomic.StoreInt32(&b.done, 1)
	b.client.Close()
}

//updateOrderingEndpoints更新订购服务的终结点
func (b *blocksProviderImpl) UpdateOrderingEndpoints(endpoints []string) {
	if !b.isEndpointsUpdated(endpoints) {
//没有为订购服务提供新的终结点
		return
	}
//我们有了一组新的端点，正在更新客户端
	logger.Debug("Updating endpoint, to %s", endpoints)
	b.client.UpdateEndpoints(endpoints)
	logger.Debug("Disconnecting so endpoints update will take effect")
//我们需要断开客户机的连接，使其重新连接
//到新更新的端点
	b.client.Disconnect(false)
}
func (b *blocksProviderImpl) isEndpointsUpdated(endpoints []string) bool {
	if len(endpoints) != len(b.client.GetEndpoints()) {
		return true
	}
//Check that endpoints was actually updated
	for _, endpoint := range endpoints {
		if !util.Contains(endpoint, b.client.GetEndpoints()) {
//找到新的终结点
			return true
		}
	}
//什么都没变
	return false
}

//每当提供程序停止时检查
func (b *blocksProviderImpl) isDone() bool {
	return atomic.LoadInt32(&b.done) == 1
}

func createGossipMsg(chainID string, payload *gossip_proto.Payload) *gossip_proto.GossipMessage {
	gossipMsg := &gossip_proto.GossipMessage{
		Nonce:   0,
		Tag:     gossip_proto.GossipMessage_CHAN_AND_ORG,
		Channel: []byte(chainID),
		Content: &gossip_proto.GossipMessage_DataMsg{
			DataMsg: &gossip_proto.DataMessage{
				Payload: payload,
			},
		},
	}
	return gossipMsg
}

func createPayload(seqNum uint64, marshaledBlock []byte) *gossip_proto.Payload {
	return &gossip_proto.Payload{
		Data:   marshaledBlock,
		SeqNum: seqNum,
	}
}
