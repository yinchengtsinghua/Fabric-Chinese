
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

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


package common

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
)

//更多的重复的实用程序应该会消失，但是这些实用程序现在在导入周期中有点混乱。
func marshalOrPanic(msg proto.Message) []byte {
	data, err := proto.Marshal(msg)
	if err != nil {
		panic("Error marshaling")
	}
	return data
}

func TestNilConfigEnvelopeAsSignedData(t *testing.T) {
	var ce *ConfigUpdateEnvelope
	_, err := ce.AsSignedData()
	if err == nil {
		t.Fatalf("Should have errored trying to convert a nil signed config item to signed data")
	}
}

func TestConfigEnvelopeAsSignedData(t *testing.T) {
	configBytes := []byte("Foo")
	signatures := [][]byte{[]byte("Signature1"), []byte("Signature2")}
	identities := [][]byte{[]byte("Identity1"), []byte("Identity2")}

	configSignatures := make([]*ConfigSignature, len(signatures))
	for i := range configSignatures {
		configSignatures[i] = &ConfigSignature{
			SignatureHeader: marshalOrPanic(&SignatureHeader{
				Creator: identities[i],
			}),
			Signature: signatures[i],
		}
	}

	ce := &ConfigUpdateEnvelope{
		ConfigUpdate: configBytes,
		Signatures:   configSignatures,
	}

	signedData, err := ce.AsSignedData()
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	for i, sigData := range signedData {
		if !bytes.Equal(sigData.Identity, identities[i]) {
			t.Errorf("Expected identity to match at index %d", i)
		}
		if !bytes.Equal(sigData.Data, append(configSignatures[i].SignatureHeader, configBytes...)) {
			t.Errorf("Expected signature over concatenation of config item bytes and signature header")
		}
		if !bytes.Equal(sigData.Signature, signatures[i]) {
			t.Errorf("Expected signature to match at index %d", i)
		}
	}
}

func TestNilEnvelopeAsSignedData(t *testing.T) {
	var env *Envelope
	_, err := env.AsSignedData()
	if err == nil {
		t.Fatalf("Should have errored trying to convert a nil envelope")
	}
}

func TestEnvelopeAsSignedData(t *testing.T) {
	identity := []byte("Foo")
	signature := []byte("Bar")

	shdrbytes, err := proto.Marshal(&SignatureHeader{Creator: identity})
	if err != nil {
		t.Fatalf("%s", err)
	}

	env := &Envelope{
		Payload: marshalOrPanic(&Payload{
			Header: &Header{
				SignatureHeader: shdrbytes,
			},
		}),
		Signature: signature,
	}

	signedData, err := env.AsSignedData()
	if err != nil {
		t.Fatalf("Unexpected error converting envelope to SignedData: %s", err)
	}

	if len(signedData) != 1 {
		t.Fatalf("Expected 1 entry of signed data, but got %d", len(signedData))
	}

	if !bytes.Equal(signedData[0].Identity, identity) {
		t.Errorf("Wrong identity bytes")
	}
	if !bytes.Equal(signedData[0].Data, env.Payload) {
		t.Errorf("Wrong data bytes")
	}
	if !bytes.Equal(signedData[0].Signature, signature) {
		t.Errorf("Wrong data bytes")
	}
}
