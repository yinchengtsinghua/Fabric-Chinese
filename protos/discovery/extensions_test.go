
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


package discovery

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestToRequest(t *testing.T) {
	sr := &SignedRequest{
		Payload: []byte{0},
	}
	r, err := sr.ToRequest()
	assert.Error(t, err)

	req := &Request{}
	b, _ := proto.Marshal(req)
	sr.Payload = b
	r, err = sr.ToRequest()
	assert.NoError(t, err)
	assert.NotNil(t, r)
}

type invalidQuery struct {
}

func (*invalidQuery) isQuery_Query() {
}

func TestGetType(t *testing.T) {
	q := &Query{
		Query: &Query_PeerQuery{
			PeerQuery: &PeerMembershipQuery{},
		},
	}
	assert.Equal(t, PeerMembershipQueryType, q.GetType())
	q = &Query{
		Query: &Query_ConfigQuery{
			ConfigQuery: &ConfigQuery{},
		},
	}
	assert.Equal(t, ConfigQueryType, q.GetType())
	q = &Query{
		Query: &Query_CcQuery{
			CcQuery: &ChaincodeQuery{},
		},
	}
	assert.Equal(t, ChaincodeQueryType, q.GetType())

	q = &Query{
		Query: &invalidQuery{},
	}
	assert.Equal(t, InvalidQueryType, q.GetType())
}
