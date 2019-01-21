
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
	"bytes"
	"encoding/gob"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	protopeer "github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

type submittedData map[string]*submittedLedgerData

type submittedLedgerData struct {
	Blocks []*ledger.BlockAndPvtData
	Txs    []*txAndPvtdata
}

func (s submittedData) initForLedger(lgrid string) {
	ld := s[lgrid]
	if ld == nil {
		ld = &submittedLedgerData{}
		s[lgrid] = ld
	}
}

func (s submittedData) recordSubmittedBlks(lgrid string, blk ...*ledger.BlockAndPvtData) {
	s.initForLedger(lgrid)
	s[lgrid].Blocks = append(s[lgrid].Blocks, blk...)
}

func (s submittedData) recordSubmittedTxs(lgrid string, tx ...*txAndPvtdata) {
	s.initForLedger(lgrid)
	s[lgrid].Txs = append(s[lgrid].Txs, tx...)
}

type sampleDataHelper struct {
	submittedData submittedData
	assert        *assert.Assertions
	t             *testing.T
}

func newSampleDataHelper(t *testing.T) *sampleDataHelper {
	return &sampleDataHelper{make(submittedData), assert.New(t), t}
}

func (d *sampleDataHelper) populateLedger(h *testhelper) {
	lgrid := h.lgrid
//BLK1部署2个链码
	txdeploy1 := h.simulateDeployTx("cc1", nil)
	txdeploy2 := h.simulateDeployTx("cc2", nil)
	blk1 := h.cutBlockAndCommitWithPvtdata()

//BLK2包含2个公共数据TXS
	txdata1 := h.simulateDataTx("txid1", func(s *simulator) {
		s.setState("cc1", "key1", d.sampleVal("value01", lgrid))
		s.setState("cc1", "key2", d.sampleVal("value02", lgrid))
	})

	txdata2 := h.simulateDataTx("txid2", func(s *simulator) {
		s.setState("cc2", "key1", d.sampleVal("value03", lgrid))
		s.setState("cc2", "key2", d.sampleVal("value04", lgrid))
	})
	blk2 := h.cutBlockAndCommitWithPvtdata()

//BLK3升级两个链码
	txupgrade1 := h.simulateUpgradeTx("cc1", d.sampleCollConf1(lgrid, "cc1"))
	txupgrade2 := h.simulateUpgradeTx("cc2", d.sampleCollConf1(lgrid, "cc2"))
	blk3 := h.cutBlockAndCommitWithPvtdata()

//BLK4包含2个带有私有数据的数据TXS
	txdata3 := h.simulateDataTx("txid3", func(s *simulator) {
		s.setPvtdata("cc1", "coll1", "key3", d.sampleVal("value05", lgrid))
		s.setPvtdata("cc1", "coll1", "key4", d.sampleVal("value06", lgrid))
	})
	txdata4 := h.simulateDataTx("txid4", func(s *simulator) {
		s.setPvtdata("cc2", "coll1", "key3", d.sampleVal("value07", lgrid))
		s.setPvtdata("cc2", "coll1", "key4", d.sampleVal("value08", lgrid))
	})
	blk4 := h.cutBlockAndCommitWithPvtdata()

//BLK5升级两个链码
	txupgrade3 := h.simulateUpgradeTx("cc1", d.sampleCollConf2(lgrid, "cc1"))
	txupgrade4 := h.simulateDeployTx("cc2", d.sampleCollConf2(lgrid, "cc2"))
	blk5 := h.cutBlockAndCommitWithPvtdata()

//BLK6包含2个带有私有数据的数据TXS
	txdata5 := h.simulateDataTx("txid5", func(s *simulator) {
		s.setPvtdata("cc1", "coll2", "key3", d.sampleVal("value09", lgrid))
		s.setPvtdata("cc1", "coll2", "key4", d.sampleVal("value10", lgrid))
	})
	txdata6 := h.simulateDataTx("txid6", func(s *simulator) {
		s.setPvtdata("cc2", "coll2", "key3", d.sampleVal("value11", lgrid))
		s.setPvtdata("cc2", "coll2", "key4", d.sampleVal("value12", lgrid))
	})
	blk6 := h.cutBlockAndCommitWithPvtdata()

//BLK7包含一个数据TXS
	txdata7 := h.simulateDataTx("txid7", func(s *simulator) {
		s.setState("cc1", "key1", d.sampleVal("value13", lgrid))
		s.DeleteState("cc1", "key2")
		s.setPvtdata("cc1", "coll1", "key3", d.sampleVal("value14", lgrid))
		s.DeletePrivateData("cc1", "coll1", "key4")
	})
	h.simulatedTrans = nil

//BLK8包含一个应标记为无效的数据Tx，因为MVCC与BLK7中的Tx冲突
	txdata8 := h.simulateDataTx("txid8", func(s *simulator) {
		s.getState("cc1", "key1")
		s.setState("cc1", "key1", d.sampleVal("value15", lgrid))
	})
	blk7 := h.committer.cutBlockAndCommitWithPvtdata(txdata7)
	blk8 := h.cutBlockAndCommitWithPvtdata()

	d.submittedData.recordSubmittedBlks(lgrid,
		blk1, blk2, blk3, blk4, blk5, blk6, blk7, blk8)
	d.submittedData.recordSubmittedTxs(lgrid,
		txdeploy1, txdeploy2, txdata1, txdata2, txupgrade1, txupgrade2,
		txdata3, txdata4, txupgrade3, txupgrade4, txdata5, txdata6, txdata7, txdata8)
}

func (d *sampleDataHelper) serilizeSubmittedData() []byte {
	gob.Register(submittedData{})
	b := bytes.Buffer{}
	encoder := gob.NewEncoder(&b)
	d.assert.NoError(encoder.Encode(d.submittedData))
	by := b.Bytes()
	d.t.Logf("Serialized submitted data to bytes of len [%d]", len(by))
	return by
}

func (d *sampleDataHelper) loadSubmittedData(b []byte) {
	gob.Register(submittedData{})
	sd := make(submittedData)
	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)
	d.assert.NoError(decoder.Decode(&sd))
	d.t.Logf("Deserialized submitted data from bytes of len [%d], submitted data = %#v", len(b), sd)
	d.submittedData = sd
}

func (d *sampleDataHelper) verifyLedgerContent(h *testhelper) {
	d.verifyState(h)
	d.verifyConfigHistory(h)
	d.verifyBlockAndPvtdata(h)
	d.verifyGetTransactionByID(h)

//如果在新运行中从磁盘加载测试分类帐，则提交的数据不可用。
//(e.g., a backup of a test lesger from a previous fabric version)
	if len(d.submittedData) != 0 {
		d.t.Log("Verifying using submitted data")
		d.verifyBlockAndPvtdataUsingSubmittedData(h)
		d.verifyGetTransactionByIDUsingSubmittedData(h)
	} else {
		d.t.Log("Skipping verifying using submitted data")
	}
}

func (d *sampleDataHelper) verifyState(h *testhelper) {
	lgrid := h.lgrid
	h.verifyPubState("cc1", "key1", d.sampleVal("value13", lgrid))
	h.verifyPubState("cc1", "key2", "")
	h.verifyPvtState("cc1", "coll1", "key3", d.sampleVal("value14", lgrid))
	h.verifyPvtState("cc1", "coll1", "key4", "")
	h.verifyPvtState("cc1", "coll2", "key3", d.sampleVal("value09", lgrid))
	h.verifyPvtState("cc1", "coll2", "key4", d.sampleVal("value10", lgrid))

	h.verifyPubState("cc2", "key1", d.sampleVal("value03", lgrid))
	h.verifyPubState("cc2", "key2", d.sampleVal("value04", lgrid))
	h.verifyPvtState("cc2", "coll1", "key3", d.sampleVal("value07", lgrid))
	h.verifyPvtState("cc2", "coll1", "key4", d.sampleVal("value08", lgrid))
	h.verifyPvtState("cc2", "coll2", "key3", d.sampleVal("value11", lgrid))
	h.verifyPvtState("cc2", "coll2", "key4", d.sampleVal("value12", lgrid))
}

func (d *sampleDataHelper) verifyConfigHistory(h *testhelper) {
	lgrid := h.lgrid
	h.verifyMostRecentCollectionConfigBelow(10, "cc1",
		&expectedCollConfInfo{5, d.sampleCollConf2(lgrid, "cc1")})

	h.verifyMostRecentCollectionConfigBelow(5, "cc1",
		&expectedCollConfInfo{3, d.sampleCollConf1(lgrid, "cc1")})

	h.verifyMostRecentCollectionConfigBelow(10, "cc2",
		&expectedCollConfInfo{5, d.sampleCollConf2(lgrid, "cc2")})

	h.verifyMostRecentCollectionConfigBelow(5, "cc2",
		&expectedCollConfInfo{3, d.sampleCollConf1(lgrid, "cc2")})
}

func (d *sampleDataHelper) verifyBlockAndPvtdata(h *testhelper) {
	lgrid := h.lgrid
	h.verifyBlockAndPvtData(2, nil, func(r *retrievedBlockAndPvtdata) {
		r.hasNumTx(2)
		r.hasNoPvtdata()
	})

	h.verifyBlockAndPvtData(4, nil, func(r *retrievedBlockAndPvtdata) {
		r.hasNumTx(2)
		r.pvtdataShouldContain(0, "cc1", "coll1", "key3", d.sampleVal("value05", lgrid))
		r.pvtdataShouldContain(1, "cc2", "coll1", "key3", d.sampleVal("value07", lgrid))
	})
}

func (d *sampleDataHelper) verifyGetTransactionByID(h *testhelper) {
	h.verifyTxValidationCode("txid7", protopeer.TxValidationCode_VALID)
	h.verifyTxValidationCode("txid8", protopeer.TxValidationCode_MVCC_READ_CONFLICT)
}

func (d *sampleDataHelper) verifyBlockAndPvtdataUsingSubmittedData(h *testhelper) {
	lgrid := h.lgrid
	submittedData := d.submittedData[lgrid]
	for _, submittedBlk := range submittedData.Blocks {
		blkNum := submittedBlk.Block.Header.Number
		if blkNum != 8 {
			h.verifyBlockAndPvtDataSameAs(uint64(blkNum), submittedBlk)
		} else {
			h.verifyBlockAndPvtData(uint64(8), nil, func(r *retrievedBlockAndPvtdata) {
				r.sameBlockHeaderAndData(submittedBlk.Block)
				r.containsValidationCode(0, protopeer.TxValidationCode_MVCC_READ_CONFLICT)
			})
		}
	}
}

func (d *sampleDataHelper) verifyGetTransactionByIDUsingSubmittedData(h *testhelper) {
	lgrid := h.lgrid
	for _, submittedTx := range d.submittedData[lgrid].Txs {
		expectedValidationCode := protopeer.TxValidationCode_VALID
		if submittedTx.Txid == "txid8" {
			expectedValidationCode = protopeer.TxValidationCode_MVCC_READ_CONFLICT
		}
		h.verifyGetTransactionByID(submittedTx.Txid,
			&protopeer.ProcessedTransaction{TransactionEnvelope: submittedTx.Envelope, ValidationCode: int32(expectedValidationCode)})
	}
}

func (d *sampleDataHelper) sampleVal(val, ledgerid string) string {
	return fmt.Sprintf("%s:%s", val, ledgerid)
}

func (d *sampleDataHelper) sampleCollConf1(ledgerid, ccName string) []*collConf {
	return []*collConf{
		{name: "coll1", members: []string{"org1", "org2"}},
		{name: ledgerid, members: []string{"org1", "org2"}},
		{name: ccName, members: []string{"org1", "org2"}},
	}
}

func (d *sampleDataHelper) sampleCollConf2(ledgerid string, ccName string) []*collConf {
	return []*collConf{
		{name: "coll1", members: []string{"org1", "org2"}},
		{name: "coll2", members: []string{"org1", "org2"}},
		{name: ledgerid, members: []string{"org1", "org2"}},
		{name: ccName, members: []string{"org1", "org2"}},
	}
}
