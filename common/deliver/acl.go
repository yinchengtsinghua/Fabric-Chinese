
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


package deliver

import (
	"time"

	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

//expiresatfunc用于提取标识过期的时间。
type ExpiresAtFunc func(identityBytes []byte) time.Time

//configSequencer提供当前配置块的序列号。
type ConfigSequencer interface {
	Sequence() uint64
}

//
//如果无法从信封中提取签名头，则返回错误。
func NewSessionAC(chain ConfigSequencer, env *common.Envelope, policyChecker PolicyChecker, channelID string, expiresAt ExpiresAtFunc) (*SessionAccessControl, error) {
	signedData, err := env.AsSignedData()
	if err != nil {
		return nil, err
	}

	return &SessionAccessControl{
		envelope:       env,
		channelID:      channelID,
		sequencer:      chain,
		policyChecker:  policyChecker,
		sessionEndTime: expiresAt(signedData[0].Identity),
	}, nil
}

//sessionaccesscontrol为公共信封保存与访问控制相关的数据
//
//与请求信封关联。
type SessionAccessControl struct {
	sequencer          ConfigSequencer
	policyChecker      PolicyChecker
	channelID          string
	envelope           *common.Envelope
	lastConfigSequence uint64
	sessionEndTime     time.Time
	usedAtLeastOnce    bool
}

//
//
//变化。
func (ac *SessionAccessControl) Evaluate() error {
	if !ac.sessionEndTime.IsZero() && time.Now().After(ac.sessionEndTime) {
		return errors.Errorf("client identity expired %v before", time.Since(ac.sessionEndTime))
	}

	policyCheckNeeded := !ac.usedAtLeastOnce

	if currentConfigSequence := ac.sequencer.Sequence(); currentConfigSequence > ac.lastConfigSequence {
		ac.lastConfigSequence = currentConfigSequence
		policyCheckNeeded = true
	}

	if !policyCheckNeeded {
		return nil
	}

	ac.usedAtLeastOnce = true
	return ac.policyChecker.CheckPolicy(ac.envelope, ac.channelID)
}
