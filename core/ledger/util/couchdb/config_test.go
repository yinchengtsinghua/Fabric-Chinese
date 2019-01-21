
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


package couchdb

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetCouchDBDefinition(t *testing.T) {
	expectedAddress := viper.GetString("ledger.state.couchDBConfig.couchDBAddress")

	couchDBDef := GetCouchDBDefinition()
	assert.Equal(t, expectedAddress, couchDBDef.URL)
	assert.Equal(t, "", couchDBDef.Username)
	assert.Equal(t, "", couchDBDef.Password)
	assert.Equal(t, 3, couchDBDef.MaxRetries)
	assert.Equal(t, 20, couchDBDef.MaxRetriesOnStartup)
	assert.Equal(t, time.Second*35, couchDBDef.RequestTimeout)
}
