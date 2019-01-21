
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


package statebased

import (
	"fmt"
)

//每当
//键的验证参数不能是
//因为它们正在更新
type ValidationParameterUpdatedError struct {
	CC     string
	Coll   string
	Key    string
	Height uint64
	Txnum  uint64
}

func (f *ValidationParameterUpdatedError) Error() string {
	return fmt.Sprintf("validation parameters for key [%s] in namespace [%s:%s] have been changed in transaction %d of block %d", f.Key, f.CC, f.Coll, f.Txnum, f.Height)
}

//验证插件依次使用keyLevelValidationParameterManager
//检索各个KVS密钥的验证参数。
//应按以下顺序调用函数：
//1）为验证某个Tx而调用的验证插件调用ExtractValidationParameterDependency
//以便经理能够确定分类帐中的验证参数
//可以使用，或者是否由该块中的事务更新。
//2）验证插件对GetValidationParameterWorkey发出0个或多个调用。
//3) the validation plugin determines the validation code for the tx and calls SetTxValidationCode.
type KeyLevelValidationParameterManager interface {
//GetValidationParameterWorkey返回的验证参数
//在指定块上由（cc，coll，key）标识的提供的kvs密钥
//height h. The function returns the validation parameter and no error in case of
//成功，否则为零，否则为错误。一个可能是
//返回的是validationParameterUpdatedErr，如果
//给定kvs密钥的验证参数已被事务更改
//with txNum smaller than the one supplied by the caller. This protects from a
//事务更改验证参数标记为有效的方案
//由VSCC执行，随后由于其他原因（如MVCC）被提交人取消。
//冲突）。此功能可能被阻止，直到有足够的信息
//已通过（通过调用ApplyRwsetUpdates和ApplyValidateRwsetupdates）的
//所有TxNum小于调用方提供的TxE。
	GetValidationParameterForKey(cc, coll, key string, blockNum, txNum uint64) ([]byte, error)

//ExtractValidationParameterDependency用于确定哪些验证参数是
//由高度为'blocknum，txnum'的事务更新。这是需要的
//确定哪些Txe依赖于特定的验证参数，并将
//确定GetValidationParameterWorksey是否可以阻止。
	ExtractValidationParameterDependency(blockNum, txNum uint64, rwset []byte)

//settxvalidationresult设置高度事务的验证结果
//`blocknum，txnum`用于指定的链码'cc`。
//这用于确定依赖项是否由
//是否提取验证参数依赖项。
	SetTxValidationResult(cc string, blockNum, txNum uint64, err error)
}
