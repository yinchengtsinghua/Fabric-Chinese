
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*


SPDX许可证标识符：Apache-2.0
**/

package java

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/ccmetadata"
	"github.com/hyperledger/fabric/core/chaincode/platforms/util"
	cutil "github.com/hyperledger/fabric/core/container/util"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = flogging.MustGetLogger("chaincode.platform.java")

//Java Java代码平台
type Platform struct {
}

//name返回此平台的名称
func (javaPlatform *Platform) Name() string {
	return pb.ChaincodeSpec_JAVA.String()
}

//ValueATPATH验证Java链码路径
func (javaPlatform *Platform) ValidatePath(rawPath string) error {
	path, err := url.Parse(rawPath)
	if err != nil || path == nil {
		logger.Errorf("invalid chaincode path %s %v", rawPath, err)
		return fmt.Errorf("invalid path: %s", err)
	}

	return nil
}

func (javaPlatform *Platform) ValidateCodePackage(code []byte) error {
	if len(code) == 0 {
//如果未包含代码包，则不验证任何内容
		return nil
	}

//要有效的文件应与第一个regexp匹配，而与第二个regexp不匹配。
	filesToMatch := regexp.MustCompile(`^(/)?src/((src|META-INF)/.*|(build\.gradle|settings\.gradle|pom\.xml))`)
	filesToIgnore := regexp.MustCompile(`.*\.class$`)
	is := bytes.NewReader(code)
	gr, err := gzip.NewReader(is)
	if err != nil {
		return fmt.Errorf("failure opening codepackage gzip stream: %s", err)
	}
	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
//只有当没有更多的条目需要扫描时，我们才能到达这里。
				break
			} else {
				return err
			}
		}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//检查符合路径的名称
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
		if !filesToMatch.MatchString(header.Name) || filesToIgnore.MatchString(header.Name) {
			return fmt.Errorf("illegal file detected in payload: \"%s\"", header.Name)
		}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//检查文件模式是否有意义
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//可接受标志：
//isreg==0100000
//-rw rw rw-=0666
//
//在此上下文中，任何其他内容都是可疑的，将被拒绝
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
		if header.Mode&^0100666 != 0 {
			return fmt.Errorf("illegal file mode detected for file %s: %o", header.Name, header.Mode)
		}
	}
	return nil
}

//WryEpkAcAd编写Java链代码包
func (javaPlatform *Platform) GetDeploymentPayload(path string) ([]byte, error) {

	logger.Debugf("Packaging java project from path %s", path)
	var err error

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//写下我们的焦油包裹
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	payload := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(payload)
	tw := tar.NewWriter(gw)

	folder := path
	if folder == "" {
		logger.Error("ChaincodeSpec's path cannot be empty")
		return nil, errors.New("ChaincodeSpec's path cannot be empty")
	}

//修剪尾随斜线（如果存在）
	if folder[len(folder)-1] == '/' {
		folder = folder[:len(folder)-1]
	}

	if err = cutil.WriteJavaProjectToPackage(tw, folder); err != nil {

		logger.Errorf("Error writing java project to tar package %s", err)
		return nil, fmt.Errorf("Error writing Chaincode package contents: %s", err)
	}

	tw.Close()
	gw.Close()

	return payload.Bytes(), nil
}

func (javaPlatform *Platform) GenerateDockerfile() (string, error) {
	var buf []string

	buf = append(buf, "FROM "+cutil.GetDockerfileFromConfig("chaincode.java.runtime"))
	buf = append(buf, "ADD binpackage.tar /root/chaincode-java/chaincode")

	dockerFileContents := strings.Join(buf, "\n")

	return dockerFileContents, nil
}

func (javaPlatform *Platform) GenerateDockerBuild(path string, code []byte, tw *tar.Writer) error {
	codepackage := bytes.NewReader(code)
	binpackage := bytes.NewBuffer(nil)
	buildOptions := util.DockerBuildOptions{
		Image:        cutil.GetDockerfileFromConfig("chaincode.java.runtime"),
		Cmd:          "./build.sh",
		InputStream:  codepackage,
		OutputStream: binpackage,
	}
	logger.Debugf("Executing docker build %v, %v", buildOptions.Image, buildOptions.Cmd)
	err := util.DockerBuild(buildOptions)
	if err != nil {
		logger.Errorf("Can't build java chaincode %v", err)
		return err
	}

	resultBytes := binpackage.Bytes()
	return cutil.WriteBytesToPackage("binpackage.tar", resultBytes, tw)
}

//GetMetadataProvider获取给定部署规范的元数据提供程序
func (javaPlatform *Platform) GetMetadataProvider(code []byte) platforms.MetadataProvider {
	return &ccmetadata.TargzMetadataProvider{Code: code}
}
