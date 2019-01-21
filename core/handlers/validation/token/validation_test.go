
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


package token_test

import (
	"testing"

	"github.com/hyperledger/fabric/core/handlers/validation/token"
	"github.com/stretchr/testify/assert"
)

func TestValidationFactory_New(t *testing.T) {
	factory := &token.ValidationFactory{}
	plugin := factory.New()
	assert.NotNil(t, plugin)
}

func TestValidation_Validate(t *testing.T) {
	factory := &token.ValidationFactory{}
	plugin := factory.New()

	err := plugin.Init()
	assert.NoError(t, err)

//验证返回零，无论什么！
	err = plugin.Validate(nil, "", 0, 0, nil)
	assert.NoError(t, err)
}
