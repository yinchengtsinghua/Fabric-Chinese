
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


package election

import (
	"bytes"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hyperledger/fabric/gossip/util"
	"github.com/spf13/viper"
)

//八卦领袖选举模块
//Algorithm properties:
//-对等端通过比较ID来破坏对称性
//-每一位同行都是领导者或追随者，
//如果会员的观点是
//所有同龄人都一样
//-如果网络分为2组或更多组，则领导者的数量
//是网络分区的数目，但当分区恢复时，
//最终只剩下一个领导者
//-同行通过八卦领导建议或声明信息进行沟通。

//伪代码中的算法：
//
//
//变量：
//leaderknown=错误
//
//Invariant：
//对等端侦听来自远程对等端的消息
//每当它收到领导宣言，
//Leaderknown设置为true
//
//SARTUP（）：
//等待成员国观点稳定，或收到领导声明
//或者启动超时过期。
//转到SteadyState（）。
//
//稳定状态（）：
//虽然真实：
//如果Leaderknown为假：
//含铅量（）
//如果你是领导者：
//广播领导宣言
//如果收到领导声明
//具有较低ID的对等机，
//成为追随者
//否则，你就是一个追随者：
//如果在
//时间阈值：
//将leaderknown设置为false
//
//铅释放（）：
//八卦领导力建议信息
//从一段时间内发送的其他对等方收集消息
//如果收到领导声明：
//返回
//遍历收集的所有建议消息。
//如果来自ID较低的对等端的建议消息
//比你自己收到的还多，回来吧。
//否则，宣布自己是领导者

//领导选举模块使用Lead Error适配器
//发送和接收消息以及获取成员信息
type LeaderElectionAdapter interface {
//流言蜚语向其他同龄人传递信息
	Gossip(Msg)

//accept返回发出消息的通道
	Accept() <-chan Msg

//创建建议消息
	CreateMessage(isDeclaration bool) Msg

//peers返回被认为是活动的对等方的列表
	Peers() []Peer
}

type leadershipCallback func(isLeader bool)

//LeaderRelationService是运行Leader选举算法的对象
type LeaderElectionService interface {
//Isleader返回这个同伴是否是领导者
	IsLeader() bool

//停止停止LeadRelationService
	Stop()

//屈服于放弃领导直到选出新的领导，
//或者超时过期
	Yield()
}

type peerID []byte

func (p peerID) String() string {
	if p == nil {
		return "<nil>"
	}
	return hex.EncodeToString(p)
}

//对等机描述远程对等机
type Peer interface {
//ID返回对等机的ID
	ID() peerID
}

//msg描述从远程对等机发送的消息
type Msg interface {
//senderid返回发送消息的对等方的ID
	SenderID() peerID
//IsProposal返回此消息是否是领导层建议
	IsProposal() bool
//IsDeclaration returns whether this message is a leadership declaration
	IsDeclaration() bool
}

func noopCallback(_ bool) {
}

//NewLeaderRelationService返回新的LeaderRelationService
func NewLeaderElectionService(adapter LeaderElectionAdapter, id string, callback leadershipCallback) LeaderElectionService {
	if len(id) == 0 {
		panic("Empty id")
	}
	le := &leaderElectionSvcImpl{
		id:            peerID(id),
		proposals:     util.NewSet(),
		adapter:       adapter,
		stopChan:      make(chan struct{}, 1),
		interruptChan: make(chan struct{}, 1),
		logger:        util.GetLogger(util.ElectionLogger, ""),
		callback:      noopCallback,
	}

	if callback != nil {
		le.callback = callback
	}

	go le.start()
	return le
}

//LeaderRelationsCmpl是LeaderRelationService的实现
type leaderElectionSvcImpl struct {
	id        peerID
	proposals *util.Set
	sync.Mutex
	stopChan      chan struct{}
	interruptChan chan struct{}
	stopWG        sync.WaitGroup
	isLeader      int32
	toDie         int32
	leaderExists  int32
	yield         int32
	sleeping      bool
	adapter       LeaderElectionAdapter
	logger        util.Logger
	callback      leadershipCallback
	yieldTimer    *time.Timer
}

func (le *leaderElectionSvcImpl) start() {
	le.stopWG.Add(2)
	go le.handleMessages()
	le.waitForMembershipStabilization(getStartupGracePeriod())
	go le.run()
}

func (le *leaderElectionSvcImpl) handleMessages() {
	le.logger.Debug(le.id, ": Entering")
	defer le.logger.Debug(le.id, ": Exiting")
	defer le.stopWG.Done()
	msgChan := le.adapter.Accept()
	for {
		select {
		case <-le.stopChan:
			le.stopChan <- struct{}{}
			return
		case msg := <-msgChan:
			if !le.isAlive(msg.SenderID()) {
				le.logger.Debug(le.id, ": Got message from", msg.SenderID(), "but it is not in the view")
				break
			}
			le.handleMessage(msg)
		}
	}
}

func (le *leaderElectionSvcImpl) handleMessage(msg Msg) {
	msgType := "proposal"
	if msg.IsDeclaration() {
		msgType = "declaration"
	}
	le.logger.Debug(le.id, ":", msg.SenderID(), "sent us", msgType)
	le.Lock()
	defer le.Unlock()

	if msg.IsProposal() {
		le.proposals.Add(string(msg.SenderID()))
	} else if msg.IsDeclaration() {
		atomic.StoreInt32(&le.leaderExists, int32(1))
		if le.sleeping && len(le.interruptChan) == 0 {
			le.interruptChan <- struct{}{}
		}
		if bytes.Compare(msg.SenderID(), le.id) < 0 && le.IsLeader() {
			le.stopBeingLeader()
		}
	} else {
//我们不应该到这里
		le.logger.Error("Got a message that's not a proposal and not a declaration")
	}
}

//等待中断休眠，直到触发中断通道
//或给定的超时过期
func (le *leaderElectionSvcImpl) waitForInterrupt(timeout time.Duration) {
	le.logger.Debug(le.id, ": Entering")
	defer le.logger.Debug(le.id, ": Exiting")
	le.Lock()
	le.sleeping = true
	le.Unlock()

	select {
	case <-le.interruptChan:
	case <-le.stopChan:
		le.stopChan <- struct{}{}
	case <-time.After(timeout):
	}

	le.Lock()
	le.sleeping = false
//我们中断中断通道。
//因为我们可能会收到两条领导声明信息
//但我们只能在上面的选择块中读取其中的1个
	le.drainInterruptChannel()
	le.Unlock()
}

func (le *leaderElectionSvcImpl) run() {
	defer le.stopWG.Done()
	for !le.shouldStop() {
		if !le.isLeaderExists() {
			le.leaderElection()
		}
//如果我们屈服，某个领导人当选，
//停止屈服
		if le.isLeaderExists() && le.isYielding() {
			le.stopYielding()
		}
		if le.shouldStop() {
			return
		}
		if le.IsLeader() {
			le.leader()
		} else {
			le.follower()
		}
	}
}

func (le *leaderElectionSvcImpl) leaderElection() {
	le.logger.Debug(le.id, ": Entering")
	defer le.logger.Debug(le.id, ": Exiting")
//如果我们屈服于其他同行，不要参与
//在领导人选举中
	if le.isYielding() {
		return
	}
//建议自己成为领导者
	le.propose()
//收集其他建议
	le.waitForInterrupt(getLeaderElectionDuration())
//如果有人宣称自己是领导者，放弃
//努力成为领导者
	if le.isLeaderExists() {
		le.logger.Info(le.id, ": Some peer is already a leader")
		return
	}

	if le.isYielding() {
		le.logger.Debug(le.id, ": Aborting leader election because yielding")
		return
	}
//领导者不存在，让我们看看是否有比我们更好的候选人
//作为一个领导者
	for _, o := range le.proposals.ToArray() {
		id := o.(string)
		if bytes.Compare(peerID(id), le.id) < 0 {
			return
		}
	}
//如果我们到了这里，就没有人提议当领袖
//that's a better candidate than us.
	le.beLeader()
	atomic.StoreInt32(&le.leaderExists, int32(1))
}

//建议向远程对等发送领导建议消息
func (le *leaderElectionSvcImpl) propose() {
	le.logger.Debug(le.id, ": Entering")
	le.logger.Debug(le.id, ": Exiting")
	leadershipProposal := le.adapter.CreateMessage(false)
	le.adapter.Gossip(leadershipProposal)
}

func (le *leaderElectionSvcImpl) follower() {
	le.logger.Debug(le.id, ": Entering")
	defer le.logger.Debug(le.id, ": Exiting")

	le.proposals.Clear()
	atomic.StoreInt32(&le.leaderExists, int32(0))
	select {
	case <-time.After(getLeaderAliveThreshold()):
	case <-le.stopChan:
		le.stopChan <- struct{}{}
	}
}

func (le *leaderElectionSvcImpl) leader() {
	leaderDeclaration := le.adapter.CreateMessage(true)
	le.adapter.Gossip(leaderDeclaration)
	le.waitForInterrupt(getLeadershipDeclarationInterval())
}

//WaeFieldMeMeBuffStur稳等待成员视图稳定
//或者直到某个时间限制到期，或者直到某个对等方声明自己是领导者。
func (le *leaderElectionSvcImpl) waitForMembershipStabilization(timeLimit time.Duration) {
	le.logger.Debug(le.id, ": Entering")
	defer le.logger.Debug(le.id, ": Exiting, peers found", len(le.adapter.Peers()))
	endTime := time.Now().Add(timeLimit)
	viewSize := len(le.adapter.Peers())
	for !le.shouldStop() {
		time.Sleep(getMembershipSampleInterval())
		newSize := len(le.adapter.Peers())
		if newSize == viewSize || time.Now().After(endTime) || le.isLeaderExists() {
			return
		}
		viewSize = newSize
	}
}

//DrainInterruptChannel清除中断通道
//如果需要
func (le *leaderElectionSvcImpl) drainInterruptChannel() {
	if len(le.interruptChan) == 1 {
		<-le.interruptChan
	}
}

//is alive返回给定ID的对等端是否被视为活动的
func (le *leaderElectionSvcImpl) isAlive(id peerID) bool {
	for _, p := range le.adapter.Peers() {
		if bytes.Equal(p.ID(), id) {
			return true
		}
	}
	return false
}

func (le *leaderElectionSvcImpl) isLeaderExists() bool {
	return atomic.LoadInt32(&le.leaderExists) == int32(1)
}

//Isleader返回这个同伴是否是领导者
func (le *leaderElectionSvcImpl) IsLeader() bool {
	isLeader := atomic.LoadInt32(&le.isLeader) == int32(1)
	le.logger.Debug(le.id, ": Returning", isLeader)
	return isLeader
}

func (le *leaderElectionSvcImpl) beLeader() {
	le.logger.Info(le.id, ": Becoming a leader")
	atomic.StoreInt32(&le.isLeader, int32(1))
	le.callback(true)
}

func (le *leaderElectionSvcImpl) stopBeingLeader() {
	le.logger.Info(le.id, "Stopped being a leader")
	atomic.StoreInt32(&le.isLeader, int32(0))
	le.callback(false)
}

func (le *leaderElectionSvcImpl) shouldStop() bool {
	return atomic.LoadInt32(&le.toDie) == int32(1)
}

func (le *leaderElectionSvcImpl) isYielding() bool {
	return atomic.LoadInt32(&le.yield) == int32(1)
}

func (le *leaderElectionSvcImpl) stopYielding() {
	le.logger.Debug("Stopped yielding")
	le.Lock()
	defer le.Unlock()
	atomic.StoreInt32(&le.yield, int32(0))
	le.yieldTimer.Stop()
}

//屈服于放弃领导直到选出新的领导，
//或者超时过期
func (le *leaderElectionSvcImpl) Yield() {
	le.Lock()
	defer le.Unlock()
	if !le.IsLeader() || le.isYielding() {
		return
	}
//打开产量标志
	atomic.StoreInt32(&le.yield, int32(1))
//别再当领导了
	le.stopBeingLeader()
//清除领导存在的旗帜，因为我们可能是领导
	atomic.StoreInt32(&le.leaderExists, int32(0))
//之后在任何情况下清除Yield标志
	le.yieldTimer = time.AfterFunc(getLeaderAliveThreshold()*6, func() {
		atomic.StoreInt32(&le.yield, int32(0))
	})
}

//停止停止LeadRelationService
func (le *leaderElectionSvcImpl) Stop() {
	le.logger.Debug(le.id, ": Entering")
	defer le.logger.Debug(le.id, ": Exiting")
	atomic.StoreInt32(&le.toDie, int32(1))
	le.stopChan <- struct{}{}
	le.stopWG.Wait()
}

//setStartupGracePeriod配置启动宽限期间隔，
//the period of time to wait until election algorithm will start
func SetStartupGracePeriod(t time.Duration) {
	viper.Set("peer.gossip.election.startupGracePeriod", t)
}

//setmembershipsampleinterval设置/初始化
//membership view should be checked
func SetMembershipSampleInterval(t time.Duration) {
	viper.Set("peer.gossip.election.membershipSampleInterval", t)
}

//setLeaderAliveThreshold配置领导者选举活动阈值
func SetLeaderAliveThreshold(t time.Duration) {
	viper.Set("peer.gossip.election.leaderAliveThreshold", t)
}

//setLeaderRelationDuration配置预期的领导层选举持续时间，
//等待领导人选举完成的时间间隔
func SetLeaderElectionDuration(t time.Duration) {
	viper.Set("peer.gossip.election.leaderElectionDuration", t)
}

func getStartupGracePeriod() time.Duration {
	return util.GetDurationOrDefault("peer.gossip.election.startupGracePeriod", time.Second*15)
}

func getMembershipSampleInterval() time.Duration {
	return util.GetDurationOrDefault("peer.gossip.election.membershipSampleInterval", time.Second)
}

func getLeaderAliveThreshold() time.Duration {
	return util.GetDurationOrDefault("peer.gossip.election.leaderAliveThreshold", time.Second*10)
}

func getLeadershipDeclarationInterval() time.Duration {
	return time.Duration(getLeaderAliveThreshold() / 2)
}

func getLeaderElectionDuration() time.Duration {
	return util.GetDurationOrDefault("peer.gossip.election.leaderElectionDuration", time.Second*5)
}

//GetMsgExpirationTimeout返回领导层消息过期超时
func GetMsgExpirationTimeout() time.Duration {
	return getLeaderAliveThreshold() * 10
}
