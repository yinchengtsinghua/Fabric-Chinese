
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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"reflect"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
)

//RevocationSecretKey包含吊销密钥
//并实现bccsp.key接口
type revocationSecretKey struct {
//sk是对吊销密钥的IDemix引用
	privKey *ecdsa.PrivateKey
//可导出如果为真，则可以通过bytes函数导出sk
	exportable bool
}

func NewRevocationSecretKey(sk *ecdsa.PrivateKey, exportable bool) *revocationSecretKey {
	return &revocationSecretKey{privKey: sk, exportable: exportable}
}

//字节将此键转换为其字节表示形式，
//如果允许此操作。
func (k *revocationSecretKey) Bytes() ([]byte, error) {
	if k.exportable {
		return k.privKey.D.Bytes(), nil
	}

	return nil, errors.New("not exportable")
}

//ski返回此密钥的主题密钥标识符。
func (k *revocationSecretKey) SKI() []byte {
//整理公钥
	raw := elliptic.Marshal(k.privKey.Curve, k.privKey.PublicKey.X, k.privKey.PublicKey.Y)

//散列
	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

//如果此密钥是对称密钥，则对称返回true，
//如果此密钥是非对称的，则为false
func (k *revocationSecretKey) Symmetric() bool {
	return false
}

//如果此密钥是私钥，则private返回true，
//否则为假。
func (k *revocationSecretKey) Private() bool {
	return true
}

//public key返回非对称公钥/私钥对的相应公钥部分。
//此方法返回对称密钥方案中的错误。
func (k *revocationSecretKey) PublicKey() (bccsp.Key, error) {
	return &revocationPublicKey{&k.privKey.PublicKey}, nil
}

type revocationPublicKey struct {
	pubKey *ecdsa.PublicKey
}

func NewRevocationPublicKey(pubKey *ecdsa.PublicKey) *revocationPublicKey {
	return &revocationPublicKey{pubKey: pubKey}
}

//字节将此键转换为其字节表示形式，
//如果允许此操作。
func (k *revocationPublicKey) Bytes() (raw []byte, err error) {
	raw, err = x509.MarshalPKIXPublicKey(k.pubKey)
	if err != nil {
		return nil, fmt.Errorf("Failed marshalling key [%s]", err)
	}
	return
}

//ski返回此密钥的主题密钥标识符。
func (k *revocationPublicKey) SKI() []byte {
//整理公钥
	raw := elliptic.Marshal(k.pubKey.Curve, k.pubKey.X, k.pubKey.Y)

//散列
	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

//如果此密钥是对称密钥，则对称返回true，
//如果此密钥是非对称的，则为false
func (k *revocationPublicKey) Symmetric() bool {
	return false
}

//如果此密钥是私钥，则private返回true，
//否则为假。
func (k *revocationPublicKey) Private() bool {
	return false
}

//public key返回非对称公钥/私钥对的相应公钥部分。
//此方法返回对称密钥方案中的错误。
func (k *revocationPublicKey) PublicKey() (bccsp.Key, error) {
	return k, nil
}

//RevocationKeyGen生成吊销密钥。
type RevocationKeyGen struct {
//exportable是允许将吊销密钥标记为exportable的标志。
//如果密钥标记为可导出，则其byte s方法将返回密钥的字节表示形式。
	Exportable bool
//吊销实现基础加密算法
	Revocation Revocation
}

func (g *RevocationKeyGen) KeyGen(opts bccsp.KeyGenOpts) (bccsp.Key, error) {
//创建新的密钥对
	key, err := g.Revocation.NewKey()
	if err != nil {
		return nil, err
	}

	return &revocationSecretKey{exportable: g.Exportable, privKey: key}, nil
}

//RevocationPublicKeyImporter导入吊销公钥
type RevocationPublicKeyImporter struct {
}

func (i *RevocationPublicKeyImporter) KeyImport(raw interface{}, opts bccsp.KeyImportOpts) (k bccsp.Key, err error) {
	der, ok := raw.([]byte)
	if !ok {
		return nil, errors.New("invalid raw, expected byte array")
	}

	if len(der) == 0 {
		return nil, errors.New("invalid raw, it must not be nil")
	}

	blockPub, _ := pem.Decode(raw.([]byte))
	if blockPub == nil {
		return nil, errors.New("Failed to decode revocation ECDSA public key")
	}
	revocationPk, err := x509.ParsePKIXPublicKey(blockPub.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse revocation ECDSA public key bytes")
	}
	ecdsaPublicKey, isECDSA := revocationPk.(*ecdsa.PublicKey)
	if !isECDSA {
		return nil, errors.Errorf("key is of type %v, not of type ECDSA", reflect.TypeOf(revocationPk))
	}

	return &revocationPublicKey{ecdsaPublicKey}, nil
}

type CriSigner struct {
	Revocation Revocation
}

func (s *CriSigner) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) ([]byte, error) {
	revocationSecretKey, ok := k.(*revocationSecretKey)
	if !ok {
		return nil, errors.New("invalid key, expected *revocationSecretKey")
	}
	criOpts, ok := opts.(*bccsp.IdemixCRISignerOpts)
	if !ok {
		return nil, errors.New("invalid options, expected *IdemixCRISignerOpts")
	}

	return s.Revocation.Sign(
		revocationSecretKey.privKey,
		criOpts.UnrevokedHandles,
		criOpts.Epoch,
		criOpts.RevocationAlgorithm,
	)
}

type CriVerifier struct {
	Revocation Revocation
}

func (v *CriVerifier) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (bool, error) {
	revocationPublicKey, ok := k.(*revocationPublicKey)
	if !ok {
		return false, errors.New("invalid key, expected *revocationPublicKey")
	}
	criOpts, ok := opts.(*bccsp.IdemixCRISignerOpts)
	if !ok {
		return false, errors.New("invalid options, expected *IdemixCRISignerOpts")
	}
	if len(signature) == 0 {
		return false, errors.New("invalid signature, it must not be empty")
	}

	err := v.Revocation.Verify(
		revocationPublicKey.pubKey,
		signature,
		criOpts.Epoch,
		criOpts.RevocationAlgorithm,
	)
	if err != nil {
		return false, err
	}

	return true, nil
}
