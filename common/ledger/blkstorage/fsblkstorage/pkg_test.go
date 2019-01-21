
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
*/


package fsblkstorage

import (
	"io/ioutil"
	"math"
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flogging.ActivateSpec("fsblkstorage=debug")
	os.Exit(m.Run())
}

func testPath() string {
	if path, err := ioutil.TempDir("", "fsblkstorage-"); err != nil {
		panic(err)
	} else {
		return path
	}
}

type testEnv struct {
	t        testing.TB
	provider *FsBlockstoreProvider
}

func newTestEnv(t testing.TB, conf *Conf) *testEnv {
	attrsToIndex := []blkstorage.IndexableAttr{
		blkstorage.IndexableAttrBlockHash,
		blkstorage.IndexableAttrBlockNum,
		blkstorage.IndexableAttrTxID,
		blkstorage.IndexableAttrBlockNumTranNum,
		blkstorage.IndexableAttrBlockTxID,
		blkstorage.IndexableAttrTxValidationCode,
	}
	return newTestEnvSelectiveIndexing(t, conf, attrsToIndex)
}

func newTestEnvSelectiveIndexing(t testing.TB, conf *Conf, attrsToIndex []blkstorage.IndexableAttr) *testEnv {
	indexConfig := &blkstorage.IndexConfig{AttrsToIndex: attrsToIndex}
	return &testEnv{t, NewProvider(conf, indexConfig).(*FsBlockstoreProvider)}
}

func (env *testEnv) Cleanup() {
	env.provider.Close()
	env.removeFSPath()
}

func (env *testEnv) removeFSPath() {
	fsPath := env.provider.conf.blockStorageDir
	os.RemoveAll(fsPath)
}

type testBlockfileMgrWrapper struct {
	t            testing.TB
	blockfileMgr *blockfileMgr
}

func newTestBlockfileWrapper(env *testEnv, ledgerid string) *testBlockfileMgrWrapper {
	blkStore, err := env.provider.OpenBlockStore(ledgerid)
	assert.NoError(env.t, err)
	return &testBlockfileMgrWrapper{env.t, blkStore.(*fsBlockStore).fileMgr}
}

func (w *testBlockfileMgrWrapper) addBlocks(blocks []*common.Block) {
	for _, blk := range blocks {
		err := w.blockfileMgr.addBlock(blk)
		assert.NoError(w.t, err, "Error while adding block to blockfileMgr")
	}
}

func (w *testBlockfileMgrWrapper) testGetBlockByHash(blocks []*common.Block) {
	for i, block := range blocks {
		hash := block.Header.Hash()
		b, err := w.blockfileMgr.retrieveBlockByHash(hash)
		assert.NoError(w.t, err, "Error while retrieving [%d]th block from blockfileMgr", i)
		assert.Equal(w.t, block, b)
	}
}

func (w *testBlockfileMgrWrapper) testGetBlockByNumber(blocks []*common.Block, startingNum uint64) {
	for i := 0; i < len(blocks); i++ {
		b, err := w.blockfileMgr.retrieveBlockByNumber(startingNum + uint64(i))
		assert.NoError(w.t, err, "Error while retrieving [%d]th block from blockfileMgr", i)
		assert.Equal(w.t, blocks[i].Header, b.Header)
	}
//
	b, err := w.blockfileMgr.retrieveBlockByNumber(math.MaxUint64)
	iLastBlock := len(blocks) - 1
	assert.NoError(w.t, err, "Error while retrieving last block from blockfileMgr")
	assert.Equal(w.t, blocks[iLastBlock], b)
}

func (w *testBlockfileMgrWrapper) close() {
	w.blockfileMgr.close()
}
