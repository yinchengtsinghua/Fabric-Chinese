
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

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


package msp

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMSPWithIntermediateCAs(t *testing.T) {
//testdata/intermediate包含具有
//1）密钥和签名证书（用于填充默认签名标识）；
//signcert不是由CA直接签名，而是由中间CA签名
//2）intermediatecert是由CA签署的中间CA。
//3）cacert是签署中间人的CA
	thisMSP := getLocalMSP(t, "testdata/intermediate")

//此MSP将信任由CA直接或由中间人签署的任何证书。

	id, err := thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

//确保我们正确验证身份
	err = thisMSP.Validate(id.GetPublicVersion())
	assert.NoError(t, err)

//确保使用中间CA验证MSP的身份
//本地MSP失败
	err = localMsp.Validate(id.GetPublicVersion())
	assert.Error(t, err)

//确保验证本地MSP的身份
//使用带有中间CA的MSP失败
	localMSPID, err := localMsp.GetDefaultSigningIdentity()
	assert.NoError(t, err)
	err = thisMSP.Validate(localMSPID.GetPublicVersion())
	assert.Error(t, err)
}

func TestMSPWithExternalIntermediateCAs(t *testing.T) {
//testdata/external包含测试MSP安装程序的凭据
//与testdata/intermediate相同，但它具有
//独立于织物环境使用
//OpenSSL。消毒证书可能导致
//从原始数据中使用的签名算法
//证书文件。原始证书字节的哈希和
//原始证书和
//导入到MSP中的一个可能会错误地失败。

	thisMSP := getLocalMSP(t, "testdata/external")

//此MSP将信任仅由中间人签名的任何证书

	id, err := thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

//确保我们正确验证身份
	err = thisMSP.Validate(id.GetPublicVersion())
	assert.NoError(t, err)
}

func TestIntermediateCAIdentityValidity(t *testing.T) {
//testdata/intermediate包含具有
//1）密钥和签名证书（用于填充默认签名标识）；
//signcert不是由CA直接签名，而是由中间CA签名
//2）intermediatecert是由CA签署的中间CA。
//3）cacert是签署中间人的CA
	thisMSP := getLocalMSP(t, "testdata/intermediate")

	id := thisMSP.(*bccspmsp).intermediateCerts[0]
	assert.Error(t, id.Validate())
}

func TestMSPWithIntermediateCAs2(t *testing.T) {
//testdata/intermediate2包含具有
//1）密钥和签名证书（用于填充默认签名标识）；
//signcert不是由CA直接签名，而是由中间CA签名
//2）intermediatecert是由CA签署的中间CA。
//3）cacert是签署中间人的CA
//4）user2 cert是由CA直接签署的身份证书。
//因此，验证应该失败。
	thisMSP := getLocalMSP(t, filepath.Join("testdata", "intermediate2"))

//默认签名标识由中间CA签名，
//验证不应返回错误
	id, err := thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)
	err = thisMSP.Validate(id.GetPublicVersion())
	assert.NoError(t, err)

//用户2证书已由根CA签名，验证必须失败
	pem, err := readPemFile(filepath.Join("testdata", "intermediate2", "users", "user2-cert.pem"))
	assert.NoError(t, err)
	id2, _, err := thisMSP.(*bccspmsp).getIdentityFromConf(pem)
	assert.NoError(t, err)
	err = thisMSP.Validate(id2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid validation chain. Parent certificate should be a leaf of the certification tree ")
}
