
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


package car

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/util"
	cutil "github.com/hyperledger/fabric/core/container/util"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//轿厢类型平台
type Platform struct {
}

//name返回此平台的名称
func (carPlatform *Platform) Name() string {
	return pb.ChaincodeSpec_CAR.String()
}

//validatePath验证汽车类型要满足的链码路径
//平台接口。此链码类型当前没有
//需要任何特定的内容，所以我们只是隐式地批准任何规范
func (carPlatform *Platform) ValidatePath(path string) error {
	return nil
}

func (carPlatform *Platform) ValidateCodePackage(codePackage []byte) error {
//汽车平台将验证链式工具中的代码包
	return nil
}

func (carPlatform *Platform) GetDeploymentPayload(path string) ([]byte, error) {

	return ioutil.ReadFile(path)
}

func (carPlatform *Platform) GenerateDockerfile() (string, error) {

	var buf []string

//让可执行文件的名称为chaincode id的名称
	buf = append(buf, "FROM "+cutil.GetDockerfileFromConfig("chaincode.car.runtime"))
	buf = append(buf, "ADD binpackage.tar /usr/local/bin")

	dockerFileContents := strings.Join(buf, "\n")

	return dockerFileContents, nil
}

func (carPlatform *Platform) GenerateDockerBuild(path string, code []byte, tw *tar.Writer) error {

//将.car文件捆绑到tar流中，以便将其传输到builder容器
	codepackage, output := io.Pipe()
	go func() {
		tw := tar.NewWriter(output)

		err := cutil.WriteBytesToPackage("codepackage.car", code, tw)

		tw.Close()
		output.CloseWithError(err)
	}()

	binpackage := bytes.NewBuffer(nil)
	err := util.DockerBuild(util.DockerBuildOptions{
		Cmd:          "java -jar /usr/local/bin/chaintool buildcar /chaincode/input/codepackage.car -o /chaincode/output/chaincode",
		InputStream:  codepackage,
		OutputStream: binpackage,
	})
	if err != nil {
		return fmt.Errorf("Error building CAR: %s", err)
	}

	return cutil.WriteBytesToPackage("binpackage.tar", binpackage.Bytes(), tw)
}

//GetMetadataProvider获取给定部署规范的元数据提供程序
func (carPlatform *Platform) GetMetadataProvider(code []byte) platforms.MetadataProvider {
	return &MetadataProvider{}
}
