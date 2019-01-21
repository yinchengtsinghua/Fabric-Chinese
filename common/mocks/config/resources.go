
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
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/configtx"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/msp"
)

type Resources struct {
//
	ConfigtxValidatorVal configtx.Validator

//
	PolicyManagerVal policies.Manager

//
	ChannelConfigVal channelconfig.Channel

//
	OrdererConfigVal channelconfig.Orderer

//
	ApplicationConfigVal channelconfig.Application

//
	ConsortiumsConfigVal channelconfig.Consortiums

//mspmanagerval作为mspmanager（）的结果返回
	MSPManagerVal msp.MSPManager

//validatenewr作为validatenew的结果返回
	ValidateNewErr error
}

//
func (r *Resources) ConfigtxValidator() configtx.Validator {
	return r.ConfigtxValidatorVal
}

//
func (r *Resources) PolicyManager() policies.Manager {
	return r.PolicyManagerVal
}

//
func (r *Resources) ChannelConfig() channelconfig.Channel {
	return r.ChannelConfigVal
}

//
func (r *Resources) OrdererConfig() (channelconfig.Orderer, bool) {
	return r.OrdererConfigVal, r.OrdererConfigVal != nil
}

//
func (r *Resources) ApplicationConfig() (channelconfig.Application, bool) {
	return r.ApplicationConfigVal, r.ApplicationConfigVal != nil
}

func (r *Resources) ConsortiumsConfig() (channelconfig.Consortiums, bool) {
	return r.ConsortiumsConfigVal, r.ConsortiumsConfigVal != nil
}

//
func (r *Resources) MSPManager() msp.MSPManager {
	return r.MSPManagerVal
}

//validateNew返回validatener
func (r *Resources) ValidateNew(res channelconfig.Resources) error {
	return r.ValidateNewErr
}
