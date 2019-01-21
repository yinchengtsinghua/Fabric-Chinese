
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
*/


package channelconfig

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/cache"
	mspprotos "github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

type pendingMSPConfig struct {
	mspConfig *mspprotos.MSPConfig
	msp       msp.MSP
}

//MSPC配置处理程序
type MSPConfigHandler struct {
	version msp.MSPVersion
	idMap   map[string]*pendingMSPConfig
}

func NewMSPConfigHandler(mspVersion msp.MSPVersion) *MSPConfigHandler {
	return &MSPConfigHandler{
		version: mspVersion,
		idMap:   make(map[string]*pendingMSPConfig),
	}
}

//
func (bh *MSPConfigHandler) ProposeMSP(mspConfig *mspprotos.MSPConfig) (msp.MSP, error) {
	var theMsp msp.MSP
	var err error

	switch mspConfig.Type {
	case int32(msp.FABRIC):
//创建bccsp msp实例
		mspInst, err := msp.New(&msp.BCCSPNewOpts{NewBaseOpts: msp.NewBaseOpts{Version: bh.version}})
		if err != nil {
			return nil, errors.WithMessage(err, "creating the MSP manager failed")
		}

//在顶部添加缓存层
		theMsp, err = cache.New(mspInst)
		if err != nil {
			return nil, errors.WithMessage(err, "creating the MSP cache failed")
		}
	case int32(msp.IDEMIX):
//创建IDemix MSP实例
		theMsp, err = msp.New(&msp.IdemixNewOpts{
			NewBaseOpts: msp.NewBaseOpts{Version: bh.version},
		})
		if err != nil {
			return nil, errors.WithMessage(err, "creating the MSP manager failed")
		}
	default:
		return nil, errors.New(fmt.Sprintf("Setup error: unsupported msp type %d", mspConfig.Type))
	}

//设置它
	err = theMsp.Setup(mspConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "setting up the MSP manager failed")
	}

//将MSP添加到挂起MSP的映射中
	mspID, _ := theMsp.GetIdentifier()

	existingPendingMSPConfig, ok := bh.idMap[mspID]
	if ok && !proto.Equal(existingPendingMSPConfig.mspConfig, mspConfig) {
		return nil, errors.New(fmt.Sprintf("Attempted to define two different versions of MSP: %s", mspID))
	}

	if !ok {
		bh.idMap[mspID] = &pendingMSPConfig{
			mspConfig: mspConfig,
			msp:       theMsp,
		}
	}

	return theMsp, nil
}

func (bh *MSPConfigHandler) CreateMSPManager() (msp.MSPManager, error) {
	mspList := make([]msp.MSP, len(bh.idMap))
	i := 0
	for _, pendingMSP := range bh.idMap {
		mspList[i] = pendingMSP.msp
		i++
	}

	manager := msp.NewMSPManager()
	err := manager.Setup(mspList)
	return manager, err
}
