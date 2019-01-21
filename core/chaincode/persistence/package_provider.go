
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
	"io/ioutil"

	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/pkg/errors"
)

//StorePackageProvider是检索
//来自chaincodeinstallpackage的代码包
type StorePackageProvider interface {
	GetChaincodeInstallPath() string
	ListInstalledChaincodes() ([]chaincode.InstalledChaincode, error)
	Load(hash []byte) (codePackage []byte, name, version string, err error)
	RetrieveHash(name, version string) (hash []byte, err error)
}

//
//来自chaincodedeploymentspec的代码包
type LegacyPackageProvider interface {
	GetChaincodeCodePackage(name, version string) (codePackage []byte, err error)
	ListInstalledChaincodes(dir string, de ccprovider.DirEnumerator, ce ccprovider.ChaincodeExtractor) ([]chaincode.InstalledChaincode, error)
}

//
type PackageParser interface {
	Parse(data []byte) (*ChaincodePackage, error)
}

//PackageProvider保留获取代码所需的依赖项
//链码的包字节
type PackageProvider struct {
	Store    StorePackageProvider
	Parser   PackageParser
	LegacyPP LegacyPackageProvider
}

//
//名称和版本。它首先搜索
//chaincodeinstallpackages，然后返回搜索
//链码部署规范
func (p *PackageProvider) GetChaincodeCodePackage(name, version string) ([]byte, error) {
	codePackage, err := p.getCodePackageFromStore(name, version)
	if err == nil {
		return codePackage, nil
	}
	if _, ok := err.(*CodePackageNotFoundErr); !ok {
//如果无法检索哈希或代码包，则返回错误
//无法从持久性存储中加载
		return nil, err
	}

	codePackage, err = p.getCodePackageFromLegacyPP(name, version)
	if err != nil {
		logger.Debug(err.Error())
		err = errors.Errorf("code package not found for chaincode with name '%s', version '%s'", name, version)
		return nil, err
	}
	return codePackage, nil
}

//getcodepackagefromstore从包中获取代码包字节
//提供程序的存储区，它保持chaincodeinstallpackages
func (p *PackageProvider) getCodePackageFromStore(name, version string) ([]byte, error) {
	hash, err := p.Store.RetrieveHash(name, version)
	if _, ok := err.(*CodePackageNotFoundErr); ok {
		return nil, err
	}
	if err != nil {
		return nil, errors.WithMessage(err, "error retrieving hash")
	}

	fsBytes, _, _, err := p.Store.Load(hash)
	if err != nil {
		return nil, errors.WithMessage(err, "error loading code package from ChaincodeInstallPackage")
	}

	ccPackage, err := p.Parser.Parse(fsBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "error parsing chaincode package")
	}

	return ccPackage.CodePackage, nil
}

//getcodepackagefromlegacypp从
//旧的包提供程序，它将持续使用chaincodedeploymentspecs
func (p *PackageProvider) getCodePackageFromLegacyPP(name, version string) ([]byte, error) {
	codePackage, err := p.LegacyPP.GetChaincodeCodePackage(name, version)
	if err != nil {
		return nil, errors.Wrap(err, "error loading code package from ChaincodeDeploymentSpec")
	}
	return codePackage, nil
}

//
//在对等机上安装的每个链码
func (p *PackageProvider) ListInstalledChaincodes() ([]chaincode.InstalledChaincode, error) {
//首先查看chaincodeinstallpackages
	installedChaincodes, err := p.Store.ListInstalledChaincodes()

	if err != nil {
//记录错误并继续
		logger.Debugf("error getting installed chaincodes from persistence store: %s", err)
	}

//然后查看CD/SCD
	installedChaincodesLegacy, err := p.LegacyPP.ListInstalledChaincodes(p.Store.GetChaincodeInstallPath(), ioutil.ReadDir, ccprovider.LoadPackage)

	if err != nil {
//记录错误并继续
		logger.Debugf("error getting installed chaincodes from ccprovider: %s", err)
	}

	for _, cc := range installedChaincodesLegacy {
		installedChaincodes = append(installedChaincodes, cc)
	}

	return installedChaincodes, nil
}
