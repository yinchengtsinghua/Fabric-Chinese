
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


package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOidFromNamedCurve(t *testing.T) {

	var (
		oidNamedCurveP224 = asn1.ObjectIdentifier{1, 3, 132, 0, 33}
		oidNamedCurveP256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}
		oidNamedCurveP384 = asn1.ObjectIdentifier{1, 3, 132, 0, 34}
		oidNamedCurveP521 = asn1.ObjectIdentifier{1, 3, 132, 0, 35}
	)

	type result struct {
		oid asn1.ObjectIdentifier
		ok  bool
	}

	var tests = []struct {
		name     string
		curve    elliptic.Curve
		expected result
	}{
		{
			name:  "P224",
			curve: elliptic.P224(),
			expected: result{
				oid: oidNamedCurveP224,
				ok:  true,
			},
		},
		{
			name:  "P256",
			curve: elliptic.P256(),
			expected: result{
				oid: oidNamedCurveP256,
				ok:  true,
			},
		},
		{
			name:  "P384",
			curve: elliptic.P384(),
			expected: result{
				oid: oidNamedCurveP384,
				ok:  true,
			},
		},
		{
			name:  "P521",
			curve: elliptic.P521(),
			expected: result{
				oid: oidNamedCurveP521,
				ok:  true,
			},
		},
		{
			name:  "T-1000",
			curve: &elliptic.CurveParams{Name: "T-1000"},
			expected: result{
				oid: nil,
				ok:  false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oid, ok := oidFromNamedCurve(test.curve)
			assert.Equal(t, oid, test.expected.oid)
			assert.Equal(t, ok, test.expected.ok)
		})
	}

}

func TestECDSAKeys(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed generating ECDSA key [%s]", err)
	}

//私钥der格式
	der, err := PrivateKeyToDER(key)
	if err != nil {
		t.Fatalf("Failed converting private key to DER [%s]", err)
	}
	keyFromDER, err := DERToPrivateKey(der)
	if err != nil {
		t.Fatalf("Failed converting DER to private key [%s]", err)
	}
	ecdsaKeyFromDer := keyFromDER.(*ecdsa.PrivateKey)
//TODO:检查曲线
	if key.D.Cmp(ecdsaKeyFromDer.D) != 0 {
		t.Fatal("Failed converting DER to private key. Invalid D.")
	}
	if key.X.Cmp(ecdsaKeyFromDer.X) != 0 {
		t.Fatal("Failed converting DER to private key. Invalid X coordinate.")
	}
	if key.Y.Cmp(ecdsaKeyFromDer.Y) != 0 {
		t.Fatal("Failed converting DER to private key. Invalid Y coordinate.")
	}

//私钥PEM格式
	rawPEM, err := PrivateKeyToPEM(key, nil)
	if err != nil {
		t.Fatalf("Failed converting private key to PEM [%s]", err)
	}
	pemBlock, _ := pem.Decode(rawPEM)
	if pemBlock.Type != "PRIVATE KEY" {
		t.Fatalf("Expected type 'PRIVATE KEY' but found '%s'", pemBlock.Type)
	}
	_, err = x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse PKCS#8 private key [%s]", err)
	}
	keyFromPEM, err := PEMtoPrivateKey(rawPEM, nil)
	if err != nil {
		t.Fatalf("Failed converting DER to private key [%s]", err)
	}
	ecdsaKeyFromPEM := keyFromPEM.(*ecdsa.PrivateKey)
//TODO:检查曲线
	if key.D.Cmp(ecdsaKeyFromPEM.D) != 0 {
		t.Fatal("Failed converting PEM to private key. Invalid D.")
	}
	if key.X.Cmp(ecdsaKeyFromPEM.X) != 0 {
		t.Fatal("Failed converting PEM to private key. Invalid X coordinate.")
	}
	if key.Y.Cmp(ecdsaKeyFromPEM.Y) != 0 {
		t.Fatal("Failed converting PEM to private key. Invalid Y coordinate.")
	}

//无私钥<->pem
	_, err = PrivateKeyToPEM(nil, nil)
	if err == nil {
		t.Fatal("PublicKeyToPEM should fail on nil")
	}

	_, err = PrivateKeyToPEM((*ecdsa.PrivateKey)(nil), nil)
	if err == nil {
		t.Fatal("PrivateKeyToPEM should fail on nil")
	}

	_, err = PrivateKeyToPEM((*rsa.PrivateKey)(nil), nil)
	if err == nil {
		t.Fatal("PrivateKeyToPEM should fail on nil")
	}

	_, err = PEMtoPrivateKey(nil, nil)
	if err == nil {
		t.Fatal("PEMtoPublicKey should fail on nil")
	}

	_, err = PEMtoPrivateKey([]byte{0, 1, 3, 4}, nil)
	if err == nil {
		t.Fatal("PEMtoPublicKey should fail invalid PEM")
	}

	_, err = DERToPrivateKey(nil)
	if err == nil {
		t.Fatal("DERToPrivateKey should fail on nil")
	}

	_, err = DERToPrivateKey([]byte{0, 1, 3, 4})
	if err == nil {
		t.Fatal("DERToPrivateKey should fail on invalid DER")
	}

	_, err = PrivateKeyToDER(nil)
	if err == nil {
		t.Fatal("DERToPrivateKey should fail on nil")
	}

//私钥加密的PEM格式
	encPEM, err := PrivateKeyToPEM(key, []byte("passwd"))
	if err != nil {
		t.Fatalf("Failed converting private key to encrypted PEM [%s]", err)
	}
	_, err = PEMtoPrivateKey(encPEM, nil)
	assert.Error(t, err)
	encKeyFromPEM, err := PEMtoPrivateKey(encPEM, []byte("passwd"))
	if err != nil {
		t.Fatalf("Failed converting DER to private key [%s]", err)
	}
	ecdsaKeyFromEncPEM := encKeyFromPEM.(*ecdsa.PrivateKey)
//TODO:检查曲线
	if key.D.Cmp(ecdsaKeyFromEncPEM.D) != 0 {
		t.Fatal("Failed converting encrypted PEM to private key. Invalid D.")
	}
	if key.X.Cmp(ecdsaKeyFromEncPEM.X) != 0 {
		t.Fatal("Failed converting encrypted PEM to private key. Invalid X coordinate.")
	}
	if key.Y.Cmp(ecdsaKeyFromEncPEM.Y) != 0 {
		t.Fatal("Failed converting encrypted PEM to private key. Invalid Y coordinate.")
	}

//公钥PEM格式
	rawPEM, err = PublicKeyToPEM(&key.PublicKey, nil)
	if err != nil {
		t.Fatalf("Failed converting public key to PEM [%s]", err)
	}
	pemBlock, _ = pem.Decode(rawPEM)
	if pemBlock.Type != "PUBLIC KEY" {
		t.Fatalf("Expected type 'PUBLIC KEY' but found '%s'", pemBlock.Type)
	}
	keyFromPEM, err = PEMtoPublicKey(rawPEM, nil)
	if err != nil {
		t.Fatalf("Failed converting DER to public key [%s]", err)
	}
	ecdsaPkFromPEM := keyFromPEM.(*ecdsa.PublicKey)
//TODO:检查曲线
	if key.X.Cmp(ecdsaPkFromPEM.X) != 0 {
		t.Fatal("Failed converting PEM to private key. Invalid X coordinate.")
	}
	if key.Y.Cmp(ecdsaPkFromPEM.Y) != 0 {
		t.Fatal("Failed converting PEM to private key. Invalid Y coordinate.")
	}

//无公钥<->pem
	_, err = PublicKeyToPEM(nil, nil)
	if err == nil {
		t.Fatal("PublicKeyToPEM should fail on nil")
	}

	_, err = PEMtoPublicKey(nil, nil)
	if err == nil {
		t.Fatal("PEMtoPublicKey should fail on nil")
	}

	_, err = PEMtoPublicKey([]byte{0, 1, 3, 4}, nil)
	if err == nil {
		t.Fatal("PEMtoPublicKey should fail on invalid PEM")
	}

//公钥加密的PEM格式
	encPEM, err = PublicKeyToPEM(&key.PublicKey, []byte("passwd"))
	if err != nil {
		t.Fatalf("Failed converting private key to encrypted PEM [%s]", err)
	}
	_, err = PEMtoPublicKey(encPEM, nil)
	assert.Error(t, err)
	pkFromEncPEM, err := PEMtoPublicKey(encPEM, []byte("passwd"))
	if err != nil {
		t.Fatalf("Failed converting DER to private key [%s]", err)
	}
	ecdsaPkFromEncPEM := pkFromEncPEM.(*ecdsa.PublicKey)
//TODO:检查曲线
	if key.X.Cmp(ecdsaPkFromEncPEM.X) != 0 {
		t.Fatal("Failed converting encrypted PEM to private key. Invalid X coordinate.")
	}
	if key.Y.Cmp(ecdsaPkFromEncPEM.Y) != 0 {
		t.Fatal("Failed converting encrypted PEM to private key. Invalid Y coordinate.")
	}

	_, err = PEMtoPublicKey(encPEM, []byte("passw"))
	if err == nil {
		t.Fatal("PEMtoPublicKey should fail on wrong password")
	}

	_, err = PEMtoPublicKey(encPEM, []byte("passw"))
	if err == nil {
		t.Fatal("PEMtoPublicKey should fail on nil password")
	}

	_, err = PEMtoPublicKey(nil, []byte("passwd"))
	if err == nil {
		t.Fatal("PEMtoPublicKey should fail on nil PEM")
	}

	_, err = PEMtoPublicKey([]byte{0, 1, 3, 4}, []byte("passwd"))
	if err == nil {
		t.Fatal("PEMtoPublicKey should fail on invalid PEM")
	}

	_, err = PEMtoPublicKey(nil, []byte("passw"))
	if err == nil {
		t.Fatal("PEMtoPublicKey should fail on nil PEM and wrong password")
	}

//公钥der格式
	der, err = PublicKeyToDER(&key.PublicKey)
	assert.NoError(t, err)
	keyFromDER, err = DERToPublicKey(der)
	assert.NoError(t, err)
	ecdsaPkFromPEM = keyFromDER.(*ecdsa.PublicKey)
//TODO:检查曲线
	if key.X.Cmp(ecdsaPkFromPEM.X) != 0 {
		t.Fatal("Failed converting PEM to private key. Invalid X coordinate.")
	}
	if key.Y.Cmp(ecdsaPkFromPEM.Y) != 0 {
		t.Fatal("Failed converting PEM to private key. Invalid Y coordinate.")
	}
}

func TestAESKey(t *testing.T) {
	k := []byte{0, 1, 2, 3, 4, 5}
	pem := AEStoPEM(k)

	k2, err := PEMtoAES(pem, nil)
	assert.NoError(t, err)
	assert.Equal(t, k, k2)

	pem, err = AEStoEncryptedPEM(k, k)
	assert.NoError(t, err)

	k2, err = PEMtoAES(pem, k)
	assert.NoError(t, err)
	assert.Equal(t, k, k2)

	_, err = PEMtoAES(pem, nil)
	assert.Error(t, err)

	_, err = AEStoEncryptedPEM(k, nil)
	assert.NoError(t, err)

	k2, err = PEMtoAES(pem, k)
	assert.NoError(t, err)
	assert.Equal(t, k, k2)
}

func TestDERToPublicKey(t *testing.T) {
	_, err := DERToPublicKey(nil)
	assert.Error(t, err)
}

func TestNil(t *testing.T) {
	_, err := PrivateKeyToEncryptedPEM(nil, nil)
	assert.Error(t, err)

	_, err = PrivateKeyToEncryptedPEM((*ecdsa.PrivateKey)(nil), nil)
	assert.Error(t, err)

	_, err = PrivateKeyToEncryptedPEM("Hello World", nil)
	assert.Error(t, err)

	_, err = PEMtoAES(nil, nil)
	assert.Error(t, err)

	_, err = AEStoEncryptedPEM(nil, nil)
	assert.Error(t, err)

	_, err = PublicKeyToPEM(nil, nil)
	assert.Error(t, err)
	_, err = PublicKeyToPEM((*ecdsa.PublicKey)(nil), nil)
	assert.Error(t, err)
	_, err = PublicKeyToPEM((*rsa.PublicKey)(nil), nil)
	assert.Error(t, err)
	_, err = PublicKeyToPEM(nil, []byte("hello world"))
	assert.Error(t, err)

	_, err = PublicKeyToPEM("hello world", nil)
	assert.Error(t, err)
	_, err = PublicKeyToPEM("hello world", []byte("hello world"))
	assert.Error(t, err)

	_, err = PublicKeyToDER(nil)
	assert.Error(t, err)
	_, err = PublicKeyToDER((*ecdsa.PublicKey)(nil))
	assert.Error(t, err)
	_, err = PublicKeyToDER((*rsa.PublicKey)(nil))
	assert.Error(t, err)
	_, err = PublicKeyToDER("hello world")
	assert.Error(t, err)

	_, err = PublicKeyToEncryptedPEM(nil, nil)
	assert.Error(t, err)
	_, err = PublicKeyToEncryptedPEM((*ecdsa.PublicKey)(nil), nil)
	assert.Error(t, err)
	_, err = PublicKeyToEncryptedPEM("hello world", nil)
	assert.Error(t, err)
	_, err = PublicKeyToEncryptedPEM("hello world", []byte("Hello world"))
	assert.Error(t, err)

}

func TestPrivateKeyToPEM(t *testing.T) {
	_, err := PrivateKeyToPEM(nil, nil)
	assert.Error(t, err)

	_, err = PrivateKeyToPEM("hello world", nil)
	assert.Error(t, err)

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	assert.NoError(t, err)
	pem, err := PrivateKeyToPEM(key, nil)
	assert.NoError(t, err)
	assert.NotNil(t, pem)
	key2, err := PEMtoPrivateKey(pem, nil)
	assert.NoError(t, err)
	assert.NotNil(t, key2)
	assert.Equal(t, key.D, key2.(*rsa.PrivateKey).D)

	pem, err = PublicKeyToPEM(&key.PublicKey, nil)
	assert.NoError(t, err)
	assert.NotNil(t, pem)
	key3, err := PEMtoPublicKey(pem, nil)
	assert.NoError(t, err)
	assert.NotNil(t, key2)
	assert.Equal(t, key.PublicKey.E, key3.(*rsa.PublicKey).E)
	assert.Equal(t, key.PublicKey.N, key3.(*rsa.PublicKey).N)
}
