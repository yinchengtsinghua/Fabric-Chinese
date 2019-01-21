
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


package deliverclient

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var logger = flogging.MustGetLogger("deliveryClient")

const (
	defaultReConnectTotalTimeThreshold = time.Second * 60 * 60
	defaultConnectionTimeout           = time.Second * 3
	defaultReConnectBackoffThreshold   = float64(time.Hour)
)

func getReConnectTotalTimeThreshold() time.Duration {
	return util.GetDurationOrDefault("peer.deliveryclient.reconnectTotalTimeThreshold", defaultReConnectTotalTimeThreshold)
}

func getConnectionTimeout() time.Duration {
	return util.GetDurationOrDefault("peer.deliveryclient.connTimeout", defaultConnectionTimeout)
}

func getReConnectBackoffThreshold() float64 {
	return util.GetFloat64OrDefault("peer.deliveryclient.reConnectBackoffThreshold", defaultReConnectBackoffThreshold)
}

//DeliverService用于与订购方通信以获取
//新块并将其发送到提交者服务
type DeliverService interface {
//StartDeliverForChannel从订购服务动态启动新块的交付
//以引导同行。
//传递完成后，将调用终结器func
	StartDeliverForChannel(chainID string, ledgerInfo blocksprovider.LedgerInfo, finalizer func()) error

//StopDeliverForChannel从订购服务动态停止新块的交付
//以引导同行。
	StopDeliverForChannel(chainID string) error

//更新端点
	UpdateEndpoints(chainID string, endpoints []string) error

//停止终止传递服务并关闭连接
	Stop()
}

//DeliverServiceImpl交付服务的实现
//维护与订购服务的连接和
//阻止提供程序
type deliverServiceImpl struct {
	conf           *Config
	blockProviders map[string]blocksprovider.BlocksProvider
	lock           sync.RWMutex
	stopping       bool
}

//config指示deliveryService的属性，
//也就是说，它如何连接到订购服务端点，
//它如何验证从它接收到的消息，
//以及它如何将消息传播给其他对等方
type Config struct {
//connfactory返回一个函数，该函数创建到端点的连接
	ConnFactory func(channelID string) func(endpoint string) (*grpc.ClientConn, error)
//abcfactory通过连接创建AtomicBroadcastClient
	ABCFactory func(*grpc.ClientConn) orderer.AtomicBroadcastClient
//Cryptosvc执行加密操作，如消息验证和签名
//和身份验证
	CryptoSvc api.MessageCryptoService
//八卦可以列举频道中的同龄人，向同龄人发送信息，
//并向八卦状态传输层添加块
	Gossip blocksprovider.GossipServiceAdapter
//端点指定排序服务的端点
	Endpoints []string
}

//要创建和初始化的NewDeliverService构造函数
//传递服务实例。它试图建立与
//配置订购服务中指定的，以防
//没有拨号，返回零
func NewDeliverService(conf *Config) (DeliverService, error) {
	ds := &deliverServiceImpl{
		conf:           conf,
		blockProviders: make(map[string]blocksprovider.BlocksProvider),
	}
	if err := ds.validateConfiguration(); err != nil {
		return nil, err
	}
	return ds, nil
}

func (d *deliverServiceImpl) UpdateEndpoints(chainID string, endpoints []string) error {
//使用chainID获取块提供程序并传递端点
//用于更新
	if bp, ok := d.blockProviders[chainID]; ok {
//我们找到了指定的频道，因此可以安全地更新它
		bp.UpdateOrderingEndpoints(endpoints)
		return nil
	}
	return errors.New(fmt.Sprintf("Channel with %s id was not found", chainID))
}

func (d *deliverServiceImpl) validateConfiguration() error {
	conf := d.conf
	if len(conf.Endpoints) == 0 {
		return errors.New("no endpoints specified")
	}
	if conf.Gossip == nil {
		return errors.New("no gossip provider specified")
	}
	if conf.ABCFactory == nil {
		return errors.New("no AtomicBroadcast factory specified")
	}
	if conf.ConnFactory == nil {
		return errors.New("no connection factory specified")
	}
	if conf.CryptoSvc == nil {
		return errors.New("no crypto service specified")
	}
	return nil
}

//StartDeliverForChannel开始阻止通道的传递
//为给定的chainID初始化grpc流，创建块提供程序实例
//从分类帐提供的位置开始读取新块。
//信息实例。
func (d *deliverServiceImpl) StartDeliverForChannel(chainID string, ledgerInfo blocksprovider.LedgerInfo, finalizer func()) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.stopping {
		errMsg := fmt.Sprintf("Delivery service is stopping cannot join a new channel %s", chainID)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}
	if _, exist := d.blockProviders[chainID]; exist {
		errMsg := fmt.Sprintf("Delivery service - block provider already exists for %s found, can't start delivery", chainID)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	} else {
		client := d.newClient(chainID, ledgerInfo)
		logger.Debug("This peer will pass blocks from orderer service to other peers for channel", chainID)
		d.blockProviders[chainID] = blocksprovider.NewBlocksProvider(chainID, client, d.conf.Gossip, d.conf.CryptoSvc)
		go d.launchBlockProvider(chainID, finalizer)
	}
	return nil
}

func (d *deliverServiceImpl) launchBlockProvider(chainID string, finalizer func()) {
	d.lock.RLock()
	pb := d.blockProviders[chainID]
	d.lock.RUnlock()
	if pb == nil {
		logger.Info("Block delivery for channel", chainID, "was stopped before block provider started")
		return
	}
	pb.DeliverBlocks()
	finalizer()
}

//StopDeliverForChannel停止通过停止通道块提供程序阻止通道的传递
func (d *deliverServiceImpl) StopDeliverForChannel(chainID string) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.stopping {
		errMsg := fmt.Sprintf("Delivery service is stopping, cannot stop delivery for channel %s", chainID)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}
	if client, exist := d.blockProviders[chainID]; exist {
		client.Stop()
		delete(d.blockProviders, chainID)
		logger.Debug("This peer will stop pass blocks from orderer service to other peers")
	} else {
		errMsg := fmt.Sprintf("Delivery service - no block provider for %s found, can't stop delivery", chainID)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

//停止所有服务并释放资源
func (d *deliverServiceImpl) Stop() {
	d.lock.Lock()
	defer d.lock.Unlock()
//标记标志以指示传递服务的关闭
	d.stopping = true

	for _, client := range d.blockProviders {
		client.Stop()
	}
}

func (d *deliverServiceImpl) newClient(chainID string, ledgerInfoProvider blocksprovider.LedgerInfo) *broadcastClient {
	reconnectBackoffThreshold := getReConnectBackoffThreshold()
	reconnectTotalTimeThreshold := getReConnectTotalTimeThreshold()
	requester := &blocksRequester{
		tls:     viper.GetBool("peer.tls.enabled"),
		chainID: chainID,
	}
	broadcastSetup := func(bd blocksprovider.BlocksDeliverer) error {
		return requester.RequestBlocks(ledgerInfoProvider)
	}
	backoffPolicy := func(attemptNum int, elapsedTime time.Duration) (time.Duration, bool) {
		if elapsedTime >= reconnectTotalTimeThreshold {
			return 0, false
		}
		sleepIncrement := float64(time.Millisecond * 500)
		attempt := float64(attemptNum)
		return time.Duration(math.Min(math.Pow(2, attempt)*sleepIncrement, reconnectBackoffThreshold)), true
	}
	connProd := comm.NewConnectionProducer(d.conf.ConnFactory(chainID), d.conf.Endpoints)
	bClient := NewBroadcastClient(connProd, d.conf.ABCFactory, broadcastSetup, backoffPolicy)
	requester.client = bClient
	return bClient
}

func DefaultConnectionFactory(channelID string) func(endpoint string) (*grpc.ClientConn, error) {
	return func(endpoint string) (*grpc.ClientConn, error) {
		dialOpts := []grpc.DialOption{grpc.WithBlock()}
//设置最大发送/接收消息大小
		dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(comm.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(comm.MaxSendMsgSize)))
//设置keepalive选项
		kaOpts := comm.DefaultKeepaliveOptions
		if viper.IsSet("peer.keepalive.deliveryClient.interval") {
			kaOpts.ClientInterval = viper.GetDuration(
				"peer.keepalive.deliveryClient.interval")
		}
		if viper.IsSet("peer.keepalive.deliveryClient.timeout") {
			kaOpts.ClientTimeout = viper.GetDuration(
				"peer.keepalive.deliveryClient.timeout")
		}
		dialOpts = append(dialOpts, comm.ClientKeepaliveOptions(kaOpts)...)

		if viper.GetBool("peer.tls.enabled") {
			creds, err := comm.GetCredentialSupport().GetDeliverServiceCredentials(channelID)
			if err != nil {
				return nil, fmt.Errorf("failed obtaining credentials for channel %s: %v", channelID, err)
			}
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
		} else {
			dialOpts = append(dialOpts, grpc.WithInsecure())
		}
		ctx, cancel := context.WithTimeout(context.Background(), getConnectionTimeout())
		defer cancel()
		return grpc.DialContext(ctx, endpoint, dialOpts...)
	}
}

func DefaultABCFactory(conn *grpc.ClientConn) orderer.AtomicBroadcastClient {
	return orderer.NewAtomicBroadcastClient(conn)
}
