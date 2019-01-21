
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


package statecouchdb

import (
	"sync"
)

//批处理在单独的goroutine中执行。
type batch interface {
	execute() error
}

//executeBatches在单独的goroutine中执行每个批，如果
//任何批在执行期间返回错误
func executeBatches(batches []batch) error {
	logger.Debugf("Executing batches = %s", batches)
	numBatches := len(batches)
	if numBatches == 0 {
		return nil
	}
	if numBatches == 1 {
		return batches[0].execute()
	}
	var batchWG sync.WaitGroup
	batchWG.Add(numBatches)
	errsChan := make(chan error, numBatches)
	defer close(errsChan)
	for _, b := range batches {
		go func(b batch) {
			defer batchWG.Done()
			if err := b.execute(); err != nil {
				errsChan <- err
				return
			}
		}(b)
	}
	batchWG.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}
	return nil
}
