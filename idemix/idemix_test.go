
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
	"bytes"
	"testing"

	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/stretchr/testify/assert"
)

func TestIdemix(t *testing.T) {
//测试弱BB信号：
//测试密钥
	rng, err := GetRand()
	assert.NoError(t, err)
	wbbsk, wbbpk := WBBKeyGen(rng)

//获取随机消息
	testmsg := RandModOrder(rng)

//测试签名
	wbbsig := WBBSign(wbbsk, testmsg)

//试验验证
	err = WBBVerify(wbbpk, wbbsig, testmsg)
	assert.NoError(t, err)

//测试IDemix功能
	AttributeNames := []string{"Attr1", "Attr2", "Attr3", "Attr4", "Attr5"}
	attrs := make([]*FP256BN.BIG, len(AttributeNames))
	for i := range AttributeNames {
		attrs[i] = FP256BN.NewBIGint(i)
	}

//测试颁发者密钥生成
	if err != nil {
		t.Fatalf("Error getting rng: \"%s\"", err)
		return
	}
//创建新的密钥对
	key, err := NewIssuerKey(AttributeNames, rng)
	if err != nil {
		t.Fatalf("Issuer key generation should have succeeded but gave error \"%s\"", err)
		return
	}

//检查钥匙是否有效
	err = key.GetIpk().Check()
	if err != nil {
		t.Fatalf("Issuer public key should be valid")
		return
	}

//确保check（）对于具有无效证明的公钥无效
	proofC := key.Ipk.GetProofC()
	key.Ipk.ProofC = BigToBytes(RandModOrder(rng))
	assert.Error(t, key.Ipk.Check(), "public key with broken zero-knowledge proof should be invalid")

//确保check（）对于具有不正确hattrs数的公钥无效
	hAttrs := key.Ipk.GetHAttrs()
	key.Ipk.HAttrs = key.Ipk.HAttrs[:0]
	assert.Error(t, key.Ipk.Check(), "public key with incorrect number of HAttrs should be invalid")
	key.Ipk.HAttrs = hAttrs

//将IPK还原为有效
	key.Ipk.ProofC = proofC
	h := key.Ipk.GetHash()
	assert.NoError(t, key.Ipk.Check(), "restored public key should be valid")
	assert.Zero(t, bytes.Compare(h, key.Ipk.GetHash()), "IPK hash changed on ipk Check")

//创建具有重复属性名的public应失败
	_, err = NewIssuerKey([]string{"Attr1", "Attr2", "Attr1"}, rng)
	assert.Error(t, err, "issuer key generation should fail with duplicate attribute names")

//测试发布
	sk := RandModOrder(rng)
	ni := RandModOrder(rng)
	m := NewCredRequest(sk, BigToBytes(ni), key.Ipk, rng)

	cred, err := NewCredential(key, m, attrs, rng)
	assert.NoError(t, err, "Failed to issue a credential: \"%s\"", err)

	assert.NoError(t, cred.Ver(sk, key.Ipk), "credential should be valid")

//使用不正确的属性量颁发凭据应失败
	_, err = NewCredential(key, m, []*FP256BN.BIG{}, rng)
	assert.Error(t, err, "issuing a credential with the incorrect amount of attributes should fail")

//破坏CredRequest的zk证明会使其无效
	proofC = m.GetProofC()
	m.ProofC = BigToBytes(RandModOrder(rng))
	assert.Error(t, m.Check(key.Ipk), "CredRequest with broken ZK proof should not be valid")

//从损坏的CredRequest创建凭据应失败
	_, err = NewCredential(key, m, attrs, rng)
	assert.Error(t, err, "creating a credential from an invalid CredRequest should fail")
	m.ProofC = proofC

//具有nil属性的凭据应无效
	attrsBackup := cred.GetAttrs()
	cred.Attrs = [][]byte{nil, nil, nil, nil, nil}
	assert.Error(t, cred.Ver(sk, key.Ipk), "credential with nil attribute should be invalid")
	cred.Attrs = attrsBackup

//生成吊销密钥对
	revocationKey, err := GenerateLongTermRevocationKey()
	assert.NoError(t, err)

//创建不包含吊销机制的CRI
	epoch := 0
	cri, err := CreateCRI(revocationKey, []*FP256BN.BIG{}, epoch, ALG_NO_REVOCATION, rng)
	assert.NoError(t, err)
	err = VerifyEpochPK(&revocationKey.PublicKey, cri.EpochPk, cri.EpochPkSig, int(cri.Epoch), RevocationAlgorithm(cri.RevocationAlg))
	assert.NoError(t, err)

//确保epoch pk在将来的epoch中无效
	err = VerifyEpochPK(&revocationKey.PublicKey, cri.EpochPk, cri.EpochPkSig, int(cri.Epoch)+1, RevocationAlgorithm(cri.RevocationAlg))
	assert.Error(t, err)

//测试输入错误
	_, err = CreateCRI(nil, []*FP256BN.BIG{}, epoch, ALG_NO_REVOCATION, rng)
	assert.Error(t, err)
	_, err = CreateCRI(revocationKey, []*FP256BN.BIG{}, epoch, ALG_NO_REVOCATION, nil)
	assert.Error(t, err)

//试验签字不披露
	Nym, RandNym := MakeNym(sk, key.Ipk, rng)

	disclosure := []byte{0, 0, 0, 0, 0}
	msg := []byte{1, 2, 3, 4, 5}
	rhindex := 4
	sig, err := NewSignature(cred, sk, Nym, RandNym, key.Ipk, disclosure, msg, rhindex, cri, rng)
	assert.NoError(t, err)

	err = sig.Ver(disclosure, key.Ipk, msg, nil, 0, &revocationKey.PublicKey, epoch)
	if err != nil {
		t.Fatalf("Signature should be valid but verification returned error: %s", err)
		return
	}

//测试签名选择性披露
	disclosure = []byte{0, 1, 1, 1, 0}
	sig, err = NewSignature(cred, sk, Nym, RandNym, key.Ipk, disclosure, msg, rhindex, cri, rng)
	assert.NoError(t, err)

	err = sig.Ver(disclosure, key.Ipk, msg, attrs, rhindex, &revocationKey.PublicKey, epoch)
	assert.NoError(t, err)

//测试名称签名
	nymsig, err := NewNymSignature(sk, Nym, RandNym, key.Ipk, []byte("testing"), rng)
	assert.NoError(t, err)

	err = nymsig.Ver(Nym, key.Ipk, []byte("testing"))
	if err != nil {
		t.Fatalf("NymSig should be valid but verification returned error: %s", err)
		return
	}
}
