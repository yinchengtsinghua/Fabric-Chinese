
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

无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/

package sw

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/bccsp/utils"
	"github.com/stretchr/testify/assert"
)

func TestInvalidStoreKey(t *testing.T) {
	t.Parallel()

	tempDir, err := ioutil.TempDir("", "bccspks")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ks, err := NewFileBasedKeyStore(nil, filepath.Join(tempDir, "bccspks"), false)
	if err != nil {
		fmt.Printf("Failed initiliazing KeyStore [%s]", err)
		os.Exit(-1)
	}

	err = ks.StoreKey(nil)
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}

	err = ks.StoreKey(&ecdsaPrivateKey{nil})
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}

	err = ks.StoreKey(&ecdsaPublicKey{nil})
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}

	err = ks.StoreKey(&rsaPublicKey{nil})
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}

	err = ks.StoreKey(&rsaPrivateKey{nil})
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}

	err = ks.StoreKey(&aesPrivateKey{nil, false})
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}

	err = ks.StoreKey(&aesPrivateKey{nil, true})
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}
}

func TestBigKeyFile(t *testing.T) {
	ksPath, err := ioutil.TempDir("", "bccspks")
	assert.NoError(t, err)
	defer os.RemoveAll(ksPath)

	ks, err := NewFileBasedKeyStore(nil, ksPath, false)
	assert.NoError(t, err)

//为要查找的密钥库生成密钥
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	cspKey := &ecdsaPrivateKey{privKey}
	ski := cspKey.SKI()
	rawKey, err := utils.PrivateKeyToPEM(privKey, nil)
	assert.NoError(t, err)

//大型填充数组，其中一些值PEM解析器将不支持
	bigBuff := make([]byte, (1 << 17))
	for i := range bigBuff {
		bigBuff[i] = '\n'
	}
	copy(bigBuff, rawKey)

//大于64K，所以文件总大小太大
	ioutil.WriteFile(filepath.Join(ksPath, "bigfile.pem"), bigBuff, 0666)

	_, err = ks.GetKey(ski)
	assert.Error(t, err)
	expected := fmt.Sprintf("Key with SKI %s not found in %s", hex.EncodeToString(ski), ksPath)
	assert.EqualError(t, err, expected)

//1K，以便找到钥匙
	ioutil.WriteFile(filepath.Join(ksPath, "smallerfile.pem"), bigBuff[0:1<<10], 0666)

	_, err = ks.GetKey(ski)
	assert.NoError(t, err)
}

func TestReInitKeyStore(t *testing.T) {
	ksPath, err := ioutil.TempDir("", "bccspks")
	assert.NoError(t, err)
	defer os.RemoveAll(ksPath)

	ks, err := NewFileBasedKeyStore(nil, ksPath, false)
	assert.NoError(t, err)
	fbKs, isFileBased := ks.(*fileBasedKeyStore)
	assert.True(t, isFileBased)
	err = fbKs.Init(nil, ksPath, false)
	assert.EqualError(t, err, "KeyStore already initilized.")
}
