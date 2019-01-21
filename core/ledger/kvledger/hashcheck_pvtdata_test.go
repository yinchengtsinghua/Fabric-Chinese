
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
	"fmt"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/stretchr/testify/assert"
)

func TestConstructValidInvalidBlocksPvtData(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	provider := testutilNewProvider(t)
	defer provider.Close()

	_, gb := testutil.NewBlockGenerator(t, "testLedger", false)
	gbHash := gb.Header.Hash()
	lg, _ := provider.Create(gb)
	defer lg.Close()

//构建pvtdata和pubrwset（即哈希rw集）
	v0 := []byte{0}
	pvtDataBlk1Tx0, pubSimResBytesBlk1Tx0 := produceSamplePvtdata(t, 0, []string{"ns-1:coll-1", "ns-1:coll-2"}, [][]byte{v0, v0})
	v1 := []byte{1}
	pvtDataBlk1Tx1, pubSimResBytesBlk1Tx1 := produceSamplePvtdata(t, 1, []string{"ns-1:coll-1", "ns-1:coll-2"}, [][]byte{v1, v1})
	v2 := []byte{2}
	pvtDataBlk1Tx2, pubSimResBytesBlk1Tx2 := produceSamplePvtdata(t, 2, []string{"ns-1:coll-1", "ns-2:coll-2"}, [][]byte{v2, v2})
	v3 := []byte{3}
	pvtDataBlk1Tx3, pubSimResBytesBlk1Tx3 := produceSamplePvtdata(t, 3, []string{"ns-1:coll-1", "ns-1:coll-2"}, [][]byte{v3, v3})
	v4 := []byte{4}
	pvtDataBlk1Tx4, pubSimResBytesBlk1Tx4 := produceSamplePvtdata(t, 4, []string{"ns-1:coll-1", "ns-4:coll-2"}, [][]byte{v4, v4})
	v5 := []byte{5}
	pvtDataBlk1Tx5, pubSimResBytesBlk1Tx5 := produceSamplePvtdata(t, 5, []string{"ns-1:coll-1", "ns-1:coll-2"}, [][]byte{v5, v5})
	v6 := []byte{6}
	pvtDataBlk1Tx6, pubSimResBytesBlk1Tx6 := produceSamplePvtdata(t, 6, []string{"ns-6:coll-2"}, [][]byte{v6})
	v7 := []byte{7}
	_, pubSimResBytesBlk1Tx7 := produceSamplePvtdata(t, 7, []string{"ns-1:coll-2"}, [][]byte{v7})
	wrongPvtDataBlk1Tx7, _ := produceSamplePvtdata(t, 7, []string{"ns-6:coll-2"}, [][]byte{v6})

	pubSimulationResults := &rwset.TxReadWriteSet{}
	err := proto.Unmarshal(pubSimResBytesBlk1Tx7, pubSimulationResults)
	assert.NoError(t, err)
	tx7PvtdataHash := pubSimulationResults.NsRwset[0].CollectionHashedRwset[0].PvtRwsetHash

//构建块1
	simulationResultsBlk1 := [][]byte{pubSimResBytesBlk1Tx0, pubSimResBytesBlk1Tx1, pubSimResBytesBlk1Tx2,
		pubSimResBytesBlk1Tx3, pubSimResBytesBlk1Tx4, pubSimResBytesBlk1Tx5,
		pubSimResBytesBlk1Tx6, pubSimResBytesBlk1Tx7}
	blk1 := testutil.ConstructBlock(t, 1, gbHash, simulationResultsBlk1, false)

//构建块1的pvtdata列表
	pvtDataBlk1 := map[uint64]*ledger.TxPvtData{
		0: pvtDataBlk1Tx0,
		1: pvtDataBlk1Tx1,
		2: pvtDataBlk1Tx2,
		4: pvtDataBlk1Tx4,
		5: pvtDataBlk1Tx5,
	}

//构建块1的MissingData列表
	missingData := make(ledger.TxMissingPvtDataMap)
	missingData.Add(3, "ns-1", "coll-1", true)
	missingData.Add(3, "ns-1", "coll-2", true)
	missingData.Add(6, "ns-6", "coll-2", true)
	missingData.Add(7, "ns-1", "coll-2", true)

//提交块1
	blockAndPvtData1 := &ledger.BlockAndPvtData{
		Block:          blk1,
		PvtData:        pvtDataBlk1,
		MissingPvtData: missingData}
	assert.NoError(t, lg.(*kvLedger).blockStore.CommitWithPvtData(blockAndPvtData1))

//从tx3、tx6和tx7中丢失的数据构造pvtdata
	blocksPvtData := []*ledger.BlockPvtData{
		{
			BlockNum: 1,
			WriteSets: map[uint64]*ledger.TxPvtData{
				3: pvtDataBlk1Tx3,
				6: pvtDataBlk1Tx6,
				7: wrongPvtDataBlk1Tx7,
//NS-6:TX7中不存在coll-2
			},
		},
	}

	expectedValidBlocksPvtData := map[uint64][]*ledger.TxPvtData{
		1: {
			pvtDataBlk1Tx3,
			pvtDataBlk1Tx6,
		},
	}

	blocksValidPvtData, hashMismatched, err := ConstructValidAndInvalidPvtData(blocksPvtData, lg.(*kvLedger).blockStore)
	assert.NoError(t, err)
	assert.Equal(t, len(expectedValidBlocksPvtData), len(blocksValidPvtData))
	assert.ElementsMatch(t, expectedValidBlocksPvtData[1], blocksValidPvtData[1])
//不应包括为tx7传递的pvtdata，即使在哈希不匹配的情况下，因为ns-6中不存在：coll-2
	assert.Len(t, hashMismatched, 0)

//用错误的pvtdata从tx7中丢失的数据构造pvtdata
	wrongPvtDataBlk1Tx7, pubSimResBytesBlk1Tx7 = produceSamplePvtdata(t, 7, []string{"ns-1:coll-2"}, [][]byte{v6})
	blocksPvtData = []*ledger.BlockPvtData{
		{
			BlockNum: 1,
			WriteSets: map[uint64]*ledger.TxPvtData{
				7: wrongPvtDataBlk1Tx7,
//NS-1:TX7中存在coll-1，但传递的pvtdata不正确
			},
		},
	}

	expectedHashMismatches := []*ledger.PvtdataHashMismatch{
		{
			BlockNum:     1,
			TxNum:        7,
			Namespace:    "ns-1",
			Collection:   "coll-2",
			ExpectedHash: tx7PvtdataHash,
		},
	}

	blocksValidPvtData, hashMismatches, err := ConstructValidAndInvalidPvtData(blocksPvtData, lg.(*kvLedger).blockStore)
	assert.NoError(t, err)
	assert.Len(t, blocksValidPvtData, 0)

	assert.ElementsMatch(t, expectedHashMismatches, hashMismatches)
}

func produceSamplePvtdata(t *testing.T, txNum uint64, nsColls []string, values [][]byte) (*ledger.TxPvtData, []byte) {
	builder := rwsetutil.NewRWSetBuilder()
	for index, nsColl := range nsColls {
		nsCollSplit := strings.Split(nsColl, ":")
		ns := nsCollSplit[0]
		coll := nsCollSplit[1]
		key := fmt.Sprintf("key-%s-%s", ns, coll)
		value := values[index]
		builder.AddToPvtAndHashedWriteSet(ns, coll, key, value)
	}
	simRes, err := builder.GetTxSimulationResults()
	assert.NoError(t, err)
	pubSimulationResultsBytes, err := proto.Marshal(simRes.PubSimulationResults)
	assert.NoError(t, err)
	return &ledger.TxPvtData{SeqInBlock: txNum, WriteSet: simRes.PvtSimulationResults}, pubSimulationResultsBytes
}
