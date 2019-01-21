
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


package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLedger(t *testing.T) {
	var info *BlockchainInfo
	info = nil
	assert.Equal(t, uint64(0), info.GetHeight())
	assert.Nil(t, info.GetCurrentBlockHash())
	assert.Nil(t, info.GetPreviousBlockHash())
	info = &BlockchainInfo{
		Height:            uint64(1),
		CurrentBlockHash:  []byte("blockhash"),
		PreviousBlockHash: []byte("previoushash"),
	}
	assert.Equal(t, uint64(1), info.GetHeight())
	assert.NotNil(t, info.GetCurrentBlockHash())
	assert.NotNil(t, info.GetPreviousBlockHash())
	info.Reset()
	assert.Equal(t, uint64(0), info.GetHeight())
	_ = info.String()
	_, _ = info.Descriptor()
	info.ProtoMessage()
}
