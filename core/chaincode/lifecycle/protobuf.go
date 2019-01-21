
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


package lifecycle

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

//protobuf定义protobuf生命周期需求的子集，并允许
//用于注入模拟封送错误。
type Protobuf interface {
	Marshal(msg proto.Message) (marshaled []byte, err error)
	Unmarshal(marshaled []byte, msg proto.Message) error
}

//Protobufimpl是用于Protobuf的标准实现
type ProtobufImpl struct{}

//元帅传给原定元帅
func (p ProtobufImpl) Marshal(msg proto.Message) ([]byte, error) {
	res, err := proto.Marshal(msg)
	return res, errors.WithStack(err)
}

//解组传递给Proto。解组
func (p ProtobufImpl) Unmarshal(marshaled []byte, msg proto.Message) error {
	return errors.WithStack(proto.Unmarshal(marshaled, msg))
}
