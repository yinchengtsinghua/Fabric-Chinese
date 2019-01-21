
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
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/idemix/handlers"
	cryptolib "github.com/hyperledger/fabric/idemix"
	"github.com/pkg/errors"
)

//凭证封装IDemix算法以生成（签名）凭证
//并验证。回想一下，颁发者根据凭证请求生成凭证，
//并由请求者进行验证。
type Credential struct {
	NewRand func() *amcl.RAND
}

//符号生成IDemix凭证。它接受输入颁发者密钥，
//串行化的凭证请求和属性值列表。
//请注意，属性不应包含其类型为IdemiMixHiddenAttribute的属性
//因为凭证需要携带所有属性值。
func (c *Credential) Sign(key handlers.IssuerSecretKey, credentialRequest []byte, attributes []bccsp.IdemixAttribute) (res []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			res = nil
			err = errors.Errorf("failure [%s]", r)
		}
	}()

	iisk, ok := key.(*IssuerSecretKey)
	if !ok {
		return nil, errors.Errorf("invalid issuer secret key, expected *Big, got [%T]", key)
	}

	cr := &cryptolib.CredRequest{}
	err = proto.Unmarshal(credentialRequest, cr)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshalling credential request")
	}

	attrValues := make([]*FP256BN.BIG, len(attributes))
	for i := 0; i < len(attributes); i++ {
		switch attributes[i].Type {
		case bccsp.IdemixBytesAttribute:
			attrValues[i] = cryptolib.HashModOrder(attributes[i].Value.([]byte))
		case bccsp.IdemixIntAttribute:
			attrValues[i] = FP256BN.NewBIGint(attributes[i].Value.(int))
		default:
			return nil, errors.Errorf("attribute type not allowed or supported [%v] at position [%d]", attributes[i].Type, i)
		}
	}

	cred, err := cryptolib.NewCredential(iisk.SK, cr, attrValues, c.NewRand())
	if err != nil {
		return nil, errors.WithMessage(err, "failed creating new credential")
	}

	return proto.Marshal(cred)
}

//验证检查IDemix凭据是否加密正确。它需要
//在输入用户密钥（sk）、颁发者公钥（ipk）、串行凭证（凭证）时，
//以及属性列表。属性列表是可选的，如果指定了该列表，请验证
//检查凭证是否具有指定的属性。
func (*Credential) Verify(sk handlers.Big, ipk handlers.IssuerPublicKey, credential []byte, attributes []bccsp.IdemixAttribute) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("failure [%s]", r)
		}
	}()

	isk, ok := sk.(*Big)
	if !ok {
		return errors.Errorf("invalid user secret key, expected *Big, got [%T]", sk)
	}
	iipk, ok := ipk.(*IssuerPublicKey)
	if !ok {
		return errors.Errorf("invalid issuer public key, expected *IssuerPublicKey, got [%T]", sk)
	}

	cred := &cryptolib.Credential{}
	err = proto.Unmarshal(credential, cred)
	if err != nil {
		return err
	}

	for i := 0; i < len(attributes); i++ {
		switch attributes[i].Type {
		case bccsp.IdemixBytesAttribute:
			if !bytes.Equal(
				cryptolib.BigToBytes(cryptolib.HashModOrder(attributes[i].Value.([]byte))),
				cred.Attrs[i]) {
				return errors.Errorf("credential does not contain the correct attribute value at position [%d]", i)
			}
		case bccsp.IdemixIntAttribute:
			if !bytes.Equal(
				cryptolib.BigToBytes(FP256BN.NewBIGint(attributes[i].Value.(int))),
				cred.Attrs[i]) {
				return errors.Errorf("credential does not contain the correct attribute value at position [%d]", i)
			}
		case bccsp.IdemixHiddenAttribute:
			continue
		default:
			return errors.Errorf("attribute type not allowed or supported [%v] at position [%d]", attributes[i].Type, i)
		}
	}

	return cred.Ver(isk.E, iipk.PK)
}
