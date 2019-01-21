
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


package policies

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImplicitMetaParserWrongTokenCount(t *testing.T) {
	errorMatch := "expected two space separated tokens, but got"

	t.Run("NoArgs", func(t *testing.T) {
		res, err := ImplicitMetaFromString("")
		assert.Nil(t, res)
		require.Error(t, err)
		assert.Regexp(t, errorMatch, err.Error())
	})

	t.Run("OneArg", func(t *testing.T) {
		res, err := ImplicitMetaFromString("ANY")
		assert.Nil(t, res)
		require.Error(t, err)
		assert.Regexp(t, errorMatch, err.Error())
	})

	t.Run("ThreeArgs", func(t *testing.T) {
		res, err := ImplicitMetaFromString("ANY of these")
		assert.Nil(t, res)
		require.Error(t, err)
		assert.Regexp(t, errorMatch, err.Error())
	})
}

func TestImplicitMetaParserBadRule(t *testing.T) {
	res, err := ImplicitMetaFromString("BAD Rule")
	assert.Nil(t, res)
	require.Error(t, err)
	assert.Regexp(t, "unknown rule type 'BAD'", err.Error())
}

func TestImplicitMetaParserGreenPath(t *testing.T) {
	for _, rule := range []cb.ImplicitMetaPolicy_Rule{cb.ImplicitMetaPolicy_ANY, cb.ImplicitMetaPolicy_ALL, cb.ImplicitMetaPolicy_MAJORITY} {
		t.Run(rule.String(), func(t *testing.T) {
			subPolicy := "foo"
			res, err := ImplicitMetaFromString(fmt.Sprintf("%v %s", rule, subPolicy))
			require.NoError(t, err)
			assert.True(t, proto.Equal(res, &cb.ImplicitMetaPolicy{
				SubPolicy: subPolicy,
				Rule:      rule,
			}))
		})
	}
}
