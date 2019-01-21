
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


package chaincode_test

import (
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode"
	"github.com/hyperledger/fabric/core/chaincode/mock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TransactionContext", func() {
	var (
		resultsIterator    *mock.QueryResultsIterator
		transactionContext *chaincode.TransactionContext
	)

	BeforeEach(func() {
		resultsIterator = &mock.QueryResultsIterator{}
		transactionContext = &chaincode.TransactionContext{}
	})

	Describe("InitializeQueryContext", func() {
		var iter1, iter2 *mock.QueryResultsIterator

		BeforeEach(func() {
			iter1 = &mock.QueryResultsIterator{}
			iter2 = &mock.QueryResultsIterator{}
		})

		It("stores a references to the results iterator", func() {
			transactionContext.InitializeQueryContext("query-id-1", iter1)
			transactionContext.InitializeQueryContext("query-id-2", iter2)

			iter := transactionContext.GetQueryIterator("query-id-1")
			Expect(iter).To(Equal(iter1))
			iter = transactionContext.GetQueryIterator("query-id-2")
			Expect(iter).To(Equal(iter2))
		})

		It("populates a pending query result", func() {
			transactionContext.InitializeQueryContext("query-id", iter1)
			pqr := transactionContext.GetPendingQueryResult("query-id")

			Expect(pqr).To(Equal(&chaincode.PendingQueryResult{}))
		})

		It("populates a total return count", func() {
			transactionContext.InitializeQueryContext("query-id", iter1)
			count := transactionContext.GetTotalReturnCount("query-id")

			Expect(*count).To(Equal(int32(0)))
		})
	})

	Describe("GetQueryIterator", func() {
		It("returns the results iteraterator provided to initialize query context", func() {
			transactionContext.InitializeQueryContext("query-id", resultsIterator)
			iter := transactionContext.GetQueryIterator("query-id")
			Expect(iter).To(Equal(resultsIterator))

			transactionContext.InitializeQueryContext("query-with-nil", nil)
			iter = transactionContext.GetQueryIterator("query-with-nil")
			Expect(iter).To(BeNil())
		})

		Context("when an unknown query id is used", func() {
			It("returns a nil query iterator", func() {
				iter := transactionContext.GetQueryIterator("unknown-id")
				Expect(iter).To(BeNil())
			})
		})
	})

	Describe("GetPendingQueryResult", func() {
		Context("when a query context has been initialized", func() {
			BeforeEach(func() {
				transactionContext.InitializeQueryContext("query-id", nil)
			})

			It("returns a non-nil pending query result", func() {
				pqr := transactionContext.GetPendingQueryResult("query-id")
				Expect(pqr).To(Equal(&chaincode.PendingQueryResult{}))
			})
		})

		Context("when a query context has not been initialized", func() {
			It("returns a nil pending query result", func() {
				pqr := transactionContext.GetPendingQueryResult("query-id")
				Expect(pqr).To(BeNil())
			})
		})
	})

	Describe("GetPendingTotalRecordCount", func() {
		Context("when a query context has been initialized", func() {
			BeforeEach(func() {
				transactionContext.InitializeQueryContext("query-id", nil)
			})

			It("returns a non-nil total record count", func() {
				retCount := transactionContext.GetTotalReturnCount("query-id")
				Expect(*retCount).To(Equal(int32(0)))
			})
		})

		Context("when a query context has not been initialized", func() {
			It("returns a nil total return count", func() {
				retCount := transactionContext.GetTotalReturnCount("query-id")
				Expect(retCount).To(BeNil())
			})
		})
	})

	Describe("CleanupQueryContext", func() {
		It("removes references to the the iterator and results", func() {
			transactionContext.InitializeQueryContext("query-id", resultsIterator)
			transactionContext.CleanupQueryContext("query-id")

			iter := transactionContext.GetQueryIterator("query-id")
			Expect(iter).To(BeNil())
			pqr := transactionContext.GetPendingQueryResult("query-id")
			Expect(pqr).To(BeNil())
			retCount := transactionContext.GetTotalReturnCount("query-id")
			Expect(retCount).To(BeNil())
		})

		It("closes the query iterator", func() {
			transactionContext.InitializeQueryContext("query-id", resultsIterator)
			transactionContext.CleanupQueryContext("query-id")

			Expect(resultsIterator.CloseCallCount()).To(Equal(1))
		})

		Context("when the query iterator is nil", func() {
			It("keeps calm and carries on", func() {
				transactionContext.InitializeQueryContext("query-id", nil)
				transactionContext.CleanupQueryContext("query-id")

				pqr := transactionContext.GetPendingQueryResult("query-id")
				Expect(pqr).To(BeNil())
			})
		})
	})

	Describe("CloseQueryIterators", func() {
		var resultsIterators []*mock.QueryResultsIterator

		BeforeEach(func() {
			for i := 0; i < 5; i++ {
				resultsIterators = append(resultsIterators, &mock.QueryResultsIterator{})
				transactionContext.InitializeQueryContext(fmt.Sprintf("query-id-%d", i+1), resultsIterators[i])
			}
		})

		It("closes all initialized results iterators", func() {
			transactionContext.CloseQueryIterators()
			for _, iter := range resultsIterators {
				Expect(iter.CloseCallCount()).To(Equal(1))
			}
		})
	})
})
