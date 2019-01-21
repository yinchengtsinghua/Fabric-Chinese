
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


package peer

import (
	"github.com/hyperledger/fabric/common/channelconfig"
)

var supportFactory SupportFactory

//SupportFactory是支持接口的工厂
type SupportFactory interface {
//NewSupport返回支持接口
	NewSupport() Support
}

//支持允许访问对等资源并避免调用静态方法
type Support interface {
//GETAPPLICATIOFIGG返回通道的CONTXXAPPLATION.SARDCONFIGG
//以及应用程序配置是否存在
	GetApplicationConfig(cid string) (channelconfig.Application, bool)
}

type supportImpl struct {
	operations Operations
}

func (s *supportImpl) GetApplicationConfig(cid string) (channelconfig.Application, bool) {
	cc := s.operations.GetChannelConfig(cid)
	if cc == nil {
		return nil, false
	}

	return cc.ApplicationConfig()
}
