
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

package client

import "github.com/pkg/errors"

//connectionconfig包含与对等方或订购方建立GRPC连接所需的数据
type ConnectionConfig struct {
	Address            string
	TlsRootCertFile    string
	ServerNameOverride string
}

//在合并令牌客户端配置的CR后，将更新client config，其中config数据
//将根据配置文件填充。
type ClientConfig struct {
	ChannelId     string
	MspDir        string
	MspId         string
	TlsEnabled    bool
	OrdererCfg    ConnectionConfig
	CommitPeerCfg ConnectionConfig
	ProverPeerCfg ConnectionConfig
}

func ValidateClientConfig(config *ClientConfig) error {
	if config == nil {
		return errors.New("client config is nil")
	}
	if config.ChannelId == "" {
		return errors.New("missing channelId")
	}

	if config.OrdererCfg.Address == "" {
		return errors.New("missing orderer address")
	}

	if config.TlsEnabled && config.OrdererCfg.TlsRootCertFile == "" {
		return errors.New("missing orderer TlsRootCertFile")
	}

	if config.OrdererCfg.Address == "" {
		return errors.New("missing commit peer address")
	}

	if config.TlsEnabled && config.OrdererCfg.TlsRootCertFile == "" {
		return errors.New("missing commit peer TlsRootCertFile")
	}

//TODO:在其他CR中添加验证程序对等验证
	return nil
}
