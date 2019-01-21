
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


package msp

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

var mspLogger = flogging.MustGetLogger("msp")

type mspManagerImpl struct {
//包含我们已设置或以其他方式添加的所有MSP的映射
	mspsMap map[string]MSP

//按提供程序类型映射MSP的映射
	mspsByProviders map[ProviderType][]MSP

//启动时可能发生的错误
	up bool
}

//new msp manager返回一个新的msp管理器实例；
//请注意，直到
//调用设置方法
func NewMSPManager() MSPManager {
	return &mspManagerImpl{}
}

//安装程序初始化此管理器的内部数据结构并创建MSP
func (mgr *mspManagerImpl) Setup(msps []MSP) error {
	if mgr.up {
		mspLogger.Infof("MSP manager already up")
		return nil
	}

	mspLogger.Debugf("Setting up the MSP manager (%d msps)", len(msps))

//创建将MSP ID分配给其管理器实例的映射-一次
	mgr.mspsMap = make(map[string]MSP)

//创建按提供程序类型对MSP排序的映射
	mgr.mspsByProviders = make(map[ProviderType][]MSP)

	for _, msp := range msps {
//将MSP添加到活动MSP的映射中
		mspID, err := msp.GetIdentifier()
		if err != nil {
			return errors.WithMessage(err, "could not extract msp identifier")
		}
		mgr.mspsMap[mspID] = msp
		providerType := msp.GetType()
		mgr.mspsByProviders[providerType] = append(mgr.mspsByProviders[providerType], msp)
	}

	mgr.up = true

	mspLogger.Debugf("MSP manager setup complete, setup %d msps", len(msps))

	return nil
}

//GetMSPS返回此管理器管理的MSP
func (mgr *mspManagerImpl) GetMSPs() (map[string]MSP, error) {
	return mgr.mspsMap, nil
}

//DeserializeIDentity返回一个标识，该标识的序列化版本作为参数提供
func (mgr *mspManagerImpl) DeserializeIdentity(serializedID []byte) (Identity, error) {
//我们首先反序列化到一个序列化的实体以获取MSP ID
	sId := &msp.SerializedIdentity{}
	err := proto.Unmarshal(serializedID, sId)
	if err != nil {
		return nil, errors.Wrap(err, "could not deserialize a SerializedIdentity")
	}

//我们现在可以尝试获得MSP
	msp := mgr.mspsMap[sId.Mspid]
	if msp == nil {
		return nil, errors.Errorf("MSP %s is unknown", sId.Mspid)
	}

	switch t := msp.(type) {
	case *bccspmsp:
		return t.deserializeIdentityInternal(sId.IdBytes)
	case *idemixmsp:
		return t.deserializeIdentityInternal(sId.IdBytes)
	default:
		return t.DeserializeIdentity(serializedID)
	}
}

func (mgr *mspManagerImpl) IsWellFormed(identity *msp.SerializedIdentity) error {
//通过其提供程序迭代所有MSP，并找到至少1个可以证明的MSP
//这个身份形成良好
	for _, mspList := range mgr.mspsByProviders {
//从设置（）的初始化开始，我们保证每个列表中至少有1个MSP。
		msp := mspList[0]
		if err := msp.IsWellFormed(identity); err == nil {
			return nil
		}
	}
	return errors.New("no MSP provider recognizes the identity")
}
