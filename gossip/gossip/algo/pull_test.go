
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


package algo

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func init() {
	util.SetupTestLogging()
	SetDigestWaitTime(time.Duration(100) * time.Millisecond)
	SetRequestWaitTime(time.Duration(200) * time.Millisecond)
	SetResponseWaitTime(time.Duration(200) * time.Millisecond)
}

type messageHook func(interface{})

type pullTestInstance struct {
	msgHooks          []messageHook
	peers             map[string]*pullTestInstance
	name              string
	nextPeerSelection []string
	msgQueue          chan interface{}
	lock              sync.Mutex
	stopChan          chan struct{}
	*PullEngine
}

type helloMsg struct {
	nonce  uint64
	source string
}

type digestMsg struct {
	nonce  uint64
	digest []string
	source string
}

type reqMsg struct {
	items  []string
	nonce  uint64
	source string
}

type resMsg struct {
	items []string
	nonce uint64
}

func newPushPullTestInstance(name string, peers map[string]*pullTestInstance) *pullTestInstance {
	inst := &pullTestInstance{
		msgHooks:          make([]messageHook, 0),
		peers:             peers,
		msgQueue:          make(chan interface{}, 100),
		nextPeerSelection: make([]string, 0),
		stopChan:          make(chan struct{}, 1),
		name:              name,
	}

	inst.PullEngine = NewPullEngine(inst, time.Duration(500)*time.Millisecond)

	peers[name] = inst
	go func() {
		for {
			select {
			case <-inst.stopChan:
				return
			case m := <-inst.msgQueue:
				inst.handleMessage(m)
			}
		}
	}()

	return inst
}

//用于测试一个对等体发送给另一个的消息。
//断言语句应通过messagehook f传递
func (p *pullTestInstance) hook(f messageHook) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.msgHooks = append(p.msgHooks, f)
}

func (p *pullTestInstance) handleMessage(m interface{}) {
	p.lock.Lock()
	for _, f := range p.msgHooks {
		f(m)
	}
	p.lock.Unlock()

	if helloMsg, isHello := m.(*helloMsg); isHello {
		p.OnHello(helloMsg.nonce, helloMsg.source)
		return
	}

	if digestMsg, isDigest := m.(*digestMsg); isDigest {
		p.OnDigest(digestMsg.digest, digestMsg.nonce, digestMsg.source)
		return
	}

	if reqMsg, isReq := m.(*reqMsg); isReq {
		p.OnReq(reqMsg.items, reqMsg.nonce, reqMsg.source)
		return
	}

	if resMsg, isRes := m.(*resMsg); isRes {
		p.OnRes(resMsg.items, resMsg.nonce)
	}
}

func (p *pullTestInstance) stop() {
	p.stopChan <- struct{}{}
	p.Stop()
}

func (p *pullTestInstance) setNextPeerSelection(selection []string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.nextPeerSelection = selection
}

func (p *pullTestInstance) SelectPeers() []string {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.nextPeerSelection
}

func (p *pullTestInstance) Hello(dest string, nonce uint64) {
	p.peers[dest].msgQueue <- &helloMsg{nonce: nonce, source: p.name}
}

func (p *pullTestInstance) SendDigest(digest []string, nonce uint64, context interface{}) {
	p.peers[context.(string)].msgQueue <- &digestMsg{source: p.name, nonce: nonce, digest: digest}
}

func (p *pullTestInstance) SendReq(dest string, items []string, nonce uint64) {
	p.peers[dest].msgQueue <- &reqMsg{nonce: nonce, source: p.name, items: items}
}

func (p *pullTestInstance) SendRes(items []string, context interface{}, nonce uint64) {
	p.peers[context.(string)].msgQueue <- &resMsg{items: items, nonce: nonce}
}

func TestPullEngine_Add(t *testing.T) {
	t.Parallel()
	peers := make(map[string]*pullTestInstance)
	inst1 := newPushPullTestInstance("p1", peers)
	defer inst1.Stop()
	inst1.Add("0")
	inst1.Add("0")
	assert.True(t, inst1.PullEngine.state.Exists("0"))
}

func TestPullEngine_Remove(t *testing.T) {
	t.Parallel()
	peers := make(map[string]*pullTestInstance)
	inst1 := newPushPullTestInstance("p1", peers)
	defer inst1.Stop()
	inst1.Add("0")
	assert.True(t, inst1.PullEngine.state.Exists("0"))
	inst1.Remove("0")
	assert.False(t, inst1.PullEngine.state.Exists("0"))
inst1.Remove("0") //去掉两次
	assert.False(t, inst1.PullEngine.state.Exists("0"))
}

func TestPullEngine_Stop(t *testing.T) {
	t.Parallel()
	peers := make(map[string]*pullTestInstance)
	inst1 := newPushPullTestInstance("p1", peers)
	inst2 := newPushPullTestInstance("p2", peers)
	defer inst2.stop()
	inst2.setNextPeerSelection([]string{"p1"})
	go func() {
		for i := 0; i < 100; i++ {
			inst1.Add(string(i))
			time.Sleep(time.Duration(10) * time.Millisecond)
		}
	}()

	time.Sleep(time.Duration(800) * time.Millisecond)
	len1 := len(inst2.state.ToArray())
	inst1.stop()
	time.Sleep(time.Duration(800) * time.Millisecond)
	len2 := len(inst2.state.ToArray())
	assert.Equal(t, len1, len2, "PullEngine was still active after Stop() was invoked!")
}

func TestPullEngineAll2AllWithIncrementalSpawning(t *testing.T) {
	t.Parallel()
//场景：生成10个节点，每个节点间隔50毫秒
//让他们在自己之间传输数据。
//Expected outcome: obviously, everything should succeed.
//这不是我们来这里的原因吗？
	instanceCount := 10
	peers := make(map[string]*pullTestInstance)

	for i := 0; i < instanceCount; i++ {
		inst := newPushPullTestInstance(fmt.Sprintf("p%d", i+1), peers)
		inst.Add(string(i + 1))
		time.Sleep(time.Duration(50) * time.Millisecond)
	}
	for i := 0; i < instanceCount; i++ {
		pID := fmt.Sprintf("p%d", i+1)
		peers[pID].setNextPeerSelection(keySet(pID, peers))
	}
	time.Sleep(time.Duration(4000) * time.Millisecond)

	for i := 0; i < instanceCount; i++ {
		pID := fmt.Sprintf("p%d", i+1)
		assert.Equal(t, instanceCount, len(peers[pID].state.ToArray()))
	}
}

func TestPullEngineSelectiveUpdates(t *testing.T) {
	t.Parallel()
//场景：inst1有1，3，inst2有0，1，2，3。
//inst1启动到inst2
//预期结果：inst1请求0,2，inst2只发送0,2
	peers := make(map[string]*pullTestInstance)
	inst1 := newPushPullTestInstance("p1", peers)
	inst2 := newPushPullTestInstance("p2", peers)
	defer inst1.stop()
	defer inst2.stop()

	inst1.Add("1", "3")
	inst2.Add("0", "1", "2", "3")

//确保inst2向inst1发送了正确的摘要
	inst1.hook(func(m interface{}) {
		if dig, isDig := m.(*digestMsg); isDig {
			assert.True(t, util.IndexInSlice(dig.digest, "0", Strcmp) != -1)
			assert.True(t, util.IndexInSlice(dig.digest, "1", Strcmp) != -1)
			assert.True(t, util.IndexInSlice(dig.digest, "2", Strcmp) != -1)
			assert.True(t, util.IndexInSlice(dig.digest, "3", Strcmp) != -1)
		}
	})

//确保inst1只要求inst2提供所需的更新
	inst2.hook(func(m interface{}) {
		if req, isReq := m.(*reqMsg); isReq {
			assert.True(t, util.IndexInSlice(req.items, "1", Strcmp) == -1)
			assert.True(t, util.IndexInSlice(req.items, "3", Strcmp) == -1)

			assert.True(t, util.IndexInSlice(req.items, "0", Strcmp) != -1)
			assert.True(t, util.IndexInSlice(req.items, "2", Strcmp) != -1)
		}
	})

//确保inst1只从inst2接收到所需的更新
	inst1.hook(func(m interface{}) {
		if res, isRes := m.(*resMsg); isRes {
			assert.True(t, util.IndexInSlice(res.items, "1", Strcmp) == -1)
			assert.True(t, util.IndexInSlice(res.items, "3", Strcmp) == -1)

			assert.True(t, util.IndexInSlice(res.items, "0", Strcmp) != -1)
			assert.True(t, util.IndexInSlice(res.items, "2", Strcmp) != -1)
		}
	})

	inst1.setNextPeerSelection([]string{"p2"})

	time.Sleep(time.Duration(2000) * time.Millisecond)
	assert.Equal(t, len(inst2.state.ToArray()), len(inst1.state.ToArray()))
}

func TestByzantineResponder(t *testing.T) {
	t.Parallel()
//场景：inst1向inst2发送hello，但inst3是拜占庭式的，因此它尝试发送摘要和对inst1的响应。
//预期结果是inst1不处理inst3的更新。
	peers := make(map[string]*pullTestInstance)
	inst1 := newPushPullTestInstance("p1", peers)
	inst2 := newPushPullTestInstance("p2", peers)
	inst3 := newPushPullTestInstance("p3", peers)
	defer inst1.stop()
	defer inst2.stop()
	defer inst3.stop()

	receivedDigestFromInst3 := int32(0)

	inst2.Add("1", "2", "3")
	inst3.Add("1", "6", "7")

	inst2.hook(func(m interface{}) {
		if _, isHello := m.(*helloMsg); isHello {
			inst3.SendDigest([]string{"5", "6", "7"}, 0, "p1")
		}
	})

	inst1.hook(func(m interface{}) {
		if dig, isDig := m.(*digestMsg); isDig {
			if dig.source == "p3" {
				atomic.StoreInt32(&receivedDigestFromInst3, int32(1))
				time.AfterFunc(time.Duration(150)*time.Millisecond, func() {
					inst3.SendRes([]string{"5", "6", "7"}, "p1", 0)
				})
			}
		}

		if res, isRes := m.(*resMsg); isRes {
//回答来自P3
			if util.IndexInSlice(res.items, "6", Strcmp) != -1 {
//inst1目前正在接受回复
				assert.Equal(t, int32(1), atomic.LoadInt32(&(inst1.acceptingResponses)), "inst1 is not accepting digests")
			}
		}
	})

	inst1.setNextPeerSelection([]string{"p2"})

	time.Sleep(time.Duration(1000) * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&receivedDigestFromInst3), "inst1 hasn't received a digest from inst3")

	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "1", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "2", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "3", Strcmp) != -1)

	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "5", Strcmp) == -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "6", Strcmp) == -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "7", Strcmp) == -1)

}

func TestMultipleInitiators(t *testing.T) {
	t.Parallel()
//场景：inst1、inst2、inst3同时与inst4启动协议。
//预期结果：inst4成功地将状态转移到所有状态
	peers := make(map[string]*pullTestInstance)
	inst1 := newPushPullTestInstance("p1", peers)
	inst2 := newPushPullTestInstance("p2", peers)
	inst3 := newPushPullTestInstance("p3", peers)
	inst4 := newPushPullTestInstance("p4", peers)
	defer inst1.stop()
	defer inst2.stop()
	defer inst3.stop()
	defer inst4.stop()

	inst4.Add("1", "2", "3", "4")
	inst1.setNextPeerSelection([]string{"p4"})
	inst2.setNextPeerSelection([]string{"p4"})
	inst3.setNextPeerSelection([]string{"p4"})

	time.Sleep(time.Duration(2000) * time.Millisecond)

	for _, inst := range []*pullTestInstance{inst1, inst2, inst3} {
		assert.True(t, util.IndexInSlice(inst.state.ToArray(), "1", Strcmp) != -1)
		assert.True(t, util.IndexInSlice(inst.state.ToArray(), "2", Strcmp) != -1)
		assert.True(t, util.IndexInSlice(inst.state.ToArray(), "3", Strcmp) != -1)
		assert.True(t, util.IndexInSlice(inst.state.ToArray(), "4", Strcmp) != -1)
	}

}

func TestLatePeers(t *testing.T) {
	t.Parallel()
//场景：inst1启动inst2（items:1,2,3,4）和inst3（items:5,6,7,8），
//但是inst2反应太慢，所有项目
//应该从inst3收到。
	peers := make(map[string]*pullTestInstance)
	inst1 := newPushPullTestInstance("p1", peers)
	inst2 := newPushPullTestInstance("p2", peers)
	inst3 := newPushPullTestInstance("p3", peers)
	defer inst1.stop()
	defer inst2.stop()
	defer inst3.stop()
	inst2.Add("1", "2", "3", "4")
	inst3.Add("5", "6", "7", "8")
	inst2.hook(func(m interface{}) {
		time.Sleep(time.Duration(600) * time.Millisecond)
	})
	inst1.setNextPeerSelection([]string{"p2", "p3"})

	time.Sleep(time.Duration(2000) * time.Millisecond)

	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "1", Strcmp) == -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "2", Strcmp) == -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "3", Strcmp) == -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "4", Strcmp) == -1)

	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "5", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "6", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "7", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "8", Strcmp) != -1)

}

func TestBiDiUpdates(t *testing.T) {
	t.Parallel()
//场景：STIM1具有{ 1, 3 }，并且STIM2具有{0}和2}，并且两者同时发起到另一个。
//预期结果：两者最终都有0,1,2,3
	peers := make(map[string]*pullTestInstance)
	inst1 := newPushPullTestInstance("p1", peers)
	inst2 := newPushPullTestInstance("p2", peers)
	defer inst1.stop()
	defer inst2.stop()

	inst1.Add("1", "3")
	inst2.Add("0", "2")

	inst1.setNextPeerSelection([]string{"p2"})
	inst2.setNextPeerSelection([]string{"p1"})

	time.Sleep(time.Duration(2000) * time.Millisecond)

	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "0", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "1", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "2", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst1.state.ToArray(), "3", Strcmp) != -1)

	assert.True(t, util.IndexInSlice(inst2.state.ToArray(), "0", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst2.state.ToArray(), "1", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst2.state.ToArray(), "2", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst2.state.ToArray(), "3", Strcmp) != -1)

}

func TestSpread(t *testing.T) {
	t.Parallel()
//场景：inst1启动inst2，inst3 inst4，每个都有0-100项。inst5也有相同的项目，但没有被选中
//预期结果：每个应答者（inst2、inst3和inst4）至少选择一次（不选择每一个应答者的概率很小）。
//inst5根本没有被选中
	peers := make(map[string]*pullTestInstance)
	inst1 := newPushPullTestInstance("p1", peers)
	inst2 := newPushPullTestInstance("p2", peers)
	inst3 := newPushPullTestInstance("p3", peers)
	inst4 := newPushPullTestInstance("p4", peers)
	inst5 := newPushPullTestInstance("p5", peers)
	defer inst1.stop()
	defer inst2.stop()
	defer inst3.stop()
	defer inst4.stop()
	defer inst5.stop()

	chooseCounters := make(map[string]int)
	chooseCounters["p2"] = 0
	chooseCounters["p3"] = 0
	chooseCounters["p4"] = 0
	chooseCounters["p5"] = 0

	lock := &sync.Mutex{}

	addToCounters := func(dest string) func(m interface{}) {
		return func(m interface{}) {
			if _, isReq := m.(*reqMsg); isReq {
				lock.Lock()
				chooseCounters[dest]++
				lock.Unlock()
			}
		}
	}

	inst2.hook(addToCounters("p2"))
	inst3.hook(addToCounters("p3"))
	inst4.hook(addToCounters("p4"))
	inst5.hook(addToCounters("p5"))

	for i := 0; i < 100; i++ {
		item := fmt.Sprintf("%d", i)
		inst2.Add(item)
		inst3.Add(item)
		inst4.Add(item)
	}

	inst1.setNextPeerSelection([]string{"p2", "p3", "p4"})

	time.Sleep(time.Duration(2000) * time.Millisecond)

	lock.Lock()
	for pI, counter := range chooseCounters {
		if pI == "p5" {
			assert.Equal(t, 0, counter)
		} else {
			assert.True(t, counter > 0, "%s was not selected!", pI)
		}
	}
	lock.Unlock()
}

func TestFilter(t *testing.T) {
	t.Parallel()
//场景：3个实例，项目[0-5]只在第一个实例中找到，其他2个没有。
//而且第一个实例只给第二个实例偶数项，而奇数项给第三个实例。
//另外，实例2和3彼此不认识。
//预期结果：inst2只有偶数项，inst3只有奇数项
	peers := make(map[string]*pullTestInstance)
	inst1 := newPushPullTestInstance("p1", peers)
	inst2 := newPushPullTestInstance("p2", peers)
	inst3 := newPushPullTestInstance("p3", peers)
	defer inst1.stop()
	defer inst2.stop()
	defer inst3.stop()

	inst1.PullEngine.digFilter = func(context interface{}) func(digestItem string) bool {
		return func(digestItem string) bool {
			n, _ := strconv.ParseInt(digestItem, 10, 64)
			if context == "p2" {
				return n%2 == 0
			}
			return n%2 == 1
		}
	}

	inst1.Add("0", "1", "2", "3", "4", "5")
	inst2.setNextPeerSelection([]string{"p1"})
	inst3.setNextPeerSelection([]string{"p1"})

	time.Sleep(time.Second * 2)

	assert.True(t, util.IndexInSlice(inst2.state.ToArray(), "0", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst2.state.ToArray(), "1", Strcmp) == -1)
	assert.True(t, util.IndexInSlice(inst2.state.ToArray(), "2", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst2.state.ToArray(), "3", Strcmp) == -1)
	assert.True(t, util.IndexInSlice(inst2.state.ToArray(), "4", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst2.state.ToArray(), "5", Strcmp) == -1)

	assert.True(t, util.IndexInSlice(inst3.state.ToArray(), "0", Strcmp) == -1)
	assert.True(t, util.IndexInSlice(inst3.state.ToArray(), "1", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst3.state.ToArray(), "2", Strcmp) == -1)
	assert.True(t, util.IndexInSlice(inst3.state.ToArray(), "3", Strcmp) != -1)
	assert.True(t, util.IndexInSlice(inst3.state.ToArray(), "4", Strcmp) == -1)
	assert.True(t, util.IndexInSlice(inst3.state.ToArray(), "5", Strcmp) != -1)

}

func TestDefaultConfig(t *testing.T) {
	preDigestWaitTime := util.GetDurationOrDefault("peer.gossip.digestWaitTime", defDigestWaitTime)
	preRequestWaitTime := util.GetDurationOrDefault("peer.gossip.requestWaitTime", defRequestWaitTime)
	preResponseWaitTime := util.GetDurationOrDefault("peer.gossip.responseWaitTime", defResponseWaitTime)
	defer func() {
		SetDigestWaitTime(preDigestWaitTime)
		SetRequestWaitTime(preRequestWaitTime)
		SetResponseWaitTime(preResponseWaitTime)
	}()

//检查当没有属性时是否可以读取默认持续时间
//在配置文件中定义。
	viper.Reset()
	assert.Equal(t, time.Duration(1000)*time.Millisecond, util.GetDurationOrDefault("peer.gossip.digestWaitTime", defDigestWaitTime))
	assert.Equal(t, time.Duration(1500)*time.Millisecond, util.GetDurationOrDefault("peer.gossip.requestWaitTime", defRequestWaitTime))
	assert.Equal(t, time.Duration(2000)*time.Millisecond, util.GetDurationOrDefault("peer.gossip.responseWaitTime", defResponseWaitTime))

//检查配置文件（core.yaml）中的属性
//设置为所需的持续时间。
	viper.Reset()
	viper.SetConfigName("core")
	viper.SetEnvPrefix("CORE")
	configtest.AddDevConfigPath(nil)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(1000)*time.Millisecond, util.GetDurationOrDefault("peer.gossip.digestWaitTime", defDigestWaitTime))
	assert.Equal(t, time.Duration(1500)*time.Millisecond, util.GetDurationOrDefault("peer.gossip.requestWaitTime", defRequestWaitTime))
	assert.Equal(t, time.Duration(2000)*time.Millisecond, util.GetDurationOrDefault("peer.gossip.responseWaitTime", defResponseWaitTime))
}

func Strcmp(a interface{}, b interface{}) bool {
	return a.(string) == b.(string)
}

func keySet(selfPeer string, m map[string]*pullTestInstance) []string {
	peers := make([]string, len(m)-1)
	i := 0
	for pID := range m {
		if pID == selfPeer {
			continue
		}
		peers[i] = pID
		i++
	}

	return peers
}
