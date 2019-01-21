
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


package lockbasedtxmgr

import (
	"fmt"

	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/pkg/errors"
)

//lockbasedTxSimulator是“lockbasedTxMgr”中使用的事务模拟器。
type lockBasedTxSimulator struct {
	lockBasedQueryExecutor
	rwsetBuilder              *rwsetutil.RWSetBuilder
	writePerformed            bool
	pvtdataQueriesPerformed   bool
	simulationResultsComputed bool
	paginatedQueriesPerformed bool
}

func newLockBasedTxSimulator(txmgr *LockBasedTxMgr, txid string) (*lockBasedTxSimulator, error) {
	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	helper := newQueryHelper(txmgr, rwsetBuilder)
	logger.Debugf("constructing new tx simulator txid = [%s]", txid)
	return &lockBasedTxSimulator{lockBasedQueryExecutor{helper, txid}, rwsetBuilder, false, false, false, false}, nil
}

//setstate在接口“ledger.txsimulator”中实现方法
func (s *lockBasedTxSimulator) SetState(ns string, key string, value []byte) error {
	if err := s.checkWritePrecondition(key, value); err != nil {
		return err
	}
	s.rwsetBuilder.AddToWriteSet(ns, key, value)
	return nil
}

//deleteState在接口“ledger.txSimulator”中实现方法
func (s *lockBasedTxSimulator) DeleteState(ns string, key string) error {
	return s.SetState(ns, key, nil)
}

//setStateEmultipleKeys在接口“ledger.txSimulator”中实现方法
func (s *lockBasedTxSimulator) SetStateMultipleKeys(namespace string, kvs map[string][]byte) error {
	for k, v := range kvs {
		if err := s.SetState(namespace, k, v); err != nil {
			return err
		}
	}
	return nil
}

//setStateMetadata在接口“ledger.txSimulator”中实现方法
func (s *lockBasedTxSimulator) SetStateMetadata(namespace, key string, metadata map[string][]byte) error {
	if err := s.checkWritePrecondition(key, nil); err != nil {
		return err
	}
	s.rwsetBuilder.AddToMetadataWriteSet(namespace, key, metadata)
	return nil
}

//deleteStateMetadata在接口“ledger.txSimulator”中实现方法
func (s *lockBasedTxSimulator) DeleteStateMetadata(namespace, key string) error {
	return s.SetStateMetadata(namespace, key, nil)
}

//setprivatedata在接口“ledger.txsimulator”中实现方法
func (s *lockBasedTxSimulator) SetPrivateData(ns, coll, key string, value []byte) error {
	if err := s.helper.validateCollName(ns, coll); err != nil {
		return err
	}
	if err := s.checkWritePrecondition(key, value); err != nil {
		return err
	}
	s.writePerformed = true
	s.rwsetBuilder.AddToPvtAndHashedWriteSet(ns, coll, key, value)
	return nil
}

//deleteprivatedata在接口'ledger.txsimulator'中实现方法
func (s *lockBasedTxSimulator) DeletePrivateData(ns, coll, key string) error {
	return s.SetPrivateData(ns, coll, key, nil)
}

//setprivatedatamultiplekeys在接口“ledger.txsimulator”中实现方法
func (s *lockBasedTxSimulator) SetPrivateDataMultipleKeys(ns, coll string, kvs map[string][]byte) error {
	for k, v := range kvs {
		if err := s.SetPrivateData(ns, coll, k, v); err != nil {
			return err
		}
	}
	return nil
}

//getprivatedatarangescaniterator在接口'ledger.txsimulator'中实现方法
func (s *lockBasedTxSimulator) GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey string) (commonledger.ResultsIterator, error) {
	if err := s.checkBeforePvtdataQueries(); err != nil {
		return nil, err
	}
	return s.lockBasedQueryExecutor.GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey)
}

//setprivatedatametadata在接口“ledger.txsimulator”中实现方法
func (s *lockBasedTxSimulator) SetPrivateDataMetadata(namespace, collection, key string, metadata map[string][]byte) error {
	if err := s.helper.validateCollName(namespace, collection); err != nil {
		return err
	}
	if err := s.checkWritePrecondition(key, nil); err != nil {
		return err
	}
	s.rwsetBuilder.AddToHashedMetadataWriteSet(namespace, collection, key, metadata)
	return nil
}

//DeletePrivateMetadata implements method in interface `ledger.TxSimulator`
func (s *lockBasedTxSimulator) DeletePrivateDataMetadata(namespace, collection, key string) error {
	return s.SetPrivateDataMetadata(namespace, collection, key, nil)
}

//ExecuteEqueryOnPrivateData在接口“ledger.txSimulator”中实现方法
func (s *lockBasedTxSimulator) ExecuteQueryOnPrivateData(namespace, collection, query string) (commonledger.ResultsIterator, error) {
	if err := s.checkBeforePvtdataQueries(); err != nil {
		return nil, err
	}
	return s.lockBasedQueryExecutor.ExecuteQueryOnPrivateData(namespace, collection, query)
}

//getStateRangeScanIteratorWithMetadata在接口“ledger.queryExecutor”中实现方法
func (s *lockBasedTxSimulator) GetStateRangeScanIteratorWithMetadata(namespace string, startKey string, endKey string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	if err := s.checkBeforePaginatedQueries(); err != nil {
		return nil, err
	}
	return s.lockBasedQueryExecutor.GetStateRangeScanIteratorWithMetadata(namespace, startKey, endKey, metadata)
}

//Excel中的元数据实现了接口'Leiger-QueRealExtuor中的方法
func (s *lockBasedTxSimulator) ExecuteQueryWithMetadata(namespace, query string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	if err := s.checkBeforePaginatedQueries(); err != nil {
		return nil, err
	}
	return s.lockBasedQueryExecutor.ExecuteQueryWithMetadata(namespace, query, metadata)
}

//gettxSimulationResults在接口“ledger.txSimulator”中实现方法
func (s *lockBasedTxSimulator) GetTxSimulationResults() (*ledger.TxSimulationResults, error) {
	if s.simulationResultsComputed {
		return nil, errors.New("this function should only be called once on a transaction simulator instance")
	}
	defer func() { s.simulationResultsComputed = true }()
	logger.Debugf("Simulation completed, getting simulation results")
	if s.helper.err != nil {
		return nil, s.helper.err
	}
	s.helper.addRangeQueryInfo()
	return s.rwsetBuilder.GetTxSimulationResults()
}

//executeUpdate在接口“ledger.txsimulator”中实现方法
func (s *lockBasedTxSimulator) ExecuteUpdate(query string) error {
	return errors.New("not supported")
}

func (s *lockBasedTxSimulator) checkWritePrecondition(key string, value []byte) error {
	if err := s.helper.checkDone(); err != nil {
		return err
	}
	if err := s.checkPvtdataQueryPerformed(); err != nil {
		return err
	}
	if err := s.checkPaginatedQueryPerformed(); err != nil {
		return err
	}
	s.writePerformed = true
	if err := s.helper.txmgr.db.ValidateKeyValue(key, value); err != nil {
		return err
	}
	return nil
}

func (s *lockBasedTxSimulator) checkBeforePvtdataQueries() error {
	if s.writePerformed {
		return &txmgr.ErrUnsupportedTransaction{
			Msg: fmt.Sprintf("txid [%s]: Queries on pvt data is supported only in a read-only transaction", s.txid),
		}
	}
	s.pvtdataQueriesPerformed = true
	return nil
}

func (s *lockBasedTxSimulator) checkPvtdataQueryPerformed() error {
	if s.pvtdataQueriesPerformed {
		return &txmgr.ErrUnsupportedTransaction{
			Msg: fmt.Sprintf("txid [%s]: Transaction has already performed queries on pvt data. Writes are not allowed", s.txid),
		}
	}
	return nil
}

func (s *lockBasedTxSimulator) checkBeforePaginatedQueries() error {
	if s.writePerformed {
		return &txmgr.ErrUnsupportedTransaction{
			Msg: fmt.Sprintf("txid [%s]: Paginated queries are supported only in a read-only transaction", s.txid),
		}
	}
	s.paginatedQueriesPerformed = true
	return nil
}

func (s *lockBasedTxSimulator) checkPaginatedQueryPerformed() error {
	if s.paginatedQueriesPerformed {
		return &txmgr.ErrUnsupportedTransaction{
			Msg: fmt.Sprintf("txid [%s]: Transaction has already performed a paginated query. Writes are not allowed", s.txid),
		}
	}
	return nil
}
