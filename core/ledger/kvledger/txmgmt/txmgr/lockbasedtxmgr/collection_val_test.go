
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
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/stretchr/testify/assert"
)

func TestCollectionValidation(t *testing.T) {
	testEnv := testEnvsMap[levelDBtestEnvName]
	testEnv.init(t, "testLedger", nil)
	defer testEnv.cleanup()
	txMgr := testEnv.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr),
		[]collConfigkey{
			{"ns1", "coll1"},
			{"ns1", "coll2"},
			{"ns2", "coll1"},
			{"ns2", "coll2"},
		},
		version.NewHeight(1, 1),
	)

	sim, err := txMgr.NewTxSimulator("tx-id1")
	assert.NoError(t, err)

	_, err = sim.GetPrivateData("ns3", "coll1", "key1")
	_, ok := err.(*ledger.CollConfigNotDefinedError)
	assert.True(t, ok)

	err = sim.SetPrivateData("ns3", "coll1", "key1", []byte("val1"))
	_, ok = err.(*ledger.CollConfigNotDefinedError)
	assert.True(t, ok)

	_, err = sim.GetPrivateData("ns1", "coll3", "key1")
	_, ok = err.(*ledger.InvalidCollNameError)
	assert.True(t, ok)

	err = sim.SetPrivateData("ns1", "coll3", "key1", []byte("val1"))
	_, ok = err.(*ledger.InvalidCollNameError)
	assert.True(t, ok)

	err = sim.SetPrivateData("ns1", "coll1", "key1", []byte("val1"))
	assert.NoError(t, err)
}

func TestPvtGetNoCollection(t *testing.T) {
	testEnv := testEnvs[0]
	testEnv.init(t, "test-pvtdata-get-no-collection", nil)
	defer testEnv.cleanup()
	txMgr := testEnv.getTxMgr().(*LockBasedTxMgr)
	queryHelper := newQueryHelper(txMgr, nil)
	valueHash, metadataBytes, err := queryHelper.getPrivateDataValueHash("cc", "coll", "key")
	assert.Nil(t, valueHash)
	assert.Nil(t, metadataBytes)
	assert.Error(t, err)
	assert.IsType(t, &ledger.CollConfigNotDefinedError{}, err)
}

func TestPvtPutNoCollection(t *testing.T) {
	testEnv := testEnvs[0]
	testEnv.init(t, "test-pvtdata-put-no-collection", nil)
	defer testEnv.cleanup()
	txMgr := testEnv.getTxMgr().(*LockBasedTxMgr)
	txsim, err := txMgr.NewTxSimulator("txid")
	assert.NoError(t, err)
	err = txsim.SetPrivateDataMetadata("cc", "coll", "key", map[string][]byte{})
	assert.Error(t, err)
	assert.IsType(t, &ledger.CollConfigNotDefinedError{}, err)
}
