
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp，SecureKey Technologies Inc.保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package decoration

import (
	"github.com/hyperledger/fabric/protos/peer"
)

//decorator修饰链码输入
type Decorator interface {
//修饰通过更改链码输入来修饰它
	Decorate(proposal *peer.Proposal, input *peer.ChaincodeInput) *peer.ChaincodeInput
}

//按提供的顺序应用装饰
func Apply(proposal *peer.Proposal, input *peer.ChaincodeInput,
	decorators ...Decorator) *peer.ChaincodeInput {
	for _, decorator := range decorators {
		input = decorator.Decorate(proposal, input)
	}

	return input
}
