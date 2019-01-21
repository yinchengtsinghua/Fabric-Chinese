
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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	var rp *retryProcess

	mockChannel := newChannel(channelNameForTest(t), defaultPartition)
	flag := false

	noErrorFn := func() error {
		flag = true
		return nil
	}

	errorFn := func() error { return fmt.Errorf("foo") }

	t.Run("Proper", func(t *testing.T) {
		exitChan := make(chan struct{})
		rp = newRetryProcess(mockRetryOptions, exitChan, mockChannel, "foo", noErrorFn)
		assert.NoError(t, rp.retry(), "Expected retry to return no errors")
		assert.Equal(t, true, flag, "Expected flag to be set to true")
	})

	t.Run("WithError", func(t *testing.T) {
		exitChan := make(chan struct{})
		rp = newRetryProcess(mockRetryOptions, exitChan, mockChannel, "foo", errorFn)
		assert.Error(t, rp.retry(), "Expected retry to return an error")
	})
}
