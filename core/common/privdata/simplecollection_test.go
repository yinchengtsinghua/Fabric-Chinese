
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


package privdata

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/msp"
	pb "github.com/hyperledger/fabric/protos/common"
	mb "github.com/hyperledger/fabric/protos/msp"
	"github.com/stretchr/testify/assert"
)

func createCollectionPolicyConfig(accessPolicy *pb.SignaturePolicyEnvelope) *pb.CollectionPolicyConfig {
	cpcSp := &pb.CollectionPolicyConfig_SignaturePolicy{
		SignaturePolicy: accessPolicy,
	}
	cpc := &pb.CollectionPolicyConfig{
		Payload: cpcSp,
	}
	return cpc
}

type mockIdentity struct {
	idBytes []byte
}

func (id *mockIdentity) Anonymous() bool {
	panic("implement me")
}

func (id *mockIdentity) ExpiresAt() time.Time {
	return time.Time{}
}

func (id *mockIdentity) SatisfiesPrincipal(p *mb.MSPPrincipal) error {
	if bytes.Compare(id.idBytes, p.Principal) == 0 {
		return nil
	}
	return errors.New("Principals do not match")
}

func (id *mockIdentity) GetIdentifier() *msp.IdentityIdentifier {
	return &msp.IdentityIdentifier{Mspid: "Mock", Id: string(id.idBytes)}
}

func (id *mockIdentity) GetMSPIdentifier() string {
	return string(id.idBytes)
}

func (id *mockIdentity) Validate() error {
	return nil
}

func (id *mockIdentity) GetOrganizationalUnits() []*msp.OUIdentifier {
	return nil
}

func (id *mockIdentity) Verify(msg []byte, sig []byte) error {
	if bytes.Compare(sig, []byte("badsigned")) == 0 {
		return errors.New("Invalid signature")
	}
	return nil
}

func (id *mockIdentity) Serialize() ([]byte, error) {
	return id.idBytes, nil
}

type mockDeserializer struct {
	fail error
}

func (md *mockDeserializer) DeserializeIdentity(serializedIdentity []byte) (msp.Identity, error) {
	if md.fail != nil {
		return nil, md.fail
	}
	return &mockIdentity{idBytes: serializedIdentity}, nil
}

func (md *mockDeserializer) IsWellFormed(_ *mb.SerializedIdentity) error {
	return nil
}

func TestSetupBadConfig(t *testing.T) {
//使用无效数据设置简单集合
	var sc SimpleCollection
	err := sc.Setup(&pb.StaticCollectionConfig{}, &mockDeserializer{})
	assert.Error(t, err)
}

func TestSetupGoodConfigCollection(t *testing.T) {
//创建成员访问策略
	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}
	policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
	accessPolicy := createCollectionPolicyConfig(policyEnvelope)

//创建静态集合配置
	collectionConfig := &pb.StaticCollectionConfig{
		Name:              "test collection",
		RequiredPeerCount: 1,
		MemberOrgsPolicy:  accessPolicy,
	}

//使用有效数据设置简单集合
	var sc SimpleCollection
	err := sc.Setup(collectionConfig, &mockDeserializer{})
	assert.NoError(t, err)

//检查名称
	assert.True(t, sc.CollectionID() == "test collection")

//检查成员
	members := sc.MemberOrgs()
	assert.True(t, members[0] == "signer0")
	assert.True(t, members[1] == "signer1")

//检查所需的对等计数
	assert.True(t, sc.RequiredPeerCount() == 1)
}

func TestSimpleCollectionFilter(t *testing.T) {
//创建成员访问策略
	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}
	policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
	accessPolicy := createCollectionPolicyConfig(policyEnvelope)

//创建静态集合配置
	collectionConfig := &pb.StaticCollectionConfig{
		Name:              "test collection",
		RequiredPeerCount: 1,
		MemberOrgsPolicy:  accessPolicy,
	}

//建立简单的馆藏
	var sc SimpleCollection
	err := sc.Setup(collectionConfig, &mockDeserializer{})
	assert.NoError(t, err)

//获取集合访问筛选器
	var cap CollectionAccessPolicy
	cap = &sc
	accessFilter := cap.AccessFilter()

//检查筛选器：不是集合的成员
	notMember := pb.SignedData{
		Identity:  []byte{1, 2, 3},
		Signature: []byte{},
		Data:      []byte{},
	}
	assert.False(t, accessFilter(notMember))

//检查筛选器：集合的成员
	member := pb.SignedData{
		Identity:  signers[0],
		Signature: []byte{},
		Data:      []byte{},
	}
	assert.True(t, accessFilter(member))
}
