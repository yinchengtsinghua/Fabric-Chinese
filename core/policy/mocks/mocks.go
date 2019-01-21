
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


package mocks

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	mspproto "github.com/hyperledger/fabric/protos/msp"
)

type MockChannelPolicyManagerGetter struct {
	Managers map[string]policies.Manager
}

func (c *MockChannelPolicyManagerGetter) Manager(channelID string) (policies.Manager, bool) {
	return c.Managers[channelID], true
}

type MockChannelPolicyManager struct {
	MockPolicy policies.Policy
}

func (m *MockChannelPolicyManager) GetPolicy(id string) (policies.Policy, bool) {
	return m.MockPolicy, true
}

func (m *MockChannelPolicyManager) Manager(path []string) (policies.Manager, bool) {
	panic("Not implemented")
}

type MockPolicy struct {
	Deserializer msp.IdentityDeserializer
}

//Evaluate获取一组SignedData并评估该组签名是否满足策略
func (m *MockPolicy) Evaluate(signatureSet []*common.SignedData) error {
	fmt.Printf("Evaluate [%s], [% x], [% x]\n", string(signatureSet[0].Identity), string(signatureSet[0].Data), string(signatureSet[0].Signature))
	identity, err := m.Deserializer.DeserializeIdentity(signatureSet[0].Identity)
	if err != nil {
		return err
	}

	return identity.Verify(signatureSet[0].Data, signatureSet[0].Signature)
}

type MockIdentityDeserializer struct {
	Identity []byte
	Msg      []byte
}

func (d *MockIdentityDeserializer) DeserializeIdentity(serializedIdentity []byte) (msp.Identity, error) {
	fmt.Printf("[DeserializeIdentity] id : [%s], [%s]\n", string(serializedIdentity), string(d.Identity))
	if bytes.Equal(d.Identity, serializedIdentity) {
		fmt.Printf("GOT : [%s], [%s]\n", string(serializedIdentity), string(d.Identity))
		return &MockIdentity{identity: d.Identity, msg: d.Msg}, nil
	}

	return nil, errors.New("Invalid Identity")
}

func (d *MockIdentityDeserializer) IsWellFormed(_ *mspproto.SerializedIdentity) error {
	return nil
}

type MockIdentity struct {
	identity []byte
	msg      []byte
}

func (id *MockIdentity) Anonymous() bool {
	panic("implement me")
}

func (id *MockIdentity) SatisfiesPrincipal(p *mspproto.MSPPrincipal) error {
	fmt.Printf("[SatisfiesPrincipal] id : [%s], [%s]\n", string(id.identity), string(p.Principal))
	if !bytes.Equal(id.identity, p.Principal) {
		return fmt.Errorf("Different identities [% x]!=[% x]", id.identity, p.Principal)
	}
	return nil
}

func (id *MockIdentity) ExpiresAt() time.Time {
	return time.Time{}
}

func (id *MockIdentity) GetIdentifier() *msp.IdentityIdentifier {
	return &msp.IdentityIdentifier{Mspid: "mock", Id: "mock"}
}

func (id *MockIdentity) GetMSPIdentifier() string {
	return "mock"
}

func (id *MockIdentity) Validate() error {
	return nil
}

func (id *MockIdentity) GetOrganizationalUnits() []*msp.OUIdentifier {
	return nil
}

func (id *MockIdentity) Verify(msg []byte, sig []byte) error {
	fmt.Printf("VERIFY [% x], [% x], [% x]\n", string(id.msg), string(msg), string(sig))
	if bytes.Equal(id.msg, msg) {
		if bytes.Equal(msg, sig) {
			return nil
		}
	}

	return errors.New("Invalid Signature")
}

func (id *MockIdentity) Serialize() ([]byte, error) {
	return []byte("cert"), nil
}

type MockMSPPrincipalGetter struct {
	Principal []byte
}

func (m *MockMSPPrincipalGetter) Get(role string) (*mspproto.MSPPrincipal, error) {
	return &mspproto.MSPPrincipal{Principal: m.Principal}, nil
}
