
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


package grpclogging_test

import (
	"context"
	"time"

	"github.com/hyperledger/fabric/common/grpclogging"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Context", func() {
	var inputFields []zapcore.Field

	BeforeEach(func() {
		inputFields = []zapcore.Field{
			zap.String("string-key", "string-value"),
			zap.Duration("duration-key", time.Second),
			zap.Int("int-key", 42),
		}
	})

	It("decorates a context with fields", func() {
		ctx := grpclogging.WithFields(context.Background(), inputFields)
		Expect(ctx).NotTo(Equal(context.Background()))

		fields := grpclogging.Fields(ctx)
		Expect(fields).NotTo(BeEmpty())
	})

	It("extracts fields from a decorated context as a slice of zapcore.Field", func() {
		ctx := grpclogging.WithFields(context.Background(), inputFields)

		fields := grpclogging.Fields(ctx)
		Expect(fields).To(ConsistOf(inputFields))
	})

	It("extracts fields from a decorated context as a slice of interface{}", func() {
		ctx := grpclogging.WithFields(context.Background(), inputFields)

		zapFields := grpclogging.ZapFields(ctx)
		Expect(zapFields).To(Equal(inputFields))
	})

	It("returns the same fields regardless of type", func() {
		ctx := grpclogging.WithFields(context.Background(), inputFields)

		fields := grpclogging.Fields(ctx)
		zapFields := grpclogging.ZapFields(ctx)
		Expect(zapFields).To(ConsistOf(fields))
	})

	Context("when the context isn't decorated", func() {
		It("returns no fields", func() {
			fields := grpclogging.Fields(context.Background())
			Expect(fields).To(BeNil())

			zapFields := grpclogging.ZapFields(context.Background())
			Expect(zapFields).To(BeNil())
		})
	})
})
