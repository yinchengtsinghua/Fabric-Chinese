
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


package mgmt

import (
	"testing"

	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/hyperledger/fabric/msp"
	"github.com/stretchr/testify/assert"
)

func TestGetManagerForChains(t *testing.T) {
//此呼叫之前不存在频道的MSPManager
	mspMgr1 := GetManagerForChain("test")
//确保设置了MSPManager
	if mspMgr1 == nil {
		t.Fatal("mspMgr1 fail")
	}

//频道的MSPManager现已存在
	mspMgr2 := GetManagerForChain("test")
//确保MSPManager返回的结果与第一个结果匹配
	if mspMgr2 != mspMgr1 {
		t.Fatal("mspMgr2 != mspMgr1 fail")
	}
}

func TestGetManagerForChains_usingMSPConfigHandlers(t *testing.T) {
	XXXSetMSPManager("foo", msp.NewMSPManager())
	msp2 := GetManagerForChain("foo")
//应设置返回值，因为MSPManager已初始化
	if msp2 == nil {
		t.FailNow()
	}
}

func TestGetIdentityDeserializer(t *testing.T) {
	XXXSetMSPManager("baz", msp.NewMSPManager())
	ids := GetIdentityDeserializer("baz")
	assert.NotNil(t, ids)
	ids = GetIdentityDeserializer("")
	assert.NotNil(t, ids)
}

func TestGetLocalSigningIdentityOrPanic(t *testing.T) {
	sid := GetLocalSigningIdentityOrPanic()
	assert.NotNil(t, sid)
}

func TestUpdateLocalMspCache(t *testing.T) {
//重置localmsp以强制在第一次调用时对其进行初始化
	localMsp = nil

//第一个调用应初始化本地MSP并返回缓存版本
	firstMsp := GetLocalMSP()
//第二个调用应返回相同的
	secondMsp := GetLocalMSP()
//第三次呼叫应返回相同的
	thirdMsp := GetLocalMSP()

//同一（如果未修补，则为非缓存）实例
	if thirdMsp != secondMsp {
		t.Fatalf("thirdMsp != secondMsp")
	}
//第一个（缓存的）和第二个（非缓存的）不同，除非修补
	if firstMsp != secondMsp {
		t.Fatalf("firstMsp != secondMsp")
	}
}

func TestNewMSPMgmtMgr(t *testing.T) {
	err := LoadMSPSetupForTesting()
	assert.Nil(t, err)

//测试不存在的通道
	mspMgmtMgr := GetManagerForChain("fake")

	id := GetLocalSigningIdentityOrPanic()
	assert.NotNil(t, id)

	serializedID, err := id.Serialize()
	if err != nil {
		t.Fatalf("Serialize should have succeeded, got err %s", err)
		return
	}

	idBack, err := mspMgmtMgr.DeserializeIdentity(serializedID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel doesn't exist")
	assert.Nil(t, idBack, "deserialized identity should have been nil")

//现有渠道测试
	mspMgmtMgr = GetManagerForChain(util.GetTestChainID())

	id = GetLocalSigningIdentityOrPanic()
	assert.NotNil(t, id)

	serializedID, err = id.Serialize()
	if err != nil {
		t.Fatalf("Serialize should have succeeded, got err %s", err)
		return
	}

	idBack, err = mspMgmtMgr.DeserializeIdentity(serializedID)
	assert.NoError(t, err)
	assert.NotNil(t, idBack, "deserialized identity should not have been nil")
}

func LoadMSPSetupForTesting() error {
	dir, err := configtest.GetDevMspDir()
	if err != nil {
		return err
	}
	conf, err := msp.GetLocalMspConfig(dir, nil, "SampleOrg")
	if err != nil {
		return err
	}

	err = GetLocalMSP().Setup(conf)
	if err != nil {
		return err
	}

	err = GetManagerForChain(util.GetTestChainID()).Setup([]msp.MSP{GetLocalMSP()})
	if err != nil {
		return err
	}

	return nil
}
