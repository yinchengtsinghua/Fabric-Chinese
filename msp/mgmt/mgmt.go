
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


package mgmt

import (
	"reflect"
	"sync"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/cache"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

//loadLocalMSPWithType从指定目录中加载具有指定类型的本地MSP
func LoadLocalMspWithType(dir string, bccspConfig *factory.FactoryOpts, mspID, mspType string) error {
	if mspID == "" {
		return errors.New("the local MSP must have an ID")
	}

	conf, err := msp.GetLocalMspConfigWithType(dir, bccspConfig, mspID, mspType)
	if err != nil {
		return err
	}

	return GetLocalMSP().Setup(conf)
}

//loadLocalMSP从指定目录加载本地MSP
func LoadLocalMsp(dir string, bccspConfig *factory.FactoryOpts, mspID string) error {
	if mspID == "" {
		return errors.New("the local MSP must have an ID")
	}

	conf, err := msp.GetLocalMspConfig(dir, bccspConfig, mspID)
	if err != nil {
		return err
	}

	return GetLocalMSP().Setup(conf)
}

//Fixme：一旦链管理代码完成，
//这些映射和helpser函数应该消失，因为
//每链MSP经理的所有权将由IT处理；
//但是在过渡期间，这些助手函数是必需的

var m sync.Mutex
var localMsp msp.MSP
var mspMap map[string]msp.MSPManager = make(map[string]msp.MSPManager)
var mspLogger = flogging.MustGetLogger("msp")

//TODO-这是一个临时解决方案，允许对等机跟踪
//已经为一个通道设置了mspmanager，它指示该通道是否
//存在与否
type mspMgmtMgr struct {
	msp.MSPManager
//跟踪此MSPManager是否已成功设置
	up bool
}

func (mgr *mspMgmtMgr) DeserializeIdentity(serializedIdentity []byte) (msp.Identity, error) {
	if !mgr.up {
		return nil, errors.New("channel doesn't exist")
	}
	return mgr.MSPManager.DeserializeIdentity(serializedIdentity)
}

func (mgr *mspMgmtMgr) Setup(msps []msp.MSP) error {
	err := mgr.MSPManager.Setup(msps)
	if err == nil {
		mgr.up = true
	}
	return err
}

//GetManagerForChain返回所提供的
//链；如果不存在此类管理器，则创建一个
func GetManagerForChain(chainID string) msp.MSPManager {
	m.Lock()
	defer m.Unlock()

	mspMgr, ok := mspMap[chainID]
	if !ok {
		mspLogger.Debugf("Created new msp manager for channel `%s`", chainID)
		mspMgmtMgr := &mspMgmtMgr{msp.NewMSPManager(), false}
		mspMap[chainID] = mspMgmtMgr
		mspMgr = mspMgmtMgr
	} else {
//检查内部mspmanagerimpl和mspmgmtmgr类型。如果不同
//找到类型是因为开发人员添加了一个新类型
//实现MSPManager接口，并应向逻辑添加事例
//上面处理它。
		if !(reflect.TypeOf(mspMgr).Elem().Name() == "mspManagerImpl" || reflect.TypeOf(mspMgr).Elem().Name() == "mspMgmtMgr") {
			panic("Found unexpected MSPManager type.")
		}
		mspLogger.Debugf("Returning existing manager for channel '%s'", chainID)
	}
	return mspMgr
}

//GetManagers返回所有已注册的管理器
func GetDeserializers() map[string]msp.IdentityDeserializer {
	m.Lock()
	defer m.Unlock()

	clone := make(map[string]msp.IdentityDeserializer)

	for key, mspManager := range mspMap {
		clone[key] = mspManager
	}

	return clone
}

//XXXSetMSPManager是从自定义MSP配置块转换的权宜之计解决方案
//分析到channelconfig.resources接口，同时保留有问题的单例
//MSP经理的性质
func XXXSetMSPManager(chainID string, manager msp.MSPManager) {
	m.Lock()
	defer m.Unlock()

	mspMap[chainID] = &mspMgmtMgr{manager, true}
}

//getlocalmsp返回本地msp（如果不存在则创建它）
func GetLocalMSP() msp.MSP {
	m.Lock()
	defer m.Unlock()

	if localMsp != nil {
		return localMsp
	}

	localMsp = loadLocaMSP()

	return localMsp
}

func loadLocaMSP() msp.MSP {
//确定MSP的类型（默认情况下，我们将使用bccspmsp）
	mspType := viper.GetString("peer.localMspType")
	if mspType == "" {
		mspType = msp.ProviderTypeToString(msp.FABRIC)
	}

	var mspOpts = map[string]msp.NewOpts{
		msp.ProviderTypeToString(msp.FABRIC): &msp.BCCSPNewOpts{NewBaseOpts: msp.NewBaseOpts{Version: msp.MSPv1_0}},
		msp.ProviderTypeToString(msp.IDEMIX): &msp.IdemixNewOpts{NewBaseOpts: msp.NewBaseOpts{Version: msp.MSPv1_1}},
	}
	newOpts, found := mspOpts[mspType]
	if !found {
		mspLogger.Panicf("msp type " + mspType + " unknown")
	}

	mspInst, err := msp.New(newOpts)
	if err != nil {
		mspLogger.Fatalf("Failed to initialize local MSP, received err %+v", err)
	}
	switch mspType {
	case msp.ProviderTypeToString(msp.FABRIC):
		mspInst, err = cache.New(mspInst)
		if err != nil {
			mspLogger.Fatalf("Failed to initialize local MSP, received err %+v", err)
		}
	case msp.ProviderTypeToString(msp.IDEMIX):
//什么也不做
	default:
		panic("msp type " + mspType + " unknown")
	}

	mspLogger.Debugf("Created new local MSP")

	return mspInst
}

//GetIdentityDeserializer返回给定链的IdentityDeserializer
func GetIdentityDeserializer(chainID string) msp.IdentityDeserializer {
	if chainID == "" {
		return GetLocalMSP()
	}

	return GetManagerForChain(chainID)
}

//GetLocalSigningIdentityOrpanic返回本地签名标识或紧急情况
//或错误
func GetLocalSigningIdentityOrPanic() msp.SigningIdentity {
	id, err := GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		mspLogger.Panicf("Failed getting local signing identity [%+v]", err)
	}
	return id
}
