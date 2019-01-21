
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


package manager_test

import (
	mockid "github.com/hyperledger/fabric/token/identity/mock"
	"github.com/hyperledger/fabric/token/tms/manager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("AllIssuingValidator", func() {
	var (
		fakeCreatorInfo          *mockid.PublicInfo
		fakeIdentityDeserializer *mockid.Deserializer
		fakeIdentity             *mockid.Identity
		policyValidator          *manager.AllIssuingValidator
	)

	BeforeEach(func() {
		fakeCreatorInfo = &mockid.PublicInfo{}
		fakeIdentityDeserializer = &mockid.Deserializer{}
		fakeIdentity = &mockid.Identity{}

		policyValidator = &manager.AllIssuingValidator{
			Deserializer: fakeIdentityDeserializer,
		}
	})

	Describe("Validate", func() {
		Context("when the creator is a member", func() {
			BeforeEach(func() {
				fakeIdentityDeserializer.DeserializeIdentityReturns(fakeIdentity, nil)
			})
			It("returns no error", func() {
				err := policyValidator.Validate(fakeCreatorInfo, "")

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the creator cannot be deserialized", func() {
			BeforeEach(func() {
				fakeIdentityDeserializer.DeserializeIdentityReturns(nil, errors.New("Deserialize, no-way-man"))
				fakeCreatorInfo.PublicReturns([]byte{1, 2, 3})
			})

			It("returns an error", func() {
				err := policyValidator.Validate(fakeCreatorInfo, "")

				Expect(err.Error()).To(Equal("identity [0x010203] cannot be deserialised: Deserialize, no-way-man"))
				Expect(fakeIdentityDeserializer.DeserializeIdentityCallCount()).To(Equal(1))
			})
		})

		Context("when identity validation fail", func() {
			BeforeEach(func() {
				fakeIdentity.ValidateReturns(errors.New("Validate, no-way-man"))
				fakeIdentityDeserializer.DeserializeIdentityReturns(fakeIdentity, nil)
				fakeCreatorInfo.PublicReturns([]byte{4, 5, 6})
			})

			It("returns an error", func() {
				err := policyValidator.Validate(fakeCreatorInfo, "")

				Expect(err.Error()).To(Equal("identity [0x040506] cannot be validated: Validate, no-way-man"))
				Expect(fakeIdentityDeserializer.DeserializeIdentityCallCount()).To(Equal(1))
				Expect(fakeIdentity.ValidateCallCount()).To(Equal(1))
			})

		})

	})
})
