
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

type Peers struct {
	UserCert string
	UserKey  string
	MSPID    string
	Server   string
	Channel  string
}

func (p Peers) SessionName() string {
	return "discover-peers"
}

func (p Peers) Args() []string {
	return []string{
		"--userCert", p.UserCert,
		"--userKey", p.UserKey,
		"--MSP", p.MSPID,
		"peers",
		"--server", p.Server,
		"--channel", p.Channel,
	}
}

type Config struct {
	UserCert string
	UserKey  string
	MSPID    string
	Server   string
	Channel  string
}

func (c Config) SessionName() string {
	return "discover-config"
}

func (c Config) Args() []string {
	return []string{
		"--userCert", c.UserCert,
		"--userKey", c.UserKey,
		"--MSP", c.MSPID,
		"config",
		"--server", c.Server,
		"--channel", c.Channel,
	}
}

type Endorsers struct {
	UserCert    string
	UserKey     string
	MSPID       string
	Server      string
	Channel     string
	Chaincode   string
	Chaincodes  []string
	Collection  string
	Collections []string
}

func (e Endorsers) SessionName() string {
	return "discover-endorsers"
}

func (e Endorsers) Args() []string {
	args := []string{
		"--userCert", e.UserCert,
		"--userKey", e.UserKey,
		"--MSP", e.MSPID,
		"endorsers",
		"--server", e.Server,
		"--channel", e.Channel,
	}
	if e.Chaincode != "" {
		args = append(args, "--chaincode", e.Chaincode)
	}
	for _, cc := range e.Chaincodes {
		args = append(args, "--chaincode", cc)
	}
	if e.Collection != "" {
		args = append(args, "--collection", e.Collection)
	}
	for _, c := range e.Collections {
		args = append(args, "--collection", c)
	}
	return args
}
