
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


package lscc

import (
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

type supportImpl struct {
}

//PutChaincodeToLocalStorage存储提供的链代码
//包到本地存储（即文件系统）
func (s *supportImpl) PutChaincodeToLocalStorage(ccpack ccprovider.CCPackage) error {
	if err := ccpack.PutChaincodeToFS(); err != nil {
		return errors.Errorf("error installing chaincode code %s:%s(%s)", ccpack.GetChaincodeData().CCName(), ccpack.GetChaincodeData().CCVersion(), err)
	}

	return nil
}

//getchaincoderomlocalstorage检索链码包
//对于请求的链码，由名称和版本指定
func (s *supportImpl) GetChaincodeFromLocalStorage(ccname string, ccversion string) (ccprovider.CCPackage, error) {
	return ccprovider.GetChaincodeFromFS(ccname, ccversion)
}

//getchaincodesfromlocalstorage返回所有链码的数组
//以前保存到本地存储的数据
func (s *supportImpl) GetChaincodesFromLocalStorage() (*pb.ChaincodeQueryResponse, error) {
	return ccprovider.GetInstalledChaincodes()
}

//GetInstantiationPolicy返回
//提供的链码（如果未指定，则为通道的默认值）
func (s *supportImpl) GetInstantiationPolicy(channel string, ccpack ccprovider.CCPackage) ([]byte, error) {
	var ip []byte
	var err error
//如果ccpack是signedcddspackage，则返回其IP，否则使用默认IP
	sccpack, isSccpack := ccpack.(*ccprovider.SignedCDSPackage)
	if isSccpack {
		ip = sccpack.GetInstantiationPolicy()
		if ip == nil {
			return nil, errors.Errorf("instantiation policy cannot be nil for a SignedCCDeploymentSpec")
		}
	} else {
//默认的实例化策略允许任何通道MSP管理员
//to be able to instantiate
		mspids := peer.GetMSPIDs(channel)

		p := cauthdsl.SignedByAnyAdmin(mspids)
		ip, err = utils.Marshal(p)
		if err != nil {
			return nil, errors.Errorf("error marshalling default instantiation policy")
		}

	}
	return ip, nil
}

//checkInstantiationPolicy检查提供的签名建议是否
//符合提供的实例化策略
func (s *supportImpl) CheckInstantiationPolicy(signedProp *pb.SignedProposal, chainName string, instantiationPolicy []byte) error {
//从策略字节创建策略对象
	mgr := mgmt.GetManagerForChain(chainName)
	if mgr == nil {
		return errors.Errorf("error checking chaincode instantiation policy: MSP manager for channel %s not found", chainName)
	}
	npp := cauthdsl.NewPolicyProvider(mgr)
	instPol, _, err := npp.NewPolicy(instantiationPolicy)
	if err != nil {
		return err
	}
	proposal, err := utils.GetProposal(signedProp.ProposalBytes)
	if err != nil {
		return err
	}
//获取提案的签名标题
	header, err := utils.GetHeader(proposal.Header)
	if err != nil {
		return err
	}
	shdr, err := utils.GetSignatureHeader(header.SignatureHeader)
	if err != nil {
		return err
	}
//构造签名数据，我们可以根据
	sd := []*common.SignedData{{
		Data:      signedProp.ProposalBytes,
		Identity:  shdr.Creator,
		Signature: signedProp.Signature,
	}}
	err = instPol.Evaluate(sd)
	if err != nil {
		return errors.WithMessage(err, "instantiation policy violation")
	}
	return nil
}
