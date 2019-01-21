
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


package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestConfig_dirExists(t *testing.T) {
	tmpF := os.TempDir()
	exists := dirExists(tmpF)
	assert.True(t, exists,
		"%s directory exists but dirExists returned false", tmpF)

	tmpF = "/blah-" + time.Now().Format(time.RFC3339Nano)
	exists = dirExists(tmpF)
	assert.False(t, exists,
		"%s directory does not exist but dirExists returned true",
		tmpF)
}

func TestConfig_InitViper(t *testing.T) {
//案例1：使用viper实例调用initviper
	v := viper.New()
	err := InitViper(v, "")
	assert.NoError(t, err, "Error returned by InitViper")

//案例2：调用initviper的默认viper实例
	err = InitViper(nil, "")
	assert.NoError(t, err, "Error returned by InitViper")
}

func TestConfig_GetPath(t *testing.T) {
//案例1：毒蛇财产不存在
	path := GetPath("foo")
	assert.Equal(t, "", path, "GetPath should have returned empty string for path 'foo'")

//Case 2: viper property that has absolute path
	viper.Set("testpath", "/test/config.yml")
	path = GetPath("testpath")
	assert.Equal(t, "/test/config.yml", path)
}

func TestConfig_TranslatePathInPlace(t *testing.T) {
//案例1：相对路径
	p := "foo"
	TranslatePathInPlace(OfficialPath, &p)
	assert.NotEqual(t, "foo", p, "TranslatePathInPlace failed to translate path %s", p)

//案例2：绝对路径
	p = "/foo"
	TranslatePathInPlace(OfficialPath, &p)
	assert.Equal(t, "/foo", p, "TranslatePathInPlace failed to translate path %s", p)
}
