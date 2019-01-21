
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


package gossip

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/msp/mgmt"
)

var saLogger = flogging.MustGetLogger("peer.gossip.sa")

//MSPSecurityAdvisor实现SecurityAdvisor接口
//使用对等的MSP。
//
//为了使系统安全，必须
//MSP是最新的。频道的MSP通过以下方式更新：
//由订购服务分发的配置事务。
//
//此实现假定这些机制都已就位并起作用。
type mspSecurityAdvisor struct {
	deserializer mgmt.DeserializersManager
}

//NewSecurityAdvisor创建MSPSecurityAdvisor的新实例
//实现了MessageCryptoService
func NewSecurityAdvisor(deserializer mgmt.DeserializersManager) api.SecurityAdvisor {
	return &mspSecurityAdvisor{deserializer: deserializer}
}

//orgByPeerIdentity返回orgIdentityType
//一个给定的对等身份。
//如果出现任何错误，则返回nil。
//此方法不验证对等标识。
//这个验证应该在执行流程中适当地完成。
func (advisor *mspSecurityAdvisor) OrgByPeerIdentity(peerIdentity api.PeerIdentityType) api.OrgIdentityType {
//验证参数
	if len(peerIdentity) == 0 {
		saLogger.Error("Invalid Peer Identity. It must be different from nil.")

		return nil
	}

//请注意，假定peerIdentity是标识的序列化。
//所以，第一步是身份反序列化

//TODO:此方法应返回由两个字段组成的结构：
//标识所属的MSP的mspidentifier之一，
//然后是这个身份所拥有的组织单位的列表。
//对于流言蜚语来说，这是我们现在需要的第一部分，
//即返回标识的msp标识符（identity.getmspidentifier（））

//首先检查本地MSP。
	identity, err := advisor.deserializer.GetLocalDeserializer().DeserializeIdentity([]byte(peerIdentity))
	if err == nil {
		return []byte(identity.GetMSPIdentifier())
	}

//与经理核对
	for chainID, mspManager := range advisor.deserializer.GetChannelDeserializers() {
//反序列化标识
		identity, err := mspManager.DeserializeIdentity([]byte(peerIdentity))
		if err != nil {
			saLogger.Debugf("Failed deserialization identity [% x] on [%s]: [%s]", peerIdentity, chainID, err)
			continue
		}

		return []byte(identity.GetMSPIdentifier())
	}

	saLogger.Warningf("Peer Identity [% x] cannot be desirialized. No MSP found able to do that.", peerIdentity)

	return nil
}
