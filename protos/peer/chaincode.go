
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


package peer

//name实现platforms.namedescriber接口
func (cs *ChaincodeSpec) Name() string {
	if cs.ChaincodeId == nil {
		return ""
	}

	return cs.ChaincodeId.Name
}

func (cs *ChaincodeSpec) Version() string {
	if cs.ChaincodeId == nil {
		return ""
	}

	return cs.ChaincodeId.Version
}

//path实现platforms.pathDescriber接口
func (cs *ChaincodeSpec) Path() string {
	if cs.ChaincodeId == nil {
		return ""
	}

	return cs.ChaincodeId.Path
}

func (cs *ChaincodeSpec) CCType() string {
	return cs.Type.String()
}

//path实现platforms.pathDescriber接口
func (cds *ChaincodeDeploymentSpec) Path() string {
	if cds.ChaincodeSpec == nil {
		return ""
	}

	return cds.ChaincodeSpec.Path()
}

//bytes实现platforms.codepackage接口
func (cds *ChaincodeDeploymentSpec) Bytes() []byte {
	return cds.CodePackage
}

func (cds *ChaincodeDeploymentSpec) CCType() string {
	if cds.ChaincodeSpec == nil {
		return ""
	}

	return cds.ChaincodeSpec.CCType()
}

func (cds *ChaincodeDeploymentSpec) Name() string {
	if cds.ChaincodeSpec == nil {
		return ""
	}

	return cds.ChaincodeSpec.Name()
}

func (cds *ChaincodeDeploymentSpec) Version() string {
	if cds.ChaincodeSpec == nil {
		return ""
	}

	return cds.ChaincodeSpec.Version()
}
