
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


package kvledger

import (
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/metrics/metricsfakes"
	lgr "github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

func TestStatsBlockchainHeight(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	testMetricProvider := testutilConstructMetricProvider()
	provider, err := NewProvider()
	assert.NoError(t, err)
	provider.Initialize(&lgr.Initializer{
		DeployedChaincodeInfoProvider: &mock.DeployedChaincodeInfoProvider{},
		MetricsProvider:               testMetricProvider.fakeProvider,
	})
	defer provider.Close()

//创建分类帐
	ledgerid := "ledger1"
	_, gb := testutil.NewBlockGenerator(t, ledgerid, false)
	l, err := provider.Create(gb)
	assert.NoError(t, err)
	ledger := l.(*kvLedger)
	defer ledger.Close()

	fakeBlockchainHeightGauge := testMetricProvider.fakeBlockchainHeightGauge
//分类帐创建期间调用
	assert.Equal(t, float64(0), fakeBlockchainHeightGauge.SetArgsForCall(0))
	assert.Equal(t, []string{"channel", ledgerid}, fakeBlockchainHeightGauge.WithArgsForCall(0))

//在提交Genesis块期间调用
	assert.Equal(t, float64(1), fakeBlockchainHeightGauge.SetArgsForCall(1))
	assert.Equal(t, []string{"channel", ledgerid}, fakeBlockchainHeightGauge.WithArgsForCall(1))

	ledger.Close()
	provider.Open(ledgerid)
//在打开现有分类帐期间调用-打开分类帐应设置当前高度
	assert.Equal(t, float64(1), fakeBlockchainHeightGauge.SetArgsForCall(2))
	assert.Equal(t, []string{"channel", ledgerid}, fakeBlockchainHeightGauge.WithArgsForCall(2))

//显式调用updateBlockStats API并使用假度量验证调用
	ledger.updateBlockStats(
		10, 1*time.Second, 2*time.Second, 3*time.Second, nil,
	)
	assert.Equal(t, []string{"channel", ledgerid}, fakeBlockchainHeightGauge.WithArgsForCall(3))
	assert.Equal(t, float64(11), fakeBlockchainHeightGauge.SetArgsForCall(3))
}

func TestStatsBlockCommit(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	testMetricProvider := testutilConstructMetricProvider()
	provider, err := NewProvider()
	assert.NoError(t, err)
	provider.Initialize(&lgr.Initializer{
		DeployedChaincodeInfoProvider: &mock.DeployedChaincodeInfoProvider{},
		MetricsProvider:               testMetricProvider.fakeProvider,
	})
	defer provider.Close()

//创建分类帐
	ledgerid := "ledger1"
	_, gb := testutil.NewBlockGenerator(t, ledgerid, false)
	l, err := provider.Create(gb)
	assert.NoError(t, err)
	ledger := l.(*kvLedger)
	defer ledger.Close()

//在提交Genesis块期间调用
	assert.Equal(t,
		[]string{"channel", ledgerid},
		testMetricProvider.fakeBlockProcessingTimeHist.WithArgsForCall(0),
	)
	assert.Equal(t,
		[]string{"channel", ledgerid},
		testMetricProvider.fakeBlockstorageCommitTimeHist.WithArgsForCall(0),
	)
	assert.Equal(t,
		[]string{"channel", ledgerid},
		testMetricProvider.fakeStatedbCommitTimeHist.WithArgsForCall(0),
	)
	assert.Equal(t,
		[]string{
			"channel", ledgerid,
			"transaction_type", common.HeaderType_CONFIG.String(),
			"chaincode", "unknown",
			"validation_code", peer.TxValidationCode_VALID.String(),
		},
		testMetricProvider.fakeTransactionsCount.WithArgsForCall(0),
	)

//显式调用updateBlockStats API并使用假度量验证调用
	ledger.updateBlockStats(
		10, 1*time.Second, 2*time.Second, 3*time.Second,
		[]*txmgr.TxStatInfo{
			{
				ValidationCode: peer.TxValidationCode_VALID,
				TxType:         common.HeaderType_ENDORSER_TRANSACTION,
				ChaincodeID:    &peer.ChaincodeID{Name: "mycc", Version: "1.0"},
				NumCollections: 2,
			},
			{
				ValidationCode: peer.TxValidationCode_INVALID_OTHER_REASON,
				TxType:         -1,
			},
		},
	)
	assert.Equal(t,
		[]string{"channel", ledgerid},
		testMetricProvider.fakeBlockProcessingTimeHist.WithArgsForCall(1),
	)
	assert.Equal(t,
		float64(1),
		testMetricProvider.fakeBlockProcessingTimeHist.ObserveArgsForCall(1),
	)
	assert.Equal(t,
		[]string{"channel", ledgerid},
		testMetricProvider.fakeBlockstorageCommitTimeHist.WithArgsForCall(1),
	)
	assert.Equal(t,
		float64(2),
		testMetricProvider.fakeBlockstorageCommitTimeHist.ObserveArgsForCall(1),
	)
	assert.Equal(t,
		[]string{"channel", ledgerid},
		testMetricProvider.fakeStatedbCommitTimeHist.WithArgsForCall(1),
	)
	assert.Equal(t,
		float64(3),
		testMetricProvider.fakeStatedbCommitTimeHist.ObserveArgsForCall(1),
	)
	assert.Equal(t,
		[]string{
			"channel", ledgerid,
			"transaction_type", common.HeaderType_ENDORSER_TRANSACTION.String(),
			"chaincode", "mycc:1.0",
			"validation_code", peer.TxValidationCode_VALID.String(),
		},
		testMetricProvider.fakeTransactionsCount.WithArgsForCall(1),
	)
	assert.Equal(t,
		float64(1),
		testMetricProvider.fakeTransactionsCount.AddArgsForCall(1),
	)

	assert.Equal(t,
		[]string{
			"channel", ledgerid,
			"transaction_type", "unknown",
			"chaincode", "unknown",
			"validation_code", peer.TxValidationCode_INVALID_OTHER_REASON.String(),
		},
		testMetricProvider.fakeTransactionsCount.WithArgsForCall(2),
	)
	assert.Equal(t,
		float64(1),
		testMetricProvider.fakeTransactionsCount.AddArgsForCall(2),
	)
}

type testMetricProvider struct {
	fakeProvider                   *metricsfakes.Provider
	fakeBlockchainHeightGauge      *metricsfakes.Gauge
	fakeBlockProcessingTimeHist    *metricsfakes.Histogram
	fakeBlockstorageCommitTimeHist *metricsfakes.Histogram
	fakeStatedbCommitTimeHist      *metricsfakes.Histogram
	fakeTransactionsCount          *metricsfakes.Counter
}

func testutilConstructMetricProvider() *testMetricProvider {
	fakeProvider := &metricsfakes.Provider{}
	fakeBlockchainHeightGauge := testutilConstructGuage()
	fakeBlockProcessingTimeHist := testutilConstructHist()
	fakeBlockstorageCommitTimeHist := testutilConstructHist()
	fakeStatedbCommitTimeHist := testutilConstructHist()
	fakeTransactionsCount := testutilConstructCounter()
	fakeProvider.NewGaugeStub = func(opts metrics.GaugeOpts) metrics.Gauge {
		switch opts.Name {
		case blockchainHeightOpts.Name:
			return fakeBlockchainHeightGauge
		}
		return nil
	}
	fakeProvider.NewHistogramStub = func(opts metrics.HistogramOpts) metrics.Histogram {
		switch opts.Name {
		case blockProcessingTimeOpts.Name:
			return fakeBlockProcessingTimeHist
		case blockstorageCommitTimeOpts.Name:
			return fakeBlockstorageCommitTimeHist
		case statedbCommitTimeOpts.Name:
			return fakeStatedbCommitTimeHist
		}
		return nil
	}

	fakeProvider.NewCounterStub = func(opts metrics.CounterOpts) metrics.Counter {
		switch opts.Name {
		case transactionCountOpts.Name:
			return fakeTransactionsCount
		}
		return nil
	}
	return &testMetricProvider{
		fakeProvider,
		fakeBlockchainHeightGauge,
		fakeBlockProcessingTimeHist,
		fakeBlockstorageCommitTimeHist,
		fakeStatedbCommitTimeHist,
		fakeTransactionsCount,
	}
}

func testutilConstructGuage() *metricsfakes.Gauge {
	fakeGauge := &metricsfakes.Gauge{}
	fakeGauge.WithStub = func(lableValues ...string) metrics.Gauge {
		return fakeGauge
	}
	return fakeGauge
}

func testutilConstructHist() *metricsfakes.Histogram {
	fakeHist := &metricsfakes.Histogram{}
	fakeHist.WithStub = func(lableValues ...string) metrics.Histogram {
		return fakeHist
	}
	return fakeHist
}

func testutilConstructCounter() *metricsfakes.Counter {
	fakeCounter := &metricsfakes.Counter{}
	fakeCounter.WithStub = func(lableValues ...string) metrics.Counter {
		return fakeCounter
	}
	return fakeCounter
}
