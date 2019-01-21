
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
	"github.com/hyperledger/fabric-amcl/amcl"
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/pkg/errors"
)

//nonrevokedprover是处理撤销的zk证明系统的证明者。
type nonRevokedProver interface {
//getfscontribution返回对fiat shamir散列的非撤销贡献，形成zkp的挑战，
	getFSContribution(rh *FP256BN.BIG, rRh *FP256BN.BIG, cri *CredentialRevocationInformation, rng *amcl.RAND) ([]byte, error)

//GetUnrevokedProof返回有关传递的质询的未吊销证明
	getNonRevokedProof(chal *FP256BN.BIG) (*NonRevocationProof, error)
}

//nopenrevokedpover是空的unrevokedpover
type nopNonRevokedProver struct{}

func (prover *nopNonRevokedProver) getFSContribution(rh *FP256BN.BIG, rRh *FP256BN.BIG, cri *CredentialRevocationInformation, rng *amcl.RAND) ([]byte, error) {
	return nil, nil
}

func (prover *nopNonRevokedProver) getNonRevokedProof(chal *FP256BN.BIG) (*NonRevocationProof, error) {
	ret := &NonRevocationProof{}
	ret.RevocationAlg = int32(ALG_NO_REVOCATION)
	return ret, nil
}

//GetUnrevocationProver返回绑定到传递的吊销算法的UnrevokedProver
func getNonRevocationProver(algorithm RevocationAlgorithm) (nonRevokedProver, error) {
	switch algorithm {
	case ALG_NO_REVOCATION:
		return &nopNonRevokedProver{}, nil
	default:
//未知的吊销算法
		return nil, errors.Errorf("unknown revocation algorithm %d", algorithm)
	}
}
