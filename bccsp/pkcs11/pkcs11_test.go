
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+构建PKCS11

/*
版权所有IBM公司。保留所有权利。

SPDX许可证标识符：Apache-2.0
**/

package pkcs11

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/asn1"
	"testing"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/miekg/pkcs11"
	"github.com/stretchr/testify/assert"
)

func TestKeyGenFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestKeyGenFailures")
	}
	var testOpts bccsp.KeyGenOpts
	ki := currentBCCSP
	_, err := ki.KeyGen(testOpts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Opts parameter. It must not be nil")
}

func TestLoadLib(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestLoadLib")
	}
//设置pkcs11库并提供初始值集
	lib, pin, label := FindPKCS11Lib()

//未指定PKCS11库的测试
	_, _, _, err := loadLib("", pin, label)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No PKCS11 library default")

//测试无效的pkcs11库
	_, _, _, err = loadLib("badLib", pin, label)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Instantiate failed")

//无效标签测试
	_, _, _, err = loadLib(lib, pin, "badLabel")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Could not find token with label")

//无引脚测试
	_, _, _, err = loadLib(lib, "", label)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No PIN set")
}

func TestOIDFromNamedCurve(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestOIDFromNamedCurve")
	}
//测试p224的有效OID
	testOID, boolValue := oidFromNamedCurve(elliptic.P224())
	assert.Equal(t, oidNamedCurveP224, testOID, "Did not receive expected OID for elliptic.P224")
	assert.Equal(t, true, boolValue, "Did not receive a true value when acquiring OID for elliptic.P224")

//测试p256的有效OID
	testOID, boolValue = oidFromNamedCurve(elliptic.P256())
	assert.Equal(t, oidNamedCurveP256, testOID, "Did not receive expected OID for elliptic.P256")
	assert.Equal(t, true, boolValue, "Did not receive a true value when acquiring OID for elliptic.P256")

//测试p384的有效OID
	testOID, boolValue = oidFromNamedCurve(elliptic.P384())
	assert.Equal(t, oidNamedCurveP384, testOID, "Did not receive expected OID for elliptic.P384")
	assert.Equal(t, true, boolValue, "Did not receive a true value when acquiring OID for elliptic.P384")

//测试p521的有效OID
	testOID, boolValue = oidFromNamedCurve(elliptic.P521())
	assert.Equal(t, oidNamedCurveP521, testOID, "Did not receive expected OID for elliptic.P521")
	assert.Equal(t, true, boolValue, "Did not receive a true value when acquiring OID for elliptic.P521")

	var testCurve elliptic.Curve
	testOID, _ = oidFromNamedCurve(testCurve)
	if testOID != nil {
		t.Fatal("Expected nil to be returned.")
	}
}

func TestNamedCurveFromOID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestNamedCurveFromOID")
	}
//有效p224椭圆曲线测试
	namedCurve := namedCurveFromOID(oidNamedCurveP224)
	assert.Equal(t, elliptic.P224(), namedCurve, "Did not receive expected named curve for oidNamedCurveP224")

//有效p256椭圆曲线测试
	namedCurve = namedCurveFromOID(oidNamedCurveP256)
	assert.Equal(t, elliptic.P256(), namedCurve, "Did not receive expected named curve for oidNamedCurveP256")

//有效p256椭圆曲线测试
	namedCurve = namedCurveFromOID(oidNamedCurveP384)
	assert.Equal(t, elliptic.P384(), namedCurve, "Did not receive expected named curve for oidNamedCurveP384")

//有效p521椭圆曲线检验
	namedCurve = namedCurveFromOID(oidNamedCurveP521)
	assert.Equal(t, elliptic.P521(), namedCurve, "Did not receive expected named curved for oidNamedCurveP521")

	testAsn1Value := asn1.ObjectIdentifier{4, 9, 15, 1}
	namedCurve = namedCurveFromOID(testAsn1Value)
	if namedCurve != nil {
		t.Fatal("Expected nil to be returned.")
	}
}

func TestPKCS11GetSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestPKCS11GetSession")
	}
	var sessions []pkcs11.SessionHandle
	for i := 0; i < 3*sessionCacheSize; i++ {
		sessions = append(sessions, currentBCCSP.(*impl).getSession())
	}

//返回所有会话，应使sessioncachesize保持缓存状态
	for _, session := range sessions {
		currentBCCSP.(*impl).returnSession(session)
	}
	sessions = nil

//让我们中断OpenSession，这样就无法打开非缓存会话
	oldSlot := currentBCCSP.(*impl).slot
	currentBCCSP.(*impl).slot = ^uint(0)

//应该能够获取sessioncachesize缓存的会话
	for i := 0; i < sessionCacheSize; i++ {
		sessions = append(sessions, currentBCCSP.(*impl).getSession())
	}

//这个应该失败
	assert.Panics(t, func() {
		currentBCCSP.(*impl).getSession()
	}, "Should not been able to create another session")

//清理
	for _, session := range sessions {
		currentBCCSP.(*impl).returnSession(session)
	}
	currentBCCSP.(*impl).slot = oldSlot
}

func TestPKCS11ECKeySignVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestPKCS11ECKeySignVerify")
	}

	msg1 := []byte("This is my very authentic message")
	msg2 := []byte("This is my very unauthentic message")
	hash1, _ := currentBCCSP.Hash(msg1, &bccsp.SHAOpts{})
	hash2, _ := currentBCCSP.Hash(msg2, &bccsp.SHAOpts{})

	var oid asn1.ObjectIdentifier
	if currentTestConfig.securityLevel == 256 {
		oid = oidNamedCurveP256
	} else if currentTestConfig.securityLevel == 384 {
		oid = oidNamedCurveP384
	}

	key, pubKey, err := currentBCCSP.(*impl).generateECKey(oid, true)
	if err != nil {
		t.Fatalf("Failed generating Key [%s]", err)
	}

	R, S, err := currentBCCSP.(*impl).signP11ECDSA(key, hash1)

	if err != nil {
		t.Fatalf("Failed signing message [%s]", err)
	}

	_, _, err = currentBCCSP.(*impl).signP11ECDSA(nil, hash1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Private key not found")

	pass, err := currentBCCSP.(*impl).verifyP11ECDSA(key, hash1, R, S, currentTestConfig.securityLevel/8)
	if err != nil {
		t.Fatalf("Error verifying message 1 [%s]", err)
	}
	if pass == false {
		t.Fatal("Signature should match!")
	}

	pass = ecdsa.Verify(pubKey, hash1, R, S)
	if pass == false {
		t.Fatal("Signature should match with software verification!")
	}

	pass, err = currentBCCSP.(*impl).verifyP11ECDSA(key, hash2, R, S, currentTestConfig.securityLevel/8)
	if err != nil {
		t.Fatalf("Error verifying message 2 [%s]", err)
	}

	if pass != false {
		t.Fatal("Signature should not match!")
	}

	pass = ecdsa.Verify(pubKey, hash2, R, S)
	if pass != false {
		t.Fatal("Signature should not match with software verification!")
	}
}
