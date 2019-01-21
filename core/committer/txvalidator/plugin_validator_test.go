
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


package txvalidator_test

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/mocks/ledger"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/committer/txvalidator/mocks"
	"github.com/hyperledger/fabric/core/committer/txvalidator/testdata"
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/capabilities"
	"github.com/hyperledger/fabric/msp"
	. "github.com/hyperledger/fabric/msp/mocks"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidateWithPlugin(t *testing.T) {
	pm := make(txvalidator.MapBasedPluginMapper)
	qec := &mocks.QueryExecutorCreator{}
	deserializer := &mocks.IdentityDeserializer{}
	capabilites := &mocks.Capabilities{}
	v := txvalidator.NewPluginValidator(pm, qec, deserializer, capabilites)
	ctx := &txvalidator.Context{
		Namespace: "mycc",
		VSCCName:  "vscc",
	}

//场景一：插件没有找到，因为地图上还没有填充任何内容
	err := v.ValidateWithPlugin(ctx)
	assert.Contains(t, err.Error(), "plugin with name vscc wasn't found")

//场景二：插件初始化失败
	factory := &mocks.PluginFactory{}
	plugin := &mocks.Plugin{}
	plugin.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("foo")).Once()
	factory.On("New").Return(plugin)
	pm["vscc"] = factory
	err = v.ValidateWithPlugin(ctx)
	assert.Contains(t, err.(*validation.ExecutionFailureError).Error(), "failed initializing plugin: foo")

//场景三：插件初始化成功，但出现执行错误。
//插件应该按原样传递错误。
	plugin.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	validationErr := &validation.ExecutionFailureError{
		Reason: "bar",
	}
	plugin.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(validationErr).Once()
	err = v.ValidateWithPlugin(ctx)
	assert.Equal(t, validationErr, err)

//场景四：插件初始化成功，验证通过
	plugin.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	plugin.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	err = v.ValidateWithPlugin(ctx)
	assert.NoError(t, err)
}

func TestSamplePlugin(t *testing.T) {
	pm := make(txvalidator.MapBasedPluginMapper)
	qec := &mocks.QueryExecutorCreator{}

	qec.On("NewQueryExecutor").Return(&ledger.MockQueryExecutor{
		State: map[string]map[string][]byte{
			"lscc": {
				"mycc": []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
		},
	}, nil)

	deserializer := &mocks.IdentityDeserializer{}
	identity := &MockIdentity{}
	identity.On("GetIdentifier").Return(&msp.IdentityIdentifier{
		Mspid: "SampleOrg",
		Id:    "foo",
	})
	deserializer.On("DeserializeIdentity", []byte{7, 8, 9}).Return(identity, nil)
	capabilites := &mocks.Capabilities{}
	capabilites.On("PrivateChannelData").Return(true)
	factory := &mocks.PluginFactory{}
	factory.On("New").Return(testdata.NewSampleValidationPlugin(t))
	pm["vscc"] = factory

	transaction := testdata.MarshaledSignedData{
		Data:      []byte{1, 2, 3},
		Signature: []byte{4, 5, 6},
		Identity:  []byte{7, 8, 9},
	}

	txnData, _ := proto.Marshal(&transaction)

	v := txvalidator.NewPluginValidator(pm, qec, deserializer, capabilites)
	acceptAllPolicyBytes, _ := proto.Marshal(cauthdsl.AcceptAllPolicy)
	ctx := &txvalidator.Context{
		Namespace: "mycc",
		VSCCName:  "vscc",
		Policy:    acceptAllPolicyBytes,
		Block: &common.Block{
			Header: &common.BlockHeader{},
			Data: &common.BlockData{
				Data: [][]byte{txnData},
			},
		},
		Channel: "mychannel",
	}
	assert.NoError(t, v.ValidateWithPlugin(ctx))
}

func TestCapabilitiesInterface(t *testing.T) {
//确保应用程序功能都由验证功能实现。
//获取应用程序功能的所有方法并确保
//应用程序功能中的每个方法都存在于验证功能中
	var appCapabilities *channelconfig.ApplicationCapabilities
	appMeta := reflect.TypeOf(appCapabilities).Elem()

	var validationCapabilities *Capabilities
	validationMeta := reflect.TypeOf(validationCapabilities).Elem()
	for i := 0; i < appMeta.NumMethod(); i++ {
		method := appMeta.Method(i).Name
		_, exists := validationMeta.MethodByName(method)
		assert.True(t, exists, "method %s doesn't exist", method)
	}
}
