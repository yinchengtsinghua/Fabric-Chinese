
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


package manager

import (
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/token/identity"
	"github.com/hyperledger/fabric/token/tms/plain"
	"github.com/hyperledger/fabric/token/transaction"
	"github.com/pkg/errors"
)

//go：生成伪造者-o mock/identity _deserializer _manager.go-fake name deserializer manager。反序列化管理器

//FabricidentityDeserializerManager实现一个DeserializerManager
//通过将呼叫路由到MSP/MGMT包
type FabricIdentityDeserializerManager struct {
}

func (*FabricIdentityDeserializerManager) Deserializer(channel string) (identity.Deserializer, error) {
	id, ok := mgmt.GetDeserializers()[channel]
	if !ok {
		return nil, errors.New("channel not found")
	}
	return id, nil
}

//管理器用于访问TMS组件。
type Manager struct {
	IdentityDeserializerManager identity.DeserializerManager
}

//gettxprocessor返回用于处理令牌事务的tmstxprocessor。
func (m *Manager) GetTxProcessor(channel string) (transaction.TMSTxProcessor, error) {
	identityDeserializerManager, err := m.IdentityDeserializerManager.Deserializer(channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting identity deserialiser manager for channel '%s'", channel)
	}

	return &plain.Verifier{IssuingValidator: &AllIssuingValidator{Deserializer: identityDeserializerManager}}, nil
}
