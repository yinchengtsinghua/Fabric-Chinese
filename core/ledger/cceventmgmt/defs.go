
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


package cceventmgmt

import (
	"fmt"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
)

//chaincodedefinition捕获有关chaincode的信息
type ChaincodeDefinition struct {
	Name              string
	Hash              []byte
	Version           string
	CollectionConfigs *common.CollectionConfigPackage
}

func (cdef *ChaincodeDefinition) String() string {
	return fmt.Sprintf("Name=%s, Version=%s, Hash=%#v", cdef.Name, cdef.Version, cdef.Hash)
}

//ChaincodeLifecycleEventListener接口启用分类帐组件（主要用于StateDB）
//能够监听链码生命周期事件。dbartifactstar'表示特定于数据库的项目
//（如索引规格）用焦油包装
type ChaincodeLifecycleEventListener interface {
//当chaincode installed+defined变为true时调用handlechaincodedeploy。
//预期的用法是创建所有必要的statedb结构（如索引）并更新
//服务发现信息。在提交状态更改之前立即调用此函数
//包含链码定义或在发生链码安装时
	HandleChaincodeDeploy(chaincodeDefinition *ChaincodeDefinition, dbArtifactsTar []byte) error
//chaincodedeploydone在完成chaincode部署后调用-`succeeded`表示
//部署是否成功完成
	ChaincodeDeployDone(succeeded bool)
}

//ChaincodeInfoProvider接口使事件管理器能够检索给定链码的链码信息
type ChaincodeInfoProvider interface {
//getdeployedchaincodeinfo检索有关已部署链代码的详细信息。
//如果未部署具有给定名称的链代码，则此函数应返回nil。
//或者部署的链码的版本或哈希与给定的版本和哈希不匹配
	GetDeployedChaincodeInfo(chainid string, chaincodeDefinition *ChaincodeDefinition) (*ledger.DeployedChaincodeInfo, error)
//RetrieveChainCodeArtifacts检查给定的链码是否安装在对等机上，如果是，
//它从链式代码包tarball中提取特定于状态数据库的工件
	RetrieveChaincodeArtifacts(chaincodeDefinition *ChaincodeDefinition) (installed bool, dbArtifactsTar []byte, err error)
}
