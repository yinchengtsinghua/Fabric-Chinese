
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


package idemix

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-amcl/amcl"
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/hyperledger/fabric/bccsp/utils"
	"github.com/pkg/errors"
)

type RevocationAlgorithm int32

const (
	ALG_NO_REVOCATION RevocationAlgorithm = iota
)

var ProofBytes = map[RevocationAlgorithm]int{
	ALG_NO_REVOCATION: 0,
}

//GenerateLongterRevocationKey生成用于吊销的长期签名密钥
func GenerateLongTermRevocationKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
}

//createcri创建特定时间段（epoch）的凭证吊销信息。
//用户可以使用CRI来证明他们没有被撤销。
//注意，当不使用撤销（即alg=alg_no_撤销）时，不使用输入的未撤销数据，
//由此产生的CRI可以被任何签名者使用。
func CreateCRI(key *ecdsa.PrivateKey, unrevokedHandles []*FP256BN.BIG, epoch int, alg RevocationAlgorithm, rng *amcl.RAND) (*CredentialRevocationInformation, error) {
	if key == nil || rng == nil {
		return nil, errors.Errorf("CreateCRI received nil input")
	}
	cri := &CredentialRevocationInformation{}
	cri.RevocationAlg = int32(alg)
	cri.Epoch = int64(epoch)

	if alg == ALG_NO_REVOCATION {
//在原型中放置一个虚拟的pk
		cri.EpochPk = Ecp2ToProto(GenG2)
	} else {
//创建epoch键
		_, epochPk := WBBKeyGen(rng)
		cri.EpochPk = Ecp2ToProto(epochPk)
	}

//用长期键标记epoch+epoch键
	bytesToSign, err := proto.Marshal(cri)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CRI")
	}

	digest := sha256.Sum256(bytesToSign)

	cri.EpochPkSig, err = key.Sign(rand.Reader, digest[:], nil)
	if err != nil {
		return nil, err
	}

	if alg == ALG_NO_REVOCATION {
		return cri, nil
	} else {
		return nil, errors.Errorf("the specified revocation algorithm is not supported.")
	}
}

//verifyepochpk验证某个时期的吊销pk是否有效，
//通过检查它是否使用长期吊销密钥签名。
//注意，即使我们不使用撤销（即alg=alg_no_撤销），我们也需要
//验证签名以确保颁发者确实签署了没有吊销的签名
//在这个时代使用。
func VerifyEpochPK(pk *ecdsa.PublicKey, epochPK *ECP2, epochPkSig []byte, epoch int, alg RevocationAlgorithm) error {
	if pk == nil || epochPK == nil {
		return errors.Errorf("EpochPK invalid: received nil input")
	}
	cri := &CredentialRevocationInformation{}
	cri.RevocationAlg = int32(alg)
	cri.EpochPk = epochPK
	cri.Epoch = int64(epoch)
	bytesToSign, err := proto.Marshal(cri)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(bytesToSign)

	r, s, err := utils.UnmarshalECDSASignature(epochPkSig)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ECDSA signature")
	}

	if !ecdsa.Verify(pk, digest[:], r, s) {
		return errors.Errorf("EpochPKSig invalid")
	}

	return nil
}
