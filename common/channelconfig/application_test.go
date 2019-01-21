
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package channelconfig

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/capabilities"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	. "github.com/onsi/gomega"
)

func TestApplicationInterface(t *testing.T) {
	_ = Application((*ApplicationConfig)(nil))
}

func TestACL(t *testing.T) {
	g := NewGomegaWithT(t)
	cgt := &cb.ConfigGroup{
		Values: map[string]*cb.ConfigValue{
			ACLsKey: {
				Value: utils.MarshalOrPanic(
					ACLValues(map[string]string{}).Value(),
				),
			},
			CapabilitiesKey: {
				Value: utils.MarshalOrPanic(
					CapabilitiesValue(map[string]bool{
						capabilities.ApplicationV1_2: true,
					}).Value(),
				),
			},
		},
	}

	t.Run("Success", func(t *testing.T) {
		cg := proto.Clone(cgt).(*cb.ConfigGroup)
		_, err := NewApplicationConfig(proto.Clone(cg).(*cb.ConfigGroup), nil)
		g.Expect(err).NotTo(HaveOccurred())
	})

	t.Run("MissingCapability", func(t *testing.T) {
		cg := proto.Clone(cgt).(*cb.ConfigGroup)
		delete(cg.Values, CapabilitiesKey)
		_, err := NewApplicationConfig(cg, nil)
		g.Expect(err).To(MatchError("ACLs may not be specified without the required capability"))
	})
}
