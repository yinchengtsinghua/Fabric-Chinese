
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

package server

import (
	"github.com/hyperledger/fabric/core/peer"
	"github.com/pkg/errors"
)

//go：生成伪造者-o mock/capability\u checker.go-forke name capability checker。能力检查人员

//CapabilityChecker用于检查通道是否支持令牌函数。
type CapabilityChecker interface {
	FabToken(channelId string) (bool, error)
}

//TokenCapabilityChecker实现CapabilityChecker接口
type TokenCapabilityChecker struct {
	PeerOps peer.Operations
}

func (c *TokenCapabilityChecker) FabToken(channelId string) (bool, error) {
	ac, ok := c.PeerOps.GetChannelConfig(channelId).ApplicationConfig()
	if !ok {
		return false, errors.Errorf("no application config found for channel %s", channelId)
	}
	return ac.Capabilities().FabToken(), nil
}
