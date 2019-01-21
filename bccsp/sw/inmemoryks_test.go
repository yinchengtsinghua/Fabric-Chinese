
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


package sw

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidStore(t *testing.T) {
	t.Parallel()

	ks := NewInMemoryKeyStore()

	err := ks.StoreKey(nil)
	assert.EqualError(t, err, "key is nil")
}

func TestInvalidLoad(t *testing.T) {
	t.Parallel()

	ks := NewInMemoryKeyStore()

	_, err := ks.GetKey(nil)
	assert.EqualError(t, err, "ski is nil or empty")
}

func TestNoKeyFound(t *testing.T) {
	t.Parallel()

	ks := NewInMemoryKeyStore()

	ski := []byte("foo")
	_, err := ks.GetKey(ski)
	assert.EqualError(t, err, fmt.Sprintf("no key found for ski %x", ski))
}

func TestStoreLoad(t *testing.T) {
	t.Parallel()

	ks := NewInMemoryKeyStore()

//为要查找的密钥库生成密钥
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)
	cspKey := &ecdsaPrivateKey{privKey}

//存储密钥
	err = ks.StoreKey(cspKey)
	assert.NoError(t, err)

//加载键
	key, err := ks.GetKey(cspKey.SKI())
	assert.NoError(t, err)

	assert.Equal(t, cspKey, key)
}

func TestReadOnly(t *testing.T) {
	t.Parallel()
	ks := NewInMemoryKeyStore()
	readonly := ks.ReadOnly()
	assert.Equal(t, false, readonly)
}

func TestStoreExisting(t *testing.T) {
	t.Parallel()

	ks := NewInMemoryKeyStore()

//为要查找的密钥库生成密钥
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)
	cspKey := &ecdsaPrivateKey{privKey}

//存储密钥
	err = ks.StoreKey(cspKey)
	assert.NoError(t, err)

//再次存储密钥
	err = ks.StoreKey(cspKey)
	assert.EqualError(t, err, fmt.Sprintf("ski %x already exists in the keystore", cspKey.SKI()))
}
