
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


package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("CORS", func() {

	var (
		sess *gexec.Session
		req  *http.Request

//runserver在临时端口上启动服务器，然后创建CORS请求
//针对同一个服务器（但不发送），必须在内部调用它
//每次测试前
		runServer func(args ...string)
	)

	BeforeEach(func() {
		runServer = func(args ...string) {
			cmd := exec.Command(configtxlatorPath, args...)
			var err error
			errBuffer := gbytes.NewBuffer()
			sess, err = gexec.Start(cmd, GinkgoWriter, io.MultiWriter(errBuffer, GinkgoWriter))
			Expect(err).NotTo(HaveOccurred())
			Consistently(sess.Exited).ShouldNot(BeClosed())
			Eventually(errBuffer).Should(gbytes.Say("Serving HTTP requests on 127.0.0.1:"))
			address := regexp.MustCompile("127.0.0.1:[0-9]+").FindString(string(errBuffer.Contents()))
			Expect(address).NotTo(BeEmpty())

req, err = http.NewRequest("OPTIONS", fmt.Sprintf("http://%s/protocator/encode/common.block“，地址），nil）
			Expect(err).NotTo(HaveOccurred())
req.Header.Add("Origin", "http://“Fo.com”
			req.Header.Add("Access-Control-Request-Method", "POST")
			req.Header.Add("Access-Control-Request-Headers", "Content-Type")
		}
	})

	AfterEach(func() {
		sess.Signal(syscall.SIGKILL)
		Eventually(sess.Exited).Should(BeClosed())
		Expect(sess.ExitCode()).To(Equal(137))
	})

	Context("when CORS options are not provided", func() {
		BeforeEach(func() {
			runServer("start", "--hostname", "127.0.0.1", "--port", "0")
		})

		It("rejects CORS OPTIONS requests", func() {
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
		})
	})

	Context("when the CORS wildcard is provided", func() {
		BeforeEach(func() {
			runServer("start", "--hostname", "127.0.0.1", "--port", "9998", "--CORS", "*")
		})

		It("it allows CORS requests from any domain", func() {
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Header.Get("Access-Control-Allow-Origin")).To(Equal("*"))
			Expect(resp.Header.Get("Access-Control-Allow-Headers")).To(Equal("Content-Type"))
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Context("when multiple CORS options are provided", func() {
		BeforeEach(func() {
runServer("start", "--hostname", "127.0.0.1", "--port", "9998", "--CORS", "http://foo.com“，--cors”，“http://bar.com”）。
		})

		It("it allows CORS requests from any of them", func() {
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
Expect(resp.Header.Get("Access-Control-Allow-Origin")).To(Equal("http://Fo.com）
			Expect(resp.Header.Get("Access-Control-Allow-Headers")).To(Equal("Content-Type"))
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})
})
