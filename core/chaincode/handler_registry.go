
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


package chaincode

import (
	"sync"

	"github.com/pkg/errors"
)

//handlerRegistry维护链式代码处理程序实例。
type HandlerRegistry struct {
allowUnsolicitedRegistration bool //从cs.userrunscc

mutex     sync.Mutex              //锁盖处理和启动
handlers  map[string]*Handler     //将cname链接到关联的处理程序
launching map[string]*LaunchState //启动链码以启动状态
}

type LaunchState struct {
	mutex    sync.Mutex
	notified bool
	done     chan struct{}
	err      error
}

func NewLaunchState() *LaunchState {
	return &LaunchState{
		done: make(chan struct{}),
	}
}

func (l *LaunchState) Done() <-chan struct{} {
	return l.done
}

func (l *LaunchState) Err() error {
	l.mutex.Lock()
	err := l.err
	l.mutex.Unlock()
	return err
}

func (l *LaunchState) Notify(err error) {
	l.mutex.Lock()
	if !l.notified {
		l.notified = true
		l.err = err
		close(l.done)
	}
	l.mutex.Unlock()
}

//NewHandlerRegistry构造一个HandlerRegistry。
func NewHandlerRegistry(allowUnsolicitedRegistration bool) *HandlerRegistry {
	return &HandlerRegistry{
		handlers:                     map[string]*Handler{},
		launching:                    map[string]*LaunchState{},
		allowUnsolicitedRegistration: allowUnsolicitedRegistration,
	}
}

//启动表示正在启动链码。洗衣店说
//返回提供了确定操作何时
//完成，是否失败。bool指示是否
//链码已经启动。
func (r *HandlerRegistry) Launching(cname string) (*LaunchState, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

//发射发生或已经发生
	if launchState, ok := r.launching[cname]; ok {
		return launchState, true
	}

//处理程序已注册，但未通过启动
	if _, ok := r.handlers[cname]; ok {
		launchState := NewLaunchState()
		launchState.Notify(nil)
		return launchState, true
	}

//第一次尝试启动，因此运行时需要启动
	launchState := NewLaunchState()
	r.launching[cname] = launchState
	return launchState, false
}

//就绪表示链码注册已完成，并且
//准备好的响应已发送到链码。
func (r *HandlerRegistry) Ready(cname string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	launchStatus := r.launching[cname]
	if launchStatus != nil {
		launchStatus.Notify(nil)
	}
}

//
func (r *HandlerRegistry) Failed(cname string, err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	launchStatus := r.launching[cname]
	if launchStatus != nil {
		launchStatus.Notify(err)
	}
}

//
func (r *HandlerRegistry) Handler(cname string) *Handler {
	r.mutex.Lock()
	h := r.handlers[cname]
	r.mutex.Unlock()
	return h
}

//寄存器向注册表添加一个链式代码处理程序。
//如果已经为注册了处理程序，
//链码。如果chaincode还没有返回错误
//已“启动”，不允许主动注册。
func (r *HandlerRegistry) Register(h *Handler) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	key := h.chaincodeID.Name

	if r.handlers[key] != nil {
		chaincodeLogger.Debugf("duplicate registered handler(key:%s) return error", key)
		return errors.Errorf("duplicate chaincodeID: %s", h.chaincodeID.Name)
	}

//对等端未启动此链码，但正在尝试
//注册。仅在开发模式下允许。
	if r.launching[key] == nil && !r.allowUnsolicitedRegistration {
		return errors.Errorf("peer will not accept external chaincode connection %v (except in dev mode)", h.chaincodeID.Name)
	}

	r.handlers[key] = h

	chaincodeLogger.Debugf("registered handler complete for chaincode %s", key)
	return nil
}

//取消注册清除对状态关联的指定链码的引用。
//作为清理的一部分，它关闭处理程序以便清理任何状态。
//如果注册表不包含提供的处理程序，则返回错误。
func (r *HandlerRegistry) Deregister(cname string) error {
	chaincodeLogger.Debugf("deregister handler: %s", cname)

	r.mutex.Lock()
	handler := r.handlers[cname]
	delete(r.handlers, cname)
	delete(r.launching, cname)
	r.mutex.Unlock()

	if handler == nil {
		return errors.Errorf("could not find handler: %s", cname)
	}

	handler.Close()

	chaincodeLogger.Debugf("deregistered handler with key: %s", cname)
	return nil
}
