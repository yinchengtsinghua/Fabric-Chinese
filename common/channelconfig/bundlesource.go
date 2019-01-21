
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


package channelconfig

import (
	"sync/atomic"

	"github.com/hyperledger/fabric/common/configtx"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/msp"
)

//BundleSource存储对当前配置包的引用
//它还提供了更新这个包的方法。各种方法
//很大程度上传递到底层包，但通过原子指针来传递
//因此，总的go例程读取不易受到无序执行内存的攻击
//
type BundleSource struct {
	bundle    atomic.Value
	callbacks []func(*Bundle)
}

//new bundlesource创建一个初始bundlesource值的新bundlesource
//每当为
//捆绑资源。注意，这些回调在该函数之前立即调用
//返回。
func NewBundleSource(bundle *Bundle, callbacks ...func(*Bundle)) *BundleSource {
	bs := &BundleSource{
		callbacks: callbacks,
	}
	bs.Update(bundle)
	return bs
}

//更新将新包设置为包源并调用任何已注册的回调
func (bs *BundleSource) Update(newBundle *Bundle) {
	bs.bundle.Store(newBundle)
	for _, callback := range bs.callbacks {
		callback(newBundle)
	}
}

//stable bundle返回指向稳定包的指针。
//它是稳定的，因为对其分类方法的调用总是返回相同的
//结果，因为基础数据结构是不可变的。例如，调用
//
//ORG，然后查询MSP以获取这些组织定义可能会导致bug，因为
//更新可能会替换中间的底层包。因此，对于操作
//这要求捆绑调用之间的一致性，调用方应首先检索
//一个稳定的，然后操作它。
func (bs *BundleSource) StableBundle() *Bundle {
	return bs.bundle.Load().(*Bundle)
}

//policyManager返回为此配置构造的策略管理器
func (bs *BundleSource) PolicyManager() policies.Manager {
	return bs.StableBundle().policyManager
}

//msp manager返回为此配置构造的msp管理器
func (bs *BundleSource) MSPManager() msp.MSPManager {
	return bs.StableBundle().mspManager
}

//channel config返回链的config.channel
func (bs *BundleSource) ChannelConfig() Channel {
	return bs.StableBundle().ChannelConfig()
}

//orderconfig返回通道的config.order
//以及医嘱者配置是否存在
func (bs *BundleSource) OrdererConfig() (Orderer, bool) {
	return bs.StableBundle().OrdererConfig()
}

//consortiums config（）返回通道的config.consortiums
//以及联合体配置是否存在
func (bs *BundleSource) ConsortiumsConfig() (Consortiums, bool) {
	return bs.StableBundle().ConsortiumsConfig()
}

//application config返回通道的应用程序配置
//以及应用程序配置是否存在
func (bs *BundleSource) ApplicationConfig() (Application, bool) {
	return bs.StableBundle().ApplicationConfig()
}

//configtx validator返回通道的configtx.validator
func (bs *BundleSource) ConfigtxValidator() configtx.Validator {
	return bs.StableBundle().ConfigtxValidator()
}

//validateNew传递到当前包
func (bs *BundleSource) ValidateNew(resources Resources) error {
	return bs.StableBundle().ValidateNew(resources)
}
