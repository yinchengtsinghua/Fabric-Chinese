
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


package common

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/peer/common/mock"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var once sync.Once

//init msp初始化msp
func InitMSP() {
	once.Do(initMSP)
}

func initMSP() {
	err := msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		panic(fmt.Errorf("Fatal error when reading MSP config: err %s", err))
	}
}

func TestDeliverClientErrors(t *testing.T) {
	InitMSP()

	mockClient := &mock.DeliverService{}
	o := &DeliverClient{
		Service: mockClient,
	}

//失败-接收返回错误
	mockClient.RecvReturns(nil, errors.New("monkey"))
	block, err := o.readBlock()
	assert.Nil(t, block)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error receiving: monkey")

//失败-接收返回状态
	statusResponse := &ab.DeliverResponse{
		Type: &ab.DeliverResponse_Status{Status: cb.Status_SUCCESS},
	}
	mockClient.RecvReturns(statusResponse, nil)
	block, err = o.readBlock()
	assert.Nil(t, block)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can't read the block")

//失败-recv返回空proto
	mockClient.RecvReturns(&ab.DeliverResponse{}, nil)
	block, err = o.readBlock()
	assert.Nil(t, block)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "response error: unknown type")

//失败-发送返回错误
//获取指定块
	mockClient.SendReturns(errors.New("gorilla"))
	block, err = o.GetSpecifiedBlock(0)
	assert.Nil(t, block)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error getting specified block: gorilla")

//获取最旧的块
	block, err = o.GetOldestBlock()
	assert.Nil(t, block)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error getting oldest block: gorilla")

//获取最新块
	block, err = o.GetNewestBlock()
	assert.Nil(t, block)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error getting newest block: gorilla")
}

func TestNewOrdererDeliverClient(t *testing.T) {
	defer viper.Reset()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()
	InitMSP()

//失败-rootcert文件不存在
	viper.Set("orderer.tls.enabled", true)
	viper.Set("orderer.tls.rootcert.file", "ukelele.crt")
	oc, err := NewDeliverClientForOrderer("ukelele")
	assert.Nil(t, oc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create deliver client: failed to load config for OrdererClient")
}

func TestNewDeliverClientForPeer(t *testing.T) {
	defer viper.Reset()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()
	InitMSP()

//失败-rootcert文件不存在
	viper.Set("peer.tls.enabled", true)
	viper.Set("peer.tls.rootcert.file", "ukelele.crt")
	pc, err := NewDeliverClientForPeer("ukelele")
	assert.Nil(t, pc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create deliver client: failed to load config for PeerClient")
}
