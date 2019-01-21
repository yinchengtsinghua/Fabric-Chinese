
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

import "io"

//aes128keygenopts包含128安全级别的aes密钥生成选项
type AES128KeyGenOpts struct {
	Temporary bool
}

//算法返回密钥生成算法标识符（要使用）。
func (opts *AES128KeyGenOpts) Algorithm() string {
	return AES128
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *AES128KeyGenOpts) Ephemeral() bool {
	return opts.Temporary
}

//aes192keygenopts包含在192安全级别生成aes密钥的选项
type AES192KeyGenOpts struct {
	Temporary bool
}

//算法返回密钥生成算法标识符（要使用）。
func (opts *AES192KeyGenOpts) Algorithm() string {
	return AES192
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *AES192KeyGenOpts) Ephemeral() bool {
	return opts.Temporary
}

//aes256keyGenopts包含256安全级别的aes密钥生成选项
type AES256KeyGenOpts struct {
	Temporary bool
}

//算法返回密钥生成算法标识符（要使用）。
func (opts *AES256KeyGenOpts) Algorithm() string {
	return AES256
}

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
func (opts *AES256KeyGenOpts) Ephemeral() bool {
	return opts.Temporary
}

//AESCBCPKCS7模式包含CBC模式下的AES加密选项
//用pkcs7填充。
//注意，iv和prng都可以为零。在这种情况下，BCCSP实现
//应该使用加密安全prng对iv进行采样。
//还要注意，iv或prng可能与nil不同。
type AESCBCPKCS7ModeOpts struct {
//iv是基础密码要使用的初始化向量。
//IV的长度必须与块的块大小相同。
//仅当与nil不同时才使用。
	IV []byte
//prng是基础密码要使用的prng的实例。
//仅当与nil不同时才使用。
	PRNG io.Reader
}
