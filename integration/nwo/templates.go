
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


package nwo

//模板可用于提供自定义模板以生成配置树。
type Templates struct {
	ConfigTx string `yaml:"configtx,omitempty"`
	Core     string `yaml:"core,omitempty"`
	Crypto   string `yaml:"crypto,omitempty"`
	Orderer  string `yaml:"orderer,omitempty"`
}

func (t *Templates) ConfigTxTemplate() string {
	if t.ConfigTx != "" {
		return t.ConfigTx
	}
	return DefaultConfigTxTemplate
}

func (t *Templates) CoreTemplate() string {
	if t.Core != "" {
		return t.Core
	}
	return DefaultCoreTemplate
}

func (t *Templates) CryptoTemplate() string {
	if t.Crypto != "" {
		return t.Crypto
	}
	return DefaultCryptoTemplate
}

func (t *Templates) OrdererTemplate() string {
	if t.Orderer != "" {
		return t.Orderer
	}
	return DefaultOrdererTemplate
}
