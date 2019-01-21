
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


package plain

import (
	"sync"

	"github.com/hyperledger/fabric/token/identity"
	"github.com/hyperledger/fabric/token/transaction"
	"github.com/pkg/errors"
)

//管理器用于访问TMS组件。
type Manager struct {
	mutex            sync.RWMutex
	policyValidators map[string]identity.IssuingValidator
}

//gettxprocessor返回用于处理令牌事务的tmstxprocessor。
func (m *Manager) GetTxProcessor(channel string) (transaction.TMSTxProcessor, error) {
	m.mutex.RLock()
	policyValidator := m.policyValidators[channel]
	m.mutex.RUnlock()
	if policyValidator == nil {
		return nil, errors.Errorf("no policy validator found for channel '%s'", channel)
	}
	return &Verifier{IssuingValidator: policyValidator}, nil
}

//setpolicyvalidator为指定通道设置策略验证程序
func (m *Manager) SetPolicyValidator(channel string, validator identity.IssuingValidator) {
	m.mutex.Lock()
	m.policyValidators[channel] = validator
	m.mutex.Unlock()
}
