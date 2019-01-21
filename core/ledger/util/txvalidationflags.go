
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


package util

import (
	"github.com/hyperledger/fabric/protos/peer"
)

//TxValidationFlags是事务验证代码的数组。它在提交者验证块时使用。
type TxValidationFlags []uint8

//newtxvalidationflags创建具有目标大小的验证代码的新对象数组。
//默认值：txvalidationcode_未验证
func NewTxValidationFlags(size int) TxValidationFlags {
	return newTxValidationFlagsSetValue(size, peer.TxValidationCode_NOT_VALIDATED)
}

//newtxvalidationFlagsetValue创建具有目标大小的验证代码的新对象数组
//以及提供的价值
func NewTxValidationFlagsSetValue(size int, value peer.TxValidationCode) TxValidationFlags {
	return newTxValidationFlagsSetValue(size, value)
}

func newTxValidationFlagsSetValue(size int, value peer.TxValidationCode) TxValidationFlags {
	inst := make(TxValidationFlags, size)
	for i := range inst {
		inst[i] = uint8(value)
	}

	return inst
}

//setflag将验证代码分配给指定的事务
func (obj TxValidationFlags) SetFlag(txIndex int, flag peer.TxValidationCode) {
	obj[txIndex] = uint8(flag)
}

//标志返回指定事务的验证代码
func (obj TxValidationFlags) Flag(txIndex int) peer.TxValidationCode {
	return peer.TxValidationCode(obj[txIndex])
}

//ISvalid检查指定的事务是否有效
func (obj TxValidationFlags) IsValid(txIndex int) bool {
	return obj.IsSetTo(txIndex, peer.TxValidationCode_VALID)
}

//如果指定的事务无效，则执行ISINVALID检查
func (obj TxValidationFlags) IsInvalid(txIndex int) bool {
	return !obj.IsValid(txIndex)
}

//如果指定的事务等于标志，则IsSetto返回true；否则返回false。
func (obj TxValidationFlags) IsSetTo(txIndex int, flag peer.TxValidationCode) bool {
	return obj.Flag(txIndex) == flag
}
