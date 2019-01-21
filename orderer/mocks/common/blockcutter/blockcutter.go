
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


package blockcutter

import (
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	cb "github.com/hyperledger/fabric/protos/common"
)

var logger = flogging.MustGetLogger("orderer.mocks.common.blockcutter")

//接收器模拟BlockCutter.Receiver接口
type Receiver struct {
//IsolatedTx导致有序返回[][]抑制匹配，[]newTx，设置为真时为假
	IsolatedTx bool

//CutSeversales导致有序返回[]Curbatch，设置为true时为true
	CutAncestors bool

//CutNext导致有序返回[]append（corbatch，newtx），设置为true时为false
	CutNext bool

//SkipPendCurbatch导致命令跳过附加到Curbatch
	SkipAppendCurBatch bool

//锁定以序列化写入对Curbatch的访问
	mutex sync.Mutex

//Curbatch是批处理中当前未处理的消息
	curBatch []*cb.Envelope

//块是从有序返回之前读取的通道，它对同步很有用。
//如果出于任何原因不希望同步，只需关闭通道
	Block chan struct{}
}

//newReceiver返回模拟blockcutter.Receiver实现
func NewReceiver() *Receiver {
	return &Receiver{
		IsolatedTx:   false,
		CutAncestors: false,
		CutNext:      false,
		Block:        make(chan struct{}),
	}
}

//订单将根据收货方的状态添加或剪切批次，返回时阻止从块读取。
func (mbc *Receiver) Ordered(env *cb.Envelope) ([][]*cb.Envelope, bool) {
	defer func() {
		<-mbc.Block
	}()

	mbc.mutex.Lock()
	defer mbc.mutex.Unlock()

	if mbc.IsolatedTx {
		logger.Debugf("Receiver: Returning dual batch")
		res := [][]*cb.Envelope{mbc.curBatch, {env}}
		mbc.curBatch = nil
		return res, false
	}

	if mbc.CutAncestors {
		logger.Debugf("Receiver: Returning current batch and appending newest env")
		res := [][]*cb.Envelope{mbc.curBatch}
		mbc.curBatch = []*cb.Envelope{env}
		return res, true
	}

	if !mbc.SkipAppendCurBatch {
		mbc.curBatch = append(mbc.curBatch, env)
	}

	if mbc.CutNext {
		logger.Debugf("Receiver: Returning regular batch")
		res := [][]*cb.Envelope{mbc.curBatch}
		mbc.curBatch = nil
		return res, false
	}

	logger.Debugf("Appending to batch")
	return nil, true
}

//CUT终止当前批，返回当前批
func (mbc *Receiver) Cut() []*cb.Envelope {
	mbc.mutex.Lock()
	defer mbc.mutex.Unlock()
	logger.Debugf("Cutting batch")
	res := mbc.curBatch
	mbc.curBatch = nil
	return res
}

func (mbc *Receiver) CurBatch() []*cb.Envelope {
	mbc.mutex.Lock()
	defer mbc.mutex.Unlock()
	return mbc.curBatch
}
