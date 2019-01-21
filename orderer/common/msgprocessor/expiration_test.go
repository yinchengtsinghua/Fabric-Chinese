
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


package msgprocessor

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/mocks/config"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createEnvelope(t *testing.T, serializedIdentity []byte) *common.Envelope {
	sHdr := utils.MakeSignatureHeader(serializedIdentity, nil)
	hdr := utils.MakePayloadHeader(&common.ChannelHeader{}, sHdr)
	payload := &common.Payload{
		Header: hdr,
	}
	payloadBytes, err := proto.Marshal(payload)
	assert.NoError(t, err)
	return &common.Envelope{
		Payload:   payloadBytes,
		Signature: []byte{1, 2, 3},
	}
}

func createX509Identity(t *testing.T, certFileName string) []byte {
	certBytes, err := ioutil.ReadFile(filepath.Join("testdata", certFileName))
	assert.NoError(t, err)
	sId := &msp.SerializedIdentity{
		IdBytes: certBytes,
	}
	idBytes, err := proto.Marshal(sId)
	assert.NoError(t, err)
	return idBytes
}

func createIdemixIdentity(t *testing.T) []byte {
	idemixId := &msp.SerializedIdemixIdentity{
		NymX: []byte{1, 2, 3},
		NymY: []byte{1, 2, 3},
		Ou:   []byte("OU1"),
	}
	idemixBytes, err := proto.Marshal(idemixId)
	assert.NoError(t, err)
	sId := &msp.SerializedIdentity{
		IdBytes: idemixBytes,
	}
	idBytes, err := proto.Marshal(sId)
	assert.NoError(t, err)
	return idBytes
}

type resourcesMock struct {
	mock.Mock
}

func (r *resourcesMock) OrdererConfig() (channelconfig.Orderer, bool) {
	args := r.Called()
	if args.Get(1).(bool) {
		return args.Get(0).(channelconfig.Orderer), true
	}
	return nil, false
}

func TestExpirationRejectRule(t *testing.T) {
	activeCapability := &config.Orderer{CapabilitiesVal: &config.OrdererCapabilities{
		ExpirationVal: true,
	}}
	inActiveCapability := &config.Orderer{CapabilitiesVal: &config.OrdererCapabilities{
		ExpirationVal: false,
	}}
	resources := &resourcesMock{}
	setupMock := func() {
//奇数调用返回活动功能
		resources.On("OrdererConfig").Return(activeCapability, true).Once()
//即使调用返回非活动功能
		resources.On("OrdererConfig").Return(inActiveCapability, true).Once()
	}
	t.Run("NoOrdererConfig", func(t *testing.T) {
		resources.On("OrdererConfig").Return(nil, false).Once()
		assert.Panics(t, func() {
			NewExpirationRejectRule(resources).Apply(&common.Envelope{})
		})
	})
	t.Run("BadEnvelope", func(t *testing.T) {
		setupMock()
		err := NewExpirationRejectRule(resources).Apply(&common.Envelope{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not convert message to signedData")

		err = NewExpirationRejectRule(resources).Apply(&common.Envelope{})
		assert.NoError(t, err)
	})
	t.Run("ExpiredX509Identity", func(t *testing.T) {
		setupMock()
		env := createEnvelope(t, createX509Identity(t, "expiredCert.pem"))
		err := NewExpirationRejectRule(resources).Apply(env)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "identity expired")

		err = NewExpirationRejectRule(resources).Apply(env)
		assert.NoError(t, err)
	})
	t.Run("IdemixIdentity", func(t *testing.T) {
		setupMock()
		env := createEnvelope(t, createIdemixIdentity(t))
		assert.Nil(t, NewExpirationRejectRule(resources).Apply(env))
		assert.Nil(t, NewExpirationRejectRule(resources).Apply(env))
	})
	t.Run("NoneExpiredX509Identity", func(t *testing.T) {
		setupMock()
		env := createEnvelope(t, createX509Identity(t, "cert.pem"))
		assert.Nil(t, NewExpirationRejectRule(resources).Apply(env))
		assert.Nil(t, NewExpirationRejectRule(resources).Apply(env))
	})
}
