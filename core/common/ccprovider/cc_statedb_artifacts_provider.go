
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


package ccprovider

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/platforms"
)

const (
	ccPackageStatedbDir = "META-INF/statedb/"
)

//tar file entry封装一个文件条目，它的内容在tar中
type TarFileEntry struct {
	FileHeader  *tar.Header
	FileContent []byte
}

//extractstatedbartifactsastarbytes从代码包tar中提取statedb工件并创建statedb工件tar。
//对于couchdb，状态db工件应该包含特定于状态db的工件，例如索引规范。
//此函数旨在在链代码实例化/升级期间使用，以便可以创建StateDB工件。
func ExtractStatedbArtifactsForChaincode(ccname, ccversion string, pr *platforms.Registry) (installed bool, statedbArtifactsTar []byte, err error) {
	ccpackage, err := GetChaincodeFromFS(ccname, ccversion)
	if err != nil {
//现在，我们假设有一个错误表明对等机上没有安装链码。
//但是，我们需要一种方法来区分“未安装”和常规错误，以便在常规错误时，
//我们可以中止链式代码实例化/升级/安装操作。
		ccproviderLogger.Infof("Error while loading installation package for ccname=%s, ccversion=%s. Err=%s", ccname, ccversion, err)
		return false, nil, nil
	}

	statedbArtifactsTar, err = ExtractStatedbArtifactsFromCCPackage(ccpackage, pr)
	return true, statedbArtifactsTar, err
}

//extractstatedbartifactsfromccpackage从代码包tar中提取statedb工件并创建statedb工件tar。
//对于couchdb，状态db工件应该包含特定于状态db的工件，例如索引规范。
//这个函数在chaincode实例化/升级（从上面）和安装过程中调用，这样就可以创建statedb工件。
func ExtractStatedbArtifactsFromCCPackage(ccpackage CCPackage, pr *platforms.Registry) (statedbArtifactsTar []byte, err error) {
	cds := ccpackage.GetDepSpec()
	metaprov, err := pr.GetMetadataProvider(cds.CCType(), cds.Bytes())
	if err != nil {
		ccproviderLogger.Infof("invalid deployment spec: %s", err)
		return nil, fmt.Errorf("invalid deployment spec")
	}
	return metaprov.GetMetadataAsTarEntries()
}

//extractFileEntries从给定的“tarbytes”中提取文件条目。文件条目包含在
//仅当它位于指示的databasetype目录下的目录中时返回结果
//链码索引示例：
//“META-INF/statedb/couchdb/indexes/indexcolorsortname.json”
//集合范围索引示例：
//“META-INF/statedb/couchdb/collections/collectionmarbles/indexes/indexcollmarbles.json”
//空字符串将具有返回所有statedb元数据的效果。这在验证
//将来使用多种数据库类型进行存档
func ExtractFileEntries(tarBytes []byte, databaseType string) (map[string][]*TarFileEntry, error) {

	indexArtifacts := map[string][]*TarFileEntry{}
	tarReader := tar.NewReader(bytes.NewReader(tarBytes))
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
//tar存档结束
			break
		}
		if err != nil {
			return nil, err
		}
//从全名中拆分目录
		dir, _ := filepath.Split(hdr.Name)
//删除结束斜线
		if strings.HasPrefix(hdr.Name, "META-INF/statedb/"+databaseType) {
			fileContent, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, err
			}
			indexArtifacts[filepath.Clean(dir)] = append(indexArtifacts[filepath.Clean(dir)], &TarFileEntry{FileHeader: hdr, FileContent: fileContent})
		}
	}

	return indexArtifacts, nil
}
