
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有State Street Corp.2018保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package channelconfig

import (
	"testing"

	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

const (
	sampleAPI1Name      = "Foo"
	sampleAPI1PolicyRef = "foo"

	sampleAPI2Name      = "Bar"
	sampleAPI2PolicyRef = "/Channel/foo"
)

var sampleAPIsProvider = map[string]*pb.APIResource{
	sampleAPI1Name: {PolicyRef: sampleAPI1PolicyRef},
	sampleAPI2Name: {PolicyRef: sampleAPI2PolicyRef},
}

func TestGreenAPIsPath(t *testing.T) {
	ag := newAPIsProvider(sampleAPIsProvider)
	assert.NotNil(t, ag)

	t.Run("PresentAPIs", func(t *testing.T) {
		assert.Equal(t, "/Channel/Application/"+sampleAPI1PolicyRef, ag.PolicyRefForAPI(sampleAPI1Name))
		assert.Equal(t, sampleAPI2PolicyRef, ag.PolicyRefForAPI(sampleAPI2Name))
	})

	t.Run("MissingAPIs", func(t *testing.T) {
		assert.Empty(t, ag.PolicyRefForAPI("missing"))
	})
}

func TestNilACLs(t *testing.T) {
	ccg := newAPIsProvider(nil)

	assert.NotNil(t, ccg)
	assert.NotNil(t, ccg.aclPolicyRefs)
	assert.Empty(t, ccg.aclPolicyRefs)
}

func TestEmptyACLs(t *testing.T) {
	ccg := newAPIsProvider(map[string]*pb.APIResource{})

	assert.NotNil(t, ccg)
	assert.NotNil(t, ccg.aclPolicyRefs)
	assert.Empty(t, ccg.aclPolicyRefs)
}
