
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
	"github.com/hyperledger/fabric/token/tms/plain"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MemoryLedger", func() {
	var (
		txID1 string
		tx1   []byte
		tx2   []byte

		namespace string

		memoryLedger *plain.MemoryLedger
	)

	BeforeEach(func() {
		memoryLedger = plain.NewMemoryLedger()

		txID1 = "1"
		tx1 = []byte{1}
		tx2 = []byte{2}

		namespace = "ledgerNamespace"
	})

	Describe("get and set", func() {
		It("sets state", func() {
			By("adding a transaction")
			err := memoryLedger.SetState(namespace, txID1, tx1)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the transaction is in the ledger")
			po, err := memoryLedger.GetState(namespace, "1")
			Expect(err).NotTo(HaveOccurred())
			Expect(po).To(Equal([]byte{1}))
		})

		Context("when an entry exists", func() {
			BeforeEach(func() {
				err := memoryLedger.SetState(namespace, txID1, tx1)
				Expect(err).NotTo(HaveOccurred())
			})

			It("overwrites the entry", func() {
				By("setting the new entry")
				err := memoryLedger.SetState(namespace, txID1, tx2)
				Expect(err).NotTo(HaveOccurred())

				By("ensuring the transaction has the new value")
				po, err := memoryLedger.GetState(namespace, "1")
				Expect(err).NotTo(HaveOccurred())
				Expect(po).To(Equal([]byte{2}))
			})
		})

		Context("when an entry does not exist", func() {
			It("returns an error", func() {
				val, err := memoryLedger.GetState(namespace, "badTxID")
				Expect(err).NotTo(HaveOccurred())
				Expect(val).To(BeNil())
			})
		})
	})

	Describe("when the dummy function is invoked", func() {
		It("returns nil", func() {
			res, err := memoryLedger.GetStateRangeScanIterator("", "", "")
			Expect(res).To(BeNil())
			Expect(err).To(BeNil())
		})
	})
})
