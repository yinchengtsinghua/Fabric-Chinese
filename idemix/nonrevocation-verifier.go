
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
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/pkg/errors"
)

//nonrevokedprove是处理撤销的zk证明系统的验证器。
type nonRevocationVerifier interface {
//recomputefscontribution重新计算非撤销证明对zkp挑战的贡献
	recomputeFSContribution(proof *NonRevocationProof, chal *FP256BN.BIG, epochPK *FP256BN.ECP2, proofSRh *FP256BN.BIG) ([]byte, error)
}

//noNonReversionVerifier是一个空的非ReversionVerifier，它产生一个空的贡献
type nopNonRevocationVerifier struct{}

func (verifier *nopNonRevocationVerifier) recomputeFSContribution(proof *NonRevocationProof, chal *FP256BN.BIG, epochPK *FP256BN.ECP2, proofSRh *FP256BN.BIG) ([]byte, error) {
	return nil, nil
}

//GetUnrevocationVerifier返回绑定到传递的吊销算法的UnrevocationVerifier
func getNonRevocationVerifier(algorithm RevocationAlgorithm) (nonRevocationVerifier, error) {
	switch algorithm {
	case ALG_NO_REVOCATION:
		return &nopNonRevocationVerifier{}, nil
	default:
//未知的吊销算法
		return nil, errors.Errorf("unknown revocation algorithm %d", algorithm)
	}
}
