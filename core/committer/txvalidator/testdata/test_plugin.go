
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


package testdata

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/capabilities"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/identities"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/policies"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/state"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

//samplevalidationplugin是验证插件的示例，
//并用于执行插件验证程序提供的依赖项
type SampleValidationPlugin struct {
	t  *testing.T
	d  IdentityDeserializer
	c  Capabilities
	sf StateFetcher
	pe PolicyEvaluator
}

//newsamplevalidationplugin返回验证插件安装程序的实例
//断言。
func NewSampleValidationPlugin(t *testing.T) *SampleValidationPlugin {
	return &SampleValidationPlugin{t: t}
}

type MarshaledSignedData struct {
	Data      []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Signature []byte `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
	Identity  []byte `protobuf:"bytes,2,opt,name=identity,proto3" json:"identity,omitempty"`
}

func (sd *MarshaledSignedData) Reset() {
	*sd = MarshaledSignedData{}
}

func (*MarshaledSignedData) String() string {
	panic("implement me")
}

func (*MarshaledSignedData) ProtoMessage() {
	panic("implement me")
}

func (p *SampleValidationPlugin) Validate(block *common.Block, namespace string, txPosition int, actionPosition int, contextData ...validation.ContextDatum) error {
	txData := block.Data.Data[0]
	txn := &MarshaledSignedData{}
	err := proto.Unmarshal(txData, txn)
	assert.NoError(p.t, err)

//检查链码是否已实例化
	state, err := p.sf.FetchState()
	if err != nil {
		return err
	}
	defer state.Done()

	results, err := state.GetStateMultipleKeys("lscc", []string{namespace})
	if err != nil {
		return err
	}

	_ = p.c.PrivateChannelData()

	if len(results) == 0 {
		return errors.New("not instantiated")
	}

//检查标识是否可以正确反序列化
	identity, err := p.d.DeserializeIdentity(txn.Identity)
	if err != nil {
		return err
	}

	identifier := identity.GetIdentityIdentifier()
	assert.Equal(p.t, "SampleOrg", identifier.Mspid)
	assert.Equal(p.t, "foo", identifier.Id)

	sd := &common.SignedData{
		Signature: txn.Signature,
		Data:      txn.Data,
		Identity:  txn.Identity,
	}
//验证策略
	pol := contextData[0].(SerializedPolicy).Bytes()
	err = p.pe.Evaluate(pol, []*common.SignedData{sd})
	if err != nil {
		return err
	}

	return nil
}

func (p *SampleValidationPlugin) Init(dependencies ...validation.Dependency) error {
	for _, dep := range dependencies {
		if deserializer, isIdentityDeserializer := dep.(IdentityDeserializer); isIdentityDeserializer {
			p.d = deserializer
		}
		if capabilities, isCapabilities := dep.(Capabilities); isCapabilities {
			p.c = capabilities
		}
		if stateFetcher, isStateFetcher := dep.(StateFetcher); isStateFetcher {
			p.sf = stateFetcher
		}
		if policyEvaluator, isPolicyFetcher := dep.(PolicyEvaluator); isPolicyFetcher {
			p.pe = policyEvaluator
		}
	}
	if p.sf == nil {
		p.t.Fatal("stateFetcher not passed in init")
	}
	if p.d == nil {
		p.t.Fatal("identityDeserializer not passed in init")
	}
	if p.c == nil {
		p.t.Fatal("capabilities not passed in init")
	}
	if p.pe == nil {
		p.t.Fatal("policy fetcher not passed in init")
	}
	return nil
}
