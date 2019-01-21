
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

	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/stretchr/testify/assert"
)

func TestRevocation(t *testing.T) {
//测试数据/撤销
//1）密钥和签名证书（用于填充默认签名标识）；
//2）cacert是签署中间人的CA；
//3）撤销签名证书的撤销名单。
	thisMSP := getLocalMSP(t, "testdata/revocation")

	id, err := thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

//与此ID关联的证书被吊销，因此验证应该失败！
	err = id.Validate()
	assert.Error(t, err)

//此MSP与前一个相同，只有1个差异：
//CRL上的签名无效
	thisMSP = getLocalMSP(t, "testdata/revocation2")

	id, err = thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

//与此ID关联的证书被吊销，但CRL上的签名无效
//所以验证应该成功
	err = id.Validate()
	assert.NoError(t, err, "Identity found revoked although the signature over the CRL is invalid")
}

func TestIdentityPolicyPrincipalAgainstRevokedIdentity(t *testing.T) {
//测试数据/撤销
//1）密钥和签名证书（用于填充默认签名标识）；
//2）cacert是签署中间人的CA；
//3）撤销签名证书的撤销名单。
	thisMSP := getLocalMSP(t, "testdata/revocation")

	id, err := thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

	idSerialized, err := id.Serialize()
	assert.NoError(t, err)

	principal := &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_IDENTITY,
		Principal:               idSerialized}

	err = id.SatisfiesPrincipal(principal)
	assert.Error(t, err)
}

func TestRevokedIntermediateCA(t *testing.T) {
//测试数据/revokedica
//1）密钥和签名证书（用于填充默认签名标识）；
//2）cacert是签署中间人的CA；
//3）吊销中间CA证书的吊销列表
	dir := "testdata/revokedica"
	conf, err := GetLocalMspConfig(dir, nil, "SampleOrg")
	assert.NoError(t, err)

	thisMSP, err := newBccspMsp(MSPv1_0)
	assert.NoError(t, err)
	ks, err := sw.NewFileBasedKeyStore(nil, filepath.Join(dir, "keystore"), true)
	assert.NoError(t, err)
	csp, err := sw.NewWithParams(256, "SHA2", ks)
	assert.NoError(t, err)
	thisMSP.(*bccspmsp).bccsp = csp

	err = thisMSP.Setup(conf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CA Certificate is not valid, ")
}
