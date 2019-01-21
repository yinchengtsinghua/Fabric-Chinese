
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


package peer

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
)

func (cpp *ChaincodeProposalPayload) StaticallyOpaqueFields() []string {
	return []string{"input"}
}

func (cpp *ChaincodeProposalPayload) StaticallyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != cpp.StaticallyOpaqueFields()[0] {
		return nil, fmt.Errorf("not a marshaled field: %s", name)
	}
	return &ChaincodeInvocationSpec{}, nil
}

func (ca *ChaincodeAction) StaticallyOpaqueFields() []string {
	return []string{"results", "events"}
}

func (ca *ChaincodeAction) StaticallyOpaqueFieldProto(name string) (proto.Message, error) {
	switch name {
	case "results":
		return &rwset.TxReadWriteSet{}, nil
	case "events":
		return &ChaincodeEvent{}, nil
	default:
		return nil, fmt.Errorf("not a marshaled field: %s", name)
	}
}
