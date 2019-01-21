
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
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-amcl/amcl"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/idemix/handlers"
	cryptolib "github.com/hyperledger/fabric/idemix"
	"github.com/pkg/errors"
)

//IssuerPublickey封装IDemix颁发者公钥。
type IssuerPublicKey struct {
	PK *cryptolib.IssuerPublicKey
}

func (o *IssuerPublicKey) Bytes() ([]byte, error) {
	return proto.Marshal(o.PK)
}

func (o *IssuerPublicKey) Hash() []byte {
	return o.PK.Hash
}

//IssuerPublickey封装IDemix颁发者密钥。
type IssuerSecretKey struct {
	SK *cryptolib.IssuerKey
}

func (o *IssuerSecretKey) Bytes() ([]byte, error) {
	return proto.Marshal(o.SK)
}

func (o *IssuerSecretKey) Public() handlers.IssuerPublicKey {
	return &IssuerPublicKey{o.SK.Ipk}
}

//颁发者封装IDemix算法以生成颁发者密钥对
type Issuer struct {
	NewRand func() *amcl.RAND
}

//new key生成新的颁发者密钥对
func (i *Issuer) NewKey(attributeNames []string) (res handlers.IssuerSecretKey, err error) {
	defer func() {
		if r := recover(); r != nil {
			res = nil
			err = errors.Errorf("failure [%s]", r)
		}
	}()

	sk, err := cryptolib.NewIssuerKey(attributeNames, i.NewRand())
	if err != nil {
		return
	}

	res = &IssuerSecretKey{SK: sk}

	return
}

func (*Issuer) NewPublicKeyFromBytes(raw []byte, attributes []string) (res handlers.IssuerPublicKey, err error) {
	defer func() {
		if r := recover(); r != nil {
			res = nil
			err = errors.Errorf("failure [%s]", r)
		}
	}()

	ipk := new(cryptolib.IssuerPublicKey)
	err = proto.Unmarshal(raw, ipk)
	if err != nil {
		return nil, errors.WithStack(&bccsp.IdemixIssuerPublicKeyImporterError{
			Type:     bccsp.IdemixIssuerPublicKeyImporterUnmarshallingError,
			ErrorMsg: "failed to unmarshal issuer public key",
			Cause:    err})
	}

	err = ipk.SetHash()
	if err != nil {
		return nil, errors.WithStack(&bccsp.IdemixIssuerPublicKeyImporterError{
			Type:     bccsp.IdemixIssuerPublicKeyImporterHashError,
			ErrorMsg: "setting the hash of the issuer public key failed",
			Cause:    err})
	}

	err = ipk.Check()
	if err != nil {
		return nil, errors.WithStack(&bccsp.IdemixIssuerPublicKeyImporterError{
			Type:     bccsp.IdemixIssuerPublicKeyImporterValidationError,
			ErrorMsg: "invalid issuer public key",
			Cause:    err})
	}

	if len(attributes) != 0 {
//检查属性
		if len(attributes) != len(ipk.AttributeNames) {
			return nil, errors.WithStack(&bccsp.IdemixIssuerPublicKeyImporterError{
				Type: bccsp.IdemixIssuerPublicKeyImporterNumAttributesError,
				ErrorMsg: fmt.Sprintf("invalid number of attributes, expected [%d], got [%d]",
					len(ipk.AttributeNames), len(attributes)),
			})
		}

		for i, attr := range attributes {
			if ipk.AttributeNames[i] != attr {
				return nil, errors.WithStack(&bccsp.IdemixIssuerPublicKeyImporterError{
					Type:     bccsp.IdemixIssuerPublicKeyImporterAttributeNameError,
					ErrorMsg: fmt.Sprintf("invalid attribute name at position [%d]", i),
				})
			}
		}
	}

	res = &IssuerPublicKey{PK: ipk}

	return
}
