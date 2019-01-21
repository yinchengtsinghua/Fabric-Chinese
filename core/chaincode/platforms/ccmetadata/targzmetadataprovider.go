
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有，State Street Corp.保留所有权利。
γ
SPDX许可证标识符：Apache-2.0
**/

package ccmetadata

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"strings"

	"github.com/pkg/errors"
)

//
//实施）。目前它只处理statedb元数据，但将在将来被通用化。
//允许使用链代码打包任意元数据。
const (
	ccPackageStatedbDir = "META-INF/statedb/"
)

//targzMetadataProvider提供以targz格式打包的链代码中的元数据
//（GO、Java和节点平台）
type TargzMetadataProvider struct {
	Code []byte
}

func (tgzProv *TargzMetadataProvider) getCode() ([]byte, error) {
	if tgzProv.Code == nil {
		return nil, errors.New("nil code package")
	}

	return tgzProv.Code, nil
}

//getmetadataastarentries从chaincodedeploymentspec中提取metata数据
func (tgzProv *TargzMetadataProvider) GetMetadataAsTarEntries() ([]byte, error) {
	code, err := tgzProv.getCode()
	if err != nil {
		return nil, err
	}

	is := bytes.NewReader(code)
	gr, err := gzip.NewReader(is)
	if err != nil {
		logger.Errorf("Failure opening codepackage gzip stream: %s", err)
		return nil, err
	}

	statedbTarBuffer := bytes.NewBuffer(nil)
	tw := tar.NewWriter(statedbTarBuffer)

	tr := tar.NewReader(gr)

//对于代码包tar中的每个文件，
//如果路径中有“statedb”，则将其添加到statedb artifact tar中
	for {
		header, err := tr.Next()
		if err == io.EOF {
//只有当没有更多的条目需要扫描时，我们才能到达这里。
			break
		}
		if err != nil {
			return nil, err
		}

		if !strings.HasPrefix(header.Name, ccPackageStatedbDir) {
			continue
		}

		if err = tw.WriteHeader(header); err != nil {
			logger.Error("Error adding header to statedb tar:", err, header.Name)
			return nil, err
		}
		if _, err := io.Copy(tw, tr); err != nil {
			logger.Error("Error copying file to statedb tar:", err, header.Name)
			return nil, err
		}
		logger.Debug("Wrote file to statedb tar:", header.Name)
	}

	if err = tw.Close(); err != nil {
		return nil, err
	}

	logger.Debug("Created metadata tar")

	return statedbTarBuffer.Bytes(), nil
}
