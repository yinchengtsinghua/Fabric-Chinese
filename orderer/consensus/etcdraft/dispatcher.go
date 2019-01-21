
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


package etcdraft

import (
	"fmt"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
)

//去：生成mokery-dir。-name messagereceiver-case下划线-output mocks

//messagereceiver接收消息
type MessageReceiver interface {
//步骤将给定的步骤请求消息传递给消息接收者
	Step(req *orderer.StepRequest, sender uint64) error

//Submit将给定的SubmitRequest消息传递给MessageReceiver
	Submit(req *orderer.SubmitRequest, sender uint64) error
}

//去：生成mokery-dir。-name receivergetter-case underline-output mocks

//Receivergetter获取给定通道ID的MessageReceiver实例
type ReceiverGetter interface {
//ReceiverByChain返回messageReceiver（如果存在），否则返回nil
	ReceiverByChain(channelID string) MessageReceiver
}

//调度程序将提交请求和步骤请求分派给指定的每个链实例
type Dispatcher struct {
	Logger        *flogging.FabricLogger
	ChainSelector ReceiverGetter
}

//onstep通知调度员在给定通道上接收来自给定发送方的step请求。
func (d *Dispatcher) OnStep(channel string, sender uint64, request *orderer.StepRequest) (*orderer.StepResponse, error) {
	receiver := d.ChainSelector.ReceiverByChain(channel)
	if receiver == nil {
		d.Logger.Warningf("An attempt to send a StepRequest to a non existing channel (%s) was made by %d", channel, sender)
		return nil, errors.Errorf("channel %s doesn't exist", channel)
	}
	return &orderer.StepResponse{}, receiver.Step(request, sender)
}

//OnSubmit通知调度程序在给定通道上接收来自给定发送者的提交请求。
func (d *Dispatcher) OnSubmit(channel string, sender uint64, request *orderer.SubmitRequest) (*orderer.SubmitResponse, error) {
	receiver := d.ChainSelector.ReceiverByChain(channel)
	if receiver == nil {
		d.Logger.Warningf("An attempt to submit a transaction to a non existing channel (%s) was made by %d", channel, sender)
		return &orderer.SubmitResponse{
			Info:   fmt.Sprintf("channel %s doesn't exist", channel),
			Status: common.Status_NOT_FOUND,
		}, nil
	}
	if err := receiver.Submit(request, sender); err != nil {
		d.Logger.Errorf("Failed handling transaction on channel %s from %d: %+v", channel, sender, err)
		return &orderer.SubmitResponse{
			Info:   err.Error(),
			Status: common.Status_INTERNAL_SERVER_ERROR,
		}, nil
	}
	return &orderer.SubmitResponse{
		Status: common.Status_SUCCESS,
	}, nil
}
