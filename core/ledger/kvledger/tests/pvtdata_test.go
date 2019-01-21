
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
)

func TestMissingCollConfig(t *testing.T) {
	env := newEnv(defaultConfig, t)
	defer env.cleanup()
	h := newTestHelperCreateLgr("ledger1", t)

	collConf := []*collConf{{name: "coll1", btl: 5}}

//部署不带coll配置的cc1
	h.simulateDeployTx("cc1", nil)
	h.cutBlockAndCommitWithPvtdata()

//由于未定义收集配置，因此pvt数据操作应给出错误。
	h.simulateDataTx("", func(s *simulator) {
		h.assertError(s.GetPrivateData("cc1", "coll1", "key"))
		h.assertError(s.SetPrivateData("cc1", "coll1", "key", []byte("value")))
		h.assertError(s.DeletePrivateData("cc1", "coll1", "key"))
	})

//升级cc1（添加collconf）
	h.simulateUpgradeTx("cc1", collConf)
	h.cutBlockAndCommitWithPvtdata()

//对coll1的操作不应产生错误
//对coll2的操作应该产生错误（因为collconf中只定义了coll1）
	h.simulateDataTx("", func(s *simulator) {
		h.assertNoError(s.GetPrivateData("cc1", "coll1", "key1"))
		h.assertNoError(s.SetPrivateData("cc1", "coll1", "key2", []byte("value")))
		h.assertNoError(s.DeletePrivateData("cc1", "coll1", "key3"))
		h.assertError(s.GetPrivateData("cc1", "coll2", "key"))
		h.assertError(s.SetPrivateData("cc1", "coll2", "key", []byte("value")))
		h.assertError(s.DeletePrivateData("cc1", "coll2", "key"))
	})
}

func TestTxWithMissingPvtdata(t *testing.T) {
	env := newEnv(defaultConfig, t)
	defer env.cleanup()
	h := newTestHelperCreateLgr("ledger1", t)

	collConf := []*collConf{{name: "coll1", btl: 5}}

//使用“collconf”部署cc1
	h.simulateDeployTx("cc1", collConf)
	h.cutBlockAndCommitWithPvtdata()

//pvtdata模拟
	h.simulateDataTx("", func(s *simulator) {
		s.setPvtdata("cc1", "coll1", "key1", "value1")
	})
//另一个pvtdata模拟
	h.simulateDataTx("", func(s *simulator) {
		s.setPvtdata("cc1", "coll1", "key2", "value2")
	})
h.simulatedTrans[0].Pvtws = nil //从第一个模拟中删除pvt写入集
	blk2 := h.cutBlockAndCommitWithPvtdata()

h.verifyPvtState("cc1", "coll1", "key2", "value2") //键2应该已提交
	h.simulateDataTx("", func(s *simulator) {
h.assertError(s.GetPrivateData("cc1", "coll1", "key1")) //对于哈希版本，key1将过时
	})

//另一个数据Tx重写键1
	h.simulateDataTx("", func(s *simulator) {
		s.setPvtdata("cc1", "coll1", "key1", "newvalue1")
	})
	blk3 := h.cutBlockAndCommitWithPvtdata()
h.verifyPvtState("cc1", "coll1", "key1", "newvalue1") //应使用新值提交key1
	h.verifyBlockAndPvtDataSameAs(2, blk2)
	h.verifyBlockAndPvtDataSameAs(3, blk3)
}

func TestTxWithWrongPvtdata(t *testing.T) {
	env := newEnv(defaultConfig, t)
	defer env.cleanup()
	h := newTestHelperCreateLgr("ledger1", t)

	collConf := []*collConf{{name: "coll1", btl: 5}}

//使用“collconf”部署cc1
	h.simulateDeployTx("cc1", collConf)
	h.cutBlockAndCommitWithPvtdata()

//pvtdata模拟
	h.simulateDataTx("", func(s *simulator) {
		s.setPvtdata("cc1", "coll1", "key1", "value1")
	})
//另一个pvtdata模拟
	h.simulateDataTx("", func(s *simulator) {
		s.setPvtdata("cc1", "coll1", "key2", "value2")
	})
h.simulatedTrans[0].Pvtws = h.simulatedTrans[1].Pvtws //在第一个模拟中放入错误的pvt writeset
//如果块中存在的集合哈希与pvtdata不匹配，则拒绝提交块。
	h.cutBlockAndCommitExpectError()
	h.verifyPvtState("cc1", "coll1", "key2", "")
}

func TestBTL(t *testing.T) {
	env := newEnv(defaultConfig, t)
	defer env.cleanup()
	h := newTestHelperCreateLgr("ledger1", t)
	collConf := []*collConf{{name: "coll1", btl: 0}, {name: "coll2", btl: 5}}

//使用“collconf”部署cc1
	h.simulateDeployTx("cc1", collConf)
	h.cutBlockAndCommitWithPvtdata()

//在块2中提交pvtdata写入。
	h.simulateDataTx("", func(s *simulator) {
s.setPvtdata("cc1", "coll1", "key1", "value1") //（键1永不过期）
s.setPvtdata("cc1", "coll2", "key2", "value2") //（键2将在块8到期）
	})
	blk2 := h.cutBlockAndCommitWithPvtdata()

//用一些随机键/val再提交5个块
	for i := 0; i < 5; i++ {
		h.simulateDataTx("", func(s *simulator) {
			s.setPvtdata("cc1", "coll1", "someOtherKey", "someOtherVal")
			s.setPvtdata("cc1", "coll2", "someOtherKey", "someOtherVal")
		})
		h.cutBlockAndCommitWithPvtdata()
	}

//提交块7后
h.verifyPvtState("cc1", "coll1", "key1", "value1") //键1仍应存在于状态中
h.verifyPvtState("cc1", "coll2", "key2", "value2") //键2应该仍然存在于状态中
h.verifyBlockAndPvtDataSameAs(2, blk2)             //key1和key2应该仍然存在于pvtdata存储中

//使用一些随机键/VAL提交块8
	h.simulateDataTx("", func(s *simulator) {
		s.setPvtdata("cc1", "coll1", "someOtherKey", "someOtherVal")
		s.setPvtdata("cc1", "coll2", "someOtherKey", "someOtherVal")
	})
	h.cutBlockAndCommitWithPvtdata()

//提交第8块后
h.verifyPvtState("cc1", "coll1", "key1", "value1")                  //键1仍应存在于状态中
h.verifyPvtState("cc1", "coll2", "key2", "")                        //键2应该已从状态中清除
h.verifyBlockAndPvtData(2, nil, func(r *retrievedBlockAndPvtdata) { //从pvtdata存储中检索块2的pvtdata
r.pvtdataShouldContain(0, "cc1", "coll1", "key1", "value1") //key1应该仍然存在于pvtdata存储中
r.pvtdataShouldNotContain("cc1", "coll2")                   //<cc1，coll2>应该已从pvtdata存储中清除。
	})
}
