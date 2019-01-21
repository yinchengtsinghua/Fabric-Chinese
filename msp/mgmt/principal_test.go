
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


package mgmt

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/stretchr/testify/assert"
)

func TestNewLocalMSPPrincipalGetter(t *testing.T) {
	assert.NotNil(t, NewLocalMSPPrincipalGetter())
}

func TestLocalMSPPrincipalGetter_Get(t *testing.T) {
	m := NewDeserializersManager()
	g := NewLocalMSPPrincipalGetter()

	_, err := g.Get("")
	assert.Error(t, err)

	p, err := g.Get(Admins)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, msp.MSPPrincipal_ROLE, p.PrincipalClassification)
	role := &msp.MSPRole{}
	proto.Unmarshal(p.Principal, role)
	assert.Equal(t, m.GetLocalMSPIdentifier(), role.MspIdentifier)
	assert.Equal(t, msp.MSPRole_ADMIN, role.Role)

	p, err = g.Get(Members)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, msp.MSPPrincipal_ROLE, p.PrincipalClassification)
	role = &msp.MSPRole{}
	proto.Unmarshal(p.Principal, role)
	assert.Equal(t, m.GetLocalMSPIdentifier(), role.MspIdentifier)
	assert.Equal(t, msp.MSPRole_MEMBER, role.Role)
}
