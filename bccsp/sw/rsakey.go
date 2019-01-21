
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/

package sw

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"

	"github.com/hyperledger/fabric/bccsp"
)

//rsapublickey反映了pkcs 1公钥的ASN.1结构。
type rsaPublicKeyASN struct {
	N *big.Int
	E int
}

type rsaPrivateKey struct {
	privKey *rsa.PrivateKey
}

//字节将此键转换为其字节表示形式，
//如果允许此操作。
func (k *rsaPrivateKey) Bytes() ([]byte, error) {
	return nil, errors.New("Not supported.")
}

//ski返回此密钥的主题密钥标识符。
func (k *rsaPrivateKey) SKI() []byte {
	if k.privKey == nil {
		return nil
	}

//整理公钥
	raw, _ := asn1.Marshal(rsaPublicKeyASN{
		N: k.privKey.N,
		E: k.privKey.E,
	})

//散列
	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

//如果此密钥是对称密钥，则对称返回true，
//假是这把钥匙是不对称的
func (k *rsaPrivateKey) Symmetric() bool {
	return false
}

//如果此密钥是非对称私钥，则private返回true，
//否则为假。
func (k *rsaPrivateKey) Private() bool {
	return true
}

//public key返回非对称公钥/私钥对的相应公钥部分。
//此方法返回对称密钥方案中的错误。
func (k *rsaPrivateKey) PublicKey() (bccsp.Key, error) {
	return &rsaPublicKey{&k.privKey.PublicKey}, nil
}

type rsaPublicKey struct {
	pubKey *rsa.PublicKey
}

//字节将此键转换为其字节表示形式，
//如果允许此操作。
func (k *rsaPublicKey) Bytes() (raw []byte, err error) {
	if k.pubKey == nil {
		return nil, errors.New("Failed marshalling key. Key is nil.")
	}
	raw, err = x509.MarshalPKIXPublicKey(k.pubKey)
	if err != nil {
		return nil, fmt.Errorf("Failed marshalling key [%s]", err)
	}
	return
}

//ski返回此密钥的主题密钥标识符。
func (k *rsaPublicKey) SKI() []byte {
	if k.pubKey == nil {
		return nil
	}

//整理公钥
	raw, _ := asn1.Marshal(rsaPublicKeyASN{
		N: k.pubKey.N,
		E: k.pubKey.E,
	})

//散列
	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

//如果此密钥是对称密钥，则对称返回true，
//假是这把钥匙是不对称的
func (k *rsaPublicKey) Symmetric() bool {
	return false
}

//如果此密钥是非对称私钥，则private返回true，
//否则为假。
func (k *rsaPublicKey) Private() bool {
	return false
}

//public key返回非对称公钥/私钥对的相应公钥部分。
//此方法返回对称密钥方案中的错误。
func (k *rsaPublicKey) PublicKey() (bccsp.Key, error) {
	return k, nil
}
