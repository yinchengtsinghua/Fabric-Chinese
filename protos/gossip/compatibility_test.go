
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


package gossip

import (
	"encoding/hex"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

var digestMsg = &GossipMessage{
	Channel: []byte("mychannel"),
	Content: &GossipMessage_DataDig{
		DataDig: &DataDigest{
			Digests: [][]byte{
				{255},
				{255, 255},
				{255, 255, 255},
				{255, 255, 255, 255},
				[]byte("100"),
			},
		},
	},
}

var requestMsg = &GossipMessage{
	Channel: []byte("mychannel"),
	Content: &GossipMessage_DataReq{
		DataReq: &DataRequest{
			Digests: [][]byte{
				{255},
				{255, 255},
				{255, 255, 255},
				{255, 255, 255, 255},
				[]byte("100"),
			},
		},
	},
}

const (
	v12DataDigestBytes  = "12096d796368616e6e656c52171201ff1202ffff1203ffffff1204ffffffff1203313030"
	v12DataRequestBytes = "12096d796368616e6e656c5a171201ff1202ffff1203ffffff1204ffffffff1203313030"
)

func TestUnmarshalV12Digests(t *testing.T) {
//此测试确保数据摘要消息和数据请求的摘要
//源于结构v1.3的内容可以被v1.2成功解析
	for msgBytes, expectedMsg := range map[string]*GossipMessage{
		v12DataDigestBytes:  digestMsg,
		v12DataRequestBytes: requestMsg,
	} {
		var err error
		v13Envelope := &Envelope{}
		v13Envelope.Payload, err = hex.DecodeString(msgBytes)
		assert.NoError(t, err)
		sMsg := &SignedGossipMessage{
			Envelope: v13Envelope,
		}
		v13Digest, err := sMsg.ToGossipMessage()
		assert.NoError(t, err)
		assert.True(t, proto.Equal(expectedMsg, v13Digest.GossipMessage))
	}
}
