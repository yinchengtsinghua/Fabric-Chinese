
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有State Street Corp.保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package endorser

import (
	"testing"

	"github.com/hyperledger/fabric/common/metrics/metricsfakes"
	. "github.com/onsi/gomega"
)

func TestNewEndorserMetrics(t *testing.T) {
	gt := NewGomegaWithT(t)

	provider := &metricsfakes.Provider{}
	provider.NewHistogramReturns(&metricsfakes.Histogram{})
	provider.NewCounterReturns(&metricsfakes.Counter{})

	endorserMetrics := NewEndorserMetrics(provider)
	gt.Expect(endorserMetrics).To(Equal(&EndorserMetrics{
		ProposalDuration:         &metricsfakes.Histogram{},
		ProposalsReceived:        &metricsfakes.Counter{},
		SuccessfulProposals:      &metricsfakes.Counter{},
		ProposalValidationFailed: &metricsfakes.Counter{},
		ProposalACLCheckFailed:   &metricsfakes.Counter{},
		InitFailed:               &metricsfakes.Counter{},
		EndorsementsFailed:       &metricsfakes.Counter{},
		DuplicateTxsFailure:      &metricsfakes.Counter{},
	}))

	gt.Expect(provider.NewHistogramCallCount()).To(Equal(1))
	gt.Expect(provider.Invocations()["NewHistogram"]).To(ConsistOf([][]interface{}{
		{proposalDurationHistogramOpts},
	}))

	gt.Expect(provider.NewCounterCallCount()).To(Equal(7))
	gt.Expect(provider.Invocations()["NewCounter"]).To(ConsistOf([][]interface{}{
		{receivedProposalsCounterOpts},
		{successfulProposalsCounterOpts},
		{proposalValidationFailureCounterOpts},
		{proposalChannelACLFailureOpts},
		{initFailureCounterOpts},
		{endorsementFailureCounterOpts},
		{duplicateTxsFailureCounterOpts},
	}))
}
