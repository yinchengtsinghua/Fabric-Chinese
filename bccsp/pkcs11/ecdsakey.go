
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

package pkcs11

import (
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/hyperledger/fabric/bccsp"
)

type ecdsaPrivateKey struct {
	ski []byte
	pub ecdsaPublicKey
}

//字节将此键转换为其字节表示形式，
//如果允许此操作。
func (k *ecdsaPrivateKey) Bytes() ([]byte, error) {
	return nil, errors.New("Not supported.")
}

//ski返回此密钥的主题密钥标识符。
func (k *ecdsaPrivateKey) SKI() []byte {
	return k.ski
}

//如果此密钥是对称密钥，则对称返回true，
//如果此密钥是非对称的，则为false
func (k *ecdsaPrivateKey) Symmetric() bool {
	return false
}

//如果此密钥是私钥，则private返回true，
//否则为假。
func (k *ecdsaPrivateKey) Private() bool {
	return true
}

//public key返回非对称公钥/私钥对的相应公钥部分。
//此方法返回对称密钥方案中的错误。
func (k *ecdsaPrivateKey) PublicKey() (bccsp.Key, error) {
	return &k.pub, nil
}

type ecdsaPublicKey struct {
	ski []byte
	pub *ecdsa.PublicKey
}

//字节将此键转换为其字节表示形式，
//如果允许此操作。
func (k *ecdsaPublicKey) Bytes() (raw []byte, err error) {
	raw, err = x509.MarshalPKIXPublicKey(k.pub)
	if err != nil {
		return nil, fmt.Errorf("Failed marshalling key [%s]", err)
	}
	return
}

//ski返回此密钥的主题密钥标识符。
func (k *ecdsaPublicKey) SKI() []byte {
	return k.ski
}

//如果此密钥是对称密钥，则对称返回true，
//如果此密钥是非对称的，则为false
func (k *ecdsaPublicKey) Symmetric() bool {
	return false
}

//如果此密钥是私钥，则private返回true，
//否则为假。
func (k *ecdsaPublicKey) Private() bool {
	return false
}

//public key返回非对称公钥/私钥对的相应公钥部分。
//此方法返回对称密钥方案中的错误。
func (k *ecdsaPublicKey) PublicKey() (bccsp.Key, error) {
	return k, nil
}
