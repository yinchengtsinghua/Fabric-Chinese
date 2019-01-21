
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package entities

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignedMessage(t *testing.T) {
	ent, err := GetEncrypterSignerEntityForTest("TEST")
	assert.NoError(t, err)
	assert.NotNil(t, ent)

	m := &SignedMessage{Payload: []byte("message"), ID: []byte(ent.ID())}

	err = m.Sign(ent)
	assert.NoError(t, err)
	v, err := m.Verify(ent)
	assert.NoError(t, err)
	assert.True(t, v)
}

func TestSignedMessageErr(t *testing.T) {
	ent, err := GetEncrypterSignerEntityForTest("TEST")
	assert.NoError(t, err)
	assert.NotNil(t, ent)

	m := &SignedMessage{Payload: []byte("message"), ID: []byte(ent.ID())}

	err = m.Sign(nil)
	assert.Error(t, err)
	_, err = m.Verify(nil)
	assert.Error(t, err)

	m = &SignedMessage{Payload: []byte("message"), Sig: []byte("barf")}
	_, err = m.Verify(nil)
	assert.Error(t, err)
}

func TestSignedMessageMarshaller(t *testing.T) {
	m1 := &SignedMessage{Payload: []byte("message"), Sig: []byte("sig"), ID: []byte("ID")}
	m2 := &SignedMessage{}
	b, err := m1.ToBytes()
	assert.NoError(t, err)
	err = m2.FromBytes(b)
	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(m1, m2))
}
