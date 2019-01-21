
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


package pvtstatepurgemgmt

import (
	fmt "fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/stretchr/testify/assert"
)

func TestExpiryKVEncoding(t *testing.T) {
	pvtdataKeys := newPvtdataKeys()
	pvtdataKeys.add("ns1", "coll-1", "key-1", []byte("key-1-hash"))
	expiryInfo := &expiryInfo{&expiryInfoKey{expiryBlk: 10, committingBlk: 2}, pvtdataKeys}
	t.Logf("expiryInfo:%s", spew.Sdump(expiryInfo))
	k, v, err := encodeKV(expiryInfo)
	assert.NoError(t, err)
	expiryInfo1, err := decodeExpiryInfo(k, v)
	assert.NoError(t, err)
	assert.Equal(t, expiryInfo.expiryInfoKey, expiryInfo1.expiryInfoKey)
	assert.True(t, proto.Equal(expiryInfo.pvtdataKeys, expiryInfo1.pvtdataKeys), "proto messages are not equal")
}

func TestExpiryKeeper(t *testing.T) {
	testenv := bookkeeping.NewTestEnv(t)
	defer testenv.Cleanup()
	expiryKeeper := newExpiryKeeper("testledger", testenv.TestProvider)

	expinfo1 := &expiryInfo{&expiryInfoKey{committingBlk: 3, expiryBlk: 13}, buildPvtdataKeysForTest(1, 1)}
	expinfo2 := &expiryInfo{&expiryInfoKey{committingBlk: 3, expiryBlk: 15}, buildPvtdataKeysForTest(2, 2)}
	expinfo3 := &expiryInfo{&expiryInfoKey{committingBlk: 4, expiryBlk: 13}, buildPvtdataKeysForTest(3, 3)}
	expinfo4 := &expiryInfo{&expiryInfoKey{committingBlk: 5, expiryBlk: 17}, buildPvtdataKeysForTest(4, 4)}

//在提交BLK 3时插入密钥条目
	expiryKeeper.updateBookkeeping([]*expiryInfo{expinfo1, expinfo2}, nil)
//在提交BLK 4和5时插入密钥条目
	expiryKeeper.updateBookkeeping([]*expiryInfo{expinfo3, expinfo4}, nil)

//通过过期块13、15和17检索条目
	listExpinfo1, _ := expiryKeeper.retrieve(13)
	assert.Len(t, listExpinfo1, 2)
	assert.Equal(t, expinfo1.expiryInfoKey, listExpinfo1[0].expiryInfoKey)
	assert.True(t, proto.Equal(expinfo1.pvtdataKeys, listExpinfo1[0].pvtdataKeys))
	assert.Equal(t, expinfo3.expiryInfoKey, listExpinfo1[1].expiryInfoKey)
	assert.True(t, proto.Equal(expinfo3.pvtdataKeys, listExpinfo1[1].pvtdataKeys))

	listExpinfo2, _ := expiryKeeper.retrieve(15)
	assert.Len(t, listExpinfo2, 1)
	assert.Equal(t, expinfo2.expiryInfoKey, listExpinfo2[0].expiryInfoKey)
	assert.True(t, proto.Equal(expinfo2.pvtdataKeys, listExpinfo2[0].pvtdataKeys))

	listExpinfo3, _ := expiryKeeper.retrieve(17)
	assert.Len(t, listExpinfo3, 1)
	assert.Equal(t, expinfo4.expiryInfoKey, listExpinfo3[0].expiryInfoKey)
	assert.True(t, proto.Equal(expinfo4.pvtdataKeys, listExpinfo3[0].pvtdataKeys))

//清除在块13和块15过期的密钥的条目，然后通过过期块13、块15和块17再次检索
	expiryKeeper.updateBookkeeping(nil, []*expiryInfoKey{expinfo1.expiryInfoKey, expinfo2.expiryInfoKey, expinfo3.expiryInfoKey})
	listExpinfo4, _ := expiryKeeper.retrieve(13)
	assert.Nil(t, listExpinfo4)

	listExpinfo5, _ := expiryKeeper.retrieve(15)
	assert.Nil(t, listExpinfo5)

	listExpinfo6, _ := expiryKeeper.retrieve(17)
	assert.Len(t, listExpinfo6, 1)
	assert.Equal(t, expinfo4.expiryInfoKey, listExpinfo6[0].expiryInfoKey)
	assert.True(t, proto.Equal(expinfo4.pvtdataKeys, listExpinfo6[0].pvtdataKeys))
}

func buildPvtdataKeysForTest(startingEntry int, numEntries int) *PvtdataKeys {
	pvtdataKeys := newPvtdataKeys()
	for i := startingEntry; i <= startingEntry+numEntries; i++ {
		pvtdataKeys.add(fmt.Sprintf("ns-%d", i), fmt.Sprintf("coll-%d", i), fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf("key-%d-hash", i)))
	}
	return pvtdataKeys
}
