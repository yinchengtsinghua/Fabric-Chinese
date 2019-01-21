
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

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("PendingQueryResult", func() {
	var pqr *chaincode.PendingQueryResult

	BeforeEach(func() {
		pqr = &chaincode.PendingQueryResult{}
	})

	Describe("Size", func() {
		It("returns the number of results in the batch", func() {
			Expect(pqr.Size()).To(Equal(0))

			for i := 1; i <= 10; i++ {
				kv := &queryresult.KV{Key: fmt.Sprintf("key-%d", i)}
				err := pqr.Add(kv)
				Expect(err).NotTo(HaveOccurred())
				Expect(pqr.Size()).To(Equal(i))
			}
		})

		Describe("Add and Cut", func() {
			It("tracks the query results", func() {
				By("adding the results")
				for i := 1; i <= 10; i++ {
					kv := &queryresult.KV{Key: fmt.Sprintf("key-%d", i)}
					err := pqr.Add(kv)
					Expect(err).NotTo(HaveOccurred())
					Expect(pqr.Size()).To(Equal(i))
				}

				By("cutting the results")
				results := pqr.Cut()
				Expect(results).To(HaveLen(10))
				for i, result := range results {
					var kv queryresult.KV
					err := proto.Unmarshal(result.ResultBytes, &kv)
					Expect(err).NotTo(HaveOccurred())
					Expect(kv.Key).To(Equal(fmt.Sprintf("key-%d", i+1)))
				}
			})

			Context("when the result cannot be marshaled", func() {
				It("returns an error", func() {
					err := pqr.Add(brokenProto{})
					Expect(err).To(MatchError("marshal-failed"))
				})
			})
		})

		Describe("Cut", func() {
			BeforeEach(func() {
				for i := 1; i <= 10; i++ {
					kv := &queryresult.KV{Key: fmt.Sprintf("key-%d", i)}
					err := pqr.Add(kv)
					Expect(err).NotTo(HaveOccurred())
					Expect(pqr.Size()).To(Equal(i))
				}
			})

			It("resets the size to 0", func() {
				Expect(pqr.Size()).To(Equal(10))
				pqr.Cut()
				Expect(pqr.Size()).To(Equal(0))
			})

			Context("when cutting an empty batch", func() {
				It("returns a nil batch", func() {
					pqr.Cut()
					results := pqr.Cut()
					Expect(results).To(BeNil())
				})
			})
		})
	})
})

type brokenProto struct{}

func (brokenProto) Reset()                   {}
func (brokenProto) String() string           { return "" }
func (brokenProto) ProtoMessage()            {}
func (brokenProto) Marshal() ([]byte, error) { return nil, errors.New("marshal-failed") }
