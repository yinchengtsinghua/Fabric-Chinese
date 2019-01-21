
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有，State Street Corp.保留所有权利。
γ
SPDX许可证标识符：Apache-2.0
**/


package ccmetadata

import (
	"github.com/hyperledger/fabric/common/flogging"
)

//此包使用的记录器
var logger = flogging.MustGetLogger("chaincode.platform.metadata")

//MetadataProvider由每个平台以特定于平台的方式实现。
//它可以处理以不同格式存储在chaincodedeploymentspec中的元数据。
//通用格式是targz。当前用户希望显示元数据
//作为tar文件条目（直接从以targz格式存储的链码中提取）。
//将来，我们希望通过扩展接口来提供更好的抽象
type MetadataProvider interface {
	GetMetadataAsTarEntries() ([]byte, error)
}
