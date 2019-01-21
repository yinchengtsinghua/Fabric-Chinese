
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//Mokery v1.0.0生成的代码
package mocks

import ledger "github.com/hyperledger/fabric/core/ledger"
import mock "github.com/stretchr/testify/mock"
import protostransientstore "github.com/hyperledger/fabric/protos/transientstore"
import rwset "github.com/hyperledger/fabric/protos/ledger/rwset"
import transientstore "github.com/hyperledger/fabric/core/transientstore"

//存储是存储类型的自动生成的模拟类型
type Store struct {
	mock.Mock
}

//GetMinTransientBlkht提供具有给定字段的模拟函数：
func (_m *Store) GetMinTransientBlkHt() (uint64, error) {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

//gettxpvtrwsetbytxid提供具有给定字段的模拟函数：txid、filter
func (_m *Store) GetTxPvtRWSetByTxid(txid string, filter ledger.PvtNsCollFilter) (transientstore.RWSetScanner, error) {
	ret := _m.Called(txid, filter)

	var r0 transientstore.RWSetScanner
	if rf, ok := ret.Get(0).(func(string, ledger.PvtNsCollFilter) transientstore.RWSetScanner); ok {
		r0 = rf(txid, filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(transientstore.RWSetScanner)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, ledger.PvtNsCollFilter) error); ok {
		r1 = rf(txid, filter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

//Persist为给定字段提供模拟函数：txid、blockheight、privateSimulationResults
func (_m *Store) Persist(txid string, blockHeight uint64, privateSimulationResults *rwset.TxPvtReadWriteSet) error {
	ret := _m.Called(txid, blockHeight, privateSimulationResults)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, uint64, *rwset.TxPvtReadWriteSet) error); ok {
		r0 = rf(txid, blockHeight, privateSimulationResults)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

//PersisteWithConfig提供具有给定字段的模拟函数：txid、blockheight、privateSimulationResultsWithConfig
func (_m *Store) PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *protostransientstore.TxPvtReadWriteSetWithConfigInfo) error {
	ret := _m.Called(txid, blockHeight, privateSimulationResultsWithConfig)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, uint64, *protostransientstore.TxPvtReadWriteSetWithConfigInfo) error); ok {
		r0 = rf(txid, blockHeight, privateSimulationResultsWithConfig)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

//purgeByHeight提供了一个具有给定字段的模拟函数：maxBlockNumtoretain
func (_m *Store) PurgeByHeight(maxBlockNumToRetain uint64) error {
	ret := _m.Called(maxBlockNumToRetain)

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64) error); ok {
		r0 = rf(maxBlockNumToRetain)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

//purgebytxids提供了一个具有给定字段的模拟函数：txids
func (_m *Store) PurgeByTxids(txids []string) error {
	ret := _m.Called(txids)

	var r0 error
	if rf, ok := ret.Get(0).(func([]string) error); ok {
		r0 = rf(txids)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

//Shutdown为给定字段提供模拟函数：
func (_m *Store) Shutdown() {
	_m.Called()
}
