
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


package plain_test

import (
	"github.com/hyperledger/fabric/protos/token"
	"github.com/hyperledger/fabric/token/tms/plain"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Issuer", func() {
	var (
		issuer        *plain.Issuer
		tokensToIssue []*token.TokenToIssue
	)

	BeforeEach(func() {
		tokensToIssue = []*token.TokenToIssue{
			{Recipient: []byte("R1"), Type: "TOK1", Quantity: 1001},
			{Recipient: []byte("R2"), Type: "TOK2", Quantity: 1002},
			{Recipient: []byte("R3"), Type: "TOK3", Quantity: 1003},
		}
		issuer = &plain.Issuer{}
	})

	It("converts an import request to a token transaction", func() {
		tt, err := issuer.RequestImport(tokensToIssue)
		Expect(err).NotTo(HaveOccurred())
		Expect(tt).To(Equal(&token.TokenTransaction{
			Action: &token.TokenTransaction_PlainAction{
				PlainAction: &token.PlainTokenAction{
					Data: &token.PlainTokenAction_PlainImport{
						PlainImport: &token.PlainImport{
							Outputs: []*token.PlainOutput{
								{Owner: []byte("R1"), Type: "TOK1", Quantity: 1001},
								{Owner: []byte("R2"), Type: "TOK2", Quantity: 1002},
								{Owner: []byte("R3"), Type: "TOK3", Quantity: 1003},
							},
						},
					},
				},
			},
		}))
	})

	Context("when tokens to issue is nil", func() {
		It("creates a token transaction with no outputs", func() {
			tt, err := issuer.RequestImport(nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(tt).To(Equal(&token.TokenTransaction{
				Action: &token.TokenTransaction_PlainAction{
					PlainAction: &token.PlainTokenAction{
						Data: &token.PlainTokenAction_PlainImport{PlainImport: &token.PlainImport{}},
					},
				},
			}))
		})
	})

	Context("when tokens to issue is empty", func() {
		It("creates a token transaction with no outputs", func() {
			tt, err := issuer.RequestImport([]*token.TokenToIssue{})
			Expect(err).NotTo(HaveOccurred())

			Expect(tt).To(Equal(&token.TokenTransaction{
				Action: &token.TokenTransaction_PlainAction{
					PlainAction: &token.PlainTokenAction{
						Data: &token.PlainTokenAction_PlainImport{PlainImport: &token.PlainImport{}},
					},
				},
			}))
		})
	})
})
