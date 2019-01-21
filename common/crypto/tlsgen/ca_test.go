
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


package tlsgen

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func createTLSService(t *testing.T, ca CA, host string) *grpc.Server {
	keyPair, err := ca.NewServerCertKeyPair(host)
	cert, err := tls.X509KeyPair(keyPair.Cert, keyPair.Key)
	assert.NoError(t, err)
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    x509.NewCertPool(),
	}
	tlsConf.ClientCAs.AppendCertsFromPEM(ca.CertBytes())
	return grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConf)))
}

func TestTLSCA(t *testing.T) {
//此测试检查CA是否可以创建证书
//

	rand.Seed(time.Now().UnixNano())
randomPort := 1234 + rand.Intn(1234) //一些随机端口

	ca, err := NewCA()
	assert.NoError(t, err)
	assert.NotNil(t, ca)

	endpoint := fmt.Sprintf("127.0.0.1:%d", randomPort)
	srv := createTLSService(t, ca, "127.0.0.1")
	l, err := net.Listen("tcp", endpoint)
	assert.NoError(t, err)
	go srv.Serve(l)
	defer srv.Stop()
	defer l.Close()

	probeTLS := func(kp *CertKeyPair) error {
		keyBytes, err := base64.StdEncoding.DecodeString(kp.PrivKeyString())
		assert.NoError(t, err)
		certBytes, err := base64.StdEncoding.DecodeString(kp.PubKeyString())
		assert.NoError(t, err)
		cert, err := tls.X509KeyPair(certBytes, keyBytes)
		tlsCfg := &tls.Config{
			RootCAs:      x509.NewCertPool(),
			Certificates: []tls.Certificate{cert},
		}
		tlsCfg.RootCAs.AppendCertsFromPEM(ca.CertBytes())
		tlsOpts := grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, fmt.Sprintf("127.0.0.1:%d", randomPort), tlsOpts, grpc.WithBlock())
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}

//
//TLS服务器启动时使用的
	kp, err := ca.NewClientCertKeyPair()
	assert.NoError(t, err)
	err = probeTLS(kp)
	assert.NoError(t, err)

//错误路径-使用从外部CA生成的证书密钥对
	foreignCA, _ := NewCA()
	kp, err = foreignCA.NewClientCertKeyPair()
	assert.NoError(t, err)
	err = probeTLS(kp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}
