
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


package comm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

//addpemtocertpool将pem编码的证书添加到证书池
func AddPemToCertPool(pemCerts []byte, pool *x509.CertPool) error {
	certs, _, err := pemToX509Certs(pemCerts)
	if err != nil {
		return err
	}
	for _, cert := range certs {
		pool.AddCert(cert)
	}
	return nil
}

//用于分析PEM编码证书的实用函数
func pemToX509Certs(pemCerts []byte) ([]*x509.Certificate, []string, error) {

//可能对多个证书进行了编码
	certs := []*x509.Certificate{}
	subjects := []string{}
	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
  /*TODO:检查MSP为什么不向PEM头添加类型
  如果是块，类型！=“证书”len（block.headers）！= 0 {
   持续
  }
  **/


		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, subjects, err
		} else {
			certs = append(certs, cert)
//提取并附加主题
			subjects = append(subjects, string(cert.RawSubject))
		}
	}
	return certs, subjects, nil
}

//bindingInspector作为参数接收GRPC上下文和信封，
//并验证消息是否包含与上下文的适当绑定
type BindingInspector func(context.Context, proto.Message) error

//certhashextractor从proto.message消息中提取证书
type CertHashExtractor func(proto.Message) []byte

//NewBindingInspector根据是否
//是否配置了MutualTLS，并根据提取的函数
//来自协议消息的TLS证书哈希
func NewBindingInspector(mutualTLS bool, extractTLSCertHash CertHashExtractor) BindingInspector {
	if extractTLSCertHash == nil {
		panic(errors.New("extractTLSCertHash parameter is nil"))
	}
	inspectMessage := mutualTLSBinding
	if !mutualTLS {
		inspectMessage = noopBinding
	}
	return func(ctx context.Context, msg proto.Message) error {
		if msg == nil {
			return errors.New("message is nil")
		}
		return inspectMessage(ctx, extractTLSCertHash(msg))
	}
}

//mutualTlsbinding强制客户端在消息中发送其tls证书哈希，
//然后将其与派生的计算哈希进行比较
//从GRPC上下文。
//如果它们不匹配，或者请求中缺少证书哈希，或者
//没有要从GRPC上下文中挖掘的TLS证书，
//返回错误。
func mutualTLSBinding(ctx context.Context, claimedTLScertHash []byte) error {
	if len(claimedTLScertHash) == 0 {
		return errors.Errorf("client didn't include its TLS cert hash")
	}
	actualTLScertHash := ExtractCertificateHashFromContext(ctx)
	if len(actualTLScertHash) == 0 {
		return errors.Errorf("client didn't send a TLS certificate")
	}
	if !bytes.Equal(actualTLScertHash, claimedTLScertHash) {
		return errors.Errorf("claimed TLS cert hash is %v but actual TLS cert hash is %v", claimedTLScertHash, actualTLScertHash)
	}
	return nil
}

//noopbinding是始终返回nil的bindingInspector
func noopBinding(_ context.Context, _ []byte) error {
	return nil
}

//ExtractCertificateHashFromContext从给定的上下文中提取证书的哈希。
//如果证书不存在，则返回零。
func ExtractCertificateHashFromContext(ctx context.Context) []byte {
	rawCert := ExtractCertificateFromContext(ctx)
	if len(rawCert) == 0 {
		return nil
	}
	h := sha256.New()
	h.Write(rawCert)
	return h.Sum(nil)
}

//ExtractCertificateFromContext返回TLS证书（如果适用）
//从GRPC流的给定上下文
func ExtractCertificateFromContext(ctx context.Context) []byte {
	pr, extracted := peer.FromContext(ctx)
	if !extracted {
		return nil
	}

	authInfo := pr.AuthInfo
	if authInfo == nil {
		return nil
	}

	tlsInfo, isTLSConn := authInfo.(credentials.TLSInfo)
	if !isTLSConn {
		return nil
	}
	certs := tlsInfo.State.PeerCertificates
	if len(certs) == 0 {
		return nil
	}
	return certs[0].Raw
}
