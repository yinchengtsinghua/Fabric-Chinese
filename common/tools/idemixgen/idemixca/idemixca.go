
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


package idemixca

import (
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/hyperledger/fabric/idemix"
	"github.com/hyperledger/fabric/msp"
	m "github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

//generate issuer key调用idemix库来生成颁发者（CA）签名密钥对。
//目前，颁发者支持四个属性：
//attributeName是组织单位名称
//attributeName是角色（成员或管理员）名称
//attributeName EnrollmentID是注册ID
//attributeName吊销句柄包含可用于吊销此用户的吊销句柄
//生成的键被序列化为字节。
func GenerateIssuerKey() ([]byte, []byte, error) {
	rng, err := idemix.GetRand()
	if err != nil {
		return nil, nil, err
	}
	AttributeNames := []string{msp.AttributeNameOU, msp.AttributeNameRole, msp.AttributeNameEnrollmentId, msp.AttributeNameRevocationHandle}
	key, err := idemix.NewIssuerKey(AttributeNames, rng)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "cannot generate CA key")
	}
	ipkSerialized, err := proto.Marshal(key.Ipk)

	return key.Isk, ipkSerialized, err
}

//GenerateSignerConfig创建新的签名者配置。
//它生成一个新的用户秘密并颁发一个凭证
//使用CA的密钥对具有四个属性（如上所述）。
func GenerateSignerConfig(roleMask int, ouString string, enrollmentId string, revocationHandle int, key *idemix.IssuerKey, revKey *ecdsa.PrivateKey) ([]byte, error) {
	attrs := make([]*FP256BN.BIG, 4)

	if ouString == "" {
		return nil, errors.Errorf("the OU attribute value is empty")
	}

	if enrollmentId == "" {
		return nil, errors.Errorf("the enrollment id value is empty")
	}

	attrs[msp.AttributeIndexOU] = idemix.HashModOrder([]byte(ouString))
	attrs[msp.AttributeIndexRole] = FP256BN.NewBIGint(roleMask)
	attrs[msp.AttributeIndexEnrollmentId] = idemix.HashModOrder([]byte(enrollmentId))
	attrs[msp.AttributeIndexRevocationHandle] = FP256BN.NewBIGint(revocationHandle)

	rng, err := idemix.GetRand()
	if err != nil {
		return nil, errors.WithMessage(err, "Error getting PRNG")
	}
	sk := idemix.RandModOrder(rng)
	ni := idemix.BigToBytes(idemix.RandModOrder(rng))
	msg := idemix.NewCredRequest(sk, ni, key.Ipk, rng)
	cred, err := idemix.NewCredential(key, msg, attrs, rng)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to generate a credential")
	}

	credBytes, err := proto.Marshal(cred)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to marshal credential")
	}

//注：目前，idemyca创建的CRI带有“alg_no_revocation”
	cri, err := idemix.CreateCRI(revKey, []*FP256BN.BIG{FP256BN.NewBIGint(revocationHandle)}, 0, idemix.ALG_NO_REVOCATION, rng)
	if err != nil {
		return nil, err
	}
	criBytes, err := proto.Marshal(cri)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to marshal CRI")
	}

	signer := &m.IdemixMSPSignerConfig{
		Cred:                            credBytes,
		Sk:                              idemix.BigToBytes(sk),
		OrganizationalUnitIdentifier:    ouString,
		Role:                            int32(roleMask),
		EnrollmentId:                    enrollmentId,
		CredentialRevocationInformation: criBytes,
	}

	return proto.Marshal(signer)
}
