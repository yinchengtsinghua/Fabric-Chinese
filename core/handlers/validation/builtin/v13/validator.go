
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


package v13

import (
	commonerrors "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
)

//StateBasedvalidator用于验证执行对
//使用密钥级认可策略的KVS密钥。应该调用此接口
//任何验证程序插件（包括默认验证程序插件）。它的功能
//接口调用如下：
//1）validator插件调用prevalidate（甚至在确定事务是否为
//有效的
//2）验证程序插件在确定验证程序的有效性之前或之后调用validate。
//基于其他考虑的交易
//3）验证程序插件确定事务的整体有效性，然后调用
//后验证
type StateBasedValidator interface {
//prevalidate设置验证之前所需的验证程序的内部数据结构
//指定块中的事务“txnum”无法继续
	PreValidate(txNum uint64, block *common.Block)

//validate确定指定通道上的事务是否处于指定高度
//根据其链码级认可策略和任何密钥级验证有效
//参数
	Validate(cc string, blockNum, txNum uint64, rwset, prp, ep []byte, endorsements []*peer.Endorsement) commonerrors.TxValidationError

//postvalidate设置验证后所需的验证程序的内部数据结构
//在指定的高度为指定通道上的事务确定了代码
	PostValidate(cc string, blockNum, txNum uint64, err error)
}
