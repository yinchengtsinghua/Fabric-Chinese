
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


package etcdraft

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/orderer"
)

//typekey是一个字符串，用于跨结构标识此一致性实现。
const TypeKey = "etcdraft"

func init() {
	orderer.ConsensusTypeMetadataMap[TypeKey] = ConsensusTypeMetadataFactory{}
}

//ConsensistypeMetadataFactory允许此实现的协议消息注册
//他们的类型和订购者的原型消息。这是原生动物工作所必需的。
type ConsensusTypeMetadataFactory struct{}

//newmessage实现order.conensustypemetadatafactory接口。
func (dogf ConsensusTypeMetadataFactory) NewMessage() proto.Message {
	return &Metadata{}
}

//marshal序列化此实现的原型消息。它由编码器包调用
//在创建医嘱者配置组的过程中。
func Marshal(md *Metadata) ([]byte, error) {
	for _, c := range md.Consenters {
//希望用户将客户端/服务器证书的配置值设置为
//本地持久化它们的路径，然后将这些文件加载到内存中。
		clientCert, err := ioutil.ReadFile(string(c.GetClientTlsCert()))
		if err != nil {
			return nil, fmt.Errorf("cannot load client cert for consenter %s:%d: %s", c.GetHost(), c.GetPort(), err)
		}
		c.ClientTlsCert = clientCert

		serverCert, err := ioutil.ReadFile(string(c.GetServerTlsCert()))
		if err != nil {
			return nil, fmt.Errorf("cannot load server cert for consenter %s:%d: %s", c.GetHost(), c.GetPort(), err)
		}
		c.ServerTlsCert = serverCert
	}
	return proto.Marshal(md)
}
