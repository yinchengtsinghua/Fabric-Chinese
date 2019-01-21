
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


package accesscontrol

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/common/util"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

var ttl = time.Minute * 10

type certHash string

type KeyGenFunc func() (*tlsgen.CertKeyPair, error)

type certMapper struct {
	keyGen KeyGenFunc
	sync.RWMutex
	m map[certHash]string
}

func newCertMapper(keyGen KeyGenFunc) *certMapper {
	return &certMapper{
		keyGen: keyGen,
		m:      make(map[certHash]string),
	}
}

func (r *certMapper) lookup(h certHash) string {
	r.RLock()
	defer r.RUnlock()
	return r.m[h]
}

func (r *certMapper) register(hash certHash, name string) {
	r.Lock()
	defer r.Unlock()
	r.m[hash] = name
	time.AfterFunc(ttl, func() {
		r.purge(hash)
	})
}

func (r *certMapper) purge(hash certHash) {
	r.Lock()
	defer r.Unlock()
	delete(r.m, hash)
}

func (r *certMapper) genCert(name string) (*tlsgen.CertKeyPair, error) {
	keyPair, err := r.keyGen()
	if err != nil {
		return nil, err
	}
	hash := util.ComputeSHA256(keyPair.TLSCert.Raw)
	r.register(certHash(hash), name)
	return keyPair, nil
}

//ExtractCertificateHash从流中提取证书的哈希
func extractCertificateHashFromContext(ctx context.Context) []byte {
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
	raw := certs[0].Raw
	if len(raw) == 0 {
		return nil
	}
	return util.ComputeSHA256(raw)
}
