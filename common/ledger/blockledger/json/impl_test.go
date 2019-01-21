
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
*/


package jsonledger

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/stretchr/testify/assert"
)

var genesisBlock = cb.NewBlock(0, nil)

func init() {
	flogging.ActivateSpec("common.ledger.blockledger.json=DEBUG")
}

type testEnv struct {
	t        *testing.T
	location string
}

func initialize(t *testing.T) (*testEnv, *jsonLedger) {
	name, err := ioutil.TempDir("", "hyperledger_fabric")
	if err != nil {
		t.Fatalf("Error creating temp dir: %s", err)
	}
	flf := New(name).(*jsonLedgerFactory)
	fl, err := flf.GetOrCreate(genesisconfig.TestChainID)
	if err != nil {
		panic(err)
	}

	fl.Append(genesisBlock)
	return &testEnv{location: name, t: t}, fl.(*jsonLedger)
}

func (tev *testEnv) tearDown() {
	err := os.RemoveAll(tev.location)
	if err != nil {
		tev.t.Fatalf("Error tearing down env: %s", err)
	}
}

func TestInitialization(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()
	assert.Equal(t, uint64(1), fl.height, "Block height should be 1")

	block, found := fl.readBlock(0)
	assert.NotNil(t, block, "Error retrieving genesis block")
	assert.True(t, found, "Error retrieving genesis block")
	assert.Equal(t, fl.lastHash, block.Header.Hash(), "Block hashes did no match")

}

func TestReinitialization(t *testing.T) {
	tev, ofl := initialize(t)
	defer tev.tearDown()
	ofl.Append(blockledger.CreateNextBlock(ofl, []*cb.Envelope{{Payload: []byte("My Data")}}))
	flf := New(tev.location)
	chains := flf.ChainIDs()
	assert.Len(t, chains, 1, "Should have recovered the chain")

	tfl, err := flf.GetOrCreate(chains[0])
	assert.Nil(t, err, "Unexpected error: %s", err)

	fl := tfl.(*jsonLedger)
	assert.Equal(t, uint64(2), fl.height, "Block height should be 2")

	block, found := fl.readBlock(1)
	assert.NotNil(t, block, "Error retrieving block")
	assert.True(t, found, "Error retrieving block")
	assert.Equal(t, fl.lastHash, block.Header.Hash(), "Block hashes did no match")
}

func TestMultiReinitialization(t *testing.T) {
	tev, _ := initialize(t)
	defer tev.tearDown()
	flf := New(tev.location)

	_, err := flf.GetOrCreate("foo")
	assert.Nil(t, err, "Error creating chain")

	_, err = flf.GetOrCreate("bar")
	assert.Nil(t, err, "Error creating chain")

	flf = New(tev.location)
	assert.Len(t, flf.ChainIDs(), 3, "Should have recovered the chains")
}

func TestAddition(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()
	prevHash := fl.lastHash
	fl.Append(blockledger.CreateNextBlock(fl, []*cb.Envelope{{Payload: []byte("My Data")}}))
	assert.Equal(t, uint64(2), fl.height, "Block height should be 2")

	block, found := fl.readBlock(1)
	assert.NotNil(t, block, "Error retrieving genesis block")
	assert.True(t, found, "Error retrieving genesis block")
	assert.Equal(t, prevHash, block.Header.PreviousHash, "Block hashes did no match")
}

func TestRetrieval(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()
	fl.Append(blockledger.CreateNextBlock(fl, []*cb.Envelope{{Payload: []byte("My Data")}}))
	it, num := fl.Iterator(&ab.SeekPosition{Type: &ab.SeekPosition_Oldest{}})
	defer it.Close()
	assert.Equal(t, uint64(0), num, "Expected genesis block iterator, but got %d", num)

	block, status := it.Next()
	assert.Equal(t, cb.Status_SUCCESS, status, "Expected to successfully read the genesis block")
	assert.Equal(t, uint64(0), block.Header.Number, "Expected to successfully retrieve the genesis block")

	block, status = it.Next()
	assert.Equal(t, cb.Status_SUCCESS, status, "Expected to successfully read the second block")
	assert.Equal(t, uint64(1), block.Header.Number, "Expected to successfully retrieve the second block but got block number %d", block.Header.Number)
}

//如果实现中没有文件锁定，则由于
//同时写入和读取块文件的竞争条件。用
//
func TestRaceCondition(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()

	it, _ := fl.Iterator(&ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: 1}}})
	defer it.Close()

	var status cb.Status

	complete := make(chan struct{})
	go func() {
		_, status = it.Next()
		close(complete)
	}()

	fl.Append(blockledger.CreateNextBlock(fl, []*cb.Envelope{{Payload: []byte("My Data")}}))
	<-complete

	assert.Equal(t, cb.Status_SUCCESS, status, "Expected to successfully read the block")
}

func TestBlockedRetrieval(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()
	it, num := fl.Iterator(&ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: 1}}})
	defer it.Close()
	assert.Equal(t, uint64(1), num, "Expected block iterator at 1, but got %d", num)

	fl.Append(blockledger.CreateNextBlock(fl, []*cb.Envelope{{Payload: []byte("My Data")}}))

	block, status := it.Next()
	assert.Equal(t, cb.Status_SUCCESS, status, "Expected to successfully read the second block")
	assert.Equal(t, uint64(1), block.Header.Number, "Expected to successfully retrieve the second block")

	go func() {
//
//
//然而，在通道运行之前，它不能保证运行。
//
//
		time.Sleep(100 * time.Millisecond)
		fl.Append(blockledger.CreateNextBlock(fl, []*cb.Envelope{{Payload: []byte("Another Data")}}))
	}()

	block, status = it.Next()
	assert.Equal(t, cb.Status_SUCCESS, status, "Expected to successfully read the third block")
	assert.Equal(t, uint64(2), block.Header.Number, "Expected to successfully retrieve the third block")
}

func TestInvalidRetrieval(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()

	it, num := fl.Iterator(&ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: 2}}})
	defer it.Close()
	assert.Equal(t, uint64(0), num, "Expected block number to be zero for invalid iterator")

	_, status := it.Next()
	assert.Equal(t, cb.Status_NOT_FOUND, status, "Expected status_NOT_FOUND for invalid iterator")
}

func TestBrokenBlockFile(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()

//
	file, err := os.OpenFile(fl.blockFilename(0), os.O_RDWR, 0700)
	assert.Nil(t, err, "Expected to successfully open block file")

	_, err = file.WriteString("Hello, world!")
	assert.Nil(t, err, "Expected to successfully write to block file")

	assert.NoError(t, file.Close(), "Expected to successfully close block file")

	it, num := fl.Iterator(&ab.SeekPosition{Type: &ab.SeekPosition_Oldest{}})
	defer it.Close()
	assert.Equal(t, uint64(0), num, "Expected genesis block iterator, but got %d", num)

	_, status := it.Next()
	assert.Equal(t, cb.Status_SERVICE_UNAVAILABLE, status, "Expected reading the genesis block to fail")
}

func TestInvalidAddition(t *testing.T) {
	tev, fl := initialize(t)
	defer tev.tearDown()

//
	{
		block := blockledger.CreateNextBlock(fl, []*cb.Envelope{{Payload: []byte("My Data")}})
		block.Header.Number++
		assert.Error(t, fl.Append(block), "Addition of block with invalid number should fail")
	}

//带有无效previouushash的append块
	{
		block := blockledger.CreateNextBlock(fl, []*cb.Envelope{{Payload: []byte("My Data")}})
		block.Header.PreviousHash = nil
		assert.Error(t, fl.Append(block), "Addition of block with invalid previousHash should fail")
	}
}
