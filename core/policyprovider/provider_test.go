
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package policyprovider

import (
	"testing"

	"github.com/hyperledger/fabric/core/policy"
	"github.com/stretchr/testify/assert"
)

func TestGetPolicyChecker(t *testing.T) {
//确保GetPolicyChecker实现policy.policyChecker
	var pc policy.PolicyChecker
	pc = GetPolicyChecker()
	assert.NotNil(t, pc)
}

func TestDefaultFactory_NewPolicyChecker(t *testing.T) {
//确保DefaultFactory执行policy.policyCheckerFactory
	var pcf policy.PolicyCheckerFactory
	pcf = &defaultFactory{}
	var pc policy.PolicyChecker
//检查我们可以获得一个新的政策检查者
	pc = pcf.NewPolicyChecker()
	assert.NotNil(t, pc)
}
