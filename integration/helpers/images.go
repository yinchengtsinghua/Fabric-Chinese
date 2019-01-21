
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package helpers

import (
	"encoding/base32"
	"fmt"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/common/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func AssertImagesExist(imageNames ...string) {
	dockerClient, err := docker.NewClientFromEnv()
	Expect(err).NotTo(HaveOccurred())

	for _, imageName := range imageNames {
		images, err := dockerClient.ListImages(docker.ListImagesOptions{
			Filter: imageName,
		})
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		if len(images) != 1 {
			Fail(fmt.Sprintf("missing required image: %s", imageName), 1)
		}
	}
}

//uniquename为容器名称生成base-32 enocded uuid。
func UniqueName() string {
	name := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(util.GenerateBytesUUID())
	return strings.ToLower(name)
}
