
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
	cb "github.com/hyperledger/fabric/protos/common"
)

const (
//consortiumsgroupkey是consortiums配置的组名
	ConsortiumsGroupKey = "Consortiums"
)

//CONSORITUMSCONFIG保存联盟配置信息
type ConsortiumsConfig struct {
	consortiums map[string]Consortium
}

//newconsortiumsconfig创建consoritums配置的新实例
func NewConsortiumsConfig(consortiumsGroup *cb.ConfigGroup, mspConfig *MSPConfigHandler) (*ConsortiumsConfig, error) {
	cc := &ConsortiumsConfig{
		consortiums: make(map[string]Consortium),
	}

	for consortiumName, consortiumGroup := range consortiumsGroup.Groups {
		var err error
		if cc.consortiums[consortiumName], err = NewConsortiumConfig(consortiumGroup, mspConfig); err != nil {
			return nil, err
		}
	}
	return cc, nil
}

//联合体返回当前联合体的映射
func (cc *ConsortiumsConfig) Consortiums() map[string]Consortium {
	return cc.consortiums
}
