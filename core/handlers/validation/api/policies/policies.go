
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


package validation

import (
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	"github.com/hyperledger/fabric/protos/common"
)

//政策评估者评估政策
type PolicyEvaluator interface {
	validation.Dependency

//Evaluate获取一组SignedData，并评估该组签名是否满足
//具有给定字节的策略
	Evaluate(policyBytes []byte, signatureSet []*common.SignedData) error
}

//序列化策略定义序列化策略
type SerializedPolicy interface {
	validation.ContextDatum

//bytes返回序列化策略的字节数
	Bytes() []byte
}
