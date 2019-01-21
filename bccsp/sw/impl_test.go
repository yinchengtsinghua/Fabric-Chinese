
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
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"hash"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/signer"
	"github.com/hyperledger/fabric/bccsp/sw/mocks"
	"github.com/hyperledger/fabric/bccsp/utils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"
)

var (
	currentTestConfig testConfig
	tempDir           string
)

type testConfig struct {
	securityLevel int
	hashFamily    string
}

func (tc testConfig) Provider(t *testing.T) (bccsp.BCCSP, bccsp.KeyStore, func()) {
	td, err := ioutil.TempDir(tempDir, "test")
	assert.NoError(t, err)
	ks, err := NewFileBasedKeyStore(nil, td, false)
	assert.NoError(t, err)
	p, err := NewWithParams(tc.securityLevel, tc.hashFamily, ks)
	assert.NoError(t, err)
	return p, ks, func() { os.RemoveAll(td) }
}

func TestMain(m *testing.M) {
	tests := []testConfig{
		{256, "SHA2"},
		{256, "SHA3"},
		{384, "SHA2"},
		{384, "SHA3"},
	}

	var err error
	tempDir, err = ioutil.TempDir("", "bccsp-sw")
	if err != nil {
		fmt.Printf("Failed to create temporary directory: %s\n\n", err)
		os.Exit(-1)
	}
	defer os.RemoveAll(tempDir)

	for _, config := range tests {
		currentTestConfig = config
		ret := m.Run()
		if ret != 0 {
			fmt.Printf("Failed testing at [%d, %s]", config.securityLevel, config.hashFamily)
			os.Exit(-1)
		}
	}
	os.Exit(0)
}

func TestInvalidNewParameter(t *testing.T) {
	t.Parallel()
	_, ks, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	r, err := NewWithParams(0, "SHA2", ks)
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}
	if r != nil {
		t.Fatal("Return value should be equal to nil in this case")
	}

	r, err = NewWithParams(256, "SHA8", ks)
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}
	if r != nil {
		t.Fatal("Return value should be equal to nil in this case")
	}

	r, err = NewWithParams(256, "SHA2", nil)
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}
	if r != nil {
		t.Fatal("Return value should be equal to nil in this case")
	}

	r, err = NewWithParams(0, "SHA3", nil)
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}
	if r != nil {
		t.Fatal("Return value should be equal to nil in this case")
	}

	r, err = NewDefaultSecurityLevel("")
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}
	if r != nil {
		t.Fatal("Return value should be equal to nil in this case")
	}
}

func TestInvalidSKI(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.GetKey(nil)
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}
	if k != nil {
		t.Fatal("Return value should be equal to nil in this case")
	}

	k, err = provider.GetKey([]byte{0, 1, 2, 3, 4, 5, 6})
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}
	if k != nil {
		t.Fatal("Return value should be equal to nil in this case")
	}
}

func TestKeyGenECDSAOpts(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

//曲线P256
	k, err := provider.KeyGen(&bccsp.ECDSAP256KeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA P256 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating ECDSA P256 key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating ECDSA P256 key. Key should be private")
	}
	if k.Symmetric() {
		t.Fatal("Failed generating ECDSA P256 key. Key should be asymmetric")
	}

	ecdsaKey := k.(*ecdsaPrivateKey).privKey
	if !elliptic.P256().IsOnCurve(ecdsaKey.X, ecdsaKey.Y) {
		t.Fatal("P256 generated key in invalid. The public key must be on the P256 curve.")
	}
	if elliptic.P256() != ecdsaKey.Curve {
		t.Fatal("P256 generated key in invalid. The curve must be P256.")
	}
	if ecdsaKey.D.Cmp(big.NewInt(0)) == 0 {
		t.Fatal("P256 generated key in invalid. Private key must be different from 0.")
	}

//曲线P38
	k, err = provider.KeyGen(&bccsp.ECDSAP384KeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA P384 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating ECDSA P384 key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating ECDSA P384 key. Key should be private")
	}
	if k.Symmetric() {
		t.Fatal("Failed generating ECDSA P384 key. Key should be asymmetric")
	}

	ecdsaKey = k.(*ecdsaPrivateKey).privKey
	if !elliptic.P384().IsOnCurve(ecdsaKey.X, ecdsaKey.Y) {
		t.Fatal("P256 generated key in invalid. The public key must be on the P384 curve.")
	}
	if elliptic.P384() != ecdsaKey.Curve {
		t.Fatal("P256 generated key in invalid. The curve must be P384.")
	}
	if ecdsaKey.D.Cmp(big.NewInt(0)) == 0 {
		t.Fatal("P256 generated key in invalid. Private key must be different from 0.")
	}

}

func TestKeyGenRSAOpts(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

//一千零二十四
	k, err := provider.KeyGen(&bccsp.RSA1024KeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA 1024 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating RSA 1024 key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating RSA 1024 key. Key should be private")
	}
	if k.Symmetric() {
		t.Fatal("Failed generating RSA 1024 key. Key should be asymmetric")
	}

	rsaKey := k.(*rsaPrivateKey).privKey
	if rsaKey.N.BitLen() != 1024 {
		t.Fatal("1024 RSA generated key in invalid. Modulus be of length 1024.")
	}
	if rsaKey.D.Cmp(big.NewInt(0)) == 0 {
		t.Fatal("1024 RSA generated key in invalid. Private key must be different from 0.")
	}
	if rsaKey.E < 3 {
		t.Fatal("1024 RSA generated key in invalid. Private key must be different from 0.")
	}

//二千零四十八
	k, err = provider.KeyGen(&bccsp.RSA2048KeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA 2048 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating RSA 2048 key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating RSA 2048 key. Key should be private")
	}
	if k.Symmetric() {
		t.Fatal("Failed generating RSA 2048 key. Key should be asymmetric")
	}

	rsaKey = k.(*rsaPrivateKey).privKey
	if rsaKey.N.BitLen() != 2048 {
		t.Fatal("2048 RSA generated key in invalid. Modulus be of length 2048.")
	}
	if rsaKey.D.Cmp(big.NewInt(0)) == 0 {
		t.Fatal("2048 RSA generated key in invalid. Private key must be different from 0.")
	}
	if rsaKey.E < 3 {
		t.Fatal("2048 RSA generated key in invalid. Private key must be different from 0.")
	}

 /*
  //跳过这些测试，因为它们运行时间太长。
  / / 3072
  k，err=provider.keygen（&bccsp.rsa3072keygenopts临时：假）
  如果犯错！= nIL{
   t.fatalf（“生成RSA 3072密钥失败[%s]”，错误）
  }
  如果k=＝nI{
   t.fatal（“生成RSA 3072密钥失败。键必须不同于零”）。
  }
  如果！私有（）
   t.fatal（“生成RSA 3072密钥失败。密钥应为私有）
  }
  如果k.对称（）
   t.fatal（“生成RSA 3072密钥失败。密钥应为非对称的“）
  }

  rsakey=k.（*rsaprivatekey）.privkey
  如果rsakey.n.bitlen（）！= 3072 {
   t.fatal（“3072 RSA生成的密钥无效。模量为3072。”）
  }
  如果rsakey.d.cmp（big.newint（0））==0
   t.fatal（“3072 RSA生成的密钥无效。私钥必须与0不同。“）
  }
  如果rsakey.e<3
   t.fatal（“3072 RSA生成的密钥无效。私钥必须与0不同。“）
  }

  / / 4096
  k，err=provider.keygen（&bccsp.rsa4096keygenopts临时：假）
  如果犯错！= nIL{
   t.fatalf（“生成RSA 4096密钥失败[%s]”，错误）
  }
  如果k=＝nI{
   t.fatal（“生成RSA 4096密钥失败。键必须不同于零”）。
  }
  如果！私有（）
   t.fatal（“生成RSA 4096密钥失败。密钥应为私有）
  }
  如果k.对称（）
   t.fatal（“生成RSA 4096密钥失败。密钥应为非对称的“）
  }

  rsakey=k.（*rsaprivatekey）.privkey
  如果rsakey.n.bitlen（）！= 4096 {
   t.fatal（“4096 RSA生成的密钥无效。模量为长度4096。”）
  }
  如果rsakey.d.cmp（big.newint（0））==0
   t.fatal（“4096 RSA生成的密钥无效。私钥必须与0不同。“）
  }
  如果rsakey.e<3
   t.fatal（“4096 RSA生成的密钥无效。私钥必须与0不同。“）
  }
 **/

}

func TestKeyGenAESOpts(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

//俄歇电子能谱128
	k, err := provider.KeyGen(&bccsp.AES128KeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating AES 128 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating AES 128 key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating AES 128 key. Key should be private")
	}
	if !k.Symmetric() {
		t.Fatal("Failed generating AES 128 key. Key should be symmetric")
	}

	aesKey := k.(*aesPrivateKey).privKey
	if len(aesKey) != 16 {
		t.Fatal("AES Key generated key in invalid. The key must have length 16.")
	}

//俄歇电子能谱192
	k, err = provider.KeyGen(&bccsp.AES192KeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating AES 192 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating AES 192 key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating AES 192 key. Key should be private")
	}
	if !k.Symmetric() {
		t.Fatal("Failed generating AES 192 key. Key should be symmetric")
	}

	aesKey = k.(*aesPrivateKey).privKey
	if len(aesKey) != 24 {
		t.Fatal("AES Key generated key in invalid. The key must have length 16.")
	}

//俄歇电子能谱256
	k, err = provider.KeyGen(&bccsp.AES256KeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating AES 256 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating AES 256 key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating AES 256 key. Key should be private")
	}
	if !k.Symmetric() {
		t.Fatal("Failed generating AES 256 key. Key should be symmetric")
	}

	aesKey = k.(*aesPrivateKey).privKey
	if len(aesKey) != 32 {
		t.Fatal("AES Key generated key in invalid. The key must have length 16.")
	}
}

func TestECDSAKeyGenEphemeral(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: true})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating ECDSA key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating ECDSA key. Key should be private")
	}
	if k.Symmetric() {
		t.Fatal("Failed generating ECDSA key. Key should be asymmetric")
	}
	raw, err := k.Bytes()
	if err == nil {
		t.Fatal("Failed marshalling to bytes. Marshalling must fail.")
	}
	if len(raw) != 0 {
		t.Fatal("Failed marshalling to bytes. Output should be 0 bytes")
	}
	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting corresponding public key [%s]", err)
	}
	if pk == nil {
		t.Fatal("Public key must be different from nil.")
	}
}

func TestECDSAPrivateKeySKI(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

	ski := k.SKI()
	if len(ski) == 0 {
		t.Fatal("SKI not valid. Zero length.")
	}
}

func TestECDSAKeyGenNonEphemeral(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating ECDSA key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating ECDSA key. Key should be private")
	}
	if k.Symmetric() {
		t.Fatal("Failed generating ECDSA key. Key should be asymmetric")
	}
}

func TestECDSAGetKeyBySKI(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

	k2, err := provider.GetKey(k.SKI())
	if err != nil {
		t.Fatalf("Failed getting ECDSA key [%s]", err)
	}
	if k2 == nil {
		t.Fatal("Failed getting ECDSA key. Key must be different from nil")
	}
	if !k2.Private() {
		t.Fatal("Failed getting ECDSA key. Key should be private")
	}
	if k2.Symmetric() {
		t.Fatal("Failed getting ECDSA key. Key should be asymmetric")
	}

//检查滑雪板是否相同
	if !bytes.Equal(k.SKI(), k2.SKI()) {
		t.Fatalf("SKIs are different [%x]!=[%x]", k.SKI(), k2.SKI())
	}
}

func TestECDSAPublicKeyFromPrivateKey(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting public key from private ECDSA key [%s]", err)
	}
	if pk == nil {
		t.Fatal("Failed getting public key from private ECDSA key. Key must be different from nil")
	}
	if pk.Private() {
		t.Fatal("Failed generating ECDSA key. Key should be public")
	}
	if pk.Symmetric() {
		t.Fatal("Failed generating ECDSA key. Key should be asymmetric")
	}
}

func TestECDSAPublicKeyBytes(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting public key from private ECDSA key [%s]", err)
	}

	raw, err := pk.Bytes()
	if err != nil {
		t.Fatalf("Failed marshalling ECDSA public key [%s]", err)
	}
	if len(raw) == 0 {
		t.Fatal("Failed marshalling ECDSA public key. Zero length")
	}
}

func TestECDSAPublicKeySKI(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting public key from private ECDSA key [%s]", err)
	}

	ski := pk.SKI()
	if len(ski) == 0 {
		t.Fatal("SKI not valid. Zero length.")
	}
}

func TestECDSAKeyReRand(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed re-randomizing ECDSA key. Re-randomized Key must be different from nil")
	}

	reRandomizedKey, err := provider.KeyDeriv(k, &bccsp.ECDSAReRandKeyOpts{Temporary: false, Expansion: []byte{1}})
	if err != nil {
		t.Fatalf("Failed re-randomizing ECDSA key [%s]", err)
	}
	if !reRandomizedKey.Private() {
		t.Fatal("Failed re-randomizing ECDSA key. Re-randomized Key should be private")
	}
	if reRandomizedKey.Symmetric() {
		t.Fatal("Failed re-randomizing ECDSA key. Re-randomized Key should be asymmetric")
	}

	k2, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting public ECDSA key from private [%s]", err)
	}
	if k2 == nil {
		t.Fatal("Failed re-randomizing ECDSA key. Re-randomized Key must be different from nil")
	}

	reRandomizedKey2, err := provider.KeyDeriv(k2, &bccsp.ECDSAReRandKeyOpts{Temporary: false, Expansion: []byte{1}})
	if err != nil {
		t.Fatalf("Failed re-randomizing ECDSA key [%s]", err)
	}

	if reRandomizedKey2.Private() {
		t.Fatal("Re-randomized public Key must remain public")
	}
	if reRandomizedKey2.Symmetric() {
		t.Fatal("Re-randomized ECDSA asymmetric key must remain asymmetric")
	}

	if false == bytes.Equal(reRandomizedKey.SKI(), reRandomizedKey2.SKI()) {
		t.Fatal("Re-randomized ECDSA Private- or Public-Keys must end up having the same SKI")
	}
}

func TestECDSASign(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(k, digest, nil)
	if err != nil {
		t.Fatalf("Failed generating ECDSA signature [%s]", err)
	}
	if len(signature) == 0 {
		t.Fatal("Failed generating ECDSA key. Signature must be different from nil")
	}
}

func TestECDSAVerify(t *testing.T) {
	t.Parallel()
	provider, ks, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(k, digest, nil)
	if err != nil {
		t.Fatalf("Failed generating ECDSA signature [%s]", err)
	}

	valid, err := provider.Verify(k, signature, digest, nil)
	if err != nil {
		t.Fatalf("Failed verifying ECDSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying ECDSA signature. Signature not valid.")
	}

	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting corresponding public key [%s]", err)
	}

	valid, err = provider.Verify(pk, signature, digest, nil)
	if err != nil {
		t.Fatalf("Failed verifying ECDSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying ECDSA signature. Signature not valid.")
	}

//存储公钥
	err = ks.StoreKey(pk)
	if err != nil {
		t.Fatalf("Failed storing corresponding public key [%s]", err)
	}

	pk2, err := ks.GetKey(pk.SKI())
	if err != nil {
		t.Fatalf("Failed retrieving corresponding public key [%s]", err)
	}

	valid, err = provider.Verify(pk2, signature, digest, nil)
	if err != nil {
		t.Fatalf("Failed verifying ECDSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying ECDSA signature. Signature not valid.")
	}
}

func TestECDSAKeyDeriv(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

	reRandomizedKey, err := provider.KeyDeriv(k, &bccsp.ECDSAReRandKeyOpts{Temporary: false, Expansion: []byte{1}})
	if err != nil {
		t.Fatalf("Failed re-randomizing ECDSA key [%s]", err)
	}

	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(reRandomizedKey, digest, nil)
	if err != nil {
		t.Fatalf("Failed generating ECDSA signature [%s]", err)
	}

	valid, err := provider.Verify(reRandomizedKey, signature, digest, nil)
	if err != nil {
		t.Fatalf("Failed verifying ECDSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying ECDSA signature. Signature not valid.")
	}
}

func TestECDSAKeyImportFromExportedKey(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

//生成ECDSA密钥
	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

//导出公钥
	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting ECDSA public key [%s]", err)
	}

	pkRaw, err := pk.Bytes()
	if err != nil {
		t.Fatalf("Failed getting ECDSA raw public key [%s]", err)
	}

//导入导出的公钥
	pk2, err := provider.KeyImport(pkRaw, &bccsp.ECDSAPKIXPublicKeyImportOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed importing ECDSA public key [%s]", err)
	}
	if pk2 == nil {
		t.Fatal("Failed importing ECDSA public key. Return BCCSP key cannot be nil.")
	}

//用导入的公钥签名和验证
	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(k, digest, nil)
	if err != nil {
		t.Fatalf("Failed generating ECDSA signature [%s]", err)
	}

	valid, err := provider.Verify(pk2, signature, digest, nil)
	if err != nil {
		t.Fatalf("Failed verifying ECDSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying ECDSA signature. Signature not valid.")
	}
}

func TestECDSAKeyImportFromECDSAPublicKey(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

//生成ECDSA密钥
	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

//导出公钥
	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting ECDSA public key [%s]", err)
	}

	pkRaw, err := pk.Bytes()
	if err != nil {
		t.Fatalf("Failed getting ECDSA raw public key [%s]", err)
	}

	pub, err := utils.DERToPublicKey(pkRaw)
	if err != nil {
		t.Fatalf("Failed converting raw to ecdsa.PublicKey [%s]", err)
	}

//导入ecdsa.publickey
	pk2, err := provider.KeyImport(pub, &bccsp.ECDSAGoPublicKeyImportOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed importing ECDSA public key [%s]", err)
	}
	if pk2 == nil {
		t.Fatal("Failed importing ECDSA public key. Return BCCSP key cannot be nil.")
	}

//用导入的公钥签名和验证
	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(k, digest, nil)
	if err != nil {
		t.Fatalf("Failed generating ECDSA signature [%s]", err)
	}

	valid, err := provider.Verify(pk2, signature, digest, nil)
	if err != nil {
		t.Fatalf("Failed verifying ECDSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying ECDSA signature. Signature not valid.")
	}
}

func TestECDSAKeyImportFromECDSAPrivateKey(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

//生成ECDSA密钥，默认为p256
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

//导入ecdsa.privatekey
	priv, err := utils.PrivateKeyToDER(key)
	if err != nil {
		t.Fatalf("Failed converting raw to ecdsa.PrivateKey [%s]", err)
	}

	sk, err := provider.KeyImport(priv, &bccsp.ECDSAPrivateKeyImportOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed importing ECDSA private key [%s]", err)
	}
	if sk == nil {
		t.Fatal("Failed importing ECDSA private key. Return BCCSP key cannot be nil.")
	}

//导入ecdsa.publickey
	pub, err := utils.PublicKeyToDER(&key.PublicKey)
	if err != nil {
		t.Fatalf("Failed converting raw to ecdsa.PublicKey [%s]", err)
	}

	pk, err := provider.KeyImport(pub, &bccsp.ECDSAPKIXPublicKeyImportOpts{Temporary: false})

	if err != nil {
		t.Fatalf("Failed importing ECDSA public key [%s]", err)
	}
	if pk == nil {
		t.Fatal("Failed importing ECDSA public key. Return BCCSP key cannot be nil.")
	}

//用导入的公钥签名和验证
	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(sk, digest, nil)
	if err != nil {
		t.Fatalf("Failed generating ECDSA signature [%s]", err)
	}

	valid, err := provider.Verify(pk, signature, digest, nil)
	if err != nil {
		t.Fatalf("Failed verifying ECDSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying ECDSA signature. Signature not valid.")
	}
}

func TestKeyImportFromX509ECDSAPublicKey(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

//生成ECDSA密钥
	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

//生成自签名证书
	testExtKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
	testUnknownExtKeyUsage := []asn1.ObjectIdentifier{[]int{1, 2, 3}, []int{2, 59, 1}}
	extraExtensionData := []byte("extra extension")
	commonName := "test.example.com"
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Σ Acme Co"},
			Country:      []string{"US"},
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  []int{2, 5, 4, 42},
					Value: "Gopher",
				},
//这应该覆盖国家，如上所述。
				{
					Type:  []int{2, 5, 4, 6},
					Value: "NL",
				},
			},
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(1 * time.Hour),

		SignatureAlgorithm: x509.ECDSAWithSHA256,

		SubjectKeyId: []byte{1, 2, 3, 4},
		KeyUsage:     x509.KeyUsageCertSign,

		ExtKeyUsage:        testExtKeyUsage,
		UnknownExtKeyUsage: testUnknownExtKeyUsage,

		BasicConstraintsValid: true,
		IsCA:                  true,

OCSPServer:            []string{"http://occurrentbccsp.example.com“，
IssuingCertificateURL: []string{"http://crt.example.com/ca1.crt“，

		DNSNames:       []string{"test.example.com"},
		EmailAddresses: []string{"gopher@golang.org"},
		IPAddresses:    []net.IP{net.IPv4(127, 0, 0, 1).To4(), net.ParseIP("2001:4860:0:2001::68")},

		PolicyIdentifiers:   []asn1.ObjectIdentifier{[]int{1, 2, 3}},
		PermittedDNSDomains: []string{".example.com", "example.com"},

CRLDistributionPoints: []string{"http://crl1.example.com/ca1.crl“，”http://crl2.example.com/ca1.crl”，

		ExtraExtensions: []pkix.Extension{
			{
				Id:    []int{1, 2, 3, 4},
				Value: extraExtensionData,
			},
		},
	}

	cryptoSigner, err := signer.New(provider, k)
	if err != nil {
		t.Fatalf("Failed initializing CyrptoSigner [%s]", err)
	}

//导出公钥
	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting ECDSA public key [%s]", err)
	}

	pkRaw, err := pk.Bytes()
	if err != nil {
		t.Fatalf("Failed getting ECDSA raw public key [%s]", err)
	}

	pub, err := utils.DERToPublicKey(pkRaw)
	if err != nil {
		t.Fatalf("Failed converting raw to ECDSA.PublicKey [%s]", err)
	}

	certRaw, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, cryptoSigner)
	if err != nil {
		t.Fatalf("Failed generating self-signed certificate [%s]", err)
	}

	cert, err := utils.DERToX509Certificate(certRaw)
	if err != nil {
		t.Fatalf("Failed generating X509 certificate object from raw [%s]", err)
	}

//导入证书的公钥
	pk2, err := provider.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: false})

	if err != nil {
		t.Fatalf("Failed importing ECDSA public key [%s]", err)
	}
	if pk2 == nil {
		t.Fatal("Failed importing ECDSA public key. Return BCCSP key cannot be nil.")
	}

//用导入的公钥签名和验证
	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(k, digest, nil)
	if err != nil {
		t.Fatalf("Failed generating ECDSA signature [%s]", err)
	}

	valid, err := provider.Verify(pk2, signature, digest, nil)
	if err != nil {
		t.Fatalf("Failed verifying ECDSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying ECDSA signature. Signature not valid.")
	}
}

func TestECDSASignatureEncoding(t *testing.T) {
	t.Parallel()

	v := []byte{0x30, 0x07, 0x02, 0x01, 0x8F, 0x02, 0x02, 0xff, 0xf1}
	_, err := asn1.Unmarshal(v, &utils.ECDSASignature{})
	if err == nil {
		t.Fatalf("Unmarshalling should fail for [% x]", v)
	}
	t.Logf("Unmarshalling correctly failed for [% x] [%s]", v, err)

	v = []byte{0x30, 0x07, 0x02, 0x01, 0x8F, 0x02, 0x02, 0x00, 0x01}
	_, err = asn1.Unmarshal(v, &utils.ECDSASignature{})
	if err == nil {
		t.Fatalf("Unmarshalling should fail for [% x]", v)
	}
	t.Logf("Unmarshalling correctly failed for [% x] [%s]", v, err)

	v = []byte{0x30, 0x07, 0x02, 0x01, 0x8F, 0x02, 0x81, 0x01, 0x01}
	_, err = asn1.Unmarshal(v, &utils.ECDSASignature{})
	if err == nil {
		t.Fatalf("Unmarshalling should fail for [% x]", v)
	}
	t.Logf("Unmarshalling correctly failed for [% x] [%s]", v, err)

	v = []byte{0x30, 0x07, 0x02, 0x01, 0x8F, 0x02, 0x81, 0x01, 0x8F}
	_, err = asn1.Unmarshal(v, &utils.ECDSASignature{})
	if err == nil {
		t.Fatalf("Unmarshalling should fail for [% x]", v)
	}
	t.Logf("Unmarshalling correctly failed for [% x] [%s]", v, err)

	v = []byte{0x30, 0x0A, 0x02, 0x01, 0x8F, 0x02, 0x05, 0x00, 0x00, 0x00, 0x00, 0x8F}
	_, err = asn1.Unmarshal(v, &utils.ECDSASignature{})
	if err == nil {
		t.Fatalf("Unmarshalling should fail for [% x]", v)
	}
	t.Logf("Unmarshalling correctly failed for [% x] [%s]", v, err)

}

func TestECDSALowS(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

//确保生成低S签名
	k, err := provider.KeyGen(&bccsp.ECDSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(k, digest, nil)
	if err != nil {
		t.Fatalf("Failed generating ECDSA signature [%s]", err)
	}

	_, S, err := utils.UnmarshalECDSASignature(signature)
	if err != nil {
		t.Fatalf("Failed unmarshalling signature [%s]", err)
	}

	if S.Cmp(utils.GetCurveHalfOrdersAt(k.(*ecdsaPrivateKey).privKey.Curve)) >= 0 {
		t.Fatal("Invalid signature. It must have low-S")
	}

	valid, err := provider.Verify(k, signature, digest, nil)
	if err != nil {
		t.Fatalf("Failed verifying ECDSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying ECDSA signature. Signature not valid.")
	}

//确保拒绝高S签名。
	var R *big.Int
	for {
		R, S, err = ecdsa.Sign(rand.Reader, k.(*ecdsaPrivateKey).privKey, digest)
		if err != nil {
			t.Fatalf("Failed generating signature [%s]", err)
		}

		if S.Cmp(utils.GetCurveHalfOrdersAt(k.(*ecdsaPrivateKey).privKey.Curve)) > 0 {
			break
		}
	}

	sig, err := utils.MarshalECDSASignature(R, S)
	if err != nil {
		t.Fatalf("Failing unmarshalling signature [%s]", err)
	}

	valid, err = provider.Verify(k, sig, digest, nil)
	if err == nil {
		t.Fatal("Failed verifying ECDSA signature. It must fail for a signature with high-S")
	}
	if valid {
		t.Fatal("Failed verifying ECDSA signature. It must fail for a signature with high-S")
	}
}

func TestAESKeyGen(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.AESKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating AES_256 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating AES_256 key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating AES_256 key. Key should be private")
	}
	if !k.Symmetric() {
		t.Fatal("Failed generating AES_256 key. Key should be symmetric")
	}

	pk, err := k.PublicKey()
	if err == nil {
		t.Fatal("Error should be different from nil in this case")
	}
	if pk != nil {
		t.Fatal("Return value should be equal to nil in this case")
	}
}

func TestAESEncrypt(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.AESKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating AES_256 key [%s]", err)
	}

	ct, err := provider.Encrypt(k, []byte("Hello World"), &bccsp.AESCBCPKCS7ModeOpts{})
	if err != nil {
		t.Fatalf("Failed encrypting [%s]", err)
	}
	if len(ct) == 0 {
		t.Fatal("Failed encrypting. Nil ciphertext")
	}
}

func TestAESDecrypt(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.AESKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating AES_256 key [%s]", err)
	}

	msg := []byte("Hello World")

	ct, err := provider.Encrypt(k, msg, &bccsp.AESCBCPKCS7ModeOpts{})
	if err != nil {
		t.Fatalf("Failed encrypting [%s]", err)
	}

	pt, err := provider.Decrypt(k, ct, bccsp.AESCBCPKCS7ModeOpts{})
	if err != nil {
		t.Fatalf("Failed decrypting [%s]", err)
	}
	if len(ct) == 0 {
		t.Fatal("Failed decrypting. Nil plaintext")
	}

	if !bytes.Equal(msg, pt) {
		t.Fatalf("Failed decrypting. Decrypted plaintext is different from the original. [%x][%x]", msg, pt)
	}
}

func TestHMACTruncated256KeyDerivOverAES256Key(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.AESKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating AES_256 key [%s]", err)
	}

	hmcaedKey, err := provider.KeyDeriv(k, &bccsp.HMACTruncated256AESDeriveKeyOpts{Temporary: false, Arg: []byte{1}})
	if err != nil {
		t.Fatalf("Failed HMACing AES_256 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed HMACing AES_256 key. HMACed Key must be different from nil")
	}
	if !hmcaedKey.Private() {
		t.Fatal("Failed HMACing AES_256 key. HMACed Key should be private")
	}
	if !hmcaedKey.Symmetric() {
		t.Fatal("Failed HMACing AES_256 key. HMACed Key should be asymmetric")
	}
	raw, err := hmcaedKey.Bytes()
	if err == nil {
		t.Fatal("Failed marshalling to bytes. Operation must be forbidden")
	}
	if len(raw) != 0 {
		t.Fatal("Failed marshalling to bytes. Operation must return 0 bytes")
	}

	msg := []byte("Hello World")

	ct, err := provider.Encrypt(hmcaedKey, msg, &bccsp.AESCBCPKCS7ModeOpts{})
	if err != nil {
		t.Fatalf("Failed encrypting [%s]", err)
	}

	pt, err := provider.Decrypt(hmcaedKey, ct, bccsp.AESCBCPKCS7ModeOpts{})
	if err != nil {
		t.Fatalf("Failed decrypting [%s]", err)
	}
	if len(ct) == 0 {
		t.Fatal("Failed decrypting. Nil plaintext")
	}

	if !bytes.Equal(msg, pt) {
		t.Fatalf("Failed decrypting. Decrypted plaintext is different from the original. [%x][%x]", msg, pt)
	}
}

func TestHMACKeyDerivOverAES256Key(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.AESKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating AES_256 key [%s]", err)
	}

	hmcaedKey, err := provider.KeyDeriv(k, &bccsp.HMACDeriveKeyOpts{Temporary: false, Arg: []byte{1}})

	if err != nil {
		t.Fatalf("Failed HMACing AES_256 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed HMACing AES_256 key. HMACed Key must be different from nil")
	}
	if !hmcaedKey.Private() {
		t.Fatal("Failed HMACing AES_256 key. HMACed Key should be private")
	}
	if !hmcaedKey.Symmetric() {
		t.Fatal("Failed HMACing AES_256 key. HMACed Key should be asymmetric")
	}
	raw, err := hmcaedKey.Bytes()
	if err != nil {
		t.Fatalf("Failed marshalling to bytes [%s]", err)
	}
	if len(raw) == 0 {
		t.Fatal("Failed marshalling to bytes. 0 bytes")
	}
}

func TestAES256KeyImport(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	raw, err := GetRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed generating AES key [%s]", err)
	}

	k, err := provider.KeyImport(raw, &bccsp.AES256ImportKeyOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed importing AES_256 key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed importing AES_256 key. Imported Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed HMACing AES_256 key. Imported Key should be private")
	}
	if !k.Symmetric() {
		t.Fatal("Failed HMACing AES_256 key. Imported Key should be asymmetric")
	}
	raw, err = k.Bytes()
	if err == nil {
		t.Fatal("Failed marshalling to bytes. Marshalling must fail.")
	}
	if len(raw) != 0 {
		t.Fatal("Failed marshalling to bytes. Output should be 0 bytes")
	}

	msg := []byte("Hello World")

	ct, err := provider.Encrypt(k, msg, &bccsp.AESCBCPKCS7ModeOpts{})
	if err != nil {
		t.Fatalf("Failed encrypting [%s]", err)
	}

	pt, err := provider.Decrypt(k, ct, bccsp.AESCBCPKCS7ModeOpts{})
	if err != nil {
		t.Fatalf("Failed decrypting [%s]", err)
	}
	if len(ct) == 0 {
		t.Fatal("Failed decrypting. Nil plaintext")
	}

	if !bytes.Equal(msg, pt) {
		t.Fatalf("Failed decrypting. Decrypted plaintext is different from the original. [%x][%x]", msg, pt)
	}
}

func TestAES256KeyImportBadPaths(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	_, err := provider.KeyImport(nil, &bccsp.AES256ImportKeyOpts{Temporary: false})
	if err == nil {
		t.Fatal("Failed importing key. Must fail on importing nil key")
	}

	_, err = provider.KeyImport([]byte{1}, &bccsp.AES256ImportKeyOpts{Temporary: false})
	if err == nil {
		t.Fatal("Failed importing key. Must fail on importing a key with an invalid length")
	}
}

func TestAES256KeyGenSKI(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.AESKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating AES_256 key [%s]", err)
	}

	k2, err := provider.GetKey(k.SKI())
	if err != nil {
		t.Fatalf("Failed getting AES_256 key [%s]", err)
	}
	if k2 == nil {
		t.Fatal("Failed getting AES_256 key. Key must be different from nil")
	}
	if !k2.Private() {
		t.Fatal("Failed getting AES_256 key. Key should be private")
	}
	if !k2.Symmetric() {
		t.Fatal("Failed getting AES_256 key. Key should be symmetric")
	}

//检查滑雪板是否相同
	if !bytes.Equal(k.SKI(), k2.SKI()) {
		t.Fatalf("SKIs are different [%x]!=[%x]", k.SKI(), k2.SKI())
	}
}

func TestSHA(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	for i := 0; i < 100; i++ {
		b, err := GetRandomBytes(i)
		if err != nil {
			t.Fatalf("Failed getting random bytes [%s]", err)
		}

		h1, err := provider.Hash(b, &bccsp.SHAOpts{})
		if err != nil {
			t.Fatalf("Failed computing SHA [%s]", err)
		}

		var h hash.Hash
		switch currentTestConfig.hashFamily {
		case "SHA2":
			switch currentTestConfig.securityLevel {
			case 256:
				h = sha256.New()
			case 384:
				h = sha512.New384()
			default:
				t.Fatalf("Invalid security level [%d]", currentTestConfig.securityLevel)
			}
		case "SHA3":
			switch currentTestConfig.securityLevel {
			case 256:
				h = sha3.New256()
			case 384:
				h = sha3.New384()
			default:
				t.Fatalf("Invalid security level [%d]", currentTestConfig.securityLevel)
			}
		default:
			t.Fatalf("Invalid hash family [%s]", currentTestConfig.hashFamily)
		}

		h.Write(b)
		h2 := h.Sum(nil)
		if !bytes.Equal(h1, h2) {
			t.Fatalf("Discrempancy found in HASH result [%x], [%x]!=[%x]", b, h1, h2)
		}
	}
}

func TestRSAKeyGenEphemeral(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: true})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating RSA key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating RSA key. Key should be private")
	}
	if k.Symmetric() {
		t.Fatal("Failed generating RSA key. Key should be asymmetric")
	}

	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed generating RSA corresponding public key [%s]", err)
	}
	if pk == nil {
		t.Fatal("PK must be different from nil")
	}

	b, err := k.Bytes()
	if err == nil {
		t.Fatal("Secret keys cannot be exported. It must fail in this case")
	}
	if len(b) != 0 {
		t.Fatal("Secret keys cannot be exported. It must be nil")
	}
}

func TestRSAPublicKeyInvalidBytes(t *testing.T) {
	t.Parallel()

	rsaKey := &rsaPublicKey{nil}
	b, err := rsaKey.Bytes()
	if err == nil {
		t.Fatal("It must fail in this case")
	}
	if len(b) != 0 {
		t.Fatal("It must be nil")
	}
}

func TestRSAPrivateKeySKI(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}

	ski := k.SKI()
	if len(ski) == 0 {
		t.Fatal("SKI not valid. Zero length.")
	}
}

func TestRSAKeyGenNonEphemeral(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}
	if k == nil {
		t.Fatal("Failed generating RSA key. Key must be different from nil")
	}
	if !k.Private() {
		t.Fatal("Failed generating RSA key. Key should be private")
	}
	if k.Symmetric() {
		t.Fatal("Failed generating RSA key. Key should be asymmetric")
	}
}

func TestRSAGetKeyBySKI(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}

	k2, err := provider.GetKey(k.SKI())
	if err != nil {
		t.Fatalf("Failed getting RSA key [%s]", err)
	}
	if k2 == nil {
		t.Fatal("Failed getting RSA key. Key must be different from nil")
	}
	if !k2.Private() {
		t.Fatal("Failed getting RSA key. Key should be private")
	}
	if k2.Symmetric() {
		t.Fatal("Failed getting RSA key. Key should be asymmetric")
	}

//检查滑雪板是否相同
	if !bytes.Equal(k.SKI(), k2.SKI()) {
		t.Fatalf("SKIs are different [%x]!=[%x]", k.SKI(), k2.SKI())
	}
}

func TestRSAPublicKeyFromPrivateKey(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}

	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting public key from private RSA key [%s]", err)
	}
	if pk == nil {
		t.Fatal("Failed getting public key from private RSA key. Key must be different from nil")
	}
	if pk.Private() {
		t.Fatal("Failed generating RSA key. Key should be public")
	}
	if pk.Symmetric() {
		t.Fatal("Failed generating RSA key. Key should be asymmetric")
	}
}

func TestRSAPublicKeyBytes(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}

	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting public key from private RSA key [%s]", err)
	}

	raw, err := pk.Bytes()
	if err != nil {
		t.Fatalf("Failed marshalling RSA public key [%s]", err)
	}
	if len(raw) == 0 {
		t.Fatal("Failed marshalling RSA public key. Zero length")
	}
}

func TestRSAPublicKeySKI(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}

	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting public key from private RSA key [%s]", err)
	}

	ski := pk.SKI()
	if len(ski) == 0 {
		t.Fatal("SKI not valid. Zero length.")
	}
}

func TestRSASign(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}

	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(k, digest, &rsa.PSSOptions{SaltLength: 32, Hash: getCryptoHashIndex(t)})
	if err != nil {
		t.Fatalf("Failed generating RSA signature [%s]", err)
	}
	if len(signature) == 0 {
		t.Fatal("Failed generating RSA key. Signature must be different from nil")
	}
}

func TestRSAVerify(t *testing.T) {
	t.Parallel()
	provider, ks, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}

	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(k, digest, &rsa.PSSOptions{SaltLength: 32, Hash: getCryptoHashIndex(t)})
	if err != nil {
		t.Fatalf("Failed generating RSA signature [%s]", err)
	}

	valid, err := provider.Verify(k, signature, digest, &rsa.PSSOptions{SaltLength: 32, Hash: getCryptoHashIndex(t)})
	if err != nil {
		t.Fatalf("Failed verifying RSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying RSA signature. Signature not valid.")
	}

	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting corresponding public key [%s]", err)
	}

	valid, err = provider.Verify(pk, signature, digest, &rsa.PSSOptions{SaltLength: 32, Hash: getCryptoHashIndex(t)})
	if err != nil {
		t.Fatalf("Failed verifying RSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying RSA signature. Signature not valid.")
	}

//存储公钥
	err = ks.StoreKey(pk)
	if err != nil {
		t.Fatalf("Failed storing corresponding public key [%s]", err)
	}

	pk2, err := ks.GetKey(pk.SKI())
	if err != nil {
		t.Fatalf("Failed retrieving corresponding public key [%s]", err)
	}

	valid, err = provider.Verify(pk2, signature, digest, &rsa.PSSOptions{SaltLength: 32, Hash: getCryptoHashIndex(t)})
	if err != nil {
		t.Fatalf("Failed verifying RSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying RSA signature. Signature not valid.")
	}

}

func TestRSAKeyImportFromRSAPublicKey(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

//生成RSA密钥
	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}

//导出公钥
	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting RSA public key [%s]", err)
	}

	pkRaw, err := pk.Bytes()
	if err != nil {
		t.Fatalf("Failed getting RSA raw public key [%s]", err)
	}

	pub, err := utils.DERToPublicKey(pkRaw)
	if err != nil {
		t.Fatalf("Failed converting raw to RSA.PublicKey [%s]", err)
	}

//导入rsa.publickey
	pk2, err := provider.KeyImport(pub, &bccsp.RSAGoPublicKeyImportOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed importing RSA public key [%s]", err)
	}
	if pk2 == nil {
		t.Fatal("Failed importing RSA public key. Return BCCSP key cannot be nil.")
	}

//用导入的公钥签名和验证
	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(k, digest, &rsa.PSSOptions{SaltLength: 32, Hash: getCryptoHashIndex(t)})
	if err != nil {
		t.Fatalf("Failed generating RSA signature [%s]", err)
	}

	valid, err := provider.Verify(pk2, signature, digest, &rsa.PSSOptions{SaltLength: 32, Hash: getCryptoHashIndex(t)})
	if err != nil {
		t.Fatalf("Failed verifying RSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying RSA signature. Signature not valid.")
	}
}

func TestKeyImportFromX509RSAPublicKey(t *testing.T) {
	t.Parallel()
	provider, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

//生成RSA密钥
	k, err := provider.KeyGen(&bccsp.RSAKeyGenOpts{Temporary: false})
	if err != nil {
		t.Fatalf("Failed generating RSA key [%s]", err)
	}

//生成自签名证书
	testExtKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
	testUnknownExtKeyUsage := []asn1.ObjectIdentifier{[]int{1, 2, 3}, []int{2, 59, 1}}
	extraExtensionData := []byte("extra extension")
	commonName := "test.example.com"
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Σ Acme Co"},
			Country:      []string{"US"},
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  []int{2, 5, 4, 42},
					Value: "Gopher",
				},
//这应该覆盖国家，如上所述。
				{
					Type:  []int{2, 5, 4, 6},
					Value: "NL",
				},
			},
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(1 * time.Hour),

		SignatureAlgorithm: x509.SHA256WithRSA,

		SubjectKeyId: []byte{1, 2, 3, 4},
		KeyUsage:     x509.KeyUsageCertSign,

		ExtKeyUsage:        testExtKeyUsage,
		UnknownExtKeyUsage: testUnknownExtKeyUsage,

		BasicConstraintsValid: true,
		IsCA:                  true,

OCSPServer:            []string{"http://occurrentbccsp.example.com“，
IssuingCertificateURL: []string{"http://crt.example.com/ca1.crt“，

		DNSNames:       []string{"test.example.com"},
		EmailAddresses: []string{"gopher@golang.org"},
		IPAddresses:    []net.IP{net.IPv4(127, 0, 0, 1).To4(), net.ParseIP("2001:4860:0:2001::68")},

		PolicyIdentifiers:   []asn1.ObjectIdentifier{[]int{1, 2, 3}},
		PermittedDNSDomains: []string{".example.com", "example.com"},

CRLDistributionPoints: []string{"http://crl1.example.com/ca1.crl“，”http://crl2.example.com/ca1.crl”，

		ExtraExtensions: []pkix.Extension{
			{
				Id:    []int{1, 2, 3, 4},
				Value: extraExtensionData,
			},
		},
	}

	cryptoSigner, err := signer.New(provider, k)
	if err != nil {
		t.Fatalf("Failed initializing CyrptoSigner [%s]", err)
	}

//导出公钥
	pk, err := k.PublicKey()
	if err != nil {
		t.Fatalf("Failed getting RSA public key [%s]", err)
	}

	pkRaw, err := pk.Bytes()
	if err != nil {
		t.Fatalf("Failed getting RSA raw public key [%s]", err)
	}

	pub, err := utils.DERToPublicKey(pkRaw)
	if err != nil {
		t.Fatalf("Failed converting raw to RSA.PublicKey [%s]", err)
	}

	certRaw, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, cryptoSigner)
	if err != nil {
		t.Fatalf("Failed generating self-signed certificate [%s]", err)
	}

	cert, err := utils.DERToX509Certificate(certRaw)
	if err != nil {
		t.Fatalf("Failed generating X509 certificate object from raw [%s]", err)
	}

//导入证书的公钥
	pk2, err := provider.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: false})

	if err != nil {
		t.Fatalf("Failed importing RSA public key [%s]", err)
	}
	if pk2 == nil {
		t.Fatal("Failed importing RSA public key. Return BCCSP key cannot be nil.")
	}

//用导入的公钥签名和验证
	msg := []byte("Hello World")

	digest, err := provider.Hash(msg, &bccsp.SHAOpts{})
	if err != nil {
		t.Fatalf("Failed computing HASH [%s]", err)
	}

	signature, err := provider.Sign(k, digest, &rsa.PSSOptions{SaltLength: 32, Hash: getCryptoHashIndex(t)})
	if err != nil {
		t.Fatalf("Failed generating RSA signature [%s]", err)
	}

	valid, err := provider.Verify(pk2, signature, digest, &rsa.PSSOptions{SaltLength: 32, Hash: getCryptoHashIndex(t)})
	if err != nil {
		t.Fatalf("Failed verifying RSA signature [%s]", err)
	}
	if !valid {
		t.Fatal("Failed verifying RSA signature. Signature not valid.")
	}
}

func TestAddWrapper(t *testing.T) {
	t.Parallel()
	p, _, cleanup := currentTestConfig.Provider(t)
	defer cleanup()

	sw, ok := p.(*CSP)
	assert.True(t, ok)

	tester := func(o interface{}, getter func(t reflect.Type) (interface{}, bool)) {
		tt := reflect.TypeOf(o)
		err := sw.AddWrapper(tt, o)
		assert.NoError(t, err)
		o2, ok := getter(tt)
		assert.True(t, ok)
		assert.Equal(t, o, o2)
	}

	tester(&mocks.KeyGenerator{}, func(t reflect.Type) (interface{}, bool) { o, ok := sw.KeyGenerators[t]; return o, ok })
	tester(&mocks.KeyDeriver{}, func(t reflect.Type) (interface{}, bool) { o, ok := sw.KeyDerivers[t]; return o, ok })
	tester(&mocks.KeyImporter{}, func(t reflect.Type) (interface{}, bool) { o, ok := sw.KeyImporters[t]; return o, ok })
	tester(&mocks.Encryptor{}, func(t reflect.Type) (interface{}, bool) { o, ok := sw.Encryptors[t]; return o, ok })
	tester(&mocks.Decryptor{}, func(t reflect.Type) (interface{}, bool) { o, ok := sw.Decryptors[t]; return o, ok })
	tester(&mocks.Signer{}, func(t reflect.Type) (interface{}, bool) { o, ok := sw.Signers[t]; return o, ok })
	tester(&mocks.Verifier{}, func(t reflect.Type) (interface{}, bool) { o, ok := sw.Verifiers[t]; return o, ok })
	tester(&mocks.Hasher{}, func(t reflect.Type) (interface{}, bool) { o, ok := sw.Hashers[t]; return o, ok })

//添加无效包装
	err := sw.AddWrapper(reflect.TypeOf(cleanup), cleanup)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "wrapper type not valid, must be on of: KeyGenerator, KeyDeriver, KeyImporter, Encryptor, Decryptor, Signer, Verifier, Hasher")
}

func getCryptoHashIndex(t *testing.T) crypto.Hash {
	switch currentTestConfig.hashFamily {
	case "SHA2":
		switch currentTestConfig.securityLevel {
		case 256:
			return crypto.SHA256
		case 384:
			return crypto.SHA384
		default:
			t.Fatalf("Invalid security level [%d]", currentTestConfig.securityLevel)
		}
	case "SHA3":
		switch currentTestConfig.securityLevel {
		case 256:
			return crypto.SHA3_256
		case 384:
			return crypto.SHA3_384
		default:
			t.Fatalf("Invalid security level [%d]", currentTestConfig.securityLevel)
		}
	default:
		t.Fatalf("Invalid hash family [%s]", currentTestConfig.hashFamily)
	}

	return crypto.SHA3_256
}
