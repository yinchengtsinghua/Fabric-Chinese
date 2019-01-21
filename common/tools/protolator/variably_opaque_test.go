
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


package protolator

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/tools/protolator/testprotos"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func extractNestedMsgPlainField(source []byte) string {
	result := &testprotos.NestedMsg{}
	err := proto.Unmarshal(source, result)
	if err != nil {
		panic(err)
	}
	return result.PlainNestedField.PlainField
}

func TestPlainVariablyOpaqueMsg(t *testing.T) {
	fromPrefix := "from"
	toPrefix := "to"
	tppff := &testProtoPlainFieldFactory{
		fromPrefix: fromPrefix,
		toPrefix:   toPrefix,
	}

	fieldFactories = []protoFieldFactory{tppff}

	pfValue := "foo"
	startMsg := &testprotos.VariablyOpaqueMsg{
		OpaqueType: "NestedMsg",
		PlainOpaqueField: utils.MarshalOrPanic(&testprotos.NestedMsg{
			PlainNestedField: &testprotos.SimpleMsg{
				PlainField: pfValue,
			},
		}),
	}

	var buffer bytes.Buffer
	assert.NoError(t, DeepMarshalJSON(&buffer, startMsg))
	newMsg := &testprotos.VariablyOpaqueMsg{}
	assert.NoError(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))
	assert.NotEqual(t, fromPrefix+toPrefix+extractNestedMsgPlainField(startMsg.PlainOpaqueField), extractNestedMsgPlainField(newMsg.PlainOpaqueField))

	fieldFactories = []protoFieldFactory{tppff, nestedFieldFactory{}, variablyOpaqueFieldFactory{}}

	buffer.Reset()
	assert.NoError(t, DeepMarshalJSON(&buffer, startMsg))
	assert.NoError(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))
	assert.Equal(t, fromPrefix+toPrefix+extractNestedMsgPlainField(startMsg.PlainOpaqueField), extractNestedMsgPlainField(newMsg.PlainOpaqueField))
}

func TestMapVariablyOpaqueMsg(t *testing.T) {
	fromPrefix := "from"
	toPrefix := "to"
	tppff := &testProtoPlainFieldFactory{
		fromPrefix: fromPrefix,
		toPrefix:   toPrefix,
	}

	fieldFactories = []protoFieldFactory{tppff}

	pfValue := "foo"
	mapKey := "bar"
	startMsg := &testprotos.VariablyOpaqueMsg{
		OpaqueType: "NestedMsg",
		MapOpaqueField: map[string][]byte{
			mapKey: utils.MarshalOrPanic(&testprotos.NestedMsg{
				PlainNestedField: &testprotos.SimpleMsg{
					PlainField: pfValue,
				},
			}),
		},
	}

	var buffer bytes.Buffer
	assert.NoError(t, DeepMarshalJSON(&buffer, startMsg))
	newMsg := &testprotos.VariablyOpaqueMsg{}
	assert.NoError(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))
	assert.NotEqual(t, fromPrefix+toPrefix+extractNestedMsgPlainField(startMsg.MapOpaqueField[mapKey]), extractNestedMsgPlainField(newMsg.MapOpaqueField[mapKey]))

	fieldFactories = []protoFieldFactory{tppff, nestedFieldFactory{}, variablyOpaqueMapFieldFactory{}}

	buffer.Reset()
	assert.NoError(t, DeepMarshalJSON(&buffer, startMsg))
	assert.NoError(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))
	assert.Equal(t, fromPrefix+toPrefix+extractNestedMsgPlainField(startMsg.MapOpaqueField[mapKey]), extractNestedMsgPlainField(newMsg.MapOpaqueField[mapKey]))
}

func TestSliceVariablyOpaqueMsg(t *testing.T) {
	fromPrefix := "from"
	toPrefix := "to"
	tppff := &testProtoPlainFieldFactory{
		fromPrefix: fromPrefix,
		toPrefix:   toPrefix,
	}

	fieldFactories = []protoFieldFactory{tppff}

	pfValue := "foo"
	startMsg := &testprotos.VariablyOpaqueMsg{
		OpaqueType: "NestedMsg",
		SliceOpaqueField: [][]byte{
			utils.MarshalOrPanic(&testprotos.NestedMsg{
				PlainNestedField: &testprotos.SimpleMsg{
					PlainField: pfValue,
				},
			}),
		},
	}

	var buffer bytes.Buffer
	assert.NoError(t, DeepMarshalJSON(&buffer, startMsg))
	newMsg := &testprotos.VariablyOpaqueMsg{}
	assert.NoError(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))
	assert.NotEqual(t, fromPrefix+toPrefix+extractNestedMsgPlainField(startMsg.SliceOpaqueField[0]), extractNestedMsgPlainField(newMsg.SliceOpaqueField[0]))

	fieldFactories = []protoFieldFactory{tppff, nestedFieldFactory{}, variablyOpaqueSliceFieldFactory{}}

	buffer.Reset()
	assert.NoError(t, DeepMarshalJSON(&buffer, startMsg))
	assert.NoError(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))
	assert.Equal(t, fromPrefix+toPrefix+extractNestedMsgPlainField(startMsg.SliceOpaqueField[0]), extractNestedMsgPlainField(newMsg.SliceOpaqueField[0]))
}
