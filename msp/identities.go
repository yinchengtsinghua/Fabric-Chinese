
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


package msp

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var mspIdentityLogger = flogging.MustGetLogger("msp.identity")

type identity struct {
//ID包含此实例的标识符（mspid和标识标识符）
	id *IdentityIdentifier

//cert包含对此实例的公钥进行签名的X.509证书
	cert *x509.Certificate

//这是此实例的公钥
	pk bccsp.Key

//引用“拥有”此标识的MSP
	msp *bccspmsp
}

func newIdentity(cert *x509.Certificate, pk bccsp.Key, msp *bccspmsp) (Identity, error) {
	if mspIdentityLogger.IsEnabledFor(zapcore.DebugLevel) {
		mspIdentityLogger.Debugf("Creating identity instance for cert %s", certToPEM(cert))
	}

//先清理证书
	cert, err := msp.sanitizeCert(cert)
	if err != nil {
		return nil, err
	}

//计算标识标识符

//将标识证书的哈希用作标识标识符中的ID
	hashOpt, err := bccsp.GetHashOpt(msp.cryptoConfig.IdentityIdentifierHashFunction)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting hash function options")
	}

	digest, err := msp.bccsp.Hash(cert.Raw, hashOpt)
	if err != nil {
		return nil, errors.WithMessage(err, "failed hashing raw certificate to compute the id of the IdentityIdentifier")
	}

	id := &IdentityIdentifier{
		Mspid: msp.name,
		Id:    hex.EncodeToString(digest)}

	return &identity{id: id, cert: cert, pk: pk, msp: msp}, nil
}

//expires at返回标识过期的时间。
func (id *identity) ExpiresAt() time.Time {
	return id.cert.NotAfter
}

//如果此实例与提供的主体匹配，则satisfiesprincipal返回空值，否则返回错误。
func (id *identity) SatisfiesPrincipal(principal *msp.MSPPrincipal) error {
	return id.msp.SatisfiesPrincipal(id, principal)
}

//GetIdentifier返回此实例的标识符（mspid/idid）
func (id *identity) GetIdentifier() *IdentityIdentifier {
	return id.id
}

//getmspidentifier返回此实例的MSP标识符
func (id *identity) GetMSPIdentifier() string {
	return id.id.Mspid
}

//如果此实例是有效标识或其他错误，则validate返回nil
func (id *identity) Validate() error {
	return id.msp.Validate(id)
}

//GetOrganizationalUnits返回此实例的OU
func (id *identity) GetOrganizationalUnits() []*OUIdentifier {
	if id.cert == nil {
		return nil
	}

	cid, err := id.msp.getCertificationChainIdentifier(id)
	if err != nil {
		mspIdentityLogger.Errorf("Failed getting certification chain identifier for [%v]: [%+v]", id, err)

		return nil
	}

	res := []*OUIdentifier{}
	for _, unit := range id.cert.Subject.OrganizationalUnit {
		res = append(res, &OUIdentifier{
			OrganizationalUnitIdentifier: unit,
			CertifiersIdentifier:         cid,
		})
	}

	return res
}

//如果此标识提供匿名性，则Anonymous返回true
func (id *identity) Anonymous() bool {
	return false
}

//NewSerializedIdentity返回序列化标识
//以PEM格式通过的MSPID和X509证书为内容。
//此方法不检查证书的有效性，也不检查
//mspid与它的任何一致性。
func NewSerializedIdentity(mspID string, certPEM []byte) ([]byte, error) {
//我们通过预先设置mspid来序列化标识
//并以PEM格式附加X509证书
	sId := &msp.SerializedIdentity{Mspid: mspID, IdBytes: certPEM}
	raw, err := proto.Marshal(sId)
	if err != nil {
		return nil, errors.Wrapf(err, "failed serializing identity [%s][%X]", mspID, certPEM)
	}
	return raw, nil
}

//根据签名和消息验证检查
//确定这个身份是否产生了
//签名；如果是，则返回零，否则返回错误
func (id *identity) Verify(msg []byte, sig []byte) error {
//mspidentitylogger.infof（“验证签名”）

//计算哈希
	hashOpt, err := id.getHashOpt(id.msp.cryptoConfig.SignatureHashFamily)
	if err != nil {
		return errors.WithMessage(err, "failed getting hash function options")
	}

	digest, err := id.msp.bccsp.Hash(msg, hashOpt)
	if err != nil {
		return errors.WithMessage(err, "failed computing digest")
	}

	if mspIdentityLogger.IsEnabledFor(zapcore.DebugLevel) {
		mspIdentityLogger.Debugf("Verify: digest = %s", hex.Dump(digest))
		mspIdentityLogger.Debugf("Verify: sig = %s", hex.Dump(sig))
	}

	valid, err := id.msp.bccsp.Verify(id.pk, sig, digest, nil)
	if err != nil {
		return errors.WithMessage(err, "could not determine the validity of the signature")
	} else if !valid {
		return errors.New("The signature is invalid")
	}

	return nil
}

//serialize返回此标识的字节数组表示形式
func (id *identity) Serialize() ([]byte, error) {
//mspidentitylogger.infof（“序列化标识%s”，id.id）

	pb := &pem.Block{Bytes: id.cert.Raw, Type: "CERTIFICATE"}
	pemBytes := pem.EncodeToMemory(pb)
	if pemBytes == nil {
		return nil, errors.New("encoding of identity failed")
	}

//我们通过预先设置mspid并附加证书的asn.1 der内容来序列化标识。
	sId := &msp.SerializedIdentity{Mspid: id.id.Mspid, IdBytes: pemBytes}
	idBytes, err := proto.Marshal(sId)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal a SerializedIdentity structure for identity %s", id.id)
	}

	return idBytes, nil
}

func (id *identity) getHashOpt(hashFamily string) (bccsp.HashOpts, error) {
	switch hashFamily {
	case bccsp.SHA2:
		return bccsp.GetHashOpt(bccsp.SHA256)
	case bccsp.SHA3:
		return bccsp.GetHashOpt(bccsp.SHA3_256)
	}
	return nil, errors.Errorf("hash familiy not recognized [%s]", hashFamily)
}

type signingidentity struct {
//我们从一个基本身份嵌入一切
	identity

//签名者对应于可以从此标识生成签名的对象
	signer crypto.Signer
}

func newSigningIdentity(cert *x509.Certificate, pk bccsp.Key, signer crypto.Signer, msp *bccspmsp) (SigningIdentity, error) {
//mspidentitylogger.infof（“正在为ID%s创建签名标识实例”，ID）
	mspId, err := newIdentity(cert, pk, msp)
	if err != nil {
		return nil, err
	}
	return &signingidentity{identity: *mspId.(*identity), signer: signer}, nil
}

//签名在由此实例签名的消息上生成签名
func (id *signingidentity) Sign(msg []byte) ([]byte, error) {
//mspidentitylogger.infof（“签名消息”）

//计算哈希
	hashOpt, err := id.getHashOpt(id.msp.cryptoConfig.SignatureHashFamily)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting hash function options")
	}

	digest, err := id.msp.bccsp.Hash(msg, hashOpt)
	if err != nil {
		return nil, errors.WithMessage(err, "failed computing digest")
	}

	if len(msg) < 32 {
		mspIdentityLogger.Debugf("Sign: plaintext: %X \n", msg)
	} else {
		mspIdentityLogger.Debugf("Sign: plaintext: %X...%X \n", msg[0:16], msg[len(msg)-16:])
	}
	mspIdentityLogger.Debugf("Sign: digest: %X \n", digest)

//符号
	return id.signer.Sign(rand.Reader, digest, nil)
}

//GetPublicVersion返回此标识的公共版本，
//也就是说，只能验证消息而不能签名的消息
func (id *signingidentity) GetPublicVersion() Identity {
	return &id.identity
}
