
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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/util"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("chaincode.persistence")

//ioreadwriter定义读取、写入、删除和
//检查是否存在指定的文件
type IOReadWriter interface {
	ReadDir(string) ([]os.FileInfo, error)
	ReadFile(string) ([]byte, error)
	Remove(name string) error
	Stat(string) (os.FileInfo, error)
	WriteFile(string, []byte, os.FileMode) error
}

//filesystemio是iowriter接口的生产实现
type FilesystemIO struct {
}

//writefile将文件写入文件系统
func (f *FilesystemIO) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

//stat检查文件系统上是否存在该文件
func (f *FilesystemIO) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

//删除从文件系统中删除一个文件-用于回滚正在运行的文件
//失败时保存操作
func (f *FilesystemIO) Remove(name string) error {
	return os.Remove(name)
}

//readfile从文件系统读取文件
func (f *FilesystemIO) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

//readdir从文件系统读取目录
func (f *FilesystemIO) ReadDir(dirname string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(dirname)
}

//存储区保存保持chaincode安装包所需的信息
type Store struct {
	Path       string
	ReadWriter IOReadWriter
}

//
//版本
func (s *Store) Save(name, version string, ccInstallPkg []byte) ([]byte, error) {
	metadataJSON, err := toJSON(name, version)
	if err != nil {
		return nil, err
	}

	hash := util.ComputeSHA256(ccInstallPkg)
	hashString := hex.EncodeToString(hash)
	metadataPath := filepath.Join(s.Path, hashString+".json")
	if _, err := s.ReadWriter.Stat(metadataPath); err == nil {
		return nil, errors.Errorf("chaincode metadata already exists at %s", metadataPath)
	}

	ccInstallPkgPath := filepath.Join(s.Path, hashString+".bin")
	if _, err := s.ReadWriter.Stat(ccInstallPkgPath); err == nil {
		return nil, errors.Errorf("ChaincodeInstallPackage already exists at %s", ccInstallPkgPath)
	}

	if err := s.ReadWriter.WriteFile(metadataPath, metadataJSON, 0600); err != nil {
		return nil, errors.Wrapf(err, "error writing metadata file to %s", metadataPath)
	}

	if err := s.ReadWriter.WriteFile(ccInstallPkgPath, ccInstallPkg, 0600); err != nil {
		err = errors.Wrapf(err, "error writing chaincode install package to %s", ccInstallPkgPath)
		logger.Error(err.Error())

//出错时需要回滚上面的元数据写入
		if err2 := s.ReadWriter.Remove(metadataPath); err2 != nil {
			logger.Errorf("error removing metadata file at %s: %s", metadataPath, err2)
		}
		return nil, err
	}

	return hash, nil
}

//加载使用给定哈希加载持久化的链码安装包字节
//并返回名称和版本
func (s *Store) Load(hash []byte) (ccInstallPkg []byte, name, version string, err error) {
	hashString := hex.EncodeToString(hash)
	ccInstallPkgPath := filepath.Join(s.Path, hashString+".bin")
	ccInstallPkg, err = s.ReadWriter.ReadFile(ccInstallPkgPath)
	if err != nil {
		err = errors.Wrapf(err, "error reading chaincode install package at %s", ccInstallPkgPath)
		return nil, "", "", err
	}

	metadataPath := filepath.Join(s.Path, hashString+".json")
	name, version, err = s.LoadMetadata(metadataPath)
	if err != nil {
		return nil, "", "", err
	}

	return ccInstallPkg, name, version, nil
}

//loadMetadata加载存储在指定路径上的链码元数据
func (s *Store) LoadMetadata(path string) (name, version string, err error) {
	metadataBytes, err := s.ReadWriter.ReadFile(path)
	if err != nil {
		err = errors.Wrapf(err, "error reading metadata at %s", path)
		return "", "", err
	}
	ccMetadata := &ChaincodeMetadata{}
	err = json.Unmarshal(metadataBytes, ccMetadata)
	if err != nil {
		err = errors.Wrapf(err, "error unmarshaling metadata at %s", path)
		return "", "", err
	}

	return ccMetadata.Name, ccMetadata.Version, nil
}

//codePackageNotFounderr是当代码包无法
//在持久性存储中找到
type CodePackageNotFoundErr struct {
	Name    string
	Version string
}

func (e *CodePackageNotFoundErr) Error() string {
	return fmt.Sprintf("chaincode install package not found with name '%s', version '%s'", e.Name, e.Version)
}

//retrievehash检索给定
//链码的名称和版本
func (s *Store) RetrieveHash(name string, version string) ([]byte, error) {
	installedChaincodes, err := s.ListInstalledChaincodes()
	if err != nil {
		return nil, errors.WithMessage(err, "error getting installed chaincodes")
	}

	for _, installedChaincode := range installedChaincodes {
		if installedChaincode.Name == name && installedChaincode.Version == version {
			return installedChaincode.Id, nil
		}
	}

	err = &CodePackageNotFoundErr{
		Name:    name,
		Version: version,
	}

	return nil, err
}

//
//持久性存储中安装的链代码
func (s *Store) ListInstalledChaincodes() ([]chaincode.InstalledChaincode, error) {
	files, err := s.ReadWriter.ReadDir(s.Path)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading chaincode directory at %s", s.Path)
	}

	installedChaincodes := []chaincode.InstalledChaincode{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			metadataPath := filepath.Join(s.Path, file.Name())
			ccName, ccVersion, err := s.LoadMetadata(metadataPath)
			if err != nil {
				logger.Warning(err.Error())
				continue
			}

//拆分文件名并获取哈希值
			hashString := strings.Split(file.Name(), ".")[0]
			hash, err := hex.DecodeString(hashString)
			if err != nil {
				return nil, errors.Wrapf(err, "error decoding hash from hex string: %s", hashString)
			}
			installedChaincode := chaincode.InstalledChaincode{
				Name:    ccName,
				Version: ccVersion,
				Id:      hash,
			}
			installedChaincodes = append(installedChaincodes, installedChaincode)
		}
	}
	return installedChaincodes, nil
}

//getchaincodeinstallpath返回链码所在的路径
//安装
func (s *Store) GetChaincodeInstallPath() string {
	return s.Path
}

//chaincodemetadata保存chaincode的名称和版本
type ChaincodeMetadata struct {
	Name    string `json:"Name"`
	Version string `json:"Version"`
}

func toJSON(name, version string) ([]byte, error) {
	metadata := &ChaincodeMetadata{
		Name:    name,
		Version: version,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling name and version into JSON")
	}

	return metadataBytes, nil
}
