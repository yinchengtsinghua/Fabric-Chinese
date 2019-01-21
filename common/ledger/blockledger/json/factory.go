
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


package jsonledger

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/golang/protobuf/jsonpb"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	"github.com/pkg/errors"
)

type jsonLedgerFactory struct {
	directory string
	ledgers   map[string]blockledger.ReadWriter
	mutex     sync.Mutex
}

//getorcreate获取现有分类帐（如果存在），或者如果不存在则创建该分类帐。
func (jlf *jsonLedgerFactory) GetOrCreate(chainID string) (blockledger.ReadWriter, error) {
	jlf.mutex.Lock()
	defer jlf.mutex.Unlock()

	key := chainID

	l, ok := jlf.ledgers[key]
	if ok {
		return l, nil
	}

	directory := filepath.Join(jlf.directory, fmt.Sprintf(chainDirectoryFormatString, chainID))

	logger.Debugf("Initializing chain %s at: %s", chainID, directory)

	if err := os.MkdirAll(directory, 0700); err != nil {
		logger.Errorf("Error initializing channel %s: %s", chainID, err)
		return nil, errors.Wrapf(err, "error initializing channel %s", chainID)
	}

	ch := newChain(directory)
	jlf.ledgers[key] = ch
	return ch, nil
}

//new chain创建由JSON分类帐支持的新链
func newChain(directory string) blockledger.ReadWriter {
	jl := &jsonLedger{
		directory: directory,
		signal:    make(chan struct{}),
		marshaler: &jsonpb.Marshaler{Indent: "  "},
	}
	jl.initializeBlockHeight()
	logger.Debugf("Initialized to block height %d with hash %x", jl.height-1, jl.lastHash)
	return jl
}

//InitializeBlockHeight验证0和块之间是否存在所有块
//高度，并填充最后一个哈希
func (jl *jsonLedger) initializeBlockHeight() {
	infos, err := ioutil.ReadDir(jl.directory)
	if err != nil {
		logger.Panic(err)
	}
	nextNumber := uint64(0)
	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		var number uint64
		_, err := fmt.Sscanf(info.Name(), blockFileFormatString, &number)
		if err != nil {
			continue
		}
		if number != nextNumber {
			logger.Panicf("Missing block %d in the chain", nextNumber)
		}
		nextNumber++
	}
	jl.height = nextNumber
	if jl.height == 0 {
		return
	}
	block, found := jl.readBlock(jl.height - 1)
	if !found {
		logger.Panicf("Block %d was in directory listing but error reading", jl.height-1)
	}
	if block == nil {
		logger.Panicf("Error reading block %d", jl.height-1)
	}
	jl.lastHash = block.Header.Hash()
}

//chainIds返回工厂知道的链ID
func (jlf *jsonLedgerFactory) ChainIDs() []string {
	jlf.mutex.Lock()
	defer jlf.mutex.Unlock()
	ids := make([]string, len(jlf.ledgers))

	i := 0
	for key := range jlf.ledgers {
		ids[i] = key
		i++
	}

	return ids
}

//关闭是JSON分类账的非操作
func (jlf *jsonLedgerFactory) Close() {
return //无事可做
}

//新建创建新的分类帐工厂
func New(directory string) blockledger.Factory {
	logger.Debugf("Initializing ledger at: %s", directory)
	if err := os.MkdirAll(directory, 0700); err != nil {
		logger.Panicf("Could not create directory %s: %s", directory, err)
	}

	jlf := &jsonLedgerFactory{
		directory: directory,
		ledgers:   make(map[string]blockledger.ReadWriter),
	}

	infos, err := ioutil.ReadDir(jlf.directory)
	if err != nil {
		logger.Panicf("Error reading from directory %s while initializing ledger: %s", jlf.directory, err)
	}

	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		var chainID string
		_, err := fmt.Sscanf(info.Name(), chainDirectoryFormatString, &chainID)
		if err != nil {
			continue
		}
		jlf.GetOrCreate(chainID)
	}

	return jlf
}
