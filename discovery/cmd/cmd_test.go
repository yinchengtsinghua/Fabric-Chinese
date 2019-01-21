
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


package discovery_test

import (
	"testing"

	"github.com/hyperledger/fabric/discovery/cmd"
	"github.com/hyperledger/fabric/discovery/cmd/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/alecthomas/kingpin.v2"
)

func TestAddCommands(t *testing.T) {
	app := kingpin.New("foo", "bar")
	cli := &mocks.CommandRegistrar{}
	configFunc := mock.AnythingOfType("common.CLICommand")
	cli.On("Command", discovery.PeersCommand, mock.Anything, configFunc).Return(app.Command(discovery.PeersCommand, ""))
	cli.On("Command", discovery.ConfigCommand, mock.Anything, configFunc).Return(app.Command(discovery.ConfigCommand, ""))
	cli.On("Command", discovery.EndorsersCommand, mock.Anything, configFunc).Return(app.Command(discovery.EndorsersCommand, ""))
	discovery.AddCommands(cli)
//确保为子命令配置了服务和通道标志
	for _, cmd := range []string{discovery.PeersCommand, discovery.ConfigCommand, discovery.EndorsersCommand} {
		assert.NotNil(t, app.GetCommand(cmd).GetFlag("server"))
		assert.NotNil(t, app.GetCommand(cmd).GetFlag("channel"))
	}
//确保为背书人调用了chaincode和collection标志。
	assert.NotNil(t, app.GetCommand(discovery.EndorsersCommand).GetFlag("chaincode"))
	assert.NotNil(t, app.GetCommand(discovery.EndorsersCommand).GetFlag("collection"))
}
