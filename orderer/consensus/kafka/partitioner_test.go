
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


package kafka

import (
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func TestStaticPartitioner(t *testing.T) {
	var partition int32 = 3
	var numberOfPartitions int32 = 6

	partitionerConstructor := newStaticPartitioner(partition)
	partitioner := partitionerConstructor(channelNameForTest(t))

	for i := 0; i < 10; i++ {
		assignedPartition, err := partitioner.Partition(new(sarama.ProducerMessage), numberOfPartitions)
		assert.NoError(t, err, "Partitioner not functioning as expected:", err)
		assert.Equal(t, partition, assignedPartition, "Partitioner not returning the expected partition - expected %d, got %v", partition, assignedPartition)
	}
}
