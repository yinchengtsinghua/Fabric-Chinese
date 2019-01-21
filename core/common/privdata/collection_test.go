
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


package privdata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCollectionKVSKey(t *testing.T) {

	chaincodeCollectionKey := BuildCollectionKVSKey("chaincodeKey")
	assert.Equal(t, "chaincodeKey~collection", chaincodeCollectionKey, "collection keys should end in ~collection")
}

func TestIsCollectionConfigKey(t *testing.T) {

	isCollection := IsCollectionConfigKey("chaincodeKey")
	assert.False(t, isCollection, "key without tilda is not a collection key and should have returned false")

	isCollection = IsCollectionConfigKey("chaincodeKey~collection")
	assert.True(t, isCollection, "key with tilda is a collection key and should have returned true")
}
