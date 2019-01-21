
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


package tlsgen

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCertEncoding(t *testing.T) {
	pair, err := newCertKeyPair(false, false, "", nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, pair)
	assert.NotEmpty(t, pair.PrivKeyString())
	assert.NotEmpty(t, pair.PubKeyString())
	pair2, err := CertKeyPairFromString(pair.PrivKeyString(), pair.PubKeyString())
	assert.Equal(t, pair.Key, pair2.Key)
	assert.Equal(t, pair.Cert, pair2.Cert)
}

func TestLoadCert(t *testing.T) {
	pair, err := newCertKeyPair(false, false, "", nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, pair)
	tlsCertPair, err := tls.X509KeyPair(pair.Cert, pair.Key)
	assert.NoError(t, err)
	assert.NotNil(t, tlsCertPair)
	block, _ := pem.Decode(pair.Cert)
	cert, err := x509.ParseCertificate(block.Bytes)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}
