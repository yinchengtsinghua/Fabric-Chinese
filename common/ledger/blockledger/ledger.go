
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
*/


package blockledger

import (
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
)

//
type Factory interface {
//
//
	GetOrCreate(chainID string) (ReadWriter, error)

//chainIds返回工厂知道的链ID
	ChainIDs() []string

//
	Close()
}

//迭代器对于链读取器在创建块时流式处理块很有用。
type Iterator interface {
//
//下一个块不再可检索
	Next() (*cb.Block, cb.Status)
//
	Close()
}

//
type Reader interface {
//
//
	Iterator(startType *ab.SeekPosition) (Iterator, uint64)
//height返回分类帐上的块数
	Height() uint64
}

//
type Writer interface {
//将新块追加到分类帐
	Append(block *cb.Block) error
}

//

//
type ReadWriter interface {
	Reader
	Writer
}
