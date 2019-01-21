
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

package handlers

import (
	"crypto/sha256"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
)

//NymSecretKey包含Nym密钥
type nymSecretKey struct {
//这个钥匙的滑雪板
	ski []byte
//sk是对nym秘密的idemix引用
	sk Big
//pk是对nym公共部分的idemix引用
	pk Ecp
//可导出如果为真，则可以通过bytes函数导出sk
	exportable bool
}

func computeSKI(serialise func() ([]byte, error)) ([]byte, error) {
	raw, err := serialise()
	if err != nil {
		return nil, err
	}

	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil), nil

}

func NewNymSecretKey(sk Big, pk Ecp, exportable bool) (*nymSecretKey, error) {
	ski, err := computeSKI(sk.Bytes)
	if err != nil {
		return nil, err
	}

	return &nymSecretKey{ski: ski, sk: sk, pk: pk, exportable: exportable}, nil
}

func (k *nymSecretKey) Bytes() ([]byte, error) {
	if k.exportable {
		return k.sk.Bytes()
	}

	return nil, errors.New("not supported")
}

func (k *nymSecretKey) SKI() []byte {
	c := make([]byte, len(k.ski))
	copy(c, k.ski)
	return c
}

func (*nymSecretKey) Symmetric() bool {
	return false
}

func (*nymSecretKey) Private() bool {
	return true
}

func (k *nymSecretKey) PublicKey() (bccsp.Key, error) {
	ski, err := computeSKI(k.pk.Bytes)
	if err != nil {
		return nil, err
	}
	return &nymPublicKey{ski: ski, pk: k.pk}, nil
}

type nymPublicKey struct {
//这个钥匙的滑雪板
	ski []byte
//pk是对nym公共部分的idemix引用
	pk Ecp
}

func NewNymPublicKey(pk Ecp) *nymPublicKey {
	return &nymPublicKey{pk: pk}
}

func (k *nymPublicKey) Bytes() ([]byte, error) {
	return k.pk.Bytes()
}

func (k *nymPublicKey) SKI() []byte {
	c := make([]byte, len(k.ski))
	copy(c, k.ski)
	return c
}

func (*nymPublicKey) Symmetric() bool {
	return false
}

func (*nymPublicKey) Private() bool {
	return false
}

func (k *nymPublicKey) PublicKey() (bccsp.Key, error) {
	return k, nil
}

//NymkeyDerivation派生Nyms
type NymKeyDerivation struct {
//exportable是允许颁发者密钥标记为exportable的标志。
//如果密钥标记为可导出，则其byte s方法将返回密钥的字节表示形式。
	Exportable bool
//用户实现底层加密算法
	User User
}

func (kd *NymKeyDerivation) KeyDeriv(k bccsp.Key, opts bccsp.KeyDerivOpts) (dk bccsp.Key, err error) {
	userSecretKey, ok := k.(*userSecretKey)
	if !ok {
		return nil, errors.New("invalid key, expected *userSecretKey")
	}
	nymKeyDerivationOpts, ok := opts.(*bccsp.IdemixNymKeyDerivationOpts)
	if !ok {
		return nil, errors.New("invalid options, expected *IdemixNymKeyDerivationOpts")
	}
	if nymKeyDerivationOpts.IssuerPK == nil {
		return nil, errors.New("invalid options, missing issuer public key")
	}
	issuerPK, ok := nymKeyDerivationOpts.IssuerPK.(*issuerPublicKey)
	if !ok {
		return nil, errors.New("invalid options, expected IssuerPK as *issuerPublicKey")
	}

	Nym, RandNym, err := kd.User.MakeNym(userSecretKey.sk, issuerPK.pk)
	if err != nil {
		return nil, err
	}

	return NewNymSecretKey(RandNym, Nym, kd.Exportable)
}

//NymPublicKeyImporter导入Nym公钥
type NymPublicKeyImporter struct {
//用户实现底层加密算法
	User User
}

func (i *NymPublicKeyImporter) KeyImport(raw interface{}, opts bccsp.KeyImportOpts) (k bccsp.Key, err error) {
	bytes, ok := raw.([]byte)
	if !ok {
		return nil, errors.New("invalid raw, expected byte array")
	}

	if len(bytes) == 0 {
		return nil, errors.New("invalid raw, it must not be nil")
	}

	pk, err := i.User.NewPublicNymFromBytes(bytes)
	if err != nil {
		return nil, err
	}

	return &nymPublicKey{pk: pk}, nil
}
