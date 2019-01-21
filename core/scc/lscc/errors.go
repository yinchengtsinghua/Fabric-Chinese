
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

import "fmt"

//InvalidFunctionerr无效函数错误
type InvalidFunctionErr string

func (f InvalidFunctionErr) Error() string {
	return fmt.Sprintf("invalid function to lscc: %s", string(f))
}

//InvalidArgslener无效参数长度错误
type InvalidArgsLenErr int

func (i InvalidArgsLenErr) Error() string {
	return fmt.Sprintf("invalid number of arguments to lscc: %d", int(i))
}

//TxNotFounderr事务未找到错误
type TXNotFoundErr string

func (t TXNotFoundErr) Error() string {
	return fmt.Sprintf("transaction not found: %s", string(t))
}

//InvalidDeploymentSpecErr invalid chaincode deployment spec error
type InvalidDeploymentSpecErr string

func (f InvalidDeploymentSpecErr) Error() string {
	return fmt.Sprintf("invalid deployment spec: %s", string(f))
}

//existser链码存在错误
type ExistsErr string

func (t ExistsErr) Error() string {
	return fmt.Sprintf("chaincode with name '%s' already exists", string(t))
}

//NotFounderr链码未注册为LSCC错误
type NotFoundErr string

func (t NotFoundErr) Error() string {
	return fmt.Sprintf("could not find chaincode with name '%s'", string(t))
}

//无效的频道名称错误无效的频道名称错误
type InvalidChannelNameErr string

func (f InvalidChannelNameErr) Error() string {
	return fmt.Sprintf("invalid channel name: %s", string(f))
}

//无效的chaincode name错误无效的chaincode name错误
type InvalidChaincodeNameErr string

func (f InvalidChaincodeNameErr) Error() string {
	return fmt.Sprintf("invalid chaincode name '%s'. Names can only consist of alphanumerics, '_', and '-'", string(f))
}

//Emptychaincodenameerr尝试升级到相同版本的chaincode
type EmptyChaincodeNameErr string

func (f EmptyChaincodeNameErr) Error() string {
	return fmt.Sprint("chaincode name not provided")
}

//无效版本错误无效版本错误
type InvalidVersionErr string

func (f InvalidVersionErr) Error() string {
	return fmt.Sprintf("invalid chaincode version '%s'. Versions can only consist of alphanumerics, '_',  '-', '+', and '.'", string(f))
}

//InvalidStateDBartifactser无效状态数据库项目错误
type InvalidStatedbArtifactsErr string

func (f InvalidStatedbArtifactsErr) Error() string {
	return fmt.Sprintf("invalid state database artifact: %s", string(f))
}

//两处的chaincode不匹配的chaincode名称不匹配
type ChaincodeMismatchErr string

func (f ChaincodeMismatchErr) Error() string {
	return fmt.Sprintf("chaincode name mismatch: %s", string(f))
}

//EmptyVersionErr空版本错误
type EmptyVersionErr string

func (f EmptyVersionErr) Error() string {
	return fmt.Sprintf("version not provided for chaincode with name '%s'", string(f))
}

//封送拆送处理程序错误
type MarshallErr string

func (m MarshallErr) Error() string {
	return fmt.Sprintf("error while marshalling: %s", string(m))
}

//尝试升级到相同版本的链码的IdenticalVersioner
type IdenticalVersionErr string

func (f IdenticalVersionErr) Error() string {
	return fmt.Sprintf("version already exists for chaincode with name '%s'", string(f))
}

//由于LSCC上的指纹与安装的CC不匹配，导致InvalidConfserRor错误
type InvalidCCOnFSError string

func (f InvalidCCOnFSError) Error() string {
	return fmt.Sprintf("chaincode fingerprint mismatch: %s", string(f))
}

//当升级CC时找不到现有的实例化策略时，InstantiationPolicyMissing
type InstantiationPolicyMissing string

func (f InstantiationPolicyMissing) Error() string {
	return "instantiation policy missing"
}

//未启用v1_2功能时不允许CollectionConfigupGrades
type CollectionsConfigUpgradesNotAllowed string

func (f CollectionsConfigUpgradesNotAllowed) Error() string {
	return "as V1_2 capability is not enabled, collection upgrades are not allowed"
}

//未启用v1_2或更高版本功能时，privatechanneldata不可用
type PrivateChannelDataNotAvailable string

func (f PrivateChannelDataNotAvailable) Error() string {
	return "as V1_2 or later capability is not enabled, private channel collections and data are not available"
}
