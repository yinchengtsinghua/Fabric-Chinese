
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


package common_test

import (
	"testing"

	"github.com/hyperledger/fabric/peer/common"
	"github.com/stretchr/testify/assert"
)

func TestGetConfig(t *testing.T) {
	assert := assert.New(t)

//失败-文件名为空
	networkConfig, err := common.GetConfig("")
	assert.Error(err)
	assert.Nil(networkConfig)

//失败-文件不存在
	networkConfig, err = common.GetConfig("fakefile.yaml")
	assert.Error(err)
	assert.Nil(networkConfig)

//失败-连接配置文件中一些bool的意外值
	networkConfig, err = common.GetConfig("testdata/connectionprofile-bad.yaml")
	assert.Error(err, "error should have been nil")
	assert.Nil(networkConfig, "network config should be set")

//成功
	networkConfig, err = common.GetConfig("testdata/connectionprofile.yaml")
	assert.NoError(err, "error should have been nil")
	assert.NotNil(networkConfig, "network config should be set")
	assert.Equal(networkConfig.Name, "connection-profile")

	channelPeers := networkConfig.Channels["mychannel"].Peers
	assert.Equal(len(channelPeers), 2)
	for _, peer := range channelPeers {
		assert.True(peer.EndorsingPeer)
	}

	peers := networkConfig.Peers
	assert.Equal(len(peers), 2)
	for _, peer := range peers {
		assert.NotEmpty(peer.TLSCACerts.Path)
	}
}
