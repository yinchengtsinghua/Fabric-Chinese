
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

//用户密钥包含用户密钥
type userSecretKey struct {
//sk是对用户密钥的IDemix引用
	sk Big
//可导出如果为真，则可以通过bytes函数导出sk
	exportable bool
}

func NewUserSecretKey(sk Big, exportable bool) *userSecretKey {
	return &userSecretKey{sk: sk, exportable: exportable}
}

func (k *userSecretKey) Bytes() ([]byte, error) {
	if k.exportable {
		return k.sk.Bytes()
	}

	return nil, errors.New("not exportable")
}

func (k *userSecretKey) SKI() []byte {
	raw, err := k.sk.Bytes()
	if err != nil {
		return nil
	}
	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

func (*userSecretKey) Symmetric() bool {
	return true
}

func (*userSecretKey) Private() bool {
	return true
}

func (k *userSecretKey) PublicKey() (bccsp.Key, error) {
	return nil, errors.New("cannot call this method on a symmetric key")
}

type UserKeyGen struct {
//exportable是允许颁发者密钥标记为exportable的标志。
//如果密钥标记为可导出，则其byte s方法将返回密钥的字节表示形式。
	Exportable bool
//用户实现底层加密算法
	User User
}

func (g *UserKeyGen) KeyGen(opts bccsp.KeyGenOpts) (bccsp.Key, error) {
	sk, err := g.User.NewKey()
	if err != nil {
		return nil, err
	}

	return &userSecretKey{exportable: g.Exportable, sk: sk}, nil
}

//用户密钥导入器导入用户密钥
type UserKeyImporter struct {
//exportable是允许将密钥标记为exportable的标志。
//如果密钥标记为可导出，则其byte s方法将返回密钥的字节表示形式。
	Exportable bool
//用户实现底层加密算法
	User User
}

func (i *UserKeyImporter) KeyImport(raw interface{}, opts bccsp.KeyImportOpts) (k bccsp.Key, err error) {
	der, ok := raw.([]byte)
	if !ok {
		return nil, errors.New("invalid raw, expected byte array")
	}

	if len(der) == 0 {
		return nil, errors.New("invalid raw, it must not be nil")
	}

	sk, err := i.User.NewKeyFromBytes(raw.([]byte))
	if err != nil {
		return nil, err
	}

	return &userSecretKey{exportable: i.Exportable, sk: sk}, nil
}
