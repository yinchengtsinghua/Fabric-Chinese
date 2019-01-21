
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

package bccsp

import (
	"crypto"
)

//revocation algorithm标识撤销算法
type RevocationAlgorithm int32

const (
//Idemix常数用于识别Idemix相关算法
	IDEMIX = "IDEMIX"
)

const (
//algorevocation意味着不支持撤销
	AlgNoRevocation RevocationAlgorithm = iota
)

//IDemix IssuerKeyGenOpts包含IDemix颁发者密钥生成选项。
//可以选择传递attributes列表
type IdemixIssuerKeyGenOpts struct {
//临时通知密钥是否是临时的
	Temporary bool
//attributeName是一个属性列表
	AttributeNames []string
}

//算法返回密钥生成算法标识符（要使用）。
func (*IdemixIssuerKeyGenOpts) Algorithm() string {
	return IDEMIX
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (o *IdemixIssuerKeyGenOpts) Ephemeral() bool {
	return o.Temporary
}

//IDemix IssuerPublickeyImportOpts包含用于导入IDemix颁发者公钥的选项。
type IdemixIssuerPublicKeyImportOpts struct {
	Temporary bool
//attributeName是一个属性列表，用于确保导入公钥具有
	AttributeNames []string
}

//算法返回密钥生成算法标识符（要使用）。
func (*IdemixIssuerPublicKeyImportOpts) Algorithm() string {
	return IDEMIX
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (o *IdemixIssuerPublicKeyImportOpts) Ephemeral() bool {
	return o.Temporary
}

//IDemix用户密钥包含用于生成IDemix凭据密钥的选项。
type IdemixUserSecretKeyGenOpts struct {
	Temporary bool
}

//算法返回密钥生成算法标识符（要使用）。
func (*IdemixUserSecretKeyGenOpts) Algorithm() string {
	return IDEMIX
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (o *IdemixUserSecretKeyGenOpts) Ephemeral() bool {
	return o.Temporary
}

//IDemix用户密钥导入选项包含用于导入IDemix凭据密钥的选项。
type IdemixUserSecretKeyImportOpts struct {
	Temporary bool
}

//算法返回密钥生成算法标识符（要使用）。
func (*IdemixUserSecretKeyImportOpts) Algorithm() string {
	return IDEMIX
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (o *IdemixUserSecretKeyImportOpts) Ephemeral() bool {
	return o.Temporary
}

//IDemixNymKeyDerivationOpts包含从
//与指定的颁发者公钥相关的凭据密钥
type IdemixNymKeyDerivationOpts struct {
//临时通知密钥是否是临时的
	Temporary bool
//Issuerpk是发行者的公钥
	IssuerPK Key
}

//算法返回密钥派生算法标识符（要使用）。
func (*IdemixNymKeyDerivationOpts) Algorithm() string {
	return IDEMIX
}

//如果派生的键必须是短暂的，则短暂返回true，
//否则为假。
func (o *IdemixNymKeyDerivationOpts) Ephemeral() bool {
	return o.Temporary
}

//IssuerPublickey返回用于派生的颁发者公钥
//一个新的无法从凭证密钥链接的假名
func (o *IdemixNymKeyDerivationOpts) IssuerPublicKey() Key {
	return o.IssuerPK
}

//idemixnympublicKeyimportopts包含用于导入笔名的公共部分的选项。
type IdemixNymPublicKeyImportOpts struct {
//临时通知密钥是否是临时的
	Temporary bool
}

//算法返回密钥派生算法标识符（要使用）。
func (*IdemixNymPublicKeyImportOpts) Algorithm() string {
	return IDEMIX
}

//如果派生的键必须是短暂的，则短暂返回true，
//否则为假。
func (o *IdemixNymPublicKeyImportOpts) Ephemeral() bool {
	return o.Temporary
}

//IDemicCredentialRequestSignerOpts包含创建IDemicCredential请求的选项。
type IdemixCredentialRequestSignerOpts struct {
//属性包含要包含在
//凭据。这些索引与IDemiMixIssuerKeyGenopts attributeName有关。
	Attributes []int
//Issuerpk是发行者的公钥
	IssuerPK Key
//ISSUERNONCE由颁发者生成，由客户端用于生成凭证请求。
//
	IssuerNonce []byte
//hashfun是要使用的哈希函数
	H crypto.Hash
}

func (o *IdemixCredentialRequestSignerOpts) HashFunc() crypto.Hash {
	return o.H
}

//IssuerPublickey返回用于派生的颁发者公钥
//一个新的无法从凭证密钥链接的假名
func (o *IdemixCredentialRequestSignerOpts) IssuerPublicKey() Key {
	return o.IssuerPK
}

//IDemix属性类型表示IDemix属性的类型
type IdemixAttributeType int

const (
//IDemixHiddenAttribute表示隐藏属性
	IdemixHiddenAttribute IdemixAttributeType = iota
//IDemix字符串属性表示一个字节序列
	IdemixBytesAttribute
//idemixintattribute表示一个int
	IdemixIntAttribute
)

type IdemixAttribute struct {
//类型是属性的类型
	Type IdemixAttributeType
//值是属性的值
	Value interface{}
}

//IDemicCredentialSignerOpts包含从凭证请求开始生成凭证的选项
type IdemixCredentialSignerOpts struct {
//要包含在凭据中的属性。此处不允许使用IDemixHiddenAttribute
	Attributes []IdemixAttribute
//Issuerpk是发行者的公钥
	IssuerPK Key
//hashfun是要使用的哈希函数
	H crypto.Hash
}

//hashfunc返回用于生成的哈希函数的标识符
//传递给signer.sign的消息，否则为零以指示否
//散列操作已完成。
func (o *IdemixCredentialSignerOpts) HashFunc() crypto.Hash {
	return o.H
}

func (o *IdemixCredentialSignerOpts) IssuerPublicKey() Key {
	return o.IssuerPK
}

//IDemisSignerOpts包含生成IDemix签名的选项
type IdemixSignerOpts struct {
//Nym是要使用的假名
	Nym Key
//Issuerpk是发行者的公钥
	IssuerPK Key
//凭证是由颁发者签名的凭证的字节表示形式。
	Credential []byte
//属性指定应该公开哪些属性，哪些不公开。
//if属性[i].type=idemixHiddenAttribute
//则第i个凭证属性不应被公开，否则第i个凭证属性
//凭证属性将被公开。
//验证时，如果第i个属性被公开（属性[i].类型！=idemixhiddenattribute），则
//则必须相应地设置属性[i].值。
	Attributes []IdemixAttribute
//RHindex是包含撤销处理程序的属性的索引。
//请注意，此属性不能被放弃
	RhIndex int
//CRI包含凭证吊销信息
	CRI []byte
//epoch是签名应针对的吊销epoch
	Epoch int
//RevocationPublicKey是吊销公钥
	RevocationPublicKey Key
//h是要使用的哈希函数。
	H crypto.Hash
}

func (o *IdemixSignerOpts) HashFunc() crypto.Hash {
	return o.H
}

//idemixnymsigneropts包含用于生成idemix假名签名的选项。
type IdemixNymSignerOpts struct {
//Nym是要使用的假名
	Nym Key
//Issuerpk是发行者的公钥
	IssuerPK Key
//h是要使用的哈希函数。
	H crypto.Hash
}

//hashfunc返回用于生成的哈希函数的标识符
//传递给signer.sign的消息，否则为零以指示否
//散列操作已完成。
func (o *IdemixNymSignerOpts) HashFunc() crypto.Hash {
	return o.H
}

//idemixRevocationKeyGenOpts包含用于生成idemix吊销密钥的选项。
type IdemixRevocationKeyGenOpts struct {
//临时通知密钥是否是临时的
	Temporary bool
}

//算法返回密钥生成算法标识符（要使用）。
func (*IdemixRevocationKeyGenOpts) Algorithm() string {
	return IDEMIX
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (o *IdemixRevocationKeyGenOpts) Ephemeral() bool {
	return o.Temporary
}

//idemixRevocationPublicKeyImportOpts包含用于导入idemix吊销公钥的选项。
type IdemixRevocationPublicKeyImportOpts struct {
	Temporary bool
}

//算法返回密钥生成算法标识符（要使用）。
func (*IdemixRevocationPublicKeyImportOpts) Algorithm() string {
	return IDEMIX
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (o *IdemixRevocationPublicKeyImportOpts) Ephemeral() bool {
	return o.Temporary
}

//IDemix CRISignerOpts包含生成IDemix CRI的选项。
//CRI应该由发行机构生成，并且
//可以使用吊销公钥公开验证。
type IdemixCRISignerOpts struct {
	Epoch               int
	RevocationAlgorithm RevocationAlgorithm
	UnrevokedHandles    [][]byte
//h是要使用的哈希函数。
	H crypto.Hash
}

func (o *IdemixCRISignerOpts) HashFunc() crypto.Hash {
	return o.H
}
