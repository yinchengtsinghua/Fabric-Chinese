
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

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/customtx"
	lgrutil "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

type customTxProcessor struct {
}

func (ctp *customTxProcessor) GenerateSimulationResults(txEnvelop *common.Envelope, simulator ledger.TxSimulator, initializingLedger bool) error {
	payload := utils.UnmarshalPayloadOrPanic(txEnvelop.Payload)
	chHdr, _ := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	chainid := chHdr.ChannelId
	kvw := &kvrwset.KVWrite{}
	if err := proto.Unmarshal(payload.Data, kvw); err != nil {
		return err
	}
	if len(kvw.Key) == 0 {
		return &customtx.InvalidTxError{Msg: "Nil key"}
	}
	return simulator.SetState(chainid, kvw.Key, kvw.Value)
}

func TestCustomProcessor(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	provider := testutilNewProvider(t)
	defer provider.Close()

//创建一个定制的Tx处理器并注册它来处理“100和101”类型的事务
	chainid := "testLedger"
	customTxProcessor := &customTxProcessor{}
	customtx.InitializeTestEnv(customtx.Processors{
		100: customTxProcessor,
		101: customTxProcessor})

//使用common.headerType_config事务创建Genesis块
	_, gb := testutil.NewBlockGenerator(t, chainid, false)
	lgr, err := provider.Create(gb)
	defer lgr.Close()
	assert.NoError(t, err)

//用三个自定义事务提交一个块
	tx1 := createCustomTx(t, 100, chainid, "custom_key1", "value1")
	tx2 := createCustomTx(t, 101, chainid, "custom_key2", "value2")
	tx3 := createCustomTx(t, 101, chainid, "", "")
	blk1 := testutil.NewBlock([]*common.Envelope{tx1, tx2, tx3}, 1, gb.Header.Hash())
	assert.NoError(t, lgr.CommitWithPvtData(&ledger.BlockAndPvtData{Block: blk1}))
//验证自定义处理器导致的状态更改是否在创建分类帐期间发生。
	qe, err := lgr.NewQueryExecutor()
	assert.NoError(t, err)
	val, err := qe.GetState(chainid, "custom_key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", string(val))

	val, err = qe.GetState(chainid, "custom_key2")
	assert.NoError(t, err)
	assert.Equal(t, "value2", string(val))
	qe.Done()

	blockPersisted, err := lgr.GetBlockByNumber(1)
	assert.NoError(t, err)
	var txFilter lgrutil.TxValidationFlags
	txFilter = blockPersisted.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER]
	assert.Equal(t, peer.TxValidationCode_VALID, txFilter.Flag(0))
	assert.Equal(t, peer.TxValidationCode_VALID, txFilter.Flag(1))
	assert.Equal(t, peer.TxValidationCode_INVALID_OTHER_REASON, txFilter.Flag(2))

	tx4 := createCustomTx(t, 100, chainid, "custom_key4", "value4")
	blk2 := testutil.NewBlock([]*common.Envelope{tx4}, 2, blk1.Header.Hash())
	assert.NoError(t, lgr.CommitWithPvtData(&ledger.BlockAndPvtData{Block: blk2}))
	qe, err = lgr.NewQueryExecutor()
	assert.NoError(t, err)
	val, err = qe.GetState(chainid, "custom_key4")
	qe.Done()
	assert.NoError(t, err)
	assert.Equal(t, "value4", string(val))
}

func createCustomTx(t *testing.T, txType common.HeaderType, chainid, key, val string) *common.Envelope {
	kvWrite := &kvrwset.KVWrite{Key: key, Value: []byte(val)}
	txEnv, err := utils.CreateSignedEnvelope(txType, chainid, nil, kvWrite, 0, 0)
	assert.NoError(t, err)
	return txEnv
}
