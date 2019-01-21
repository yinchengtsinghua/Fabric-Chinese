
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/stretchr/testify/assert"
)

const (
	AESKEY1   = "01234567890123456789012345678901"
	AESKEY2   = "01234567890123456789012345678902"
	ECDSAKEY1 = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIH4Uv66F9kZMdOQxwNegkGm8c3AB3nGPOtxNKi6wb/ZooAoGCCqGSM49
AwEHoUQDQgAEEPE+VLOh+e4NpwIjI/b/fKYHi4weU7r9OTEYPiAJiJBQY6TZnvF5
oRMvwO4MCYxFtpIRO4UxIgcZBj4NCBxKqQ==
-----END EC PRIVATE KEY-----`
	ECDSAKEY2 = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIE8Seyc9TXx+yQfnGPuzjkuEfMbkq203IYdfyvMd0r3OoAoGCCqGSM49
AwEHoUQDQgAE4dcGMMroH2LagI/s5i/Bx4t4ggGDoJPNVkKBDBlIaMYjJFYD1obk
JOWqAZxKKsBxBC5Ssu+fS26VPfdNWxDsFQ==
-----END EC PRIVATE KEY-----`
	IV1 = "0123456789012345"
)

func TestInit(t *testing.T) {
	factory.InitFactories(nil)

	scc := new(EncCC)
	stub := shim.NewMockStub("enccc", scc)
	stub.MockTransactionStart("a")
	res := scc.Init(stub)
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))
}

//不幸的是，由于
//模拟垫片不支持瞬态。我们测试失败场景
//下面的测试通过调用
//与CC相反的功能
func TestInvoke(t *testing.T) {
	factory.InitFactories(nil)

	scc := &EncCC{factory.GetDefault()}
	stub := shim.NewMockStub("enccc", scc)

	res := stub.MockInvoke("tx", [][]byte{[]byte("barf")})
	assert.NotEqual(t, res.Status, int32(shim.OK))
	res = stub.MockInvoke("tx", [][]byte{[]byte("ENC")})
	assert.NotEqual(t, res.Status, int32(shim.OK))
	res = stub.MockInvoke("tx", [][]byte{[]byte("SIG")})
	assert.NotEqual(t, res.Status, int32(shim.OK))
	res = stub.MockInvoke("tx", [][]byte{[]byte("RANGE")})
	assert.NotEqual(t, res.Status, int32(shim.OK))
}

func TestEnc(t *testing.T) {
	factory.InitFactories(nil)

	scc := &EncCC{factory.GetDefault()}
	stub := shim.NewMockStub("enccc", scc)

//成功
	stub.MockTransactionStart("a")
	res := scc.Encrypter(stub, []string{"key", "value"}, []byte(AESKEY1), nil)
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))

//故障-坏钥匙
	stub.MockTransactionStart("a")
	res = scc.Encrypter(stub, []string{"key", "value"}, []byte("badkey"), nil)
	stub.MockTransactionEnd("a")
	assert.NotEqual(t, res.Status, int32(shim.OK))

//失败-参数不足
	stub.MockTransactionStart("a")
	res = scc.Encrypter(stub, []string{"key"}, []byte(AESKEY1), nil)
	stub.MockTransactionEnd("a")
	assert.NotEqual(t, res.Status, int32(shim.OK))

//成功
	stub.MockTransactionStart("a")
	res = scc.Decrypter(stub, []string{"key"}, []byte(AESKEY1), nil)
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))
	assert.True(t, bytes.Equal(res.Payload, []byte("value")))

//失败-参数不足
	stub.MockTransactionStart("a")
	res = scc.Decrypter(stub, []string{}, []byte(AESKEY1), nil)
	stub.MockTransactionEnd("a")
	assert.NotEqual(t, res.Status, int32(shim.OK))

//失败的KVS密钥
	stub.MockTransactionStart("a")
	res = scc.Decrypter(stub, []string{"badkey"}, []byte(AESKEY1), nil)
	stub.MockTransactionEnd("a")
	assert.NotEqual(t, res.Status, int32(shim.OK))

//故障-坏钥匙
	stub.MockTransactionStart("a")
	res = scc.Decrypter(stub, []string{"key"}, []byte(AESKEY2), nil)
	stub.MockTransactionEnd("a")
	assert.NotEqual(t, res.Status, int32(shim.OK))
}

func TestSig(t *testing.T) {
	factory.InitFactories(nil)

	scc := &EncCC{factory.GetDefault()}
	stub := shim.NewMockStub("enccc", scc)

//成功
	stub.MockTransactionStart("a")
	res := scc.EncrypterSigner(stub, []string{"key", "value"}, []byte(AESKEY1), []byte(ECDSAKEY1))
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))

//故障-坏钥匙
	stub.MockTransactionStart("a")
	res = scc.EncrypterSigner(stub, []string{"key", "value"}, []byte(AESKEY1), []byte("barf"))
	stub.MockTransactionEnd("a")
	assert.NotEqual(t, res.Status, int32(shim.OK))

//故障-坏的ARG
	stub.MockTransactionStart("a")
	res = scc.EncrypterSigner(stub, []string{"key"}, []byte(AESKEY1), []byte("barf"))
	stub.MockTransactionEnd("a")
	assert.NotEqual(t, res.Status, int32(shim.OK))

//失败-签名密钥错误
	stub.MockTransactionStart("a")
	res = scc.DecrypterVerify(stub, []string{"key"}, []byte(AESKEY1), []byte(ECDSAKEY2))
	stub.MockTransactionEnd("a")
	assert.NotEqual(t, res.Status, int32(shim.OK))

//故障-坏的ARG
	stub.MockTransactionStart("a")
	res = scc.DecrypterVerify(stub, []string{}, []byte(AESKEY1), []byte(ECDSAKEY1))
	stub.MockTransactionEnd("a")
	assert.NotEqual(t, res.Status, int32(shim.OK))

//失败的KVS密钥
	stub.MockTransactionStart("a")
	res = scc.DecrypterVerify(stub, []string{"badkey"}, []byte(AESKEY1), []byte(ECDSAKEY1))
	stub.MockTransactionEnd("a")
	assert.NotEqual(t, res.Status, int32(shim.OK))

//成功
	stub.MockTransactionStart("a")
	res = scc.EncrypterSigner(stub, []string{"key", "value"}, []byte(AESKEY1), []byte(ECDSAKEY1))
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))

//成功
	stub.MockTransactionStart("a")
	res = scc.DecrypterVerify(stub, []string{"key"}, []byte(AESKEY1), []byte(ECDSAKEY1))
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))
	assert.True(t, bytes.Equal(res.Payload, []byte("value")))
}

func TestEncCC_RangeDecrypter(t *testing.T) {
	factory.InitFactories(nil)

	scc := &EncCC{factory.GetDefault()}
	stub := shim.NewMockStub("enccc", scc)

	stub.MockTransactionStart("a")
	res := scc.Encrypter(stub, []string{"key1", "value1"}, []byte(AESKEY1), nil)
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))

	stub.MockTransactionStart("a")
	res = scc.Encrypter(stub, []string{"key2", "value2"}, []byte(AESKEY1), nil)
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))

	stub.MockTransactionStart("a")
	res = scc.Encrypter(stub, []string{"key3", "value3"}, []byte(AESKEY1), nil)
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))

//范围查询失败
	res = scc.RangeDecrypter(stub, nil)
	assert.NotEqual(t, res.Status, int32(shim.OK))

//成功范围查询
	stub.MockTransactionStart("a")
	res = scc.RangeDecrypter(stub, []byte(AESKEY1))
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))
	keys := []keyValuePair{}
	err := json.Unmarshal(res.Payload, &keys)
	assert.NoError(t, err)
	assert.Equal(t, keys[0].Key, "key1")
	assert.Equal(t, string(keys[0].Value), "value1")
	assert.Equal(t, keys[1].Key, "key2")
	assert.Equal(t, string(keys[1].Value), "value2")
	assert.Equal(t, keys[2].Key, "key3")
	assert.Equal(t, string(keys[2].Value), "value3")

	_, err = getStateByRangeAndDecrypt(stub, nil, string([]byte{0}), string([]byte{0}))
	assert.Error(t, err)
}

func TestDeterministicEncryption(t *testing.T) {
	factory.InitFactories(nil)

	scc := &EncCC{factory.GetDefault()}
	stub := shim.NewMockStub("enccc", scc)

	stub.MockTransactionStart("a")
	res := scc.Encrypter(stub, []string{"key1", "value1"}, []byte(AESKEY1), []byte(IV1))
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))

	c1, err := stub.GetState("key1")
	assert.NoError(t, err)
	assert.NotNil(t, c1)

	stub.MockTransactionStart("a")
	res = scc.Encrypter(stub, []string{"key1", "value1"}, []byte(AESKEY1), []byte(IV1))
	stub.MockTransactionEnd("a")
	assert.Equal(t, res.Status, int32(shim.OK))

	c2, err := stub.GetState("key1")
	assert.NoError(t, err)
	assert.NotNil(t, c1)
	assert.True(t, bytes.Equal(c1, c2))
}
