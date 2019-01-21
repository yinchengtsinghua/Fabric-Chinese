
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


package semaphore_test

import (
	"context"
	"testing"

	"github.com/hyperledger/fabric/common/semaphore"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
)

func TestNewSemaphorePanic(t *testing.T) {
	assert.PanicsWithValue(t, "count must be greater than 0", func() { semaphore.New(0) })
}

func TestSemaphoreBlocking(t *testing.T) {
	gt := NewGomegaWithT(t)

	sema := semaphore.New(5)
	for i := 0; i < 5; i++ {
		err := sema.Acquire(context.Background())
		gt.Expect(err).NotTo(HaveOccurred())
	}

	done := make(chan struct{})
	go func() {
		err := sema.Acquire(context.Background())
		gt.Expect(err).NotTo(HaveOccurred())

		close(done)
		sema.Release()
	}()

	gt.Consistently(done).ShouldNot(BeClosed())
	sema.Release()
	gt.Eventually(done).Should(BeClosed())
}

func TestSemaphoreContextError(t *testing.T) {
	gt := NewGomegaWithT(t)

	sema := semaphore.New(1)
	err := sema.Acquire(context.Background())
	gt.Expect(err).NotTo(HaveOccurred())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- sema.Acquire(ctx) }()

	gt.Eventually(errCh).Should(Receive(Equal(context.Canceled)))
}

func TestSemaphoreReleaseTooMany(t *testing.T) {
	sema := semaphore.New(1)
	assert.PanicsWithValue(t, "semaphore buffer is empty", func() { sema.Release() })
}
