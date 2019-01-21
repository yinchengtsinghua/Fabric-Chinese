
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp，SecureKey Technologies Inc.保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package auth

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

func TestChainFilters(t *testing.T) {
	iterations := 15
	filters := createNFilters(iterations)
	endorser := &mockEndorserServer{}
	initialProposal := &peer.SignedProposal{ProposalBytes: make([]byte, 4)}
	binary.BigEndian.PutUint32(initialProposal.ProposalBytes, 0)

	firstFilter := ChainFilters(endorser, filters...)
	firstFilter.ProcessProposal(nil, initialProposal)
	for i := 0; i < iterations; i++ {
		assert.Equal(t, uint32(i), filters[i].(*mockAuthFilter).sequence,
			"Expected filters to be invoked in the provided sequence")
	}

	assert.Equal(t, uint32(iterations), endorser.sequence,
		"Expected endorser to be invoked after filters")

//无过滤器测试
	binary.BigEndian.PutUint32(initialProposal.ProposalBytes, 0)
	firstFilter = ChainFilters(endorser)
	firstFilter.ProcessProposal(nil, initialProposal)
	assert.Equal(t, uint32(0), endorser.sequence,
		"Expected endorser to be invoked first")
}

func createNFilters(n int) []Filter {
	filters := make([]Filter, n)
	for i := 0; i < n; i++ {
		filters[i] = &mockAuthFilter{}
	}
	return filters
}

type mockEndorserServer struct {
	sequence uint32
}

func (es *mockEndorserServer) ProcessProposal(ctx context.Context, prop *peer.SignedProposal) (*peer.ProposalResponse, error) {
	es.sequence = binary.BigEndian.Uint32(prop.ProposalBytes)
	binary.BigEndian.PutUint32(prop.ProposalBytes, es.sequence+1)
	return nil, nil
}

type mockAuthFilter struct {
	sequence uint32
	next     peer.EndorserServer
}

func (f *mockAuthFilter) ProcessProposal(ctx context.Context, prop *peer.SignedProposal) (*peer.ProposalResponse, error) {
	f.sequence = binary.BigEndian.Uint32(prop.ProposalBytes)
	binary.BigEndian.PutUint32(prop.ProposalBytes, f.sequence+1)
	return f.next.ProcessProposal(ctx, prop)
}

func (f *mockAuthFilter) Init(next peer.EndorserServer) {
	f.next = next
}
