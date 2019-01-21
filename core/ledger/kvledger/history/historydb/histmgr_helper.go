
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package historydb

import (
	"bytes"

	"github.com/hyperledger/fabric/common/ledger/util"
)

//compositekeysep是一个零字节，用作复合键不同组件之间的分隔符。
var CompositeKeySep = []byte{0x00}

//constructcompositehistorykey生成命名空间~key~blocknum~trannum的历史键
//使用保留顺序的编码，以便按高度排序历史查询结果
func ConstructCompositeHistoryKey(ns string, key string, blocknum uint64, trannum uint64) []byte {

	var compositeKey []byte
	compositeKey = append(compositeKey, []byte(ns)...)
	compositeKey = append(compositeKey, CompositeKeySep...)
	compositeKey = append(compositeKey, []byte(key)...)
	compositeKey = append(compositeKey, CompositeKeySep...)
	compositeKey = append(compositeKey, util.EncodeOrderPreservingVarUint64(blocknum)...)
	compositeKey = append(compositeKey, util.EncodeOrderPreservingVarUint64(trannum)...)

	return compositeKey
}

//constructPartialCompositeHistoryKey生成部分历史键命名空间~key~
//用于历史键范围查询
func ConstructPartialCompositeHistoryKey(ns string, key string, endkey bool) []byte {
	var compositeKey []byte
	compositeKey = append(compositeKey, []byte(ns)...)
	compositeKey = append(compositeKey, CompositeKeySep...)
	compositeKey = append(compositeKey, []byte(key)...)
	compositeKey = append(compositeKey, CompositeKeySep...)
	if endkey {
		compositeKey = append(compositeKey, []byte{0xff}...)
	}
	return compositeKey
}

//SplitCompositeHistoryKey使用分隔符拆分键字节
func SplitCompositeHistoryKey(bytesToSplit []byte, separator []byte) ([]byte, []byte) {
	split := bytes.SplitN(bytesToSplit, separator, 2)
	return split[0], split[1]
}
