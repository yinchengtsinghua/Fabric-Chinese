
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


package tests

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger"
	lgrutil "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	protopeer "github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

//Verifier提供的函数可以帮助测试查询分类账时使用的详细代码更少。
//用预期结果验证实际结果
//对于直进函数，测试可以直接在分类帐上调用它们。
type verifier struct {
	lgr    ledger.PeerLedger
	assert *assert.Assertions
	t      *testing.T
}

func newVerifier(lgr ledger.PeerLedger, t *testing.T) *verifier {
	return &verifier{lgr, assert.New(t), t}
}

func (v *verifier) verifyLedgerHeight(expectedHt uint64) {
	info, err := v.lgr.GetBlockchainInfo()
	v.assert.NoError(err)
	v.assert.Equal(expectedHt, info.Height)
}

func (v *verifier) verifyPubState(ns, key string, expectedVal string) {
	qe, err := v.lgr.NewQueryExecutor()
	v.assert.NoError(err)
	defer qe.Done()
	committedVal, err := qe.GetState(ns, key)
	v.assert.NoError(err)
	v.t.Logf("val=%s", committedVal)
	var expectedValBytes []byte
	if expectedVal != "" {
		expectedValBytes = []byte(expectedVal)
	}
	v.assert.Equal(expectedValBytes, committedVal)
}

func (v *verifier) verifyPvtState(ns, coll, key string, expectedVal string) {
	qe, err := v.lgr.NewQueryExecutor()
	v.assert.NoError(err)
	defer qe.Done()
	committedVal, err := qe.GetPrivateData(ns, coll, key)
	v.assert.NoError(err)
	v.t.Logf("val=%s", committedVal)
	var expectedValBytes []byte
	if expectedVal != "" {
		expectedValBytes = []byte(expectedVal)
	}
	v.assert.Equal(expectedValBytes, committedVal)
}

func (v *verifier) verifyMostRecentCollectionConfigBelow(blockNum uint64, chaincodeName string, expectOut *expectedCollConfInfo) {
	configHistory, err := v.lgr.GetConfigHistoryRetriever()
	v.assert.NoError(err)
	actualCollectionConfigInfo, err := configHistory.MostRecentCollectionConfigBelow(blockNum, chaincodeName)
	v.assert.NoError(err)
	if expectOut == nil {
		v.assert.Nil(actualCollectionConfigInfo)
		return
	}
	v.t.Logf("Retrieved CollectionConfigInfo=%s", spew.Sdump(actualCollectionConfigInfo))
	actualCommittingBlockNum := actualCollectionConfigInfo.CommittingBlockNum
	actualCollConf := convertFromCollConfigProto(actualCollectionConfigInfo.CollectionConfig)
	v.assert.Equal(expectOut.committingBlockNum, actualCommittingBlockNum)
	v.assert.Equal(expectOut.collConfs, actualCollConf)
}

func (v *verifier) verifyBlockAndPvtData(blockNum uint64, filter ledger.PvtNsCollFilter, verifyLogic func(r *retrievedBlockAndPvtdata)) {
	out, err := v.lgr.GetPvtDataAndBlockByNum(blockNum, filter)
	v.assert.NoError(err)
	v.t.Logf("Retrieved Block = %s, pvtdata = %s", spew.Sdump(out.Block), spew.Sdump(out.PvtData))
	verifyLogic(&retrievedBlockAndPvtdata{out, v.assert})
}

func (v *verifier) verifyBlockAndPvtDataSameAs(blockNum uint64, expectedOut *ledger.BlockAndPvtData) {
	v.verifyBlockAndPvtData(blockNum, nil, func(r *retrievedBlockAndPvtdata) {
		r.sameAs(expectedOut)
	})
}

func (v *verifier) verifyGetTransactionByID(txid string, expectedOut *protopeer.ProcessedTransaction) {
	tran, err := v.lgr.GetTransactionByID(txid)
	v.assert.NoError(err)
	envelopEqual := proto.Equal(expectedOut.TransactionEnvelope, tran.TransactionEnvelope)
	v.assert.True(envelopEqual)
	v.assert.Equal(expectedOut.ValidationCode, tran.ValidationCode)
}

func (v *verifier) verifyTxValidationCode(txid string, expectedCode protopeer.TxValidationCode) {
	tran, err := v.lgr.GetTransactionByID(txid)
	v.assert.NoError(err)
	v.assert.Equal(int32(expectedCode), tran.ValidationCode)
}

/////////////////////////////////////////////////////
type expectedCollConfInfo struct {
	committingBlockNum uint64
	collConfs          []*collConf
}

type retrievedBlockAndPvtdata struct {
	*ledger.BlockAndPvtData
	assert *assert.Assertions
}

func (r *retrievedBlockAndPvtdata) sameAs(expectedBlockAndPvtdata *ledger.BlockAndPvtData) {
	r.samePvtdata(expectedBlockAndPvtdata.PvtData)
	r.sameBlockHeaderAndData(expectedBlockAndPvtdata.Block)
	r.sameMetadata(expectedBlockAndPvtdata.Block)
}

func (r *retrievedBlockAndPvtdata) hasNumTx(numTx int) {
	r.assert.Len(r.Block.Data.Data, numTx)
}

func (r *retrievedBlockAndPvtdata) hasNoPvtdata() {
	r.assert.Len(r.PvtData, 0)
}

func (r *retrievedBlockAndPvtdata) pvtdataShouldContain(txSeq int, ns, coll, key, value string) {
	txPvtData := r.BlockAndPvtData.PvtData[uint64(txSeq)]
	for _, nsdata := range txPvtData.WriteSet.NsPvtRwset {
		if nsdata.Namespace == ns {
			for _, colldata := range nsdata.CollectionPvtRwset {
				if colldata.CollectionName == coll {
					rwset := &kvrwset.KVRWSet{}
					r.assert.NoError(proto.Unmarshal(colldata.Rwset, rwset))
					for _, w := range rwset.Writes {
						if w.Key == key {
							r.assert.Equal([]byte(value), w.Value)
							return
						}
					}
				}
			}
		}
	}
	r.assert.FailNow("Requested kv not found")
}

func (r *retrievedBlockAndPvtdata) pvtdataShouldNotContain(ns, coll string) {
	allTxPvtData := r.BlockAndPvtData.PvtData
	for _, txPvtData := range allTxPvtData {
		r.assert.False(txPvtData.Has(ns, coll))
	}
}

func (r *retrievedBlockAndPvtdata) sameBlockHeaderAndData(expectedBlock *common.Block) {
	r.assert.True(proto.Equal(expectedBlock.Data, r.BlockAndPvtData.Block.Data))
	r.assert.True(proto.Equal(expectedBlock.Header, r.BlockAndPvtData.Block.Header))
}

func (r *retrievedBlockAndPvtdata) sameMetadata(expectedBlock *common.Block) {
//编组/解组处理可以互换处理一个零字节和空字节（根据选择的方案是proto和gob）
//所以显式比较每个元数据
	retrievedMetadata := r.Block.Metadata.Metadata
	expectedMetadata := expectedBlock.Metadata.Metadata
	r.assert.Equal(len(expectedMetadata), len(retrievedMetadata))
	for i := 0; i < len(retrievedMetadata); i++ {
		if len(expectedMetadata[i])+len(retrievedMetadata[i]) != 0 {
			r.assert.Equal(expectedMetadata[i], retrievedMetadata[i])
		}
	}
}

func (r *retrievedBlockAndPvtdata) containsValidationCode(txSeq int, validationCode protopeer.TxValidationCode) {
	var txFilter lgrutil.TxValidationFlags
	txFilter = r.BlockAndPvtData.Block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER]
	r.assert.Equal(validationCode, txFilter.Flag(txSeq))
}

func (r *retrievedBlockAndPvtdata) samePvtdata(expectedPvtdata map[uint64]*ledger.TxPvtData) {
	r.assert.Equal(len(expectedPvtdata), len(r.BlockAndPvtData.PvtData))
	for txNum, pvtData := range expectedPvtdata {
		actualPvtData := r.BlockAndPvtData.PvtData[txNum]
		r.assert.Equal(pvtData.SeqInBlock, actualPvtData.SeqInBlock)
		r.assert.True(proto.Equal(pvtData.WriteSet, actualPvtData.WriteSet))
	}
}
