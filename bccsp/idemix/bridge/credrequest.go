
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

package bridge

import (
	"bytes"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-amcl/amcl"
	"github.com/hyperledger/fabric/bccsp/idemix/handlers"
	cryptolib "github.com/hyperledger/fabric/idemix"
	"github.com/pkg/errors"
)

//CredRequest封装IDemix算法以生成（签名）凭据请求
//并验证。回想一下，凭证请求是由用户生成的，
//在凭证创建时由颁发者进行验证。
type CredRequest struct {
	NewRand func() *amcl.RAND
}

//签名生成IDemix凭证请求。它接受输入用户密钥和
//颁发者公钥。
func (cr *CredRequest) Sign(sk handlers.Big, ipk handlers.IssuerPublicKey, nonce []byte) (res []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			res = nil
			err = errors.Errorf("failure [%s]", r)
		}
	}()

	isk, ok := sk.(*Big)
	if !ok {
		return nil, errors.Errorf("invalid user secret key, expected *Big, got [%T]", sk)
	}
	iipk, ok := ipk.(*IssuerPublicKey)
	if !ok {
		return nil, errors.Errorf("invalid issuer public key, expected *IssuerPublicKey, got [%T]", ipk)
	}
	if len(nonce) != cryptolib.FieldBytes {
		return nil, errors.Errorf("invalid issuer nonce, expected length %d, got %d", cryptolib.FieldBytes, len(nonce))
	}

	rng := cr.NewRand()

	credRequest := cryptolib.NewCredRequest(
		isk.E,
		nonce,
		iipk.PK,
		rng)

	return proto.Marshal(credRequest)
}

//验证通过的凭证请求是否相对于通过的
//颁发者公钥。
func (*CredRequest) Verify(credentialRequest []byte, ipk handlers.IssuerPublicKey, nonce []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("failure [%s]", r)
		}
	}()

	credRequest := &cryptolib.CredRequest{}
	err = proto.Unmarshal(credentialRequest, credRequest)
	if err != nil {
		return err
	}

	iipk, ok := ipk.(*IssuerPublicKey)
	if !ok {
		return errors.Errorf("invalid issuer public key, expected *IssuerPublicKey, got [%T]", ipk)
	}

	err = credRequest.Check(iipk.PK)
	if err != nil {
		return err
	}

//临时支票
	if len(nonce) != cryptolib.FieldBytes {
		return errors.Errorf("invalid issuer nonce, expected length %d, got %d", cryptolib.FieldBytes, len(nonce))
	}
	if !bytes.Equal(nonce, credRequest.IssuerNonce) {
		return errors.Errorf("invalid nonce, expected [%v], got [%v]", nonce, credRequest.IssuerNonce)
	}

	return nil
}
