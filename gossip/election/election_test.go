
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
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	testTimeout      = 5 * time.Second
	testPollInterval = time.Millisecond * 300
)

func init() {
	util.SetupTestLogging()
	SetStartupGracePeriod(time.Millisecond * 500)
	SetMembershipSampleInterval(time.Millisecond * 100)
	SetLeaderAliveThreshold(time.Millisecond * 500)
	SetLeaderElectionDuration(time.Millisecond * 500)
}

type msg struct {
	sender   string
	proposal bool
}

func (m *msg) SenderID() peerID {
	return peerID(m.sender)
}

func (m *msg) IsProposal() bool {
	return m.proposal
}

func (m *msg) IsDeclaration() bool {
	return !m.proposal
}

type peer struct {
	mockedMethods map[string]struct{}
	mock.Mock
	id                 string
	peers              map[string]*peer
	sharedLock         *sync.RWMutex
	msgChan            chan Msg
	leaderFromCallback bool
	callbackInvoked    bool
	lock               sync.RWMutex
	LeaderElectionService
}

func (p *peer) On(methodName string, arguments ...interface{}) *mock.Call {
	p.sharedLock.Lock()
	defer p.sharedLock.Unlock()
	p.mockedMethods[methodName] = struct{}{}
	return p.Mock.On(methodName, arguments...)
}

func (p *peer) ID() peerID {
	return peerID(p.id)
}

func (p *peer) Gossip(m Msg) {
	p.sharedLock.RLock()
	defer p.sharedLock.RUnlock()

	if _, isMocked := p.mockedMethods["Gossip"]; isMocked {
		p.Called(m)
		return
	}

	for _, peer := range p.peers {
		if peer.id == p.id {
			continue
		}
		peer.msgChan <- m.(*msg)
	}
}

func (p *peer) Accept() <-chan Msg {
	p.sharedLock.RLock()
	defer p.sharedLock.RUnlock()

	if _, isMocked := p.mockedMethods["Accept"]; isMocked {
		args := p.Called()
		return args.Get(0).(<-chan Msg)
	}
	return (<-chan Msg)(p.msgChan)
}

func (p *peer) CreateMessage(isDeclaration bool) Msg {
	return &msg{proposal: !isDeclaration, sender: p.id}
}

func (p *peer) Peers() []Peer {
	p.sharedLock.RLock()
	defer p.sharedLock.RUnlock()

	if _, isMocked := p.mockedMethods["Peers"]; isMocked {
		args := p.Called()
		return args.Get(0).([]Peer)
	}

	var peers []Peer
	for id := range p.peers {
		peers = append(peers, &peer{id: id})
	}
	return peers
}

func (p *peer) leaderCallback(isLeader bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.leaderFromCallback = isLeader
	p.callbackInvoked = true
}

func (p *peer) isLeaderFromCallback() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.leaderFromCallback
}

func (p *peer) isCallbackInvoked() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.callbackInvoked
}

func createPeers(spawnInterval time.Duration, ids ...int) []*peer {
	peers := make([]*peer, len(ids))
	peerMap := make(map[string]*peer)
	l := &sync.RWMutex{}
	for i, id := range ids {
		p := createPeer(id, peerMap, l)
		if spawnInterval != 0 {
			time.Sleep(spawnInterval)
		}
		peers[i] = p
	}
	return peers
}

func createPeer(id int, peerMap map[string]*peer, l *sync.RWMutex) *peer {
	idStr := fmt.Sprintf("p%d", id)
	c := make(chan Msg, 100)
	p := &peer{id: idStr, peers: peerMap, sharedLock: l, msgChan: c, mockedMethods: make(map[string]struct{}), leaderFromCallback: false, callbackInvoked: false}
	p.LeaderElectionService = NewLeaderElectionService(p, idStr, p.leaderCallback)
	l.Lock()
	peerMap[idStr] = p
	l.Unlock()
	return p

}

func waitForMultipleLeadersElection(t *testing.T, peers []*peer, leadersNum int) []string {
	end := time.Now().Add(testTimeout)
	for time.Now().Before(end) {
		var leaders []string
		for _, p := range peers {
			if p.IsLeader() {
				leaders = append(leaders, p.id)
			}
		}
		if len(leaders) >= leadersNum {
			return leaders
		}
		time.Sleep(testPollInterval)
	}
	t.Fatal("No leader detected")
	return nil
}

func waitForLeaderElection(t *testing.T, peers []*peer) []string {
	return waitForMultipleLeadersElection(t, peers, 1)
}

func TestInitPeersAtSameTime(t *testing.T) {
	t.Parallel()
//场景：同时生成对等体
//预期结果：最低ID的同行是领导者
	peers := createPeers(0, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0)
	time.Sleep(getStartupGracePeriod() + getLeaderElectionDuration())
	leaders := waitForLeaderElection(t, peers)
	isP0leader := peers[len(peers)-1].IsLeader()
	assert.True(t, isP0leader, "p0 isn't a leader. Leaders are: %v", leaders)
	assert.Len(t, leaders, 1, "More than 1 leader elected")
	waitForBoolFunc(t, peers[len(peers)-1].isLeaderFromCallback, true, "Leadership callback result is wrong for ", peers[len(peers)-1].id)
}

func TestInitPeersStartAtIntervals(t *testing.T) {
	t.Parallel()
//场景：缓慢地一个接一个地生成对等体
//预期结果：第一个对等方是领导者，尽管其ID最高
	peers := createPeers(getStartupGracePeriod()+getLeadershipDeclarationInterval(), 3, 2, 1, 0)
	waitForLeaderElection(t, peers)
	assert.True(t, peers[0].IsLeader())
}

func TestStop(t *testing.T) {
	t.Parallel()
//场景：同时生成对等体
//然后停止。我们计算他们调用的八卦（）调用的数量
//after they stop, and it should not increase after they are stopped
	peers := createPeers(0, 3, 2, 1, 0)
	var gossipCounter int32
	for i, p := range peers {
		p.On("Gossip", mock.Anything).Run(func(args mock.Arguments) {
			msg := args.Get(0).(Msg)
			atomic.AddInt32(&gossipCounter, int32(1))
			for j := range peers {
				if i == j {
					continue
				}
				peers[j].msgChan <- msg
			}
		})
	}
	waitForLeaderElection(t, peers)
	for _, p := range peers {
		p.Stop()
	}
	time.Sleep(getLeaderAliveThreshold())
	gossipCounterAfterStop := atomic.LoadInt32(&gossipCounter)
	time.Sleep(getLeaderAliveThreshold() * 5)
	assert.Equal(t, gossipCounterAfterStop, atomic.LoadInt32(&gossipCounter))
}

func TestConvergence(t *testing.T) {
//场景：2个对等组聚合其视图
//预期结果：2人中只剩下1人。
//那个领导者是身份证最低的领导者
	t.Parallel()
	peers1 := createPeers(0, 3, 2, 1, 0)
	peers2 := createPeers(0, 4, 5, 6, 7)
	leaders1 := waitForLeaderElection(t, peers1)
	leaders2 := waitForLeaderElection(t, peers2)
	assert.Len(t, leaders1, 1, "Peer group 1 was suppose to have 1 leader exactly")
	assert.Len(t, leaders2, 1, "Peer group 2 was suppose to have 1 leader exactly")
	combinedPeers := append(peers1, peers2...)

	var allPeerIds []Peer
	for _, p := range combinedPeers {
		allPeerIds = append(allPeerIds, &peer{id: p.id})
	}

	for i, p := range combinedPeers {
		index := i
		gossipFunc := func(args mock.Arguments) {
			msg := args.Get(0).(Msg)
			for j := range combinedPeers {
				if index == j {
					continue
				}
				combinedPeers[j].msgChan <- msg
			}
		}
		p.On("Gossip", mock.Anything).Run(gossipFunc)
		p.On("Peers").Return(allPeerIds)
	}

	time.Sleep(getLeaderAliveThreshold() * 5)
	finalLeaders := waitForLeaderElection(t, combinedPeers)
	assert.Len(t, finalLeaders, 1, "Combined peer group was suppose to have 1 leader exactly")
	assert.Equal(t, leaders1[0], finalLeaders[0], "Combined peer group has different leader than expected:")

	for _, p := range combinedPeers {
		if p.id == finalLeaders[0] {
			waitForBoolFunc(t, p.isLeaderFromCallback, true, "Leadership callback result is wrong for ", p.id)
			waitForBoolFunc(t, p.isCallbackInvoked, true, "Leadership callback wasn't invoked for ", p.id)
		} else {
			waitForBoolFunc(t, p.isLeaderFromCallback, false, "Leadership callback result is wrong for ", p.id)
			if p.id == leaders2[0] {
				waitForBoolFunc(t, p.isCallbackInvoked, true, "Leadership callback wasn't invoked for ", p.id)
			}
		}
	}
}

func TestLeadershipTakeover(t *testing.T) {
	t.Parallel()
//场景：对等方按降序逐个生成。
//过了一会儿，领队同伴停了下来。
//预期结果：接收的对等方是ID最低的对等方
	peers := createPeers(getStartupGracePeriod()+getLeadershipDeclarationInterval(), 5, 4, 3, 2)
	leaders := waitForLeaderElection(t, peers)
	assert.Len(t, leaders, 1, "Only 1 leader should have been elected")
	assert.Equal(t, "p5", leaders[0])
	peers[0].Stop()
	time.Sleep(getLeadershipDeclarationInterval() + getLeaderAliveThreshold()*3)
	leaders = waitForLeaderElection(t, peers[1:])
	assert.Len(t, leaders, 1, "Only 1 leader should have been elected")
	assert.Equal(t, "p2", leaders[0])
}

func TestYield(t *testing.T) {
	t.Parallel()
//场景：同龄人繁殖，选出领导者。
//过了一会儿，领导让步了。
//（调用yield两次以确保只调用一个回调）
//预期结果：
//（一）选举新的领导人；
//（2）老领导不收回领导权
	peers := createPeers(0, 0, 1, 2, 3, 4, 5)
	leaders := waitForLeaderElection(t, peers)
	assert.Len(t, leaders, 1, "Only 1 leader should have been elected")
	assert.Equal(t, "p0", leaders[0])
	peers[0].Yield()
//确保调用回调时使用了“false”
	assert.True(t, peers[0].isCallbackInvoked())
	assert.False(t, peers[0].isLeaderFromCallback())
//清除回调调用标志
	peers[0].lock.Lock()
	peers[0].callbackInvoked = false
	peers[0].lock.Unlock()
//Yield again and ensure it isn't called again
	peers[0].Yield()
	assert.False(t, peers[0].isCallbackInvoked())

	ensureP0isNotAleader := func() bool {
		leaders := waitForLeaderElection(t, peers)
		return len(leaders) == 1 && leaders[0] != "p0"
	}
//选出新的领导人，而不是p0
	waitForBoolFunc(t, ensureP0isNotAleader, true)
	time.Sleep(getLeaderAliveThreshold() * 2)
//一段时间后，p0没有恢复其领导地位。
	waitForBoolFunc(t, ensureP0isNotAleader, true)
}

func TestYieldSinglePeer(t *testing.T) {
	t.Parallel()
//场景：生成一个对等体并让它产生。
//确保在一段时间后恢复领导地位。
	peers := createPeers(0, 0)
	waitForLeaderElection(t, peers)
	peers[0].Yield()
	assert.False(t, peers[0].IsLeader())
	waitForLeaderElection(t, peers)
}

func TestYieldAllPeers(t *testing.T) {
	t.Parallel()
//场景：产生2个同伴，在重新获得领导权后让他们都屈服。
//在双方都让步后，确保第一方是最终的领导者。
	peers := createPeers(0, 0, 1)
	leaders := waitForLeaderElection(t, peers)
	assert.Len(t, leaders, 1, "Only 1 leader should have been elected")
	assert.Equal(t, "p0", leaders[0])
	peers[0].Yield()
	leaders = waitForLeaderElection(t, peers)
	assert.Len(t, leaders, 1, "Only 1 leader should have been elected")
	assert.Equal(t, "p1", leaders[0])
	peers[1].Yield()
	leaders = waitForLeaderElection(t, peers)
	assert.Len(t, leaders, 1, "Only 1 leader should have been elected")
	assert.Equal(t, "p0", leaders[0])
}

func TestPartition(t *testing.T) {
	t.Parallel()
//场景：对等端一起产生，然后在一段时间后发生网络分区
//没有一个对等方可以与另一个对等方通信
//预期结果1：每个同行都是领导者
//在此之后，我们将分区修复为统一视图
//预期结果2:p0再次成为领导者
	peers := createPeers(0, 5, 4, 3, 2, 1, 0)
	leaders := waitForLeaderElection(t, peers)
	assert.Len(t, leaders, 1, "Only 1 leader should have been elected")
	assert.Equal(t, "p0", leaders[0])
	waitForBoolFunc(t, peers[len(peers)-1].isLeaderFromCallback, true, "Leadership callback result is wrong for %s", peers[len(peers)-1].id)

	for _, p := range peers {
		p.On("Peers").Return([]Peer{})
		p.On("Gossip", mock.Anything)
	}
	time.Sleep(getLeadershipDeclarationInterval() + getLeaderAliveThreshold()*2)
	leaders = waitForMultipleLeadersElection(t, peers, 6)
	assert.Len(t, leaders, 6)
	for _, p := range peers {
		waitForBoolFunc(t, p.isLeaderFromCallback, true, "Leadership callback result is wrong for %s", p.id)
	}

	for _, p := range peers {
		p.sharedLock.Lock()
		p.mockedMethods = make(map[string]struct{})
		p.callbackInvoked = false
		p.sharedLock.Unlock()
	}
	time.Sleep(getLeadershipDeclarationInterval() + getLeaderAliveThreshold()*2)
	leaders = waitForLeaderElection(t, peers)
	assert.Len(t, leaders, 1, "Only 1 leader should have been elected")
	assert.Equal(t, "p0", leaders[0])
	for _, p := range peers {
		if p.id == leaders[0] {
			waitForBoolFunc(t, p.isLeaderFromCallback, true, "Leadership callback result is wrong for %s", p.id)
		} else {
			waitForBoolFunc(t, p.isLeaderFromCallback, false, "Leadership callback result is wrong for %s", p.id)
			waitForBoolFunc(t, p.isCallbackInvoked, true, "Leadership callback wasn't invoked for %s", p.id)
		}
	}
}

func Test_peerIDString(t *testing.T) {
	tests := []struct {
		input    peerID
		expected string
	}{
		{nil, "<nil>"},
		{peerID{}, ""},
		{peerID{0, 1, 2, 3}, "00010203"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.input.String())
	}
}

func TestConfigFromFile(t *testing.T) {
	preStartupGracePeriod := getStartupGracePeriod()
	preMembershipSampleInterval := getMembershipSampleInterval()
	preLeaderAliveThreshold := getLeaderAliveThreshold()
	preLeaderElectionDuration := getLeaderElectionDuration()

//恢复配置值以避免影响其他测试
	defer func() {
		SetStartupGracePeriod(preStartupGracePeriod)
		SetMembershipSampleInterval(preMembershipSampleInterval)
		SetLeaderAliveThreshold(preLeaderAliveThreshold)
		SetLeaderElectionDuration(preLeaderElectionDuration)
	}()

//验证在缺少配置时是否使用默认值
	viper.Reset()
	assert.Equal(t, time.Second*15, getStartupGracePeriod())
	assert.Equal(t, time.Second, getMembershipSampleInterval())
	assert.Equal(t, time.Second*10, getLeaderAliveThreshold())
	assert.Equal(t, time.Second*5, getLeaderElectionDuration())
	assert.Equal(t, getLeaderAliveThreshold()/2, getLeadershipDeclarationInterval())

//验证是否从配置文件读取值
	viper.Reset()
	viper.SetConfigName("core")
	viper.SetEnvPrefix("CORE")
	configtest.AddDevConfigPath(nil)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	assert.NoError(t, err)
	assert.Equal(t, time.Second*15, getStartupGracePeriod())
	assert.Equal(t, time.Second, getMembershipSampleInterval())
	assert.Equal(t, time.Second*10, getLeaderAliveThreshold())
	assert.Equal(t, time.Second*5, getLeaderElectionDuration())
	assert.Equal(t, getLeaderAliveThreshold()/2, getLeadershipDeclarationInterval())
}

func waitForBoolFunc(t *testing.T, f func() bool, expectedValue bool, msgAndArgs ...interface{}) {
	end := time.Now().Add(testTimeout)
	for time.Now().Before(end) {
		if f() == expectedValue {
			return
		}
		time.Sleep(testPollInterval)
	}
	assert.Fail(t, fmt.Sprintf("Should be %t", expectedValue), msgAndArgs...)
}
