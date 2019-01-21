
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
	"sync"
	"sync/atomic"
	"time"

	"github.com/hyperledger/fabric/gossip/util"
	"github.com/spf13/viper"
)

/*pullengine是一个执行基于pull的八卦并维护项目内部状态的对象。
   由字符串编号标识。
   协议如下：
   1）发起程序向一组远程对等端发送带有特定nonce的hello消息。
   2）每个远程对等端都用其消息摘要响应，并返回该nonce。
   3）发起者检查接收到的nonce的有效性，汇总摘要，
      然后创建一个包含要从每个远程对等机接收的特定项目ID的请求，然后
      将每个请求发送到其对应的对等方。
   4）如果每个对等方仍保留请求的项目和nonce，则它将返回包含这些项目的响应。

    其他对等发起程序
  O<-------你好
 /\-------摘要<[3,5,8，10…]，nonce>——————————/\
  <-------请求<[3,8]，nonce>-----------------
 /\-------响应<[item3，item8]，nonce>------/\

**/


const (
	defDigestWaitTime   = time.Duration(1000) * time.Millisecond
	defRequestWaitTime  = time.Duration(1500) * time.Millisecond
	defResponseWaitTime = time.Duration(2000) * time.Millisecond
)

//setDigestWaitTime设置摘要等待时间
func SetDigestWaitTime(time time.Duration) {
	viper.Set("peer.gossip.digestWaitTime", time)
}

//setrequestwaittime设置请求等待时间
func SetRequestWaitTime(time time.Duration) {
	viper.Set("peer.gossip.requestWaitTime", time)
}

//setResponseWaitTime设置响应等待时间
func SetResponseWaitTime(time time.Duration) {
	viper.Set("peer.gossip.responseWaitTime", time)
}

//DigestFilter筛选要发送到远程对等机的摘要
//根据消息的上下文发送了问候或请求
type DigestFilter func(context interface{}) func(digestItem string) bool

//拉具发动机需要拉转接器，以便
//向远程pullengine实例发送消息。
//pullengine应该用
//OnHello, OnDigest, OnReq, OnRes when the respective message arrives
//从远程拉发动机
type PullAdapter interface {
//selectpeers返回一个对等机切片，引擎将使用该切片启动协议
	SelectPeers() []string

//hello发送hello消息以启动协议
//并返回一个期望返回的nonce
//在摘要消息中。
	Hello(dest string, nonce uint64)

//sendDigest将摘要发送到远程pullengine。
//上下文参数指定要发送到的远程引擎。
	SendDigest(digest []string, nonce uint64, context interface{})

//sendreq将一组项目发送到某个已标识的远程拉器引擎
//用绳子
	SendReq(dest string, items []string, nonce uint64)

//sendres将一组项目发送到由上下文标识的远程pullengine。
	SendRes(items []string, context interface{}, nonce uint64)
}

//pullengine是实际调用pull算法的组件
//在PullAdapter的帮助下
type PullEngine struct {
	PullAdapter
	stopFlag           int32
	state              *util.Set
	item2owners        map[string][]string
	peers2nonces       map[string]uint64
	nonces2peers       map[uint64]string
	acceptingDigests   int32
	acceptingResponses int32
	lock               sync.Mutex
	outgoingNONCES     *util.Set
	incomingNONCES     *util.Set
	digFilter          DigestFilter

	digestWaitTime   time.Duration
	requestWaitTime  time.Duration
	responseWaitTime time.Duration
}

//newpullenginewithfilter创建具有特定睡眠时间的pullengine实例
//在拉式启动之间，并在发送摘要和响应时使用给定的筛选器
func NewPullEngineWithFilter(participant PullAdapter, sleepTime time.Duration, df DigestFilter) *PullEngine {
	engine := &PullEngine{
		PullAdapter:        participant,
		stopFlag:           int32(0),
		state:              util.NewSet(),
		item2owners:        make(map[string][]string),
		peers2nonces:       make(map[string]uint64),
		nonces2peers:       make(map[uint64]string),
		acceptingDigests:   int32(0),
		acceptingResponses: int32(0),
		incomingNONCES:     util.NewSet(),
		outgoingNONCES:     util.NewSet(),
		digFilter:          df,
		digestWaitTime:     util.GetDurationOrDefault("peer.gossip.digestWaitTime", defDigestWaitTime),
		requestWaitTime:    util.GetDurationOrDefault("peer.gossip.requestWaitTime", defRequestWaitTime),
		responseWaitTime:   util.GetDurationOrDefault("peer.gossip.responseWaitTime", defResponseWaitTime),
	}

	go func() {
		for !engine.toDie() {
			time.Sleep(sleepTime)
			if engine.toDie() {
				return
			}
			engine.initiatePull()
		}
	}()

	return engine
}

//newpullengine创建具有特定睡眠时间的pullengine实例
//在拉动启动之间
func NewPullEngine(participant PullAdapter, sleepTime time.Duration) *PullEngine {
	acceptAllFilter := func(_ interface{}) func(string) bool {
		return func(_ string) bool {
			return true
		}
	}
	return NewPullEngineWithFilter(participant, sleepTime, acceptAllFilter)
}

func (engine *PullEngine) toDie() bool {
	return atomic.LoadInt32(&(engine.stopFlag)) == int32(1)
}

func (engine *PullEngine) acceptResponses() {
	atomic.StoreInt32(&(engine.acceptingResponses), int32(1))
}

func (engine *PullEngine) isAcceptingResponses() bool {
	return atomic.LoadInt32(&(engine.acceptingResponses)) == int32(1)
}

func (engine *PullEngine) acceptDigests() {
	atomic.StoreInt32(&(engine.acceptingDigests), int32(1))
}

func (engine *PullEngine) isAcceptingDigests() bool {
	return atomic.LoadInt32(&(engine.acceptingDigests)) == int32(1)
}

func (engine *PullEngine) ignoreDigests() {
	atomic.StoreInt32(&(engine.acceptingDigests), int32(0))
}

//停止停止发动机
func (engine *PullEngine) Stop() {
	atomic.StoreInt32(&(engine.stopFlag), int32(1))
}

func (engine *PullEngine) initiatePull() {
	engine.lock.Lock()
	defer engine.lock.Unlock()

	engine.acceptDigests()
	for _, peer := range engine.SelectPeers() {
		nonce := engine.newNONCE()
		engine.outgoingNONCES.Add(nonce)
		engine.nonces2peers[nonce] = peer
		engine.peers2nonces[peer] = nonce
		engine.Hello(peer, nonce)
	}

	time.AfterFunc(engine.digestWaitTime, func() {
		engine.processIncomingDigests()
	})
}

func (engine *PullEngine) processIncomingDigests() {
	engine.ignoreDigests()

	engine.lock.Lock()
	defer engine.lock.Unlock()

	requestMapping := make(map[string][]string)
	for n, sources := range engine.item2owners {
//选择随机源
		source := sources[util.RandomInt(len(sources))]
		if _, exists := requestMapping[source]; !exists {
			requestMapping[source] = make([]string, 0)
		}
//将数字追加到该源
		requestMapping[source] = append(requestMapping[source], n)
	}

	engine.acceptResponses()

	for dest, seqsToReq := range requestMapping {
		engine.SendReq(dest, seqsToReq, engine.peers2nonces[dest])
	}

	time.AfterFunc(engine.responseWaitTime, engine.endPull)
}

func (engine *PullEngine) endPull() {
	engine.lock.Lock()
	defer engine.lock.Unlock()

	atomic.StoreInt32(&(engine.acceptingResponses), int32(0))
	engine.outgoingNONCES.Clear()

	engine.item2owners = make(map[string][]string)
	engine.peers2nonces = make(map[string]uint64)
	engine.nonces2peers = make(map[uint64]string)
}

//OnDigest通知引擎摘要已到达
func (engine *PullEngine) OnDigest(digest []string, nonce uint64, context interface{}) {
	if !engine.isAcceptingDigests() || !engine.outgoingNONCES.Exists(nonce) {
		return
	}

	engine.lock.Lock()
	defer engine.lock.Unlock()

	for _, n := range digest {
		if engine.state.Exists(n) {
			continue
		}

		if _, exists := engine.item2owners[n]; !exists {
			engine.item2owners[n] = make([]string, 0)
		}

		engine.item2owners[n] = append(engine.item2owners[n], engine.nonces2peers[nonce])
	}
}

//添加将项添加到状态
func (engine *PullEngine) Add(seqs ...string) {
	for _, seq := range seqs {
		engine.state.Add(seq)
	}
}

//移除从状态中移除项
func (engine *PullEngine) Remove(seqs ...string) {
	for _, seq := range seqs {
		engine.state.Remove(seq)
	}
}

//OnHello通知引擎已到达Hello
func (engine *PullEngine) OnHello(nonce uint64, context interface{}) {
	engine.incomingNONCES.Add(nonce)

	time.AfterFunc(engine.requestWaitTime, func() {
		engine.incomingNONCES.Remove(nonce)
	})

	a := engine.state.ToArray()
	var digest []string
	filter := engine.digFilter(context)
	for _, item := range a {
		dig := item.(string)
		if !filter(dig) {
			continue
		}
		digest = append(digest, dig)
	}
	if len(digest) == 0 {
		return
	}
	engine.SendDigest(digest, nonce, context)
}

//OnReq通知引擎请求已到达
func (engine *PullEngine) OnReq(items []string, nonce uint64, context interface{}) {
	if !engine.incomingNONCES.Exists(nonce) {
		return
	}
	engine.lock.Lock()
	defer engine.lock.Unlock()

	filter := engine.digFilter(context)
	var items2Send []string
	for _, item := range items {
		if engine.state.Exists(item) && filter(item) {
			items2Send = append(items2Send, item)
		}
	}

	if len(items2Send) == 0 {
		return
	}

	go engine.SendRes(items2Send, context, nonce)
}

//OnRes通知引擎响应已到达
func (engine *PullEngine) OnRes(items []string, nonce uint64) {
	if !engine.outgoingNONCES.Exists(nonce) || !engine.isAcceptingResponses() {
		return
	}

	engine.Add(items...)
}

func (engine *PullEngine) newNONCE() uint64 {
	n := uint64(0)
	for {
		n = util.RandomUInt64()
		if !engine.outgoingNONCES.Exists(n) {
			return n
		}
	}
}
