
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package tms

import "github.com/hyperledger/fabric/protos/token"

//TransactionData结构包含令牌事务和结构事务ID。
//令牌事务由验证方对等方创建，但仅创建事务ID
//稍后由客户机执行（使用令牌事务和nonce）。在验证和提交时
//提交对等机需要时间、令牌事务和事务ID。
//将它们存储在一个结构中有助于处理它们。
type TransactionData struct {
	Tx *token.TokenTransaction
//结构事务ID
	TxID string
}
