
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


package statebased

import (
	"fmt"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/handlers/validation/api/state"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("vscc")

/*********************************************************************/
/*********************************************************************/

type ledgerKeyID struct {
	cc   string
	coll string
	key  string
}

func newLedgerKeyID(cc, coll, key string) *ledgerKeyID {
	return &ledgerKeyID{cc, coll, key}
}

/*********************************************************************/
/*********************************************************************/

//TxDependency为事务提供同步机制
//VSCC作用域中的依赖项，其中事务在每个-
//命名空间基础：
//-）waitForDepinserted（）/signalDepinserted（）对用于同步
//插入依赖项
//-）waitforandretrievevalidationresult（）/signalvalidationresult（）对
//用于同步给定命名空间的验证结果
type txDependency struct {
	mutex               sync.Mutex
	cond                *sync.Cond
	validationResultMap map[string]error
	depInserted         chan struct{}
}

func newTxDependency() *txDependency {
	txd := &txDependency{
		depInserted:         make(chan struct{}),
		validationResultMap: make(map[string]error),
	}
	txd.cond = sync.NewCond(&txd.mutex)
	return txd
}

//waitForDepinserted等待事务引入的依赖项
//已插入。函数一返回
//d.depinserted已被signaldepinserted关闭
func (d *txDependency) waitForDepInserted() {
	<-d.depInserted
}

//signalDepinserted表示引入了事务依赖项
//已按事务d.txnum插入。函数
//关闭d.depinserted，使WaitForDepinserted的所有调用方
//返回。此函数只能在此对象上调用一次
func (d *txDependency) signalDepInserted() {
	close(d.depInserted)
}

//WaitForAndRetrieveValidationResult返回验证结果
//对于命名空间“ns”-可能正在等待相应的调用
//以先完成信号验证结果。
func (d *txDependency) waitForAndRetrieveValidationResult(ns string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	err, ok := d.validationResultMap[ns]
	if ok {
		return err
	}

	for !ok {
		d.cond.Wait()
		err, ok = d.validationResultMap[ns]
	}

	return err
}

//signalvalidationresult表示命名空间“ns”的验证
//对于事务'd.txnum'已完成，但出现错误'err'。结果
//缓存到映射中。我们还广播一个条件变量
//唤醒waitforandretrievevalidationresult的可能调用方
func (d *txDependency) signalValidationResult(ns string, err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.validationResultMap[ns] = err
	d.cond.Broadcast()
}

/*********************************************************************/
/*********************************************************************/

//validationContext捕获单个块中的所有依赖项
type validationContext struct {
//互斥确保只有一个GODUTAN在
//时间将修改块高度，depsbytxnummap
//或depsbyledgerkeyidmap
	mutex                sync.RWMutex
	blockHeight          uint64
	depsByTxnumMap       map[uint64]*txDependency
	depsByLedgerKeyIDMap map[ledgerKeyID]map[uint64]*txDependency
}

func (c *validationContext) forBlock(newHeight uint64) *validationContext {
	c.mutex.RLock()
	curHeight := c.blockHeight
	c.mutex.RUnlock()

	if curHeight > newHeight {
		logger.Panicf("programming error: block with number %d validated after block with number %d", newHeight, curHeight)
	}

//0区是起源区，所以第一个到这里的街区
//实际上是块1，强制重置
	if curHeight < newHeight {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		if c.blockHeight < newHeight {
			c.blockHeight = newHeight
			c.depsByLedgerKeyIDMap = map[ledgerKeyID]map[uint64]*txDependency{}
			c.depsByTxnumMap = map[uint64]*txDependency{}
		}
	}

	return c
}

func (c *validationContext) addDependency(kid *ledgerKeyID, txnum uint64, dep *txDependency) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

//必要时创建地图
	_, ok := c.depsByLedgerKeyIDMap[*kid]
	if !ok {
		c.depsByLedgerKeyIDMap[*kid] = map[uint64]*txDependency{}
	}

	c.depsByLedgerKeyIDMap[*kid][txnum] = dep
}

func (c *validationContext) dependenciesForTxnum(kid *ledgerKeyID, txnum uint64) []*txDependency {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var deps []*txDependency

	dl, in := c.depsByLedgerKeyIDMap[*kid]
	if in {
		deps = make([]*txDependency, 0, len(dl))
		for depTxnum, dep := range dl {
			if depTxnum < txnum {
				deps = append(deps, dep)
			}
		}
	}

	return deps
}

func (c *validationContext) getOrCreateDependencyByTxnum(txnum uint64) *txDependency {
	c.mutex.RLock()
	dep, ok := c.depsByTxnumMap[txnum]
	c.mutex.RUnlock()

	if !ok {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		dep, ok = c.depsByTxnumMap[txnum]
		if !ok {
			dep = newTxDependency()
			c.depsByTxnumMap[txnum] = dep
		}
	}

	return dep
}

func (c *validationContext) waitForValidationResults(kid *ledgerKeyID, blockNum uint64, txnum uint64) error {
//在下面的代码中，我们将看到此块中是否有任何事务
//在txnum之前，它引入了依赖关系。我们这样做
//从映射中提取txnum的所有txdependency实例
//严格低于我们的标准并重新获得验证
//结果。如果其中*任何*的验证结果为零
//错误，我们有依赖关系。否则我们就没有依赖性。
//请注意，depsmap是以不可预测的顺序迭代的。
//这不会违反正确性，因为hasDependencies
//如果引入了*any*依赖项，则应返回true

//我们分两步进行：
//1）在保持互斥的同时，我们获取所有依赖项的快照
//影响我们并把它们放在一个局部区域；然后我们释放
//互斥体
//2）我们遍历依赖项切片，并为每个依赖项检索
//验证结果
//需要采用两步方法来避免死锁，其中
//consumer (the caller of this function) holds the mutex and thus
//阻止生产者（信号验证结果的调用方）执行以下操作：
//产生结果。

	for _, dep := range c.dependenciesForTxnum(kid, txnum) {
		if valErr := dep.waitForAndRetrieveValidationResult(kid.cc); valErr == nil {
			return &ValidationParameterUpdatedError{
				CC:     kid.cc,
				Coll:   kid.coll,
				Key:    kid.key,
				Height: blockNum,
				Txnum:  txnum,
			}
		}
	}
	return nil
}

/*********************************************************************/
/*********************************************************************/

type KeyLevelValidationParameterManagerImpl struct {
	StateFetcher  validation.StateFetcher
	validationCtx validationContext
}

//ExtractValidationParameterDependency实现了
//keyLevelValidationParameterManager接口的相同名称
//Note that this function doesn't take any namespace argument. 这是
//因为我们要检查此事务所针对的所有命名空间
//修改元数据。
func (m *KeyLevelValidationParameterManagerImpl) ExtractValidationParameterDependency(blockNum, txNum uint64, rwsetBytes []byte) {
	vCtx := m.validationCtx.forBlock(blockNum)

//此对象表示事务（blocknum、txnum）引入的依赖项
	dep := vCtx.getOrCreateDependencyByTxnum(txNum)

	rwset := &rwsetutil.TxRwSet{}
	err := rwset.FromProtoBytes(rwsetBytes)
//注意，我们会悄悄地丢弃损坏的读写
//集合-分类帐仍将使它们无效
	if err == nil {
//在这里，我们循环使用此事务生成的所有元数据更新
//并通知事务（blocknum、txnum）修改它们，以便
//所有后续事务都知道必须等待验证
//事务（blocknum、txnum）才能继续
		for _, rws := range rwset.NsRwSets {
			for _, mw := range rws.KvRwSet.MetadataWrites {
//记录此密钥依赖于我们的Tx的事实
				vCtx.addDependency(newLedgerKeyID(rws.NameSpace, "", mw.Key), txNum, dep)
			}

			for _, cw := range rws.CollHashedRwSets {
				for _, mw := range cw.HashedRwSet.MetadataWrites {
//记录此（pvt）密钥依赖于我们的Tx的事实
					vCtx.addDependency(newLedgerKeyID(rws.NameSpace, cw.CollectionName, string(mw.KeyHash)), txNum, dep)
				}
			}
		}
	} else {
		logger.Warningf("unmarshalling the read write set returned error '%s', skipping", err)
	}

//表示我们已经引入了此事务的所有依赖项
	dep.signalDepInserted()
}

//GetValidationParameterWorkey实现的方法
//keyLevelValidationParameterManager接口的相同名称
func (m *KeyLevelValidationParameterManagerImpl) GetValidationParameterForKey(cc, coll, key string, blockNum, txNum uint64) ([]byte, error) {
	vCtx := m.validationCtx.forBlock(blockNum)

//等到所有的txe都引入依赖项
	for i := int64(txNum) - 1; i >= 0; i-- {
		txdep := vCtx.getOrCreateDependencyByTxnum(uint64(i))
		txdep.waitForDepInserted()
	}

//等待CC命名空间中所有依赖项的验证结果可用。
//如果同时更新了验证参数，则退出
	err := vCtx.waitForValidationResults(newLedgerKeyID(cc, coll, key), blockNum, txNum)
	if err != nil {
		logger.Errorf(err.Error())
		return nil, err
	}

//如果我们在这里，这意味着检索验证是安全的。
//从分类帐中请求的键的参数

	state, err := m.StateFetcher.FetchState()
	if err != nil {
		err = errors.WithMessage(err, "could not retrieve ledger")
		logger.Errorf(err.Error())
		return nil, err
	}
	defer state.Done()

	var mdMap map[string][]byte
	if coll == "" {
		mdMap, err = state.GetStateMetadata(cc, key)
		if err != nil {
			err = errors.WithMessage(err, fmt.Sprintf("could not retrieve metadata for %s:%s", cc, key))
			logger.Errorf(err.Error())
			return nil, err
		}
	} else {
		mdMap, err = state.GetPrivateDataMetadataByHash(cc, coll, []byte(key))
		if err != nil {
			err = errors.WithMessage(err, fmt.Sprintf("could not retrieve metadata for %s:%s:%x", cc, coll, []byte(key)))
			logger.Errorf(err.Error())
			return nil, err
		}
	}

	return mdMap[pb.MetaDataKeys_VALIDATION_PARAMETER.String()], nil
}

//settxvalidationcode实现与
//KeyLevelValidationParameterManager接口。注意
//此函数接收一个命名空间参数，以便记录
//此事务和链码的验证结果。
func (m *KeyLevelValidationParameterManagerImpl) SetTxValidationResult(ns string, blockNum, txNum uint64, err error) {
	vCtx := m.validationCtx.forBlock(blockNum)

//此对象表示调用方的事务引入的依赖项
	dep := vCtx.getOrCreateDependencyByTxnum(txNum)

//发送此Tx的验证状态信号
	dep.signalValidationResult(ns, err)
}
