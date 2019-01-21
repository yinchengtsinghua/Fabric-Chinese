
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


package kafka

import (
	"fmt"
	"time"

	localconfig "github.com/hyperledger/fabric/orderer/common/localconfig"
)

type retryProcess struct {
	shortPollingInterval, shortTimeout time.Duration
	longPollingInterval, longTimeout   time.Duration
	exit                               chan struct{}
	channel                            channel
	msg                                string
	fn                                 func() error
}

func newRetryProcess(retryOptions localconfig.Retry, exit chan struct{}, channel channel, msg string, fn func() error) *retryProcess {
	return &retryProcess{
		shortPollingInterval: retryOptions.ShortInterval,
		shortTimeout:         retryOptions.ShortTotal,
		longPollingInterval:  retryOptions.LongInterval,
		longTimeout:          retryOptions.LongTotal,
		exit:                 exit,
		channel:              channel,
		msg:                  msg,
		fn:                   fn,
	}
}

func (rp *retryProcess) retry() error {
	if err := rp.try(rp.shortPollingInterval, rp.shortTimeout); err != nil {
		logger.Debugf("[channel: %s] Switching to the long retry interval", rp.channel.topic())
		return rp.try(rp.longPollingInterval, rp.longTimeout)
	}
	return nil
}

func (rp *retryProcess) try(interval, total time.Duration) (err error) {
//配置验证不允许非正值的ticker值
//（这会导致恐慌）。下面的路径适用于这些测试用例
//当我们不能避免创建一个可重新审判的程序，但我们希望
//立即终止。
	if rp.shortPollingInterval == 0 {
		return fmt.Errorf("illegal value")
	}

//如果初始操作成功，我们不需要启动重试过程。
	logger.Debugf("[channel: %s] "+rp.msg, rp.channel.topic())
	if err = rp.fn(); err == nil {
		logger.Debugf("[channel: %s] Error is nil, breaking the retry loop", rp.channel.topic())
		return
	}

	logger.Debugf("[channel: %s] Initial attempt failed = %s", rp.channel.topic(), err)

	tickInterval := time.NewTicker(interval)
	tickTotal := time.NewTicker(total)
	defer tickTotal.Stop()
	defer tickInterval.Stop()
	logger.Debugf("[channel: %s] Retrying every %s for a total of %s", rp.channel.topic(), interval.String(), total.String())

	for {
		select {
		case <-rp.exit:
			fmt.Println("exit channel")
			exitErr := fmt.Errorf("[channel: %s] process asked to exit", rp.channel.topic())
logger.Warning(exitErr.Error()) //在警告级别记录
			return exitErr
		case <-tickTotal.C:
			return
		case <-tickInterval.C:
			logger.Debugf("[channel: %s] "+rp.msg, rp.channel.topic())
			if err = rp.fn(); err == nil {
				logger.Debugf("[channel: %s] Error is nil, breaking the retry loop", rp.channel.topic())
				return
			}

			logger.Debugf("[channel: %s] Need to retry because process failed = %s", rp.channel.topic(), err)
		}
	}
}
