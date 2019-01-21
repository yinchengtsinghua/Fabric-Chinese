
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


package admin

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	google_protobuf "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockEvaluator struct {
	err error
}

func (e *mockEvaluator) Evaluate(signatureSet []*common.SignedData) error {
	return e.err
}

func TestValidate(t *testing.T) {
	v := &validator{}
	_, err := v.validate(nil, nil)
	assert.Contains(t, err.Error(), "nil context")

	env := validRequest()
	v.ace = &mockEvaluator{err: errors.New("access denied")}
	_, err = v.validate(context.Background(), env)
	assert.Equal(t, accessDenied, err)

	v.ace = &mockEvaluator{err: nil}
	op2, err := v.validate(context.Background(), env)
	op := &peer.AdminOperation{
		Content: &peer.AdminOperation_LogReq{
			LogReq: &peer.LogLevelRequest{LogLevel: "foo"},
		},
	}
	assert.Equal(t, op, op2)
	assert.NoError(t, err)
}

func TestValidateStructureRequestBadInput(t *testing.T) {
	ctx := context.Background()
	op, sd, err := validateStructure(ctx, nil)
	assert.Nil(t, op)
	assert.Nil(t, sd)
	assert.Equal(t, "nil envelope", err.Error())

	op, sd, err = validateStructure(nil, &common.Envelope{})
	assert.Nil(t, op)
	assert.Nil(t, sd)
	assert.Equal(t, "nil context", err.Error())

	op, sd, err = validateStructure(ctx, &common.Envelope{})
	assert.Nil(t, op)
	assert.Nil(t, sd)
	assert.Contains(t, err.Error(), "envelope must have a Header")

	pl := &common.Payload{}
	pl.Header = &common.Header{
		ChannelHeader: []byte{1, 2, 3, 4, 5, 6},
	}
	plBytes, _ := proto.Marshal(pl)
	op, sd, err = validateStructure(ctx, &common.Envelope{Payload: plBytes})
	assert.Nil(t, op)
	assert.Nil(t, sd)
	assert.Contains(t, err.Error(), "error unmarshaling ChannelHeader")

	ch := &common.ChannelHeader{
		Type: int32(common.HeaderType_PEER_ADMIN_OPERATION),
	}
	chBytes, _ := proto.Marshal(ch)
	pl = &common.Payload{}
	pl.Header = &common.Header{
		ChannelHeader: chBytes,
	}
	plBytes, _ = proto.Marshal(pl)
	op, sd, err = validateStructure(ctx, &common.Envelope{Payload: plBytes})
	assert.Nil(t, op)
	assert.Nil(t, sd)
	assert.Contains(t, err.Error(), "empty timestamp")

	ch = &common.ChannelHeader{
		Type:      int32(common.HeaderType_PEER_ADMIN_OPERATION),
		Timestamp: &google_protobuf.Timestamp{},
	}
	chBytes, _ = proto.Marshal(ch)
	pl = &common.Payload{}
	pl.Header = &common.Header{
		ChannelHeader: chBytes,
	}
	plBytes, _ = proto.Marshal(pl)
	op, sd, err = validateStructure(ctx, &common.Envelope{Payload: plBytes})
	assert.Nil(t, op)
	assert.Nil(t, sd)
	assert.Contains(t, err.Error(), "access denied")

	now := time.Now()
	ch = &common.ChannelHeader{
		Type: int32(common.HeaderType_PEER_ADMIN_OPERATION),
		Timestamp: &google_protobuf.Timestamp{
			Seconds: now.UnixNano() / 1000 / 1000 / 1000,
		},
	}
	chBytes, _ = proto.Marshal(ch)
	pl = &common.Payload{
		Data: []byte{1, 2, 3, 4, 5, 6},
	}
	pl.Header = &common.Header{
		ChannelHeader: chBytes,
	}
	plBytes, _ = proto.Marshal(pl)
	op, sd, err = validateStructure(ctx, &common.Envelope{Payload: plBytes})
	assert.Nil(t, op)
	assert.Nil(t, sd)
	assert.Contains(t, err.Error(), "error unmarshaling message")
}

func TestValidateStructureRequestGoodInput(t *testing.T) {
	op := &peer.AdminOperation{
		Content: &peer.AdminOperation_LogReq{
			LogReq: &peer.LogLevelRequest{LogLevel: "foo"},
		},
	}

	env := validRequest()
	sd, _ := env.AsSignedData()

	op2, sd, err := validateStructure(context.Background(), env)
	assert.NoError(t, err)
	assert.Equal(t, op, op2)
	assert.Equal(t, []*common.SignedData{{
		Data:      sd[0].Data,
		Signature: sd[0].Signature,
		Identity:  sd[0].Identity,
	}}, sd)
}

func validRequest() *common.Envelope {
	now := time.Now()
	ch := &common.ChannelHeader{
		Type: int32(common.HeaderType_PEER_ADMIN_OPERATION),
		Timestamp: &google_protobuf.Timestamp{
			Seconds: now.UnixNano() / 1000 / 1000 / 1000,
		},
	}
	op := &peer.AdminOperation{
		Content: &peer.AdminOperation_LogReq{
			LogReq: &peer.LogLevelRequest{LogLevel: "foo"},
		},
	}
	opBytes, _ := proto.Marshal(op)
	chBytes, _ := proto.Marshal(ch)
	pl := &common.Payload{
		Data: opBytes,
	}
	pl.Header = &common.Header{
		ChannelHeader: chBytes,
	}
	plBytes, _ := proto.Marshal(pl)
	return &common.Envelope{Payload: plBytes}
}
