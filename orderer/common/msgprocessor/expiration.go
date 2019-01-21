
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


package msgprocessor

import (
	"time"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

type resources interface {
//orderconfig返回通道的config.order
//以及医嘱者配置是否存在
	OrdererConfig() (channelconfig.Orderer, bool)
}

//NewExpirationRejectRule返回拒绝由标识签名的邮件的规则
//由于该功能处于活动状态，谁的身份已过期
func NewExpirationRejectRule(filterSupport resources) Rule {
	return &expirationRejectRule{filterSupport: filterSupport}
}

type expirationRejectRule struct {
	filterSupport resources
}

//应用检查创建信封的标识是否已过期
func (exp *expirationRejectRule) Apply(message *common.Envelope) error {
	ordererConf, ok := exp.filterSupport.OrdererConfig()
	if !ok {
		logger.Panic("Programming error: orderer config not found")
	}
	if !ordererConf.Capabilities().ExpirationCheck() {
		return nil
	}
	signedData, err := message.AsSignedData()

	if err != nil {
		return errors.Errorf("could not convert message to signedData: %s", err)
	}
	expirationTime := crypto.ExpiresAt(signedData[0].Identity)
//标识不能过期，或者标识尚未过期
	if expirationTime.IsZero() || time.Now().Before(expirationTime) {
		return nil
	}
	return errors.New("identity expired")
}
