
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


package entities

import (
	"encoding/pem"
	"reflect"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
)

/*******************/
/*结构定义*/
/*******************/

//BCCSpentity是实体接口的实现
//保存BCCSP实例
type BCCSPEntity struct {
	IDstr string
	BCCSP bccsp.BCCSP
}

//BCCSpsingentity是签名接口的一个实现
type BCCSPSignerEntity struct {
	BCCSPEntity
	SKey  bccsp.Key
	SOpts bccsp.SignerOpts
	HOpts bccsp.HashOpts
}

//bccspencrypterEntity是encrypterEntity接口的一个实现
type BCCSPEncrypterEntity struct {
	BCCSPEntity
	EKey  bccsp.Key
	EOpts bccsp.EncrypterOpts
	DOpts bccsp.DecrypterOpts
}

//bccspencyptersignentity是encryptersignentity接口的一个实现
type BCCSPEncrypterSignerEntity struct {
	BCCSPEncrypterEntity
	BCCSPSignerEntity
}

/*****/
/*构造函数*/
/*****/

//newaes256encrypterentity返回的加密程序实体是
//能够使用pkcs 7填充执行aes 256位加密。
//可选地，在这种情况下，可以提供静脉输液
//加密；奥特耶韦斯，随机产生一个。
func NewAES256EncrypterEntity(ID string, b bccsp.BCCSP, key, IV []byte) (*BCCSPEncrypterEntity, error) {
	if b == nil {
		return nil, errors.New("nil BCCSP")
	}

	k, err := b.KeyImport(key, &bccsp.AES256ImportKeyOpts{Temporary: true})
	if err != nil {
		return nil, errors.WithMessage(err, "bccspInst.KeyImport failed")
	}

	return NewEncrypterEntity(ID, b, k, &bccsp.AESCBCPKCS7ModeOpts{IV: IV}, &bccsp.AESCBCPKCS7ModeOpts{})
}

//NewEncrypterEntity returns an EncrypterEntity that is capable
//使用i）提供的bccsp实例进行加密；
//
//和解密选项。提供实体的标识符
//作为一种论据，呼叫者有责任
//以有意义的方式选择它
func NewEncrypterEntity(ID string, bccsp bccsp.BCCSP, eKey bccsp.Key, eOpts bccsp.EncrypterOpts, dOpts bccsp.DecrypterOpts) (*BCCSPEncrypterEntity, error) {
	if ID == "" {
		return nil, errors.New("NewEntity error: empty ID")
	}

	if bccsp == nil {
		return nil, errors.New("NewEntity error: nil bccsp")
	}

	if eKey == nil {
		return nil, errors.New("NewEntity error: nil keys")
	}

	return &BCCSPEncrypterEntity{
		BCCSPEntity: BCCSPEntity{
			IDstr: ID,
			BCCSP: bccsp,
		},
		EKey:  eKey,
		EOpts: eOpts,
		DOpts: dOpts,
	}, nil
}

//newecdsasignentity返回能够使用ecdsa签名的签名者实体
func NewECDSASignerEntity(ID string, b bccsp.BCCSP, signKeyBytes []byte) (*BCCSPSignerEntity, error) {
	if b == nil {
		return nil, errors.New("nil BCCSP")
	}

	bl, _ := pem.Decode(signKeyBytes)
	if bl == nil {
		return nil, errors.New("pem.Decode returns nil")
	}

	signKey, err := b.KeyImport(bl.Bytes, &bccsp.ECDSAPrivateKeyImportOpts{Temporary: true})
	if err != nil {
		return nil, errors.WithMessage(err, "bccspInst.KeyImport failed")
	}

	return NewSignerEntity(ID, b, signKey, nil, &bccsp.SHA256Opts{})
}

//NewECDSaverifiedEntity返回能够使用ECDSA进行验证的验证程序实体
func NewECDSAVerifierEntity(ID string, b bccsp.BCCSP, signKeyBytes []byte) (*BCCSPSignerEntity, error) {
	if b == nil {
		return nil, errors.New("nil BCCSP")
	}

	bl, _ := pem.Decode(signKeyBytes)
	if bl == nil {
		return nil, errors.New("pem.Decode returns nil")
	}

	signKey, err := b.KeyImport(bl.Bytes, &bccsp.ECDSAPKIXPublicKeyImportOpts{Temporary: true})
	if err != nil {
		return nil, errors.WithMessage(err, "bccspInst.KeyImport failed")
	}

	return NewSignerEntity(ID, b, signKey, nil, &bccsp.SHA256Opts{})
}

//NewSignerenty返回签名
func NewSignerEntity(ID string, bccsp bccsp.BCCSP, sKey bccsp.Key, sOpts bccsp.SignerOpts, hOpts bccsp.HashOpts) (*BCCSPSignerEntity, error) {
	if ID == "" {
		return nil, errors.New("NewSignerEntity error: empty ID")
	}

	if bccsp == nil {
		return nil, errors.New("NewSignerEntity error: nil bccsp")
	}

	if sKey == nil {
		return nil, errors.New("NewSignerEntity error: nil key")
	}

	return &BCCSPSignerEntity{
		BCCSPEntity: BCCSPEntity{
			IDstr: ID,
			BCCSP: bccsp,
		},
		SKey:  sKey,
		SOpts: sOpts,
		HOpts: hOpts,
	}, nil
}

//newaes256encrypterecdsasignerenty返回的加密程序实体是
//
//使用ECDSA签名
func NewAES256EncrypterECDSASignerEntity(ID string, b bccsp.BCCSP, encKeyBytes, signKeyBytes []byte) (*BCCSPEncrypterSignerEntity, error) {
	if b == nil {
		return nil, errors.New("nil BCCSP")
	}

	encKey, err := b.KeyImport(encKeyBytes, &bccsp.AES256ImportKeyOpts{Temporary: true})
	if err != nil {
		return nil, errors.WithMessage(err, "bccspInst.KeyImport failed")
	}

	bl, _ := pem.Decode(signKeyBytes)
	if bl == nil {
		return nil, errors.New("pem.Decode returns nil")
	}

	signKey, err := b.KeyImport(bl.Bytes, &bccsp.ECDSAPrivateKeyImportOpts{Temporary: true})
	if err != nil {
		return nil, errors.WithMessage(err, "bccspInst.KeyImport failed")
	}

	return NewEncrypterSignerEntity(ID, b, encKey, signKey, &bccsp.AESCBCPKCS7ModeOpts{}, &bccsp.AESCBCPKCS7ModeOpts{}, nil, &bccsp.SHA256Opts{})
}

//newencryptersignentity返回一个encryptersignentity
//（也是一种加密技术）能够
//执行加密并使用
//i）提供的bccsp实例；i i）提供的加密
//以及签名密钥和iii）提供的加密、解密，
//签名和哈希选项。实体的标识符为
//作为参数提供-这是调用方的责任
//以有意义的方式选择它
func NewEncrypterSignerEntity(ID string, bccsp bccsp.BCCSP, eKey, sKey bccsp.Key, eOpts bccsp.EncrypterOpts, dOpts bccsp.DecrypterOpts, sOpts bccsp.SignerOpts, hOpts bccsp.HashOpts) (*BCCSPEncrypterSignerEntity, error) {
	if ID == "" {
		return nil, errors.New("NewEntity error: empty ID")
	}

	if bccsp == nil {
		return nil, errors.New("NewEntity error: nil bccsp")
	}

	if eKey == nil || sKey == nil {
		return nil, errors.New("NewEntity error: nil keys")
	}

	return &BCCSPEncrypterSignerEntity{
		BCCSPEncrypterEntity: BCCSPEncrypterEntity{
			BCCSPEntity: BCCSPEntity{
				IDstr: ID,
				BCCSP: bccsp,
			},
			EKey:  eKey,
			EOpts: eOpts,
			DOpts: dOpts,
		},
		BCCSPSignerEntity: BCCSPSignerEntity{
			BCCSPEntity: BCCSPEntity{
				IDstr: ID,
				BCCSP: bccsp,
			},
			SKey:  sKey,
			SOpts: sOpts,
			HOpts: hOpts,
		},
	}, nil
}

/*****/
/*方法*/
/*****/

func (e *BCCSPEntity) ID() string {
	return e.IDstr
}

func (e *BCCSPEncrypterEntity) Encrypt(plaintext []byte) ([]byte, error) {
	return e.BCCSP.Encrypt(e.EKey, plaintext, e.EOpts)
}

func (e *BCCSPEncrypterEntity) Decrypt(ciphertext []byte) ([]byte, error) {
	return e.BCCSP.Decrypt(e.EKey, ciphertext, e.DOpts)
}

func (this *BCCSPEncrypterEntity) Equals(e Entity) bool {
	if that, rightType := e.(*BCCSPEncrypterEntity); rightType {
		return compare(this.EKey, that.EKey)
	}

	return false
}

func (pe *BCCSPEncrypterEntity) Public() (Entity, error) {
	var err error
	eKeyPub := pe.EKey

	if !pe.EKey.Symmetric() {
		if eKeyPub, err = pe.EKey.PublicKey(); err != nil {
			return nil, errors.WithMessage(err, "public error, eKey.PublicKey returned")
		}
	}

	return &BCCSPEncrypterEntity{
		BCCSPEntity: BCCSPEntity{
			IDstr: pe.IDstr,
			BCCSP: pe.BCCSP,
		},
		DOpts: pe.DOpts,
		EOpts: pe.EOpts,
		EKey:  eKeyPub,
	}, nil
}

func (this *BCCSPSignerEntity) Equals(e Entity) bool {
	if that, rightType := e.(*BCCSPSignerEntity); rightType {
		return compare(this.SKey, that.SKey)
	}

	return false
}

func (e *BCCSPSignerEntity) Public() (Entity, error) {
	var err error
	sKeyPub := e.SKey

	if !e.SKey.Symmetric() {
		if sKeyPub, err = e.SKey.PublicKey(); err != nil {
			return nil, errors.WithMessage(err, "public error, sKey.PublicKey returned")
		}
	}

	return &BCCSPSignerEntity{
		BCCSPEntity: BCCSPEntity{
			IDstr: e.IDstr,
			BCCSP: e.BCCSP,
		},
		HOpts: e.HOpts,
		SOpts: e.SOpts,
		SKey:  sKeyPub,
	}, nil
}

func (e *BCCSPSignerEntity) Sign(msg []byte) ([]byte, error) {
	h, err := e.BCCSP.Hash(msg, e.HOpts)
	if err != nil {
		return nil, errors.WithMessage(err, "sign error: bccsp.Hash returned")
	}

	return e.BCCSP.Sign(e.SKey, h, e.SOpts)
}

func (e *BCCSPSignerEntity) Verify(signature, msg []byte) (bool, error) {
	h, err := e.BCCSP.Hash(msg, e.HOpts)
	if err != nil {
		return false, errors.WithMessage(err, "sign error: bccsp.Hash returned")
	}

	return e.BCCSP.Verify(e.SKey, signature, h, e.SOpts)
}

func (pe *BCCSPEncrypterSignerEntity) Public() (Entity, error) {
	var err error
	eKeyPub := pe.EKey

	if !pe.EKey.Symmetric() {
		if eKeyPub, err = pe.EKey.PublicKey(); err != nil {
			return nil, errors.WithMessage(err, "public error, eKey.PublicKey returned")
		}
	}

	sKeyPub, err := pe.SKey.PublicKey()
	if err != nil {
		return nil, errors.WithMessage(err, "public error, sKey.PublicKey returned")
	}

	return &BCCSPEncrypterSignerEntity{
		BCCSPEncrypterEntity: BCCSPEncrypterEntity{
			BCCSPEntity: BCCSPEntity{
				IDstr: pe.BCCSPEncrypterEntity.IDstr,
				BCCSP: pe.BCCSPEncrypterEntity.BCCSP,
			},
			EKey:  eKeyPub,
			EOpts: pe.EOpts,
			DOpts: pe.DOpts,
		},
		BCCSPSignerEntity: BCCSPSignerEntity{
			BCCSPEntity: BCCSPEntity{
				IDstr: pe.BCCSPEncrypterEntity.IDstr,
				BCCSP: pe.BCCSPEncrypterEntity.BCCSP,
			},
			SKey:  sKeyPub,
			HOpts: pe.HOpts,
			SOpts: pe.SOpts,
		},
	}, nil
}

func (this *BCCSPEncrypterSignerEntity) Equals(e Entity) bool {
	if that, rightType := e.(*BCCSPEncrypterSignerEntity); rightType {
		return compare(this.SKey, that.SKey) && compare(this.EKey, that.EKey)
	} else {
		return false
	}
}

func (e *BCCSPEncrypterSignerEntity) ID() string {
	return e.BCCSPEncrypterEntity.ID()
}

/**************/
/*帮助程序函数*/
/**************/

//
//
//公共版本。这是必需的，因为当我们比较
//two entities, we might compare the public and the private
//同一实体的版本，并期望被告知
//实体是等价的
func compare(this, that bccsp.Key) bool {
	var err error
	if this.Private() {
		this, err = this.PublicKey()
		if err != nil {
			return false
		}
	}
	if that.Private() {
		that, err = that.PublicKey()
		if err != nil {
			return false
		}
	}

	return reflect.DeepEqual(this, that)
}
