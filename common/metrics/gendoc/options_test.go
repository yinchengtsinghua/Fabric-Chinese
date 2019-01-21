
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


package gendoc_test

import (
	"github.com/hyperledger/fabric/common/metrics/gendoc"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Options", func() {
	It("finds standard options", func() {
		f, err := ParseFile("testdata/basic.go")
		Expect(err).NotTo(HaveOccurred())
		Expect(f).NotTo(BeNil())

		options, err := gendoc.FileOptions(f)
		Expect(err).NotTo(HaveOccurred())
		Expect(options).To(HaveLen(3))
	})

	It("finds options that use named imports", func() {
		f, err := ParseFile("testdata/named_import.go")
		Expect(err).NotTo(HaveOccurred())
		Expect(f).NotTo(BeNil())

		options, err := gendoc.FileOptions(f)
		Expect(err).NotTo(HaveOccurred())
		Expect(options).To(HaveLen(3))
	})

	It("ignores variables that are tagged", func() {
		f, err := ParseFile("testdata/ignored.go")
		Expect(err).NotTo(HaveOccurred())
		Expect(f).NotTo(BeNil())

		options, err := gendoc.FileOptions(f)
		Expect(err).NotTo(HaveOccurred())
		Expect(options).To(BeEmpty())
	})
})
