
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


package persistence

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

//chaincode包只是一个.tar.gz文件。目前，我们
//假设包包含chaincode-package-metadata.json文件
//其中包含“type”和可选的“path”。将来，它会
//如果我们搬到一个更buildpack类型的系统，而不是下面的系统，那就好了
//提供了JAR+清单类型系统，但为了方便和增量更改，
//在用户可检查工件的原型格式上移动到tar格式
//好像是个好步骤。

const (
	ChaincodePackageMetadataFile = "Chaincode-Package-Metadata.json"
)

//chaincode package表示chaincode包的非tar ed格式。
type ChaincodePackage struct {
	Metadata    *ChaincodePackageMetadata
	CodePackage []byte
}

//chaincodepackagemetata包含理解所需的信息
//嵌入的代码包。
type ChaincodePackageMetadata struct {
	Type string `json:"Type"`
	Path string `json:"Path"`
}

//
type ChaincodePackageParser struct{}

//解析将一组字节解析为链码包
//并将解析后的包作为结构返回
func (ccpp ChaincodePackageParser) Parse(source []byte) (*ChaincodePackage, error) {
	gzReader, err := gzip.NewReader(bytes.NewBuffer(source))
	if err != nil {
		return nil, errors.Wrapf(err, "error reading as gzip stream")
	}

	tarReader := tar.NewReader(gzReader)

	var codePackage []byte
	var ccPackageMetadata *ChaincodePackageMetadata
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, errors.Wrapf(err, "error inspecting next tar header")
		}

		if header.Typeflag != tar.TypeReg {
			return nil, errors.Errorf("tar entry %s is not a regular file, type %v", header.Name, header.Typeflag)
		}

		fileBytes, err := ioutil.ReadAll(tarReader)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read %s from tar", header.Name)
		}

		if header.Name == ChaincodePackageMetadataFile {
			ccPackageMetadata = &ChaincodePackageMetadata{}
			err := json.Unmarshal(fileBytes, ccPackageMetadata)
			if err != nil {
				return nil, errors.Wrapf(err, "could not unmarshal %s as json", ChaincodePackageMetadataFile)
			}

			continue
		}

		if codePackage != nil {
			return nil, errors.Errorf("found too many files in archive, cannot identify which file is the code-package")
		}

		codePackage = fileBytes
	}

	if codePackage == nil {
		return nil, errors.Errorf("did not find a code package inside the package")
	}

	if ccPackageMetadata == nil {
		return nil, errors.Errorf("did not find any package metadata (missing %s)", ChaincodePackageMetadataFile)
	}

	return &ChaincodePackage{
		Metadata:    ccPackageMetadata,
		CodePackage: codePackage,
	}, nil
}
