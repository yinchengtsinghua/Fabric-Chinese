
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package e2e

import (
	"encoding/json"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/hyperledger/fabric/integration/nwo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

func TestEndToEnd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EndToEnd Suite")
}

var components *nwo.Components

var _ = SynchronizedBeforeSuite(func() []byte {
	components = &nwo.Components{}
	components.Build()

	payload, err := json.Marshal(components)
	Expect(err).NotTo(HaveOccurred())

	return payload
}, func(payload []byte) {
	err := json.Unmarshal(payload, &components)
	Expect(err).NotTo(HaveOccurred())
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	components.Cleanup()
})

func BasePort() int {
	return 30000 + 1000*GinkgoParallelNode()
}

type DatagramReader struct {
	buffer    *gbytes.Buffer
	errCh     chan error
	sock      *net.UDPConn
	doneCh    chan struct{}
	closeOnce sync.Once
	err       error
}

func NewDatagramReader() *DatagramReader {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())
	sock, err := net.ListenUDP("udp", udpAddr)
	Expect(err).NotTo(HaveOccurred())
	err = sock.SetReadBuffer(1024 * 1024)
	Expect(err).NotTo(HaveOccurred())

	return &DatagramReader{
		buffer: gbytes.NewBuffer(),
		sock:   sock,
		errCh:  make(chan error, 1),
		doneCh: make(chan struct{}),
	}
}

func (dr *DatagramReader) Buffer() *gbytes.Buffer {
	return dr.buffer
}

func (dr *DatagramReader) Address() string {
	return dr.sock.LocalAddr().String()
}

func (dr *DatagramReader) String() string {
	return string(dr.buffer.Contents())
}

func (dr *DatagramReader) Start() {
	buf := make([]byte, 1024*1024)
	for {
		select {
		case <-dr.doneCh:
			dr.errCh <- nil
			return

		default:
			n, _, err := dr.sock.ReadFrom(buf)
			if err != nil {
				dr.errCh <- err
				return
			}
			_, err = dr.buffer.Write(buf[0:n])
			if err != nil {
				dr.errCh <- err
				return
			}
		}
	}
}

func (dr *DatagramReader) Close() error {
	dr.closeOnce.Do(func() {
		close(dr.doneCh)
		err := dr.sock.Close()
		dr.err = <-dr.errCh
		if dr.err == nil && err != nil && err != io.EOF {
			dr.err = err
		}
	})
	return dr.err
}
