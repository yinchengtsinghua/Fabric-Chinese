
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


package fileledger

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/stretchr/testify/assert"
)

type mockBlockStoreProvider struct {
	blockstore blkstorage.BlockStore
	exists     bool
	list       []string
	error      error
}

func (mbsp *mockBlockStoreProvider) CreateBlockStore(ledgerid string) (blkstorage.BlockStore, error) {
	return mbsp.blockstore, mbsp.error
}

func (mbsp *mockBlockStoreProvider) OpenBlockStore(ledgerid string) (blkstorage.BlockStore, error) {
	return mbsp.blockstore, mbsp.error
}

func (mbsp *mockBlockStoreProvider) Exists(ledgerid string) (bool, error) {
	return mbsp.exists, mbsp.error
}

func (mbsp *mockBlockStoreProvider) List() ([]string, error) {
	return mbsp.list, mbsp.error
}

func (mbsp *mockBlockStoreProvider) Close() {
}

func TestBlockstoreProviderError(t *testing.T) {
	flf := &fileLedgerFactory{
		blkstorageProvider: &mockBlockStoreProvider{error: fmt.Errorf("blockstorage provider error")},
		ledgers:            make(map[string]blockledger.ReadWriter),
	}
	assert.Panics(
		t,
		func() { flf.ChainIDs() },
		"Expected ChainIDs to panic if storage provider cannot list chain IDs")

	_, err := flf.GetOrCreate("foo")
	assert.Error(t, err, "Expected GetOrCreate to return error if blockstorage provider cannot open")
	assert.Empty(t, flf.ledgers, "Expected no new ledger is created")
}

func TestMultiReinitialization(t *testing.T) {
	dir, err := ioutil.TempDir("", "hyperledger_fabric")
	assert.NoError(t, err, "Error creating temp dir: %s", err)

	flf := New(dir)
	_, err = flf.GetOrCreate(genesisconfig.TestChainID)
	assert.NoError(t, err, "Error GetOrCreate chain")
	assert.Equal(t, 1, len(flf.ChainIDs()), "Expected 1 chain")
	flf.Close()

	flf = New(dir)
	_, err = flf.GetOrCreate("foo")
	assert.NoError(t, err, "Error creating chain")
	assert.Equal(t, 2, len(flf.ChainIDs()), "Expected chain to be recovered")
	flf.Close()

	flf = New(dir)
	_, err = flf.GetOrCreate("bar")
	assert.NoError(t, err, "Error creating chain")
	assert.Equal(t, 3, len(flf.ChainIDs()), "Expected chain to be recovered")
	flf.Close()
}
