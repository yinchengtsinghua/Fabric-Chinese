
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


package crypto

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/stretchr/testify/assert"
)

func TestX509CertExpiresAt(t *testing.T) {
	certBytes, err := ioutil.ReadFile(filepath.Join("testdata", "cert.pem"))
	assert.NoError(t, err)
	sId := &msp.SerializedIdentity{
		IdBytes: certBytes,
	}
	serializedIdentity, err := proto.Marshal(sId)
	assert.NoError(t, err)
	expirationTime := ExpiresAt(serializedIdentity)
	assert.Equal(t, time.Date(2027, 8, 17, 12, 19, 48, 0, time.UTC), expirationTime)
}

func TestX509InvalidCertExpiresAt(t *testing.T) {
	certBytes, err := ioutil.ReadFile(filepath.Join("testdata", "badCert.pem"))
	assert.NoError(t, err)
	sId := &msp.SerializedIdentity{
		IdBytes: certBytes,
	}
	serializedIdentity, err := proto.Marshal(sId)
	assert.NoError(t, err)
	expirationTime := ExpiresAt(serializedIdentity)
	assert.True(t, expirationTime.IsZero())
}

func TestIdemixIdentityExpiresAt(t *testing.T) {
	idemixId := &msp.SerializedIdemixIdentity{
		NymX: []byte{1, 2, 3},
		NymY: []byte{1, 2, 3},
		Ou:   []byte("OU1"),
	}
	idemixBytes, err := proto.Marshal(idemixId)
	assert.NoError(t, err)
	sId := &msp.SerializedIdentity{
		IdBytes: idemixBytes,
	}
	serializedIdentity, err := proto.Marshal(sId)
	assert.NoError(t, err)
	expirationTime := ExpiresAt(serializedIdentity)
	assert.True(t, expirationTime.IsZero())
}

func TestInvalidIdentityExpiresAt(t *testing.T) {
	expirationTime := ExpiresAt([]byte{1, 2, 3})
	assert.True(t, expirationTime.IsZero())
}
