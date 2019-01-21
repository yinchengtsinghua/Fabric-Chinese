
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


package msgprocessor

import (
	"fmt"

	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
)

//支持定义创建此筛选器所需的通道支持的子集
type Support interface {
	BatchSize() *ab.BatchSize
}

//新建创建一个大小筛选器，该筛选器拒绝大于maxbytes的邮件
func NewSizeFilter(support Support) *MaxBytesRule {
	return &MaxBytesRule{support: support}
}

//MaxBytesRule实现规则接口。
type MaxBytesRule struct {
	support Support
}

//如果消息超过配置的绝对最大批处理大小，则APPLY返回错误。
func (r *MaxBytesRule) Apply(message *cb.Envelope) error {
	maxBytes := r.support.BatchSize().AbsoluteMaxBytes
	if size := messageByteSize(message); size > maxBytes {
		return fmt.Errorf("message payload is %d bytes and exceeds maximum allowed %d bytes", size, maxBytes)
	}
	return nil
}

func messageByteSize(message *cb.Envelope) uint32 {
//这是一个很好的近似值，但由于proto封送处理中的字段说明符，因此将短一些字节。
//这可能需要填充以确定真正的精确编组大小
	return uint32(len(message.Payload) + len(message.Signature))
}
