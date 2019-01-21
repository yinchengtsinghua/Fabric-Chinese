
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


//包msgprocessor提供处理分类消息的实现
//可通过广播进入系统的类型。
package msgprocessor

import (
	"errors"

	"github.com/hyperledger/fabric/common/flogging"
	cb "github.com/hyperledger/fabric/protos/common"
)

const (
//一旦启用，这些最终应来自通道支持
	msgVersion = int32(0)
	epoch      = 0
)

var logger = flogging.MustGetLogger("orderer.common.msgprocessor")

//系统通道返回errChanneldoesNoteList，用于
//不用于系统通道ID，并且不尝试创建新通道
var ErrChannelDoesNotExist = errors.New("channel does not exist")

//errPermissionDenied由事务引起的错误返回
//由于授权失败而不允许。
var ErrPermissionDenied = errors.New("permission denied")

//分类表示系统可能的消息类型。
type Classification int

const (
//normalmsg是标准（代言人或其他非配置）消息的类。
//此类型的消息应由processnormalmsg处理。
	NormalMsg Classification = iota

//configupdatemsg表示config_update类型的消息。
//此类型的消息应由processconfigupdatemsg处理。
	ConfigUpdateMsg

//configmsg表示order_transaction或config类型的消息。
//此类型的消息应由processconfigmsg处理
	ConfigMsg
)

//处理器提供必要的方法来分类和处理
//通过广播接口到达。
type Processor interface {
//ClassifyMSG检查消息头以确定需要哪种类型的处理
	ClassifyMsg(chdr *cb.ChannelHeader) Classification

//processnormalmsg将根据当前配置检查消息的有效性。它返回电流
//配置序列号，成功时为零，如果消息无效则为错误
	ProcessNormalMsg(env *cb.Envelope) (configSeq uint64, err error)

//processconfigupdatemsg将尝试将配置更新应用于当前配置，如果成功
//返回生成的配置消息和从中计算配置的configseq。如果配置更新消息
//无效，返回错误。
	ProcessConfigUpdateMsg(env *cb.Envelope) (config *cb.Envelope, configSeq uint64, err error)

//processconfigmsg接受'order_tx'或'config'类型的消息，解压缩嵌入的configupdate信封
//并调用“processconfigupdatemsg”生成与原始消息类型相同的新配置消息。
//此方法用于重新验证和复制配置消息，如果它被认为不再有效。
	ProcessConfigMsg(env *cb.Envelope) (*cb.Envelope, uint64, error)
}
