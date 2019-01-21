
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package cluster

import (
	"bytes"
	"crypto/x509"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

//远程验证程序验证到远程主机的连接
type RemoteVerifier func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error

//去：生成mokery-dir。-name secureddialer-case-underline-output./mocks/

//安全拨号程序连接到远程地址
type SecureDialer interface {
	Dial(address string, verifyFunc RemoteVerifier) (*grpc.ClientConn, error)
}

//去：生成mokery-dir。-name connectionmapper-case underline-output./mocks/

//ConnectionMapper将证书映射到连接
type ConnectionMapper interface {
	Lookup(cert []byte) (*grpc.ClientConn, bool)
	Put(cert []byte, conn *grpc.ClientConn)
	Remove(cert []byte)
}

//ConnectionStore存储到远程节点的连接
type ConnectionStore struct {
	certsByEndpoints atomic.Value
	lock             sync.RWMutex
	Connections      ConnectionMapper
	dialer           SecureDialer
}

//NewConnectionStore使用给定的SecureDialer创建新的ConnectionStore
func NewConnectionStore(dialer SecureDialer) *ConnectionStore {
	connMapping := &ConnectionStore{
		Connections: make(ConnByCertMap),
		dialer:      dialer,
	}
	return connMapping
}

//verifyhandshake返回一个谓词，该谓词验证远程节点的身份验证
//自身具有给定的TLS证书
func (c *ConnectionStore) verifyHandshake(endpoint string, certificate []byte) RemoteVerifier {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		if bytes.Equal(certificate, rawCerts[0]) {
			return nil
		}
		return errors.Errorf("certificate presented by %s doesn't match any authorized certificate", endpoint)
	}
}

//断开连接关闭映射到给定证书的GRPC连接
func (c *ConnectionStore) Disconnect(expectedServerCert []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()

	conn, connected := c.Connections.Lookup(expectedServerCert)
	if !connected {
		return
	}
	conn.Close()
	c.Connections.Remove(expectedServerCert)
}

//连接获取到给定端点的连接，并需要给定的服务器证书
//由远程节点显示
func (c *ConnectionStore) Connection(endpoint string, expectedServerCert []byte) (*grpc.ClientConn, error) {
	c.lock.RLock()
	conn, alreadyConnected := c.Connections.Lookup(expectedServerCert)
	c.lock.RUnlock()

	if alreadyConnected {
		return conn, nil
	}

//否则，我们需要连接到远程端点
	return c.connect(endpoint, expectedServerCert)
}

//Connect连接到给定的终结点，需要给定的TLS服务器证书
//在认证时提交
func (c *ConnectionStore) connect(endpoint string, expectedServerCert []byte) (*grpc.ClientConn, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
//再次检查其他Goroutine是否已连接
//我们在等锁
	conn, alreadyConnected := c.Connections.Lookup(expectedServerCert)
	if alreadyConnected {
		return conn, nil
	}

	v := c.verifyHandshake(endpoint, expectedServerCert)
	conn, err := c.dialer.Dial(endpoint, v)
	if err != nil {
		return nil, err
	}

	c.Connections.Put(expectedServerCert, conn)
	return conn, nil
}
