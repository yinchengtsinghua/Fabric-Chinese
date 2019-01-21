
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
	"testing"

	cb "github.com/hyperledger/fabric/protos/common"
	mspprotos "github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

//这个文件中的测试都是相对无意义的，因为所有这些函数都是被执行的。
//在正常的启动路径中，如果它们被破坏，事情会非常糟糕。
//另外，如果不简单地重新实现函数，就没有什么可以测试的了
//在测试中，也没有提供任何值。但是，不包括这些会产生人为的
//代码覆盖率很低，所以在这里。

func basicTest(t *testing.T, sv *StandardConfigValue) {
	assert.NotNil(t, sv)
	assert.NotEmpty(t, sv.Key())
	assert.NotNil(t, sv.Value())
}

func TestUtilsBasic(t *testing.T) {
	basicTest(t, ConsortiumValue("foo"))
	basicTest(t, HashingAlgorithmValue())
	basicTest(t, BlockDataHashingStructureValue())
	basicTest(t, OrdererAddressesValue([]string{"foo:1", "bar:2"}))
	basicTest(t, ConsensusTypeValue("foo", []byte("bar")))
	basicTest(t, BatchSizeValue(1, 2, 3))
	basicTest(t, BatchTimeoutValue("1s"))
	basicTest(t, ChannelRestrictionsValue(7))
	basicTest(t, KafkaBrokersValue([]string{"foo:1", "bar:2"}))
	basicTest(t, MSPValue(&mspprotos.MSPConfig{}))
	basicTest(t, CapabilitiesValue(map[string]bool{"foo": true, "bar": false}))
	basicTest(t, AnchorPeersValue([]*pb.AnchorPeer{{}, {}}))
	basicTest(t, ChannelCreationPolicyValue(&cb.Policy{}))
	basicTest(t, ACLValues(map[string]string{"foo": "fooval", "bar": "barval"}))
}
