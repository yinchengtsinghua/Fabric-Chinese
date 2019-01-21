
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和

*/


package shim

import (
	pb "github.com/hyperledger/fabric/protos/peer"
)

const (
//OK常量-状态代码小于400，背书人将背书。
//OK表示初始化或调用成功。
	OK = 200

//错误阈值常量-大于或等于400的状态代码将被视为错误并被背书人拒绝。
	ERRORTHRESHOLD = 400

//错误常量-默认错误值
	ERROR = 500
)

func Success(payload []byte) pb.Response {
	return pb.Response{
		Status:  OK,
		Payload: payload,
	}
}

func Error(msg string) pb.Response {
	return pb.Response{
		Status:  ERROR,
		Message: msg,
	}
}
