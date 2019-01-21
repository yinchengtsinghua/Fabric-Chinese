
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


package config

import (
	"fmt"
	"net"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/flogging"
	mspconstants "github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/discovery"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("discovery.config")

//CurrentConfigBlockGetter允许获取最后一个配置块
type CurrentConfigBlockGetter interface {
//getcurrconfigblock返回给定通道的当前配置块
	GetCurrConfigBlock(channel string) *common.Block
}

//CurrentConfigBlockGetterFunc enables to fetch the last config block
type CurrentConfigBlockGetterFunc func(channel string) *common.Block

//CurrentConfigBlockGetterfUnc允许获取最后一个配置块
func (f CurrentConfigBlockGetterFunc) GetCurrConfigBlock(channel string) *common.Block {
	return f(channel)
}

//DiscoverySupport实现用于服务发现的支持
//与配置有关
type DiscoverySupport struct {
	CurrentConfigBlockGetter
}

//新建DiscoverySupport创建新的DiscoverySupport
func NewDiscoverySupport(getLastConfigBlock CurrentConfigBlockGetter) *DiscoverySupport {
	return &DiscoverySupport{
		CurrentConfigBlockGetter: getLastConfigBlock,
	}
}

//config返回通道的配置
func (s *DiscoverySupport) Config(channel string) (*discovery.ConfigResult, error) {
	block := s.GetCurrConfigBlock(channel)
	if block == nil {
		return nil, errors.Errorf("could not get last config block for channel %s", channel)
	}
	if block.Data == nil || len(block.Data.Data) == 0 {
		return nil, errors.Errorf("no transactions in block")
	}
	env := &common.Envelope{}
	if err := proto.Unmarshal(block.Data.Data[0], env); err != nil {
		return nil, errors.Wrap(err, "failed unmarshaling envelope")
	}
	pl := &common.Payload{}
	if err := proto.Unmarshal(env.Payload, pl); err != nil {
		return nil, errors.Wrap(err, "failed unmarshaling payload")
	}
	ce := &common.ConfigEnvelope{}
	if err := proto.Unmarshal(pl.Data, ce); err != nil {
		return nil, errors.Wrap(err, "failed unmarshaling config envelope")
	}

	if err := ValidateConfigEnvelope(ce); err != nil {
		return nil, errors.Wrap(err, "config envelope is invalid")
	}

	res := &discovery.ConfigResult{
		Msps:     make(map[string]*msp.FabricMSPConfig),
		Orderers: make(map[string]*discovery.Endpoints),
	}
	ordererGrp := ce.Config.ChannelGroup.Groups[channelconfig.OrdererGroupKey].Groups
	appGrp := ce.Config.ChannelGroup.Groups[channelconfig.ApplicationGroupKey].Groups

	ordererAddresses := &common.OrdererAddresses{}
	if err := proto.Unmarshal(ce.Config.ChannelGroup.Values[channelconfig.OrdererAddressesKey].Value, ordererAddresses); err != nil {
		return nil, errors.Wrap(err, "failed unmarshaling orderer addresses")
	}

	ordererEndpoints, err := computeOrdererEndpoints(ordererGrp, ordererAddresses)
	if err != nil {
		return nil, errors.Wrap(err, "failed computing orderer addresses")
	}
	res.Orderers = ordererEndpoints

	if err := appendMSPConfigs(ordererGrp, appGrp, res.Msps); err != nil {
		return nil, errors.WithStack(err)
	}
	return res, nil

}

func computeOrdererEndpoints(ordererGrp map[string]*common.ConfigGroup, ordererAddresses *common.OrdererAddresses) (map[string]*discovery.Endpoints, error) {
	res := make(map[string]*discovery.Endpoints)
	for name, group := range ordererGrp {
		mspConfig := &msp.MSPConfig{}
		if err := proto.Unmarshal(group.Values[channelconfig.MSPKey].Value, mspConfig); err != nil {
			return nil, errors.Wrap(err, "failed parsing MSPConfig")
		}
//跳过非结构的MSP，因为它们不携带用于服务发现的有用信息。
//IDemix MSP不应出现在医嘱者组中，但这不是致命错误
//对于发现服务，我们可以忽略它。
		if mspConfig.Type != int32(mspconstants.FABRIC) {
			logger.Error("Orderer group", name, "is not a FABRIC MSP, but is of type", mspConfig.Type)
			continue
		}
		fabricConfig := &msp.FabricMSPConfig{}
		if err := proto.Unmarshal(mspConfig.Config, fabricConfig); err != nil {
			return nil, errors.Wrap(err, "failed marshaling FabricMSPConfig")
		}
		res[fabricConfig.Name] = &discovery.Endpoints{}
		for _, endpoint := range ordererAddresses.Addresses {
			host, portStr, err := net.SplitHostPort(endpoint)
			if err != nil {
				return nil, errors.Errorf("failed parsing orderer endpoint %s", endpoint)
			}
			port, err := strconv.ParseInt(portStr, 10, 32)
			if err != nil {
				return nil, errors.Errorf("%s is not a valid port number", portStr)
			}
			res[fabricConfig.Name].Endpoint = append(res[fabricConfig.Name].Endpoint, &discovery.Endpoint{
				Host: host,
				Port: uint32(port),
			})
		}
	}
	return res, nil
}

func appendMSPConfigs(ordererGrp, appGrp map[string]*common.ConfigGroup, output map[string]*msp.FabricMSPConfig) error {
	for _, group := range []map[string]*common.ConfigGroup{ordererGrp, appGrp} {
		for _, grp := range group {
			mspConfig := &msp.MSPConfig{}
			if err := proto.Unmarshal(grp.Values[channelconfig.MSPKey].Value, mspConfig); err != nil {
				return errors.Wrap(err, "failed parsing MSPConfig")
			}
//跳过非结构MSP，因为它们不携带用于服务发现的有用信息
			if mspConfig.Type != int32(mspconstants.FABRIC) {
				continue
			}
			fabricConfig := &msp.FabricMSPConfig{}
			if err := proto.Unmarshal(mspConfig.Config, fabricConfig); err != nil {
				return errors.Wrap(err, "failed marshaling FabricMSPConfig")
			}
			if _, exists := output[fabricConfig.Name]; exists {
				continue
			}
			output[fabricConfig.Name] = fabricConfig
		}
	}

	return nil
}

func ValidateConfigEnvelope(ce *common.ConfigEnvelope) error {
	if ce.Config == nil {
		return fmt.Errorf("field Config is nil")
	}
	if ce.Config.ChannelGroup == nil {
		return fmt.Errorf("field Config.ChannelGroup is nil")
	}
	grps := ce.Config.ChannelGroup.Groups
	if grps == nil {
		return fmt.Errorf("field Config.ChannelGroup.Groups is nil")
	}
	for _, field := range []string{channelconfig.OrdererGroupKey, channelconfig.ApplicationGroupKey} {
		grp, exists := grps[field]
		if !exists {
			return fmt.Errorf("key Config.ChannelGroup.Groups[%s] is missing", field)
		}
		if grp.Groups == nil {
			return fmt.Errorf("key Config.ChannelGroup.Groups[%s].Groups is nil", field)
		}
	}
	if ce.Config.ChannelGroup.Values == nil {
		return fmt.Errorf("field Config.ChannelGroup.Values is nil")
	}
	if _, exists := ce.Config.ChannelGroup.Values[channelconfig.OrdererAddressesKey]; !exists {
		return fmt.Errorf("field Config.ChannelGroup.Values is empty")
	}
	return nil
}
