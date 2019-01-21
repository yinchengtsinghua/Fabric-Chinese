
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
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/testutil"
	protopeer "github.com/hyperledger/fabric/protos/peer"
)

//testv11测试由v1.1创建的ledgersdata文件夹是否可以以向后兼容的方式与将来的版本一起使用
func TestV11(t *testing.T) {
	fsPath := defaultConfig["peer.fileSystemPath"].(string)
//此测试数据由v1.1代码https://gerrit.hyperledger.org/r//c/22749/6/core/ledger/kvledger/tests/v11_generate_test.go@22生成。
	testutil.CopyDir("testdata/v11/sample_ledgers/ledgersData", fsPath)
	env := newEnv(defaultConfig, t)
	defer env.cleanup()

	h1, h2 := newTestHelperOpenLgr("ledger1", t), newTestHelperOpenLgr("ledger2", t)
	dataHelper := &v11SampleDataHelper{}

	dataHelper.verifyBeforeStateRebuild(h1)
	dataHelper.verifyBeforeStateRebuild(h2)

	env.closeAllLedgersAndDrop(rebuildableStatedb)
	h1, h2 = newTestHelperOpenLgr("ledger1", t), newTestHelperOpenLgr("ledger2", t)
	dataHelper.verifyAfterStateRebuild(h1)
	dataHelper.verifyAfterStateRebuild(h2)

	env.closeAllLedgersAndDrop(rebuildableStatedb + rebuildableBlockIndex + rebuildableConfigHistory)
	h1, h2 = newTestHelperOpenLgr("ledger1", t), newTestHelperOpenLgr("ledger2", t)
	dataHelper.verifyAfterStateRebuild(h1)
	dataHelper.verifyAfterStateRebuild(h2)
}

//v11示例数据帮助程序提供了一组用于验证分类帐的功能。这将在假设
//分类账是由v1.1中的代码生成的（https://gerrit.hyperledger.org/r//c/22749/1/core/ledger/kvledger/tests/v11_generate_test.go@22）。
//总之，上面的生成函数构造了两个分类账并填充分类账使用了此代码
//（https://gerrit.hyperledger.org/r//c/22749/1/core/ledger/kvledger/tests/utilu sample_data.go@55）
type v11SampleDataHelper struct {
}

func (d *v11SampleDataHelper) verifyBeforeStateRebuild(h *testhelper) {
	dataHelper := &v11SampleDataHelper{}
	dataHelper.verifyState(h)
	dataHelper.verifyBlockAndPvtdata(h)
	dataHelper.verifyGetTransactionByID(h)
	dataHelper.verifyConfigHistoryDoesNotExist(h)
}

func (d *v11SampleDataHelper) verifyAfterStateRebuild(h *testhelper) {
	dataHelper := &v11SampleDataHelper{}
	dataHelper.verifyState(h)
	dataHelper.verifyBlockAndPvtdata(h)
	dataHelper.verifyGetTransactionByID(h)
	dataHelper.verifyConfigHistory(h)
}

func (d *v11SampleDataHelper) verifyState(h *testhelper) {
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

func (d *v11SampleDataHelper) verifyConfigHistory(h *testhelper) {
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

func (d *v11SampleDataHelper) verifyConfigHistoryDoesNotExist(h *testhelper) {
	h.verifyMostRecentCollectionConfigBelow(10, "cc1", nil)
	h.verifyMostRecentCollectionConfigBelow(10, "cc2", nil)
}

func (d *v11SampleDataHelper) verifyBlockAndPvtdata(h *testhelper) {
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

func (d *v11SampleDataHelper) verifyGetTransactionByID(h *testhelper) {
	h.verifyTxValidationCode("txid7", protopeer.TxValidationCode_VALID)
	h.verifyTxValidationCode("txid8", protopeer.TxValidationCode_MVCC_READ_CONFLICT)
}

func (d *v11SampleDataHelper) sampleVal(val, ledgerid string) string {
	return fmt.Sprintf("%s:%s", val, ledgerid)
}

func (d *v11SampleDataHelper) sampleCollConf1(ledgerid, ccName string) []*collConf {
	return []*collConf{
		{name: "coll1", members: []string{"org1", "org2"}},
		{name: ledgerid, members: []string{"org1", "org2"}},
		{name: ccName, members: []string{"org1", "org2"}},
	}
}

func (d *v11SampleDataHelper) sampleCollConf2(ledgerid string, ccName string) []*collConf {
	return []*collConf{
		{name: "coll1", members: []string{"org1", "org2"}},
		{name: "coll2", members: []string{"org1", "org2"}},
		{name: ledgerid, members: []string{"org1", "org2"}},
		{name: ccName, members: []string{"org1", "org2"}},
	}
}
