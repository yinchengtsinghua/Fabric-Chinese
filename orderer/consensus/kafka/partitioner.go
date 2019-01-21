
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

import "github.com/Shopify/sarama"

type staticPartitioner struct {
	partitionID int32
}

//NewStaticPartitioner返回的PartitionerConstructor
//返回始终选择指定分区的分区程序。
func newStaticPartitioner(partition int32) sarama.PartitionerConstructor {
	return func(topic string) sarama.Partitioner {
		return &staticPartitioner{partition}
	}
}

//分区接受消息和分区计数并选择一个分区。
func (prt *staticPartitioner) Partition(message *sarama.ProducerMessage, numPartitions int32) (int32, error) {
	return prt.partitionID, nil
}

//RequiresConsistency向分区用户指示
//key->partition的映射是否一致。
func (prt *staticPartitioner) RequiresConsistency() bool {
	return true
}
