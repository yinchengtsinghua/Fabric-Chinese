
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


package endorsement

import (
	"github.com/hyperledger/fabric/protos/peer"
)

//参数定义了用于背书的参数
type Argument interface {
	Dependency
//arg返回参数的字节数
	Arg() []byte
}

//依赖项标记传递给init（）方法的依赖项
type Dependency interface {
}

//插件认可建议响应
type Plugin interface {
//认可对给定的有效负载（proposalResponsePayLoad字节）进行签名，并可选地对其进行变异。
//返回：
//背书：有效载荷上的签名，以及用于验证签名的标识。
//作为输入给出的有效负载（可以在此函数中修改）
//或失败时出错
	Endorse(payload []byte, sp *peer.SignedProposal) (*peer.Endorsement, []byte, error)

//init将依赖项插入插件的实例中
	Init(dependencies ...Dependency) error
}

//PluginFactory创建插件的新实例
type PluginFactory interface {
	New() Plugin
}
