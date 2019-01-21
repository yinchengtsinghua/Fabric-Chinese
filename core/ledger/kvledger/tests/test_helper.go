
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

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/stretchr/testify/assert"
)

//testhhelper嵌入（1）一个客户机，（2）一个提交者和（3）一个验证器，这三个都在
//一个分类帐实例，并在分类帐API上添加帮助/可恢复功能，有助于
//避免在实际测试代码中重复。
//“客户机”增加了模拟关系API的价值，“提交者”有助于减少
//下一个块并提交块，最后，验证器帮助实现
//分类帐API根据提交的块返回正确的值
type testhelper struct {
	*client
	*committer
	*verifier
	lgr    ledger.PeerLedger
	lgrid  string
	assert *assert.Assertions
}

//newtestHelperCreateLgr创建一个新的分类帐，并为分类帐重试“testHelper”
func newTestHelperCreateLgr(id string, t *testing.T) *testhelper {
	genesisBlk, err := constructTestGenesisBlock(id)
	assert.NoError(t, err)
	lgr, err := ledgermgmt.CreateLedger(genesisBlk)
	assert.NoError(t, err)
	client, committer, verifier := newClient(lgr, t), newCommitter(lgr, t), newVerifier(lgr, t)
	return &testhelper{client, committer, verifier, lgr, id, assert.New(t)}
}

//NealTestLePulOpenLGR打开一个现有的分类帐，并为分类账重写一个“TestelPar”。
func newTestHelperOpenLgr(id string, t *testing.T) *testhelper {
	lgr, err := ledgermgmt.OpenLedger(id)
	assert.NoError(t, err)
	client, committer, verifier := newClient(lgr, t), newCommitter(lgr, t), newVerifier(lgr, t)
	return &testhelper{client, committer, verifier, lgr, id, assert.New(t)}
}

//CutBlockAndCommitWithPvtData收集测试代码（通过调用
//“客户机”中可用的函数，并剪切下一个块并提交到分类帐
func (h *testhelper) cutBlockAndCommitWithPvtdata() *ledger.BlockAndPvtData {
	defer func() { h.simulatedTrans = nil }()
	return h.committer.cutBlockAndCommitWithPvtdata(h.simulatedTrans...)
}

func (h *testhelper) cutBlockAndCommitExpectError() (*ledger.BlockAndPvtData, error) {
	defer func() { h.simulatedTrans = nil }()
	return h.committer.cutBlockAndCommitExpectError(h.simulatedTrans...)
}

//assertError是一个辅助函数，可以作为assertError（f（））调用，其中“f”是其他函数
//此函数假定函数“f”的最后一个返回类型为“error”
func (h *testhelper) assertError(output ...interface{}) {
	lastParam := output[len(output)-1]
	assert.NotNil(h.t, lastParam)
	h.assert.Error(lastParam.(error))
}

//assertNoError请参见函数“assertError”的注释
func (h *testhelper) assertNoError(output ...interface{}) {
	h.assert.Nil(output[len(output)-1])
}
