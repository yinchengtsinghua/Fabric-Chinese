
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+构建PKCS11

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
	"github.com/hyperledger/fabric/bccsp/pkcs11"
	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/pkg/errors"
)

const (
//pkcs11basedfactoryname是基于hsm的BCCSP实现的工厂名称。
	PKCS11BasedFactoryName = "PKCS11"
)

//PKCS11工厂是基于HSM的BCCSP的工厂。
type PKCS11Factory struct{}

//name返回此工厂的名称
func (f *PKCS11Factory) Name() string {
	return PKCS11BasedFactoryName
}

//get返回使用opts的bccsp实例。
func (f *PKCS11Factory) Get(config *FactoryOpts) (bccsp.BCCSP, error) {
//验证参数
	if config == nil || config.Pkcs11Opts == nil {
		return nil, errors.New("Invalid config. It must not be nil.")
	}

	p11Opts := config.Pkcs11Opts

//TODO:pkcs11不需要密钥库，但我们尚未将所有pkcs11 bccsp迁移到pkcs11。
	var ks bccsp.KeyStore
	if p11Opts.Ephemeral == true {
		ks = sw.NewDummyKeyStore()
	} else if p11Opts.FileKeystore != nil {
		fks, err := sw.NewFileBasedKeyStore(nil, p11Opts.FileKeystore.KeyStorePath, false)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to initialize software key store")
		}
		ks = fks
	} else {
//默认为dummykeystore
		ks = sw.NewDummyKeyStore()
	}
	return pkcs11.New(*p11Opts, ks)
}
