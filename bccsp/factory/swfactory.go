
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/

package factory

import (
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/pkg/errors"
)

const (
//SoftwareBasedFactoryName是基于软件的BCCSP实现的工厂名称。
	SoftwareBasedFactoryName = "SW"
)

//SWFactory是基于软件的BCCSP的工厂。
type SWFactory struct{}

//name返回此工厂的名称
func (f *SWFactory) Name() string {
	return SoftwareBasedFactoryName
}

//get返回使用opts的bccsp实例。
func (f *SWFactory) Get(config *FactoryOpts) (bccsp.BCCSP, error) {
//验证参数
	if config == nil || config.SwOpts == nil {
		return nil, errors.New("Invalid config. It must not be nil.")
	}

	swOpts := config.SwOpts

	var ks bccsp.KeyStore
	if swOpts.Ephemeral == true {
		ks = sw.NewDummyKeyStore()
	} else if swOpts.FileKeystore != nil {
		fks, err := sw.NewFileBasedKeyStore(nil, swOpts.FileKeystore.KeyStorePath, false)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to initialize software key store")
		}
		ks = fks
	} else if swOpts.InmemKeystore != nil {
		ks = sw.NewInMemoryKeyStore()
	} else {
//默认为临时密钥存储
		ks = sw.NewDummyKeyStore()
	}

	return sw.NewWithParams(swOpts.SecLevel, swOpts.HashFamily, ks)
}

//swopts包含swfactory的选项
type SwOpts struct {
//未指定时的默认算法（是否已弃用？）
	SecLevel   int    `mapstructure:"security" json:"security" yaml:"Security"`
	HashFamily string `mapstructure:"hash" json:"hash" yaml:"Hash"`

//密钥存储选项
	Ephemeral     bool               `mapstructure:"tempkeys,omitempty" json:"tempkeys,omitempty"`
	FileKeystore  *FileKeystoreOpts  `mapstructure:"filekeystore,omitempty" json:"filekeystore,omitempty" yaml:"FileKeyStore"`
	DummyKeystore *DummyKeystoreOpts `mapstructure:"dummykeystore,omitempty" json:"dummykeystore,omitempty"`
	InmemKeystore *InmemKeystoreOpts `mapstructure:"inmemkeystore,omitempty" json:"inmemkeystore,omitempty"`
}

//可插入的密钥库，可以添加jks、p12等。
type FileKeystoreOpts struct {
	KeyStorePath string `mapstructure:"keystore" yaml:"KeyStore"`
}

type DummyKeystoreOpts struct{}

//inmemkeystoreopts-空，因为内存中的密钥库没有配置
type InmemKeystoreOpts struct{}
