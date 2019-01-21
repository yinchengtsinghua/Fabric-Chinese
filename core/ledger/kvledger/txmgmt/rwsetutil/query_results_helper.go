
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


package rwsetutil

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	bccspfactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/pkg/errors"
)

//用于表示merkle树级别的merkletelevel
type MerkleTreeLevel uint32

//哈希表示哈希的字节数
type Hash []byte

const (
	leafLevel = MerkleTreeLevel(1)
)

var (
	hashOpts = &bccsp.SHA256Opts{}
)

//RangeQueryResultsHelper有助于在验证期间为幻象项检测准备范围查询结果。
//在迭代过程中，结果将被发送。
//如果“hashingnabled”设置为true，则会在结果上使用散列构建merkle树。
//Merkle树有助于减小rwset的大小，否则将需要存储所有原始kvreads
//
//树的心理模型可以描述如下：
//所有结果都被视为树的叶节点（级别0）。树的下一级是通过收集“maxdegree+1”来构建的。
//上一级别的项，并对整个集合进行哈希处理。
//树的更高层次以类似的方式构建，但是唯一的区别是不同于0级
//（其中集合由原始kvreads组成），级别1及以上的集合由哈希组成
//（上一级的集合）。
//重复这个过程，直到我们达到一个水平，我们剩下的项目数小于或等于“maxdegree”。
//在上一个集合中，项数可以小于“maxdegree”（除非这是给定级别上唯一的集合）。
//
//因此，如果输入结果总数小于或等于“maxdegree”，则根本不执行哈希操作。
//计算的最终输出是原始结果的集合（如果小于或等于“maxdegree”），或者
//树中某一级别的哈希（等于“maxdegree”）的集合。
//
//应调用“addresult”函数以提供下一个结果，并在末尾调用“done”函数。
//“done”函数执行最终处理并返回最终输出
type RangeQueryResultsHelper struct {
	pendingResults []*kvrwset.KVRead
	mt             *merkleTree
	maxDegree      uint32
	hashingEnabled bool
}

//NewRangeQueryResultShelper构造RangeQueryResultShelper
func NewRangeQueryResultsHelper(enableHashing bool, maxDegree uint32) (*RangeQueryResultsHelper, error) {
	helper := &RangeQueryResultsHelper{pendingResults: nil,
		hashingEnabled: enableHashing,
		maxDegree:      maxDegree,
		mt:             nil}
	if enableHashing {
		var err error
		if helper.mt, err = newMerkleTree(maxDegree); err != nil {
			return nil, err
		}
	}
	return helper, nil
}

//addresult添加新的查询结果进行处理。
//将结果放入挂起结果列表。如果挂起的结果数超过“maxdegree”，
//使用结果以逐步更新Merkle树
func (helper *RangeQueryResultsHelper) AddResult(kvRead *kvrwset.KVRead) error {
	logger.Debug("Adding a result")
	helper.pendingResults = append(helper.pendingResults, kvRead)
	if helper.hashingEnabled && uint32(len(helper.pendingResults)) > helper.maxDegree {
		logger.Debug("Processing the accumulated results")
		if err := helper.processPendingResults(); err != nil {
			return err
		}
	}
	return nil
}

//完成如果需要处理任何挂起的结果
//这将返回最终挂起的结果（即[]*kvread）和结果的散列（即*merklesummary）
//这两个结果中只有一个是非零的（除非从未添加任何结果）。
//如果且仅当'enableHashing'设置为false，则'merklesummary'将为nil
//或者总结果数小于'maxdegree'
func (helper *RangeQueryResultsHelper) Done() ([]*kvrwset.KVRead, *kvrwset.QueryReadsMerkleSummary, error) {
//如果总结果小于或等于“maxdegree”，则merkle树将为空。
//也就是说，即使在处理结果进行哈希处理之后
	if !helper.hashingEnabled || helper.mt.isEmpty() {
		return helper.pendingResults, nil, nil
	}
	if len(helper.pendingResults) != 0 {
		logger.Debug("Processing the pending results")
		if err := helper.processPendingResults(); err != nil {
			return helper.pendingResults, nil, err
		}
	}
	helper.mt.done()
	return helper.pendingResults, helper.mt.getSummery(), nil
}

//getmerklesummary返回merklesummary的当前状态
//Merkle树的这种中间状态有助于在验证期间及早检测到不匹配。
//这有助于在验证期间不需要构建完整的Merkle树。
//如果结果集的早期部分不匹配。
func (helper *RangeQueryResultsHelper) GetMerkleSummary() *kvrwset.QueryReadsMerkleSummary {
	if !helper.hashingEnabled {
		return nil
	}
	return helper.mt.getSummery()
}

func (helper *RangeQueryResultsHelper) processPendingResults() error {
	var b []byte
	var err error
	if b, err = serializeKVReads(helper.pendingResults); err != nil {
		return err
	}
	helper.pendingResults = nil
	hash, err := bccspfactory.GetDefault().Hash(b, hashOpts)
	if err != nil {
		return err
	}
	helper.mt.update(hash)
	return nil
}

func serializeKVReads(kvReads []*kvrwset.KVRead) ([]byte, error) {
	return proto.Marshal(&kvrwset.QueryReads{KvReads: kvReads})
}

/////////merkle树建筑规范

type merkleTree struct {
	tree      map[MerkleTreeLevel][]Hash
	maxLevel  MerkleTreeLevel
	maxDegree uint32
}

func newMerkleTree(maxDegree uint32) (*merkleTree, error) {
	if maxDegree < 2 {
		return nil, errors.Errorf("maxDegree [%d] should not be less than 2 in the merkle tree", maxDegree)
	}
	return &merkleTree{make(map[MerkleTreeLevel][]Hash), 1, maxDegree}, nil
}

//更新采用一个散列值，该散列值形成了Merkle树中的下一个叶级（级别1）节点。
//另外，添加这个新的叶节点，尽可能地完成Merkle树-
//即递归地构建更高级别的节点并删除底层的子树。
func (m *merkleTree) update(nextLeafLevelHash Hash) error {
	logger.Debugf("Before update() = %s", m)
	defer logger.Debugf("After update() = %s", m)
	m.tree[leafLevel] = append(m.tree[leafLevel], nextLeafLevelHash)
	currentLevel := leafLevel
	for {
		currentLevelHashes := m.tree[currentLevel]
		if uint32(len(currentLevelHashes)) <= m.maxDegree {
			return nil
		}
		nextLevelHash, err := computeCombinedHash(currentLevelHashes)
		if err != nil {
			return err
		}
		delete(m.tree, currentLevel)
		nextLevel := currentLevel + 1
		m.tree[nextLevel] = append(m.tree[nextLevel], nextLevelHash)
		if nextLevel > m.maxLevel {
			m.maxLevel = nextLevel
		}
		currentLevel = nextLevel
	}
}

//梅克尔树完成了。
//可能有一些节点的级别低于maxlevel（到目前为止树看到的最大级别）。
//从这些节点中生成父节点，直到我们在maxlevel（或maxlevel+1）级别完成树。
func (m *merkleTree) done() error {
	logger.Debugf("Before done() = %s", m)
	defer logger.Debugf("After done() = %s", m)
	currentLevel := leafLevel
	var h Hash
	var err error
	for currentLevel < m.maxLevel {
		currentLevelHashes := m.tree[currentLevel]
		switch len(currentLevelHashes) {
		case 0:
			currentLevel++
			continue
		case 1:
			h = currentLevelHashes[0]
		default:
			if h, err = computeCombinedHash(currentLevelHashes); err != nil {
				return err
			}
		}
		delete(m.tree, currentLevel)
		currentLevel++
		m.tree[currentLevel] = append(m.tree[currentLevel], h)
	}

	finalHashes := m.tree[m.maxLevel]
	if uint32(len(finalHashes)) > m.maxDegree {
		delete(m.tree, m.maxLevel)
		m.maxLevel++
		combinedHash, err := computeCombinedHash(finalHashes)
		if err != nil {
			return err
		}
		m.tree[m.maxLevel] = []Hash{combinedHash}
	}
	return nil
}

func (m *merkleTree) getSummery() *kvrwset.QueryReadsMerkleSummary {
	return &kvrwset.QueryReadsMerkleSummary{MaxDegree: m.maxDegree,
		MaxLevel:       uint32(m.getMaxLevel()),
		MaxLevelHashes: hashesToBytes(m.getMaxLevelHashes())}
}

func (m *merkleTree) getMaxLevel() MerkleTreeLevel {
	return m.maxLevel
}

func (m *merkleTree) getMaxLevelHashes() []Hash {
	return m.tree[m.maxLevel]
}

func (m *merkleTree) isEmpty() bool {
	return m.maxLevel == 1 && len(m.tree[m.maxLevel]) == 0
}

func (m *merkleTree) String() string {
	return fmt.Sprintf("tree := %#v", m.tree)
}

func computeCombinedHash(hashes []Hash) (Hash, error) {
	combinedHash := []byte{}
	for _, h := range hashes {
		combinedHash = append(combinedHash, h...)
	}
	return bccspfactory.GetDefault().Hash(combinedHash, hashOpts)
}

func hashesToBytes(hashes []Hash) [][]byte {
	b := [][]byte{}
	for _, hash := range hashes {
		b = append(b, hash)
	}
	return b
}
