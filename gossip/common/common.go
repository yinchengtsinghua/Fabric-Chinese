
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


package common

import (
	"bytes"
	"encoding/hex"
)

func init() {
//这只是为了满足代码覆盖工具
//错过任何方法
	switch true {

	}
}

//pki id type定义保存pki ID的类型
//它是对等机的安全标识符
type PKIidType []byte

func (p PKIidType) String() string {
	if p == nil {
		return "<nil>"
	}
	return hex.EncodeToString(p)
}

//不是相同的筛选器生成筛选器函数
//提供谓词以在当前ID
//等于另一个。
func (id PKIidType) IsNotSameFilter(that PKIidType) bool {
	return !bytes.Equal(id, that)
}

//messageacceptor是一个谓词，用于
//确定在哪些消息中创建了
//MessageAcceptor的实例感兴趣。
type MessageAcceptor func(interface{}) bool

//有效负载定义包含分类帐块的对象
type Payload struct {
ChainID ChainID //块的通道ID
Data    []byte  //消息的内容，可能加密或签名
Hash    string  //消息哈希
SeqNum  uint64  //消息序列号
}

//chainID定义链的标识表示形式
type ChainID []byte

//messagereplacingpolicy返回：
//如果此消息使
//如果此消息被此消息无效，则消息无效
//消息\u否则不执行\u操作
type MessageReplacingPolicy func(this interface{}, that interface{}) InvalidationResult

//InvalidationResult确定消息如何影响其他消息
//当它被放进八卦信息商店时
type InvalidationResult int

const (
//messagenoaction表示消息没有关系
	MessageNoAction InvalidationResult = iota
//message invalidates表示消息使其他消息无效
	MessageInvalidates
//message invalidated表示消息被另一条消息失效。
	MessageInvalidated
)
