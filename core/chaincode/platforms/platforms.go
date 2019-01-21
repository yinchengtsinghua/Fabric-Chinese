
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


package platforms

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metadata"
	cutil "github.com/hyperledger/fabric/core/container/util"
)

//MetadataProvider由每个平台以特定于平台的方式实现。
//它可以处理以不同格式存储在chaincodedeploymentspec中的元数据。
//通用格式是targz。当前用户希望显示元数据
//作为tar文件条目（直接从以targz格式存储的链码中提取）。
//将来，我们希望通过扩展接口来提供更好的抽象
type MetadataProvider interface {
	GetMetadataAsTarEntries() ([]byte, error)
}

//用于验证规范和编写的包的接口
//给定平台
type Platform interface {
	Name() string
	ValidatePath(path string) error
	ValidateCodePackage(code []byte) error
	GetDeploymentPayload(path string) ([]byte, error)
	GenerateDockerfile() (string, error)
	GenerateDockerBuild(path string, code []byte, tw *tar.Writer) error
	GetMetadataProvider(code []byte) MetadataProvider
}

type PackageWriter interface {
	Write(name string, payload []byte, tw *tar.Writer) error
}

type PackageWriterWrapper func(name string, payload []byte, tw *tar.Writer) error

func (pw PackageWriterWrapper) Write(name string, payload []byte, tw *tar.Writer) error {
	return pw(name, payload, tw)
}

type Registry struct {
	Platforms     map[string]Platform
	PackageWriter PackageWriter
}

var logger = flogging.MustGetLogger("chaincode.platform")

func NewRegistry(platformTypes ...Platform) *Registry {
	platforms := make(map[string]Platform)
	for _, platform := range platformTypes {
		if _, ok := platforms[platform.Name()]; ok {
			logger.Panicf("Multiple platforms of the same name specified: %s", platform.Name())
		}
		platforms[platform.Name()] = platform
	}
	return &Registry{
		Platforms:     platforms,
		PackageWriter: PackageWriterWrapper(cutil.WriteBytesToPackage),
	}
}

func (r *Registry) ValidateSpec(ccType, path string) error {
	platform, ok := r.Platforms[ccType]
	if !ok {
		return fmt.Errorf("Unknown chaincodeType: %s", ccType)
	}
	return platform.ValidatePath(path)
}

func (r *Registry) ValidateDeploymentSpec(ccType string, codePackage []byte) error {
	platform, ok := r.Platforms[ccType]
	if !ok {
		return fmt.Errorf("Unknown chaincodeType: %s", ccType)
	}
	return platform.ValidateCodePackage(codePackage)
}

func (r *Registry) GetMetadataProvider(ccType string, codePackage []byte) (MetadataProvider, error) {
	platform, ok := r.Platforms[ccType]
	if !ok {
		return nil, fmt.Errorf("Unknown chaincodeType: %s", ccType)
	}
	return platform.GetMetadataProvider(codePackage), nil
}

func (r *Registry) GetDeploymentPayload(ccType, path string) ([]byte, error) {
	platform, ok := r.Platforms[ccType]
	if !ok {
		return nil, fmt.Errorf("Unknown chaincodeType: %s", ccType)
	}

	return platform.GetDeploymentPayload(path)
}

func (r *Registry) GenerateDockerfile(ccType, name, version string) (string, error) {
	platform, ok := r.Platforms[ccType]
	if !ok {
		return "", fmt.Errorf("Unknown chaincodeType: %s", ccType)
	}

	var buf []string

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	base, err := platform.GenerateDockerfile()
	if err != nil {
		return "", fmt.Errorf("Failed to generate platform-specific Dockerfile: %s", err)
	}
	buf = append(buf, base)

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//添加一些方便的标签
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	buf = append(buf, fmt.Sprintf(`LABEL %s.chaincode.id.name="%s" \`, metadata.BaseDockerLabel, name))
	buf = append(buf, fmt.Sprintf(`      %s.chaincode.id.version="%s" \`, metadata.BaseDockerLabel, version))
	buf = append(buf, fmt.Sprintf(`      %s.chaincode.type="%s" \`, metadata.BaseDockerLabel, ccType))
	buf = append(buf, fmt.Sprintf(`      %s.version="%s" \`, metadata.BaseDockerLabel, metadata.Version))
	buf = append(buf, fmt.Sprintf(`      %s.base.version="%s"`, metadata.BaseDockerLabel, metadata.BaseVersion))
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//然后用任何通用选项来扩充它
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//附加版本，以便将链码生成版本与对等生成版本进行比较
	buf = append(buf, fmt.Sprintf("ENV CORE_CHAINCODE_BUILDLEVEL=%s", metadata.Version))

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//定稿
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	contents := strings.Join(buf, "\n")
	logger.Debugf("\n%s", contents)

	return contents, nil
}

func (r *Registry) StreamDockerBuild(ccType, path string, codePackage []byte, inputFiles map[string][]byte, tw *tar.Writer) error {
	var err error

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//根据规范确定平台驱动程序
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	platform, ok := r.Platforms[ccType]
	if !ok {
		return fmt.Errorf("could not find platform of type: %s", ccType)
	}

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//首先输出静态输入文件
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	for name, data := range inputFiles {
		err = r.PackageWriter.Write(name, data, tw)
		if err != nil {
			return fmt.Errorf(`Failed to inject "%s": %s`, name, err)
		}
	}

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//现在让平台有机会为构建贡献自己的上下文
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	err = platform.GenerateDockerBuild(path, codePackage, tw)
	if err != nil {
		return fmt.Errorf("Failed to generate platform-specific docker build: %s", err)
	}

	return nil
}

func (r *Registry) GenerateDockerBuild(ccType, path, name, version string, codePackage []byte) (io.Reader, error) {

	inputFiles := make(map[string][]byte)

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//生成特定于上下文的Dockerfile
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	dockerFile, err := r.GenerateDockerfile(ccType, name, version)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate a Dockerfile: %s", err)
	}

	inputFiles["Dockerfile"] = []byte(dockerFile)

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//最后，启动一个异步进程，将上述所有内容流式传输到Docker构建上下文中。
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	input, output := io.Pipe()

	go func() {
		gw := gzip.NewWriter(output)
		tw := tar.NewWriter(gw)
		err := r.StreamDockerBuild(ccType, path, codePackage, inputFiles, tw)
		if err != nil {
			logger.Error(err)
		}

		tw.Close()
		gw.Close()
		output.CloseWithError(err)
	}()

	return input, nil
}
