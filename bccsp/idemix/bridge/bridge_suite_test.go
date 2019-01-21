
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

package bridge_test

import (
	"testing"

	"github.com/hyperledger/fabric-amcl/amcl"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPlain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plain Suite")
}

//newrandanpanic是一个实用程序测试函数，调用时总是死机
func NewRandPanic() *amcl.RAND {
	panic("new rand panic")
}
