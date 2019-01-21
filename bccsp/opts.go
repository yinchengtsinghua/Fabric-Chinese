
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package bccsp

const (
//ECDSA椭圆曲线数字签名算法（密钥生成、导入、签名、验证）
//默认安全级别。
//每个BCCSP可能支持，也可能不支持默认安全级别。如果不支持，则
//将返回一个错误。
	ECDSA = "ECDSA"

//P-256曲线上的ECDSA椭圆曲线数字签名算法
	ECDSAP256 = "ECDSAP256"

//基于P-384曲线的ECDSA椭圆曲线数字签名算法
	ECDSAP384 = "ECDSAP384"

//ecdsarerand ecdsa密钥重新随机化
	ECDSAReRand = "ECDSA_RERAND"

//RSA处于默认安全级别。
//每个BCCSP可能支持，也可能不支持默认安全级别。如果不支持，则
//将返回一个错误。
	RSA = "RSA"
//1024位安全级别的RSA。
	RSA1024 = "RSA1024"
//2048位安全级别的RSA。
	RSA2048 = "RSA2048"
//RSA的3072位安全级别。
	RSA3072 = "RSA3072"
//RSA处于4096位安全级别。
	RSA4096 = "RSA4096"

//默认安全级别的AES高级加密标准。
//每个BCCSP可能支持，也可能不支持默认安全级别。如果不支持，则
//将返回一个错误。
	AES = "AES"
//128位安全级别的高级加密标准
	AES128 = "AES128"
//高级加密标准，192位安全级别
	AES192 = "AES192"
//256位安全级别的高级加密标准
	AES256 = "AES256"

//HMAC键控哈希消息验证代码
	HMAC = "HMAC"
//hmac truncated 256 hmac以256位截断。
	HMACTruncated256 = "HMAC_TRUNCATED_256"

//使用默认族的安全哈希算法。
//每个BCCSP可能支持，也可能不支持默认安全级别。如果不支持，则
//将返回一个错误。
	SHA = "SHA"

//sha2是sha2哈希家族的标识符
	SHA2 = "SHA2"
//sha3是sha3散列族的标识符
	SHA3 = "SHA3"

//沙256
	SHA256 = "SHA256"
//沙38
	SHA384 = "SHA384"
//Sa3Y256
	SHA3_256 = "SHA3_256"
//Sa3Y38
	SHA3_384 = "SHA3_384"

//用于X509证书相关操作的X509证书标签
	X509Certificate = "X509Certificate"
)

//ecdsakeygenopts包含用于生成ecdsa密钥的选项。
type ECDSAKeyGenOpts struct {
	Temporary bool
}

//算法返回密钥生成算法标识符（要使用）。
func (opts *ECDSAKeyGenOpts) Algorithm() string {
	return ECDSA
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *ECDSAKeyGenOpts) Ephemeral() bool {
	return opts.Temporary
}

//ecdsapkixpublickeyimportopts包含pkix格式的ecdsa公钥导入选项
type ECDSAPKIXPublicKeyImportOpts struct {
	Temporary bool
}

//算法返回密钥导入算法标识符（要使用）。
func (opts *ECDSAPKIXPublicKeyImportOpts) Algorithm() string {
	return ECDSA
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *ECDSAPKIXPublicKeyImportOpts) Ephemeral() bool {
	return opts.Temporary
}

//ecdsaprivatekeimportopts包含用于以der格式导入ecdsa密钥的选项
//或pkcs 8格式。
type ECDSAPrivateKeyImportOpts struct {
	Temporary bool
}

//算法返回密钥导入算法标识符（要使用）。
func (opts *ECDSAPrivateKeyImportOpts) Algorithm() string {
	return ECDSA
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *ECDSAPrivateKeyImportOpts) Ephemeral() bool {
	return opts.Temporary
}

//ecdsagopublickeyportopts包含从ecdsa.publickey导入ecdsa密钥的选项
type ECDSAGoPublicKeyImportOpts struct {
	Temporary bool
}

//算法返回密钥导入算法标识符（要使用）。
func (opts *ECDSAGoPublicKeyImportOpts) Algorithm() string {
	return ECDSA
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *ECDSAGoPublicKeyImportOpts) Ephemeral() bool {
	return opts.Temporary
}

//ecdsarerandkeyopts包含用于ecdsa密钥重新随机化的选项。
type ECDSAReRandKeyOpts struct {
	Temporary bool
	Expansion []byte
}

//算法返回密钥派生算法标识符（要使用）。
func (opts *ECDSAReRandKeyOpts) Algorithm() string {
	return ECDSAReRand
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *ECDSAReRandKeyOpts) Ephemeral() bool {
	return opts.Temporary
}

//ExpansionValue返回重新随机化因子
func (opts *ECDSAReRandKeyOpts) ExpansionValue() []byte {
	return opts.Expansion
}

//aeskeygenopts包含用于在默认安全级别生成aes密钥的选项
type AESKeyGenOpts struct {
	Temporary bool
}

//算法返回密钥生成算法标识符（要使用）。
func (opts *AESKeyGenOpts) Algorithm() string {
	return AES
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *AESKeyGenOpts) Ephemeral() bool {
	return opts.Temporary
}

//hmactruncated256aesDeriveKeyOpts包含用于截断hmac的选项
//256位的密钥派生。
type HMACTruncated256AESDeriveKeyOpts struct {
	Temporary bool
	Arg       []byte
}

//算法返回密钥派生算法标识符（要使用）。
func (opts *HMACTruncated256AESDeriveKeyOpts) Algorithm() string {
	return HMACTruncated256
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *HMACTruncated256AESDeriveKeyOpts) Ephemeral() bool {
	return opts.Temporary
}

//参数返回要传递给HMAC的参数
func (opts *HMACTruncated256AESDeriveKeyOpts) Argument() []byte {
	return opts.Arg
}

//hmacDeliveKeyOpts包含用于hmac密钥派生的选项。
type HMACDeriveKeyOpts struct {
	Temporary bool
	Arg       []byte
}

//算法返回密钥派生算法标识符（要使用）。
func (opts *HMACDeriveKeyOpts) Algorithm() string {
	return HMAC
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *HMACDeriveKeyOpts) Ephemeral() bool {
	return opts.Temporary
}

//参数返回要传递给HMAC的参数
func (opts *HMACDeriveKeyOpts) Argument() []byte {
	return opts.Arg
}

//aes256importkeypts包含用于导入aes 256键的选项。
type AES256ImportKeyOpts struct {
	Temporary bool
}

//算法返回密钥导入算法标识符（要使用）。
func (opts *AES256ImportKeyOpts) Algorithm() string {
	return AES
}

//如果生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *AES256ImportKeyOpts) Ephemeral() bool {
	return opts.Temporary
}

//hmacimportkeyopts包含导入hmac键的选项。
type HMACImportKeyOpts struct {
	Temporary bool
}

//算法返回密钥导入算法标识符（要使用）。
func (opts *HMACImportKeyOpts) Algorithm() string {
	return HMAC
}

//如果生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *HMACImportKeyOpts) Ephemeral() bool {
	return opts.Temporary
}

//Shaopts包含计算sha的选项。
type SHAOpts struct {
}

//Algorithm返回哈希算法标识符（要使用）。
func (opts *SHAOpts) Algorithm() string {
	return SHA
}

//rsakeygenopts包含用于生成rsa密钥的选项。
type RSAKeyGenOpts struct {
	Temporary bool
}

//算法返回密钥生成算法标识符（要使用）。
func (opts *RSAKeyGenOpts) Algorithm() string {
	return RSA
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *RSAKeyGenOpts) Ephemeral() bool {
	return opts.Temporary
}

//ecdsagopublickeyportopts包含从rsa.publickey导入rsa密钥的选项
type RSAGoPublicKeyImportOpts struct {
	Temporary bool
}

//算法返回密钥导入算法标识符（要使用）。
func (opts *RSAGoPublicKeyImportOpts) Algorithm() string {
	return RSA
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *RSAGoPublicKeyImportOpts) Ephemeral() bool {
	return opts.Temporary
}

//X509PublicKeyImportOpts包含从X509证书导入公钥的选项
type X509PublicKeyImportOpts struct {
	Temporary bool
}

//算法返回密钥导入算法标识符（要使用）。
func (opts *X509PublicKeyImportOpts) Algorithm() string {
	return X509Certificate
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *X509PublicKeyImportOpts) Ephemeral() bool {
	return opts.Temporary
}
