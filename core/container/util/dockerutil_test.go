
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有2016年伦敦证券交易所版权所有。

SPDX许可证标识符：Apache-2.0
**/


package util

import (
	"runtime"
	"testing"

	"github.com/hyperledger/fabric/common/metadata"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestUtil_DockerfileTemplateParser(t *testing.T) {
	expected := "FROM foo:" + runtime.GOARCH + "-" + metadata.Version
	actual := ParseDockerfileTemplate("FROM foo:$(ARCH)-$(PROJECT_VERSION)")
	assert.Equal(t, expected, actual, "Error parsing Dockerfile Template. Expected \"%s\", got \"%s\"",
		expected, actual)
}

func TestUtil_GetDockerfileFromConfig(t *testing.T) {
	expected := "FROM " + metadata.DockerNamespace + ":" + runtime.GOARCH + "-" + metadata.Version
	path := "dt"
	viper.Set(path, "FROM $(DOCKER_NS):$(ARCH)-$(PROJECT_VERSION)")
	actual := GetDockerfileFromConfig(path)
	assert.Equal(t, expected, actual, "Error parsing Dockerfile Template. Expected \"%s\", got \"%s\"",
		expected, actual)
}

func TestUtil_GetDockertClient(t *testing.T) {
viper.Set("vm.endpoint", "unix:///var/run/docker.sock“）
	_, err := NewDockerClient()
	assert.NoError(t, err, "Error getting docker client")
}
