
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司。保留所有权利。
γ
SPDX许可证标识符：Apache-2.0
**/


package node

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/ccmetadata"
	"github.com/hyperledger/fabric/core/chaincode/platforms/util"
	cutil "github.com/hyperledger/fabric/core/container/util"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = flogging.MustGetLogger("chaincode.platform.node")

//Go中编写的链码平台
type Platform struct {
}

//返回给定的文件或目录是否存在
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

//name返回此平台的名称
func (nodePlatform *Platform) Name() string {
	return pb.ChaincodeSpec_NODE.String()
}

//validatespec验证go chaincodes
func (nodePlatform *Platform) ValidatePath(rawPath string) error {
	path, err := url.Parse(rawPath)
	if err != nil || path == nil {
		return fmt.Errorf("invalid path: %s", err)
	}

//将空方案视为本地文件系统路径
	if path.Scheme == "" {
		pathToCheck, err := filepath.Abs(rawPath)
		if err != nil {
			return fmt.Errorf("error obtaining absolute path of the chaincode: %s", err)
		}

		exists, err := pathExists(pathToCheck)
		if err != nil {
			return fmt.Errorf("error validating chaincode path: %s", err)
		}
		if !exists {
			return fmt.Errorf("path to chaincode does not exist: %s", rawPath)
		}
	}
	return nil
}

func (nodePlatform *Platform) ValidateCodePackage(code []byte) error {

	if len(code) == 0 {
//如果未包含代码包，则不验证任何内容
		return nil
	}

//FAB-2122：扫描提供的tarball，确保它只包含以下源代码
//SRC文件夹。
//
//应该注意的是，我们不能用这些技术抓住每一个威胁。因此，
//容器本身需要是最后一道防线，并配置为
//在执行约束时有弹性。但是，我们还是应该尽最大努力保持
//垃圾尽可能从系统中排出。
	re := regexp.MustCompile(`^(/)?(src|META-INF)/.*`)
	is := bytes.NewReader(code)
	gr, err := gzip.NewReader(is)
	if err != nil {
		return fmt.Errorf("failure opening codepackage gzip stream: %s", err)
	}
	tr := tar.NewReader(gr)

	var foundPackageJson = false
	for {
		header, err := tr.Next()
		if err != nil {
//只有当没有更多的条目需要扫描时，我们才能到达这里。
			break
		}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//检查符合路径的名称
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
		if !re.MatchString(header.Name) {
			return fmt.Errorf("illegal file detected in payload: \"%s\"", header.Name)
		}
		if header.Name == "src/package.json" {
			foundPackageJson = true
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
	if !foundPackageJson {
		return fmt.Errorf("no package.json found at the root of the chaincode package")
	}

	return nil
}

//通过将源文件放在.tar.gz格式的src/$file条目中生成部署负载
func (nodePlatform *Platform) GetDeploymentPayload(path string) ([]byte, error) {

	var err error

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//写下我们的焦油包裹
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	payload := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(payload)
	tw := tar.NewWriter(gw)

	folder := path
	if folder == "" {
		return nil, errors.New("ChaincodeSpec's path cannot be empty")
	}

//修剪尾随斜线（如果存在）
	if folder[len(folder)-1] == '/' {
		folder = folder[:len(folder)-1]
	}

	logger.Debugf("Packaging node.js project from path %s", folder)

	if err = cutil.WriteFolderToTarPackage(tw, folder, []string{"node_modules"}, nil, nil); err != nil {

		logger.Errorf("Error writing folder to tar package %s", err)
		return nil, fmt.Errorf("Error writing Chaincode package contents: %s", err)
	}

//写出tar文件
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("Error writing Chaincode package contents: %s", err)
	}

	tw.Close()
	gw.Close()

	return payload.Bytes(), nil
}

func (nodePlatform *Platform) GenerateDockerfile() (string, error) {

	var buf []string

	buf = append(buf, "FROM "+cutil.GetDockerfileFromConfig("chaincode.node.runtime"))
	buf = append(buf, "ADD binpackage.tar /usr/local/src")

	dockerFileContents := strings.Join(buf, "\n")

	return dockerFileContents, nil
}

func (nodePlatform *Platform) GenerateDockerBuild(path string, code []byte, tw *tar.Writer) error {

	codepackage := bytes.NewReader(code)
	binpackage := bytes.NewBuffer(nil)
	err := util.DockerBuild(util.DockerBuildOptions{
		Cmd:          fmt.Sprint("cp -R /chaincode/input/src/. /chaincode/output && cd /chaincode/output && npm install --production"),
		InputStream:  codepackage,
		OutputStream: binpackage,
	})
	if err != nil {
		return err
	}

	return cutil.WriteBytesToPackage("binpackage.tar", binpackage.Bytes(), tw)
}

//GetMetadataProvider获取给定部署规范的元数据提供程序
func (nodePlatform *Platform) GetMetadataProvider(code []byte) platforms.MetadataProvider {
	return &ccmetadata.TargzMetadataProvider{Code: code}
}
