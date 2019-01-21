
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


package jsonledger

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

//
func TestErrorMkdir(t *testing.T) {
	name, err := ioutil.TempDir("", "hyperledger_fabric")
	assert.Nil(t, err, "Error creating temp dir: %s", err)
	defer os.RemoveAll(name)
	ledgerPath := path.Join(name, "jsonledger")
	assert.NoError(t, ioutil.WriteFile(ledgerPath, nil, 0700))

	assert.Panics(t, func() { New(ledgerPath) }, "Should have failed to create factory")
}

//
//
//
//
func TestIgnoreInvalidObjectInDir(t *testing.T) {
	name, err := ioutil.TempDir("", "hyperledger_fabric")
	assert.Nil(t, err, "Error creating temp dir: %s", err)
	defer os.RemoveAll(name)
	file, err := ioutil.TempFile(name, "chain_")
	assert.Nil(t, err, "Errot creating temp file: %s", err)
	defer file.Close()
	_, err = ioutil.TempDir(name, "invalid_chain_")
	assert.Nil(t, err, "Error creating temp dir: %s", err)

	jlf := New(name)
	assert.Empty(t, jlf.ChainIDs(), "Expected invalid objects to be ignored while restoring chains from directory")
}

//
func TestInvalidChain(t *testing.T) {
	name, err := ioutil.TempDir("", "hyperledger_fabric")
	assert.Nil(t, err, "Error creating temp dir: %s", err)
	defer os.RemoveAll(name)

	chainDir, err := ioutil.TempDir(name, "chain_")
	assert.Nil(t, err, "Error creating temp dir: %s", err)

//
	secondBlock := path.Join(chainDir, fmt.Sprintf(blockFileFormatString, 1))
	assert.NoError(t, ioutil.WriteFile(secondBlock, nil, 0700))

	t.Run("MissingBlock", func(t *testing.T) {
		assert.Panics(t, func() { New(name) }, "Expected initialization panics if block is missing")
	})

	t.Run("SkipDir", func(t *testing.T) {
		invalidBlock := path.Join(chainDir, fmt.Sprintf(blockFileFormatString, 0))
		assert.NoError(t, os.Mkdir(invalidBlock, 0700))
		assert.Panics(t, func() { New(name) }, "Expected initialization skips directory in chain dir")
		assert.NoError(t, os.RemoveAll(invalidBlock))
	})

	firstBlock := path.Join(chainDir, fmt.Sprintf(blockFileFormatString, 0))
	assert.NoError(t, ioutil.WriteFile(firstBlock, nil, 0700))

	t.Run("MalformedBlock", func(t *testing.T) {
		assert.Panics(t, func() { New(name) }, "Expected initialization panics if block is malformed")
	})
}

//
func TestIgnoreInvalidBlockFileName(t *testing.T) {
	name, err := ioutil.TempDir("", "hyperledger_fabric")
	assert.Nil(t, err, "Error creating temp dir: %s", err)
	defer os.RemoveAll(name)

	chainDir, err := ioutil.TempDir(name, "chain_")
	assert.Nil(t, err, "Error creating temp dir: %s", err)

	invalidBlock := path.Join(chainDir, "invalid_block")
	assert.NoError(t, ioutil.WriteFile(invalidBlock, nil, 0700))
	jfl := New(name)
	assert.Equal(t, 1, len(jfl.ChainIDs()), "Expected factory initialized with 1 chain")

	chain, err := jfl.GetOrCreate(jfl.ChainIDs()[0])
	assert.Nil(t, err, "Should have retrieved chain")
	assert.Zero(t, chain.Height(), "Expected chain to be empty")
}

func TestClose(t *testing.T) {
	name, err := ioutil.TempDir("", "hyperledger_fabric")
	assert.Nil(t, err, "Error creating temp dir: %s", err)
	defer os.RemoveAll(name)

	jlf := New(name)
	assert.NotPanics(t, func() { jlf.Close() }, "Noop should not pannic")
}
