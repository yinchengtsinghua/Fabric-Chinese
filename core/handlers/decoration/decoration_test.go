
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有SecureKey Technologies Inc.保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package decoration

import (
	"encoding/binary"
	"testing"

	"github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

const (
	decorationKey = "sequence"
)

func TestApplyDecorations(t *testing.T) {
	iterations := 15
	decorators := createNDecorators(iterations)
	initialInput := &peer.ChaincodeInput{Decorations: make(map[string][]byte)}
	seq := make([]byte, 4)
	binary.BigEndian.PutUint32(seq, 0)
	initialInput.Decorations[decorationKey] = seq

	finalInput := Apply(nil, initialInput, decorators...)
	for i := 0; i < iterations; i++ {
		assert.Equal(t, uint32(i), decorators[i].(*mockDecorator).sequence,
			"Expected decorators to be applied in the provided sequence")
	}

	assert.Equal(t, uint32(iterations), binary.BigEndian.Uint32(finalInput.Decorations[decorationKey]),
		"Expected decorators to be applied in the provided sequence")
}

func createNDecorators(n int) []Decorator {
	decorators := make([]Decorator, n)
	for i := 0; i < n; i++ {
		decorators[i] = &mockDecorator{}
	}
	return decorators
}

type mockDecorator struct {
	sequence uint32
}

func (d *mockDecorator) Decorate(proposal *peer.Proposal,
	input *peer.ChaincodeInput) *peer.ChaincodeInput {

	d.sequence = binary.BigEndian.Uint32(input.Decorations[decorationKey])
	binary.BigEndian.PutUint32(input.Decorations[decorationKey], d.sequence+1)

	return input
}
