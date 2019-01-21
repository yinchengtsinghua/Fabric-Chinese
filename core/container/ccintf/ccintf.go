
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


package ccintf

//此包定义支持运行时和
//链码和对等端之间的通信（链码支持）。
//当前InprocController使用它。DockerController没有。

import (
	"fmt"

	pb "github.com/hyperledger/fabric/protos/peer"
)

//对等端和链码实例之间的流的链码流接口。
type ChaincodeStream interface {
	Send(*pb.ChaincodeMessage) error
	Recv() (*pb.ChaincodeMessage, error)
}

//CCSUPPORT必须由对等机中的链码支持端实现。
//（如链码支持）
type CCSupport interface {
	HandleChaincodeStream(ChaincodeStream) error
}

//getcchandlerkey用于通过上下文传递CCSUPPORT
func GetCCHandlerKey() string {
	return "CCHANDLER"
}

//CCID封装了链代码ID
type CCID struct {
	Name    string
	Version string
}

//getname返回基于ccid字段的规范链码名称
func (ccid *CCID) GetName() string {
	if ccid.Version != "" {
		return fmt.Sprintf("%s-%s", ccid.Name, ccid.Version)
	}
	return ccid.Name
}
