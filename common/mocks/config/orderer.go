
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package config

import (
	"time"

	"github.com/hyperledger/fabric/common/channelconfig"
	ab "github.com/hyperledger/fabric/protos/orderer"
)

//
type Orderer struct {
//consensistypeval作为consensistype（）的结果返回
	ConsensusTypeVal string
//
	ConsensusMetadataVal []byte
//
	BatchSizeVal *ab.BatchSize
//batchTimeoutVal作为batchTimeout（）的结果返回
	BatchTimeoutVal time.Duration
//
	KafkaBrokersVal []string
//
	MaxChannelsCountVal uint64
//OrganizationsVal作为Organizations（）的结果返回
	OrganizationsVal map[string]channelconfig.Org
//
	CapabilitiesVal channelconfig.OrdererCapabilities
}

//
func (o *Orderer) ConsensusType() string {
	return o.ConsensusTypeVal
}

//
func (o *Orderer) ConsensusMetadata() []byte {
	return o.ConsensusMetadataVal
}

//
func (o *Orderer) BatchSize() *ab.BatchSize {
	return o.BatchSizeVal
}

//
func (o *Orderer) BatchTimeout() time.Duration {
	return o.BatchTimeoutVal
}

//KafkAbrokers返回KafkAbrokersVal
func (o *Orderer) KafkaBrokers() []string {
	return o.KafkaBrokersVal
}

//
func (o *Orderer) MaxChannelsCount() uint64 {
	return o.MaxChannelsCountVal
}

//
func (o *Orderer) Organizations() map[string]channelconfig.Org {
	return o.OrganizationsVal
}

//
func (o *Orderer) Capabilities() channelconfig.OrdererCapabilities {
	return o.CapabilitiesVal
}

//ordercapabilities模拟channelconfig.ordercapabilities接口
type OrdererCapabilities struct {
//SUPPORTEDER由SUPPORTED（）返回
	SupportedErr error

//
	PredictableChannelTemplateVal bool

//通过重新提交（）返回重新提交值
	ResubmissionVal bool

//expirationval由expirationcheck（）返回
	ExpirationVal bool
}

//支持的返回支持者
func (oc *OrdererCapabilities) Supported() error {
	return oc.SupportedErr
}

//
func (oc *OrdererCapabilities) PredictableChannelTemplate() bool {
	return oc.PredictableChannelTemplateVal
}

//
func (oc *OrdererCapabilities) Resubmission() bool {
	return oc.ResubmissionVal
}

//ExpirationCheck指定订购方是否检查标识过期检查
//验证消息时
func (oc *OrdererCapabilities) ExpirationCheck() bool {
	return oc.ExpirationVal
}
