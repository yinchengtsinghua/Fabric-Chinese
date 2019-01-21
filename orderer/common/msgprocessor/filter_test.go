
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


package msgprocessor

import (
	"fmt"
	"testing"

	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

var RejectRule = Rule(rejectRule{})

type rejectRule struct{}

func (r rejectRule) Apply(message *cb.Envelope) error {
	return fmt.Errorf("Rejected")
}

func TestEmptyRejectRule(t *testing.T) {
	t.Run("Reject", func(t *testing.T) {
		assert.NotNil(t, EmptyRejectRule.Apply(&cb.Envelope{}))
	})
	t.Run("Accept", func(t *testing.T) {
		assert.Nil(t, EmptyRejectRule.Apply(&cb.Envelope{Payload: []byte("fakedata")}))
	})
}

func TestAcceptRule(t *testing.T) {
	assert.Nil(t, AcceptRule.Apply(&cb.Envelope{}))
}

func TestRuleSet(t *testing.T) {
	t.Run("RejectAccept", func(t *testing.T) {
		assert.NotNil(t, NewRuleSet([]Rule{RejectRule, AcceptRule}).Apply(&cb.Envelope{}))
	})
	t.Run("AcceptReject", func(t *testing.T) {
		assert.NotNil(t, NewRuleSet([]Rule{AcceptRule, RejectRule}).Apply(&cb.Envelope{}))
	})
	t.Run("AcceptAccept", func(t *testing.T) {
		assert.Nil(t, NewRuleSet([]Rule{AcceptRule, AcceptRule}).Apply(&cb.Envelope{}))
	})
	t.Run("Empty", func(t *testing.T) {
		assert.Nil(t, NewRuleSet(nil).Apply(&cb.Envelope{}))
	})
}
