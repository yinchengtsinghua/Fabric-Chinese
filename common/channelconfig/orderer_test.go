
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


package channelconfig

import (
	"testing"

	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/stretchr/testify/assert"
)

func TestBatchSize(t *testing.T) {
	validMaxMessageCount := uint32(10)
	validAbsoluteMaxBytes := uint32(1000)
	validPreferredMaxBytes := uint32(500)

	oc := &OrdererConfig{protos: &OrdererProtos{BatchSize: &ab.BatchSize{MaxMessageCount: validMaxMessageCount, AbsoluteMaxBytes: validAbsoluteMaxBytes, PreferredMaxBytes: validPreferredMaxBytes}}}
	assert.NoError(t, oc.validateBatchSize(), "BatchSize was valid")

	oc = &OrdererConfig{protos: &OrdererProtos{BatchSize: &ab.BatchSize{MaxMessageCount: 0, AbsoluteMaxBytes: validAbsoluteMaxBytes, PreferredMaxBytes: validPreferredMaxBytes}}}
	assert.Error(t, oc.validateBatchSize(), "MaxMessageCount was zero")

	oc = &OrdererConfig{protos: &OrdererProtos{BatchSize: &ab.BatchSize{MaxMessageCount: validMaxMessageCount, AbsoluteMaxBytes: 0, PreferredMaxBytes: validPreferredMaxBytes}}}
	assert.Error(t, oc.validateBatchSize(), "AbsoluteMaxBytes was zero")

	oc = &OrdererConfig{protos: &OrdererProtos{BatchSize: &ab.BatchSize{MaxMessageCount: validMaxMessageCount, AbsoluteMaxBytes: validAbsoluteMaxBytes, PreferredMaxBytes: validAbsoluteMaxBytes + 1}}}
	assert.Error(t, oc.validateBatchSize(), "PreferredMaxBytes larger to AbsoluteMaxBytes")
}

func TestBatchTimeout(t *testing.T) {
	oc := &OrdererConfig{protos: &OrdererProtos{BatchTimeout: &ab.BatchTimeout{Timeout: "1s"}}}
	assert.NoError(t, oc.validateBatchTimeout(), "Valid batch timeout")

	oc = &OrdererConfig{protos: &OrdererProtos{BatchTimeout: &ab.BatchTimeout{Timeout: "-1s"}}}
	assert.Error(t, oc.validateBatchTimeout(), "Negative batch timeout")

	oc = &OrdererConfig{protos: &OrdererProtos{BatchTimeout: &ab.BatchTimeout{Timeout: "0s"}}}
	assert.Error(t, oc.validateBatchTimeout(), "Zero batch timeout")
}

func TestKafkaBrokers(t *testing.T) {
	oc := &OrdererConfig{protos: &OrdererProtos{KafkaBrokers: &ab.KafkaBrokers{Brokers: []string{"127.0.0.1:9092", "foo.bar:9092"}}}}
	assert.NoError(t, oc.validateKafkaBrokers(), "Valid kafka brokers")

	oc = &OrdererConfig{protos: &OrdererProtos{KafkaBrokers: &ab.KafkaBrokers{Brokers: []string{"127.0.0.1", "foo.bar", "127.0.0.1:-1", "localhost:65536", "foo.bar.:9092", ".127.0.0.1:9092", "-foo.bar:9092"}}}}
	assert.Error(t, oc.validateKafkaBrokers(), "Invalid kafka brokers")
}
