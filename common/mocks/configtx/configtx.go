
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
*/


package configtx

import (
	cb "github.com/hyperledger/fabric/protos/common"
)

//
type Validator struct {
//
	ChainIDVal string

//
	SequenceVal uint64

//ApplyVal由Apply返回
	ApplyVal error

//
	AppliedConfigUpdateEnvelope *cb.ConfigEnvelope

//validateval由validate返回
	ValidateVal error

//
	ProposeConfigUpdateError error

//
	ProposeConfigUpdateVal *cb.ConfigEnvelope

//configProtoval作为configProtoval（）的值返回
	ConfigProtoVal *cb.Config
}

//
func (cm *Validator) ConfigProto() *cb.Config {
	return cm.ConfigProtoVal
}

//
func (cm *Validator) ChainID() string {
	return cm.ChainIDVal
}

//batchsize返回batchsizeval
func (cm *Validator) Sequence() uint64 {
	return cm.SequenceVal
}

//建议配置更新
func (cm *Validator) ProposeConfigUpdate(update *cb.Envelope) (*cb.ConfigEnvelope, error) {
	return cm.ProposeConfigUpdateVal, cm.ProposeConfigUpdateError
}

//
func (cm *Validator) Apply(configEnv *cb.ConfigEnvelope) error {
	cm.AppliedConfigUpdateEnvelope = configEnv
	return cm.ApplyVal
}

//validate返回validateval
func (cm *Validator) Validate(configEnv *cb.ConfigEnvelope) error {
	return cm.ValidateVal
}
