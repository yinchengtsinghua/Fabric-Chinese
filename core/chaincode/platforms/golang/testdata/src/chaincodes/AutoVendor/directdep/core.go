
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 *版权所有Greg Haskins保留所有权利
 *
 *SPDX许可证标识符：Apache-2.0
 *
 *有关详细信息，请参阅github.com/hyperledger/fabric/test/chaincodes/autovendor/chaincode/main.go。
 **/

package directdep

import (
	"chaincodes/AutoVendor/indirectdep"
)

func PointlessFunction() {
//授权我们间接依赖
	indirectdep.PointlessFunction()
}
