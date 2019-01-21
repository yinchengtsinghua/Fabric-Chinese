
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


package version

import "github.com/hyperledger/fabric/common/ledger/util"

//Height表示区块链中交易的高度
type Height struct {
	BlockNum uint64
	TxNum    uint64
}

//NewHeight constructs a new instance of Height
func NewHeight(blockNum, txNum uint64) *Height {
	return &Height{blockNum, txNum}
}

//NewHeightFromBytes从序列化字节构造高度的新实例
func NewHeightFromBytes(b []byte) (*Height, int) {
	blockNum, n1 := util.DecodeOrderPreservingVarUint64(b)
	txNum, n2 := util.DecodeOrderPreservingVarUint64(b[n1:])
	return NewHeight(blockNum, txNum), n1 + n2
}

//ToBytes序列化高度
func (h *Height) ToBytes() []byte {
	blockNumBytes := util.EncodeOrderPreservingVarUint64(h.BlockNum)
	txNumBytes := util.EncodeOrderPreservingVarUint64(h.TxNum)
	return append(blockNumBytes, txNumBytes...)
}

//根据这个高度是否比较A返回1，0，或1。
//分别小于、等于或大于指定高度。
func (h *Height) Compare(h1 *Height) int {
	res := 0
	switch {
	case h.BlockNum != h1.BlockNum:
		res = int(h.BlockNum - h1.BlockNum)
		break
	case h.TxNum != h1.TxNum:
		res = int(h.TxNum - h1.TxNum)
		break
	default:
		return 0
	}
	if res > 0 {
		return 1
	}
	return -1
}

//如果两个高度都为零或相等，则返回“相同”。
func AreSame(h1 *Height, h2 *Height) bool {
	if h1 == nil {
		return h2 == nil
	}
	if h2 == nil {
		return false
	}
	return h1.Compare(h2) == 0
}
