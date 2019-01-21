
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
*/

package handlers_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//
//go：生成伪造者-o模拟/发行者_secret_key.go-伪造名称发行者secret key。发行密钥
//go:生成伪造者-o mock/issuer_public_key.go-fake name issuer public key。发行人公钥
//go：生成仿冒者-o mock/user.go-forke name user。用户
//去：生成伪造者-o mock/big.go-伪造名字big。大的
//go：生成伪造者-o mock/ecp.go-伪造名称ecp。欧洲经委会
//go：生成伪造者-o mock/credrequest.go-fake name credrequest。克里德里斯特
//go：生成伪造者-o mock/credential.go-伪造姓名凭证。凭据
//go：生成伪造者-o mock/revocation.go-伪造名称撤销。撤销
//go：生成伪造者-o mock/signature_scheme.go-fake name signature scheme。签字仪式
//go：生成伪造者-o mock/nymsignature_scheme.go-伪造名称nymsignature scheme。姓名签名

func TestPlain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plain Suite")
}
