
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


//package main是订购方二进制文件的入口点
//并且只调用server.main（）函数。没有其他
//此包中应包含函数。
package main

import "github.com/hyperledger/fabric/orderer/common/server"

func main() {
	server.Main()
}
