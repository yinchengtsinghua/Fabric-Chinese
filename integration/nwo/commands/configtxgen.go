
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


package commands

type OutputBlock struct {
	ChannelID   string
	Profile     string
	ConfigPath  string
	OutputBlock string
}

func (o OutputBlock) SessionName() string {
	return "configtxgen-output-block"
}

func (o OutputBlock) Args() []string {
	return []string{
		"-channelID", o.ChannelID,
		"-profile", o.Profile,
		"-configPath", o.ConfigPath,
		"-outputBlock", o.OutputBlock,
	}
}

type CreateChannelTx struct {
	ChannelID             string
	Profile               string
	ConfigPath            string
	OutputCreateChannelTx string
}

func (c CreateChannelTx) SessionName() string {
	return "configtxgen-create-channel-tx"
}

func (c CreateChannelTx) Args() []string {
	return []string{
		"-channelID", c.ChannelID,
		"-profile", c.Profile,
		"-configPath", c.ConfigPath,
		"-outputCreateChannelTx", c.OutputCreateChannelTx,
	}
}

type OutputAnchorPeersUpdate struct {
	ChannelID               string
	Profile                 string
	ConfigPath              string
	AsOrg                   string
	OutputAnchorPeersUpdate string
}

func (o OutputAnchorPeersUpdate) SessionName() string {
	return "configtxgen-output-anchor-peers-update"
}

func (o OutputAnchorPeersUpdate) Args() []string {
	return []string{
		"-channelID", o.ChannelID,
		"-profile", o.Profile,
		"-configPath", o.ConfigPath,
		"-asOrg", o.AsOrg,
		"-outputAnchorPeersUpdate", o.OutputAnchorPeersUpdate,
	}
}
