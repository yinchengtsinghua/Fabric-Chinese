
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


package middleware_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/hyperledger/fabric/core/middleware"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Chain", func() {
	var (
		one, two, three middleware.Middleware
		chain           middleware.Chain

		hello http.Handler

		req  *http.Request
		resp *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		one = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("1:before,"))
				next.ServeHTTP(w, r)
				w.Write([]byte("1:after"))
			})
		}
		two = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("2:before,"))
				next.ServeHTTP(w, r)
				w.Write([]byte("2:after,"))
			})
		}
		three = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("3:before,"))
				next.ServeHTTP(w, r)
				w.Write([]byte("3:after,"))
			})
		}
		chain = middleware.NewChain(one, two, three)

		hello = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello!,"))
		})

		req = httptest.NewRequest("GET", "/", nil)
		resp = httptest.NewRecorder()
	})

	It("calls middleware in the specified order", func() {
		chain.Handler(hello).ServeHTTP(resp, req)
		Expect(resp.Body.String()).To(Equal("1:before,2:before,3:before,Hello!,3:after,2:after,1:after"))
	})

	Context("when the chain is empty", func() {
		BeforeEach(func() {
			chain = middleware.NewChain()
		})

		It("calls the handler", func() {
			chain.Handler(hello).ServeHTTP(resp, req)
			Expect(resp.Body.String()).To(Equal("Hello!,"))
		})
	})

	Context("when the handler is nil", func() {
		It("uses the DefaultServerMux", func() {
			chain.Handler(nil).ServeHTTP(resp, req)
			Expect(resp.Body.String()).To(ContainSubstring("404 page not found"))
		})
	})
})
