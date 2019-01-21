
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


package testutil

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/common/util"
	lutils "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	ptestutils "github.com/hyperledger/fabric/protos/testutils"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

//BlockGenerator生成一系列用于测试的块
type BlockGenerator struct {
	blockNum     uint64
	previousHash []byte
	signTxs      bool
	t            *testing.T
}

type TxDetails struct {
	TxID                            string
	ChaincodeName, ChaincodeVersion string
	SimulationResults               []byte
}

type BlockDetails struct {
	BlockNum     uint64
	PreviousHash []byte
	Txs          []*TxDetails
}

//
func NewBlockGenerator(t *testing.T, ledgerID string, signTxs bool) (*BlockGenerator, *common.Block) {
	gb, err := test.MakeGenesisBlock(ledgerID)
	assert.NoError(t, err)
	gb.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = lutils.NewTxValidationFlagsSetValue(len(gb.Data.Data), pb.TxValidationCode_VALID)
	return &BlockGenerator{1, gb.GetHeader().Hash(), signTxs, t}, gb
}

//
func (bg *BlockGenerator) NextBlock(simulationResults [][]byte) *common.Block {
	block := ConstructBlock(bg.t, bg.blockNum, bg.previousHash, simulationResults, bg.signTxs)
	bg.blockNum++
	bg.previousHash = block.Header.Hash()
	return block
}

//nextBlockWithTxID按顺序构造下一个块，其中包含许多事务-每个模拟结果一个
func (bg *BlockGenerator) NextBlockWithTxid(simulationResults [][]byte, txids []string) *common.Block {
//
	if len(simulationResults) != len(txids) {
		return nil
	}
	block := ConstructBlockWithTxid(bg.t, bg.blockNum, bg.previousHash, simulationResults, txids, bg.signTxs)
	bg.blockNum++
	bg.previousHash = block.Header.Hash()
	return block
}

//
func (bg *BlockGenerator) NextTestBlock(numTx int, txSize int) *common.Block {
	simulationResults := [][]byte{}
	for i := 0; i < numTx; i++ {
		simulationResults = append(simulationResults, ConstructRandomBytes(bg.t, txSize))
	}
	return bg.NextBlock(simulationResults)
}

//
func (bg *BlockGenerator) NextTestBlocks(numBlocks int) []*common.Block {
	blocks := []*common.Block{}
	numTx := 10
	for i := 0; i < numBlocks; i++ {
		block := bg.NextTestBlock(numTx, 100)
		block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = lutils.NewTxValidationFlagsSetValue(numTx, pb.TxValidationCode_VALID)
		blocks = append(blocks, block)
	}
	return blocks
}

//
func ConstructTransaction(_ *testing.T, simulationResults []byte, txid string, sign bool) (*common.Envelope, string, error) {
	return ConstructTransactionFromTxDetails(
		&TxDetails{
			ChaincodeName:     "foo",
			ChaincodeVersion:  "v1",
			TxID:              txid,
			SimulationResults: simulationResults,
		},
		sign,
	)
}

func ConstructTransactionFromTxDetails(txDetails *TxDetails, sign bool) (*common.Envelope, string, error) {
	ccid := &pb.ChaincodeID{
		Name:    txDetails.ChaincodeName,
		Version: txDetails.ChaincodeVersion,
	}
	var txEnv *common.Envelope
	var err error
	var txID string
	if sign {
		txEnv, txID, err = ptestutils.ConstructSignedTxEnvWithDefaultSigner(util.GetTestChainID(), ccid, nil, txDetails.SimulationResults, txDetails.TxID, nil, nil)
	} else {
		txEnv, txID, err = ptestutils.ConstructUnsignedTxEnv(util.GetTestChainID(), ccid, nil, txDetails.SimulationResults, txDetails.TxID, nil, nil)
	}
	return txEnv, txID, err
}

func ConstructBlockFromBlockDetails(t *testing.T, blockDetails *BlockDetails, sign bool) *common.Block {
	var envs []*common.Envelope
	for _, txDetails := range blockDetails.Txs {
		env, _, err := ConstructTransactionFromTxDetails(txDetails, sign)
		if err != nil {
			t.Fatalf("ConstructTestTransaction failed, err %s", err)
		}
		envs = append(envs, env)
	}
	return NewBlock(envs, blockDetails.BlockNum, blockDetails.PreviousHash)
}

func ConstructBlockWithTxid(t *testing.T, blockNum uint64, previousHash []byte, simulationResults [][]byte, txids []string, sign bool) *common.Block {
	envs := []*common.Envelope{}
	for i := 0; i < len(simulationResults); i++ {
		env, _, err := ConstructTransaction(t, simulationResults[i], txids[i], sign)
		if err != nil {
			t.Fatalf("ConstructTestTransaction failed, err %s", err)
		}
		envs = append(envs, env)
	}
	return NewBlock(envs, blockNum, previousHash)
}

//
func ConstructBlock(t *testing.T, blockNum uint64, previousHash []byte, simulationResults [][]byte, sign bool) *common.Block {
	envs := []*common.Envelope{}
	for i := 0; i < len(simulationResults); i++ {
		env, _, err := ConstructTransaction(t, simulationResults[i], "", sign)
		if err != nil {
			t.Fatalf("ConstructTestTransaction failed, err %s", err)
		}
		envs = append(envs, env)
	}
	return NewBlock(envs, blockNum, previousHash)
}

//
func ConstructTestBlock(t *testing.T, blockNum uint64, numTx int, txSize int) *common.Block {
	simulationResults := [][]byte{}
	for i := 0; i < numTx; i++ {
		simulationResults = append(simulationResults, ConstructRandomBytes(t, txSize))
	}
	return ConstructBlock(t, blockNum, ConstructRandomBytes(t, 32), simulationResults, false)
}

//
//
func ConstructTestBlocks(t *testing.T, numBlocks int) []*common.Block {
	bg, gb := NewBlockGenerator(t, util.GetTestChainID(), false)
	blocks := []*common.Block{}
	if numBlocks != 0 {
		blocks = append(blocks, gb)
	}
	return append(blocks, bg.NextTestBlocks(numBlocks-1)...)
}

//constructBytesProposalResponsePayLoad使用给定的链码版本和模拟结果构造一个ProposalResponse字节用于测试
func ConstructBytesProposalResponsePayload(version string, simulationResults []byte) ([]byte, error) {
	ccid := &pb.ChaincodeID{
		Name:    "foo",
		Version: version,
	}
	return ptestutils.ConstructBytesProposalResponsePayload(util.GetTestChainID(), ccid, nil, simulationResults)
}

func NewBlock(env []*common.Envelope, blockNum uint64, previousHash []byte) *common.Block {
	block := common.NewBlock(blockNum, previousHash)
	for i := 0; i < len(env); i++ {
		txEnvBytes, _ := proto.Marshal(env[i])
		block.Data.Data = append(block.Data.Data, txEnvBytes)
	}
	block.Header.DataHash = block.Data.Hash()
	utils.InitBlockMetadata(block)

	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = lutils.NewTxValidationFlagsSetValue(len(env), pb.TxValidationCode_VALID)

	return block
}
