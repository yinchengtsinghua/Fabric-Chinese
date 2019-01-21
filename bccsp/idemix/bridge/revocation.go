
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
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/hyperledger/fabric/bccsp"
	cryptolib "github.com/hyperledger/fabric/idemix"
	"github.com/pkg/errors"
)

//吊销封装了用于吊销的IDemix算法
type Revocation struct {
}

//new key生成新的吊销密钥对。
func (*Revocation) NewKey() (*ecdsa.PrivateKey, error) {
	return cryptolib.GenerateLongTermRevocationKey()
}

//符号生成一个新的CRI，与传递的无吊销句柄、epoch和吊销算法有关。
func (*Revocation) Sign(key *ecdsa.PrivateKey, unrevokedHandles [][]byte, epoch int, alg bccsp.RevocationAlgorithm) (res []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			res = nil
			err = errors.Errorf("failure [%s]", r)
		}
	}()

	handles := make([]*FP256BN.BIG, len(unrevokedHandles))
	for i := 0; i < len(unrevokedHandles); i++ {
		handles[i] = FP256BN.FromBytes(unrevokedHandles[i])
	}
	cri, err := cryptolib.CreateCRI(key, handles, epoch, cryptolib.RevocationAlgorithm(alg), NewRandOrPanic())
	if err != nil {
		return nil, errors.WithMessage(err, "failed creating CRI")
	}

	return proto.Marshal(cri)
}

//验证是否检查传递的序列化CRI（CRIRAW）对于传递的吊销公钥有效，
//epoch和撤销算法。
func (*Revocation) Verify(pk *ecdsa.PublicKey, criRaw []byte, epoch int, alg bccsp.RevocationAlgorithm) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("failure [%s]", r)
		}
	}()

	cri := &cryptolib.CredentialRevocationInformation{}
	err = proto.Unmarshal(criRaw, cri)
	if err != nil {
		return err
	}

	return cryptolib.VerifyEpochPK(
		pk,
		cri.EpochPk,
		cri.EpochPkSig,
		int(cri.Epoch),
		cryptolib.RevocationAlgorithm(cri.RevocationAlg),
	)
}
