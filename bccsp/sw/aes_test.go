
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

package sw

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"io"
	"math/big"
	mrand "math/rand"
	"testing"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/mocks"
	"github.com/hyperledger/fabric/bccsp/utils"
	"github.com/stretchr/testify/assert"
)

//TESTCBCPKCS7加密CBCPKCS7使用CBCPKCS7解密加密使用CBCPKCS7加密和解密。
func TestCBCPKCS7EncryptCBCPKCS7Decrypt(t *testing.T) {
	t.Parallel()

//注：本试验的目的不是在CBC模式下测试AES-256的强度。
//…而不是验证包装/解开密码的代码。
	key := make([]byte, 32)
	rand.Reader.Read(key)

//1234567890123456789012345678901234567890123456789012
	var ptext = []byte("a message with arbitrary length (42 bytes)")

	encrypted, encErr := AESCBCPKCS7Encrypt(key, ptext)
	if encErr != nil {
		t.Fatalf("Error encrypting '%s': %s", ptext, encErr)
	}

	decrypted, dErr := AESCBCPKCS7Decrypt(key, encrypted)
	if dErr != nil {
		t.Fatalf("Error decrypting the encrypted '%s': %v", ptext, dErr)
	}

	if string(ptext[:]) != string(decrypted[:]) {
		t.Fatal("Decrypt( Encrypt( ptext ) ) != ptext: Ciphertext decryption with the same key must result in the original plaintext!")
	}
}

//testpkcs7padding使用人类可读的明文验证pkcs 7 padding。
func TestPKCS7Padding(t *testing.T) {
	t.Parallel()

//0字节/长度ptext
	ptext := []byte("")
	expected := []byte{16, 16, 16, 16,
		16, 16, 16, 16,
		16, 16, 16, 16,
		16, 16, 16, 16}
	result := pkcs7Padding(ptext)

	if !bytes.Equal(expected, result) {
		t.Fatal("Padding error! Expected: ", expected, "', received: '", result, "'")
	}

//1字节/长度ptext
	ptext = []byte("1")
	expected = []byte{'1', 15, 15, 15,
		15, 15, 15, 15,
		15, 15, 15, 15,
		15, 15, 15, 15}
	result = pkcs7Padding(ptext)

	if !bytes.Equal(expected, result) {
		t.Fatal("Padding error! Expected: '", expected, "', received: '", result, "'")
	}

//2字节/长度ptext
	ptext = []byte("12")
	expected = []byte{'1', '2', 14, 14,
		14, 14, 14, 14,
		14, 14, 14, 14,
		14, 14, 14, 14}
	result = pkcs7Padding(ptext)

	if !bytes.Equal(expected, result) {
		t.Fatal("Padding error! Expected: '", expected, "', received: '", result, "'")
	}

//3到aes.blocksize-1字节纯文本
	ptext = []byte("1234567890ABCDEF")
	for i := 3; i < aes.BlockSize; i++ {
		result := pkcs7Padding(ptext[:i])

		padding := aes.BlockSize - i
		expectedPadding := bytes.Repeat([]byte{byte(padding)}, padding)
		expected = append(ptext[:i], expectedPadding...)

		if !bytes.Equal(result, expected) {
			t.Fatal("Padding error! Expected: '", expected, "', received: '", result, "'")
		}
	}

//aes.blocksize长度ptext
	ptext = bytes.Repeat([]byte{byte('x')}, aes.BlockSize)
	result = pkcs7Padding(ptext)

	expectedPadding := bytes.Repeat([]byte{byte(aes.BlockSize)}, aes.BlockSize)
	expected = append(ptext, expectedPadding...)

	if len(result) != 2*aes.BlockSize {
		t.Fatal("Padding error: expected the length of the returned slice to be 2 times aes.BlockSize")
	}

	if !bytes.Equal(expected, result) {
		t.Fatal("Padding error! Expected: '", expected, "', received: '", result, "'")
	}
}

//testpkcs7unpadding使用人类可读的明文验证pkcs 7 unpadding。
func TestPKCS7UnPadding(t *testing.T) {
	t.Parallel()

//0字节/长度ptext
	expected := []byte("")
	ptext := []byte{16, 16, 16, 16,
		16, 16, 16, 16,
		16, 16, 16, 16,
		16, 16, 16, 16}

	result, _ := pkcs7UnPadding(ptext)

	if !bytes.Equal(expected, result) {
		t.Fatal("UnPadding error! Expected: '", expected, "', received: '", result, "'")
	}

//1字节/长度ptext
	expected = []byte("1")
	ptext = []byte{'1', 15, 15, 15,
		15, 15, 15, 15,
		15, 15, 15, 15,
		15, 15, 15, 15}

	result, _ = pkcs7UnPadding(ptext)

	if !bytes.Equal(expected, result) {
		t.Fatal("UnPadding error! Expected: '", expected, "', received: '", result, "'")
	}

//2字节/长度ptext
	expected = []byte("12")
	ptext = []byte{'1', '2', 14, 14,
		14, 14, 14, 14,
		14, 14, 14, 14,
		14, 14, 14, 14}

	result, _ = pkcs7UnPadding(ptext)

	if !bytes.Equal(expected, result) {
		t.Fatal("UnPadding error! Expected: '", expected, "', received: '", result, "'")
	}

//3到aes.blocksize-1字节纯文本
	base := []byte("1234567890ABCDEF")
	for i := 3; i < aes.BlockSize; i++ {
		iPad := aes.BlockSize - i
		padding := bytes.Repeat([]byte{byte(iPad)}, iPad)
		ptext = append(base[:i], padding...)

		expected := base[:i]
		result, _ := pkcs7UnPadding(ptext)

		if !bytes.Equal(result, expected) {
			t.Fatal("UnPadding error! Expected: '", expected, "', received: '", result, "'")
		}
	}

//aes.blocksize长度ptext
	expected = bytes.Repeat([]byte{byte('x')}, aes.BlockSize)
	padding := bytes.Repeat([]byte{byte(aes.BlockSize)}, aes.BlockSize)
	ptext = append(expected, padding...)

	result, _ = pkcs7UnPadding(ptext)

	if !bytes.Equal(expected, result) {
		t.Fatal("UnPadding error! Expected: '", expected, "', received: '", result, "'")
	}
}

//testcbccencryptcbckcs7解密blocksizelengthplaintext验证cbckcs7解密是否返回错误
//试图解密不可恢复长度的密文时。
func TestCBCEncryptCBCPKCS7Decrypt_BlockSizeLengthPlaintext(t *testing.T) {
	t.Parallel()

//本测试的目的之一是记录并澄清预期行为，即
//根据PKCS 7 v1.5规范（见RFC-2315 P.21），在填充阶段将块附加到消息中。
	key := make([]byte, 32)
	rand.Reader.Read(key)

//1234567890123456
	var ptext = []byte("a 16 byte messag")

	encrypted, encErr := aesCBCEncrypt(key, ptext)
	if encErr != nil {
		t.Fatalf("Error encrypting '%s': %v", ptext, encErr)
	}

	decrypted, dErr := AESCBCPKCS7Decrypt(key, encrypted)
	if dErr == nil {
		t.Fatalf("Expected an error decrypting ptext '%s'. Decrypted to '%v'", dErr, decrypted)
	}
}

//testccpkcs7encryptcbccdecrypt_expectingcorruptmessage验证cbcdecrypt是否可以解密未添加的
//密文的版本，块大小长度的消息。
func TestCBCPKCS7EncryptCBCDecrypt_ExpectingCorruptMessage(t *testing.T) {
	t.Parallel()

//本测试的目的之一是记录并澄清预期行为，即
//根据PKCS 7 v1.5规范（见RFC-2315 P.21），在填充阶段将块附加到消息中。
	key := make([]byte, 32)
	rand.Reader.Read(key)

//0123456789abcdef
	var ptext = []byte("a 16 byte messag")

	encrypted, encErr := AESCBCPKCS7Encrypt(key, ptext)
	if encErr != nil {
		t.Fatalf("Error encrypting ptext %v", encErr)
	}

	decrypted, dErr := aesCBCDecrypt(key, encrypted)
	if dErr != nil {
		t.Fatalf("Error encrypting ptext %v, %v", dErr, decrypted)
	}

	if string(ptext[:]) != string(decrypted[:aes.BlockSize]) {
		t.Log("ptext: ", ptext)
		t.Log("decrypted: ", decrypted[:aes.BlockSize])
		t.Fatal("Encryption->Decryption with same key should result in original ptext")
	}

	if !bytes.Equal(decrypted[aes.BlockSize:], bytes.Repeat([]byte{byte(aes.BlockSize)}, aes.BlockSize)) {
		t.Fatal("Expected extra block with padding in encrypted ptext", decrypted)
	}
}

//testccpkcs7加密emptyplainText加密并填充空ptext。验证密文长度是否符合预期。
func TestCBCPKCS7Encrypt_EmptyPlaintext(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	rand.Reader.Read(key)

	t.Log("Generated key: ", key)

	var emptyPlaintext = []byte("")
	t.Log("Plaintext length: ", len(emptyPlaintext))

	ciphertext, encErr := AESCBCPKCS7Encrypt(key, emptyPlaintext)
	if encErr != nil {
		t.Fatalf("Error encrypting '%v'", encErr)
	}

//预期的密文长度：32（=32）
//作为填充的一部分，至少有一个块被加密（而第一个块是IV）
	const expectedLength = aes.BlockSize + aes.BlockSize
	if len(ciphertext) != expectedLength {
		t.Fatalf("Wrong ciphertext length. Expected %d, received %d", expectedLength, len(ciphertext))
	}

	t.Log("Ciphertext length: ", len(ciphertext))
	t.Log("Cipher: ", ciphertext)
}

//testcbccencrypt_emptyplaintext对空消息进行加密。验证密文长度是否符合预期。
func TestCBCEncrypt_EmptyPlaintext(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	rand.Reader.Read(key)
	t.Log("Generated key: ", key)

	var emptyPlaintext = []byte("")
	t.Log("Message length: ", len(emptyPlaintext))

	ciphertext, encErr := aesCBCEncrypt(key, emptyPlaintext)
	assert.NoError(t, encErr)

	t.Log("Ciphertext length: ", len(ciphertext))

//预期密码长度：aes.blocksize，第一个也是唯一一个块是iv
	var expectedLength = aes.BlockSize

	if len(ciphertext) != expectedLength {
		t.Fatalf("Wrong ciphertext length. Expected: '%d', received: '%d'", expectedLength, len(ciphertext))
	}
	t.Log("Ciphertext: ", ciphertext)
}

//testccpkcs7用同一密钥加密两次verifyrandomvs。如果随机生成IV，则前16个字节应该不同。
func TestCBCPKCS7Encrypt_VerifyRandomIVs(t *testing.T) {
	t.Parallel()

	key := make([]byte, aes.BlockSize)
	rand.Reader.Read(key)
	t.Log("Key 1", key)

	var ptext = []byte("a message to encrypt")

	ciphertext1, err := AESCBCPKCS7Encrypt(key, ptext)
	if err != nil {
		t.Fatalf("Error encrypting '%s': %s", ptext, err)
	}

//如果相同的消息使用相同的密钥加密，则需要不同的IV
	ciphertext2, err := AESCBCPKCS7Encrypt(key, ptext)
	if err != nil {
		t.Fatalf("Error encrypting '%s': %s", ptext, err)
	}

	iv1 := ciphertext1[:aes.BlockSize]
	iv2 := ciphertext2[:aes.BlockSize]

	t.Log("Ciphertext1: ", iv1)
	t.Log("Ciphertext2: ", iv2)
	t.Log("bytes.Equal: ", bytes.Equal(iv1, iv2))

	if bytes.Equal(iv1, iv2) {
		t.Fatal("Error: ciphertexts contain identical initialization vectors (IVs)")
	}
}

//testccpkcs7encrypt_correctephertextlengthcheck验证返回的密文长度是否符合预期。
func TestCBCPKCS7Encrypt_CorrectCiphertextLengthCheck(t *testing.T) {
	t.Parallel()

	key := make([]byte, aes.BlockSize)
	rand.Reader.Read(key)

//消息长度（字节）==aes.blocksize（16字节）
//期望的密码长度=IV长度（1个块）+1个块消息

	var ptext = []byte("0123456789ABCDEF")

	for i := 1; i < aes.BlockSize; i++ {
		ciphertext, err := AESCBCPKCS7Encrypt(key, ptext[:i])
		if err != nil {
			t.Fatal("Error encrypting '", ptext, "'")
		}

		expectedLength := aes.BlockSize + aes.BlockSize
		if len(ciphertext) != expectedLength {
			t.Fatalf("Incorrect ciphertext incorrect: expected '%d', received '%d'", expectedLength, len(ciphertext))
		}
	}
}

//testcbcencryptcbccdecrypt_keyMismatch尝试使用与用于加密的密钥不同的密钥进行解密。
func TestCBCEncryptCBCDecrypt_KeyMismatch(t *testing.T) {
	t.Parallel()

//生成随机密钥
	key := make([]byte, aes.BlockSize)
	rand.Reader.Read(key)

//克隆和篡改密钥
	wrongKey := make([]byte, aes.BlockSize)
	copy(wrongKey, key[:])
	wrongKey[0] = key[0] + 1

	var ptext = []byte("1234567890ABCDEF")
	encrypted, encErr := aesCBCEncrypt(key, ptext)
	if encErr != nil {
		t.Fatalf("Error encrypting '%s': %v", ptext, encErr)
	}

	decrypted, decErr := aesCBCDecrypt(wrongKey, encrypted)
	if decErr != nil {
		t.Fatalf("Error decrypting '%s': %v", ptext, decErr)
	}

	if string(ptext[:]) == string(decrypted[:]) {
		t.Fatal("Decrypting a ciphertext with a different key than the one used for encrypting it - should not result in the original plaintext.")
	}
}

//testcbcencryptcbccdecrypt用cbcencrypt加密，用cbcdecrypt解密。
func TestCBCEncryptCBCDecrypt(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	rand.Reader.Read(key)

//1234567890123456
	var ptext = []byte("a 16 byte messag")

	encrypted, encErr := aesCBCEncrypt(key, ptext)
	if encErr != nil {
		t.Fatalf("Error encrypting '%s': %v", ptext, encErr)
	}

	decrypted, decErr := aesCBCDecrypt(key, encrypted)
	if decErr != nil {
		t.Fatalf("Error decrypting '%s': %v", ptext, decErr)
	}

	if string(ptext[:]) != string(decrypted[:]) {
		t.Fatal("Encryption->Decryption with same key should result in the original plaintext.")
	}
}

//testcbcencryptwithrandcbcdecrypt使用传递的prng对cbccencrypt进行加密，并使用cbcdecrypt进行解密。
func TestCBCEncryptWithRandCBCDecrypt(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	rand.Reader.Read(key)

//1234567890123456
	var ptext = []byte("a 16 byte messag")

	encrypted, encErr := aesCBCEncryptWithRand(rand.Reader, key, ptext)
	if encErr != nil {
		t.Fatalf("Error encrypting '%s': %v", ptext, encErr)
	}

	decrypted, decErr := aesCBCDecrypt(key, encrypted)
	if decErr != nil {
		t.Fatalf("Error decrypting '%s': %v", ptext, decErr)
	}

	if string(ptext[:]) != string(decrypted[:]) {
		t.Fatal("Encryption->Decryption with same key should result in the original plaintext.")
	}
}

//testcbcencryptwithivcbccdecrypt使用传递的iv对cbcencrypt进行加密，并使用cbcdecrypt进行解密。
func TestCBCEncryptWithIVCBCDecrypt(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	rand.Reader.Read(key)

//1234567890123456
	var ptext = []byte("a 16 byte messag")

	iv := make([]byte, aes.BlockSize)
	_, err := io.ReadFull(rand.Reader, iv)
	assert.NoError(t, err)

	encrypted, encErr := aesCBCEncryptWithIV(iv, key, ptext)
	if encErr != nil {
		t.Fatalf("Error encrypting '%s': %v", ptext, encErr)
	}

	decrypted, decErr := aesCBCDecrypt(key, encrypted)
	if decErr != nil {
		t.Fatalf("Error decrypting '%s': %v", ptext, decErr)
	}

	if string(ptext[:]) != string(decrypted[:]) {
		t.Fatal("Encryption->Decryption with same key should result in the original plaintext.")
	}
}

//testesrelatedutilfunctions测试与aes相关的结构中常用的各种功能
func TestAESRelatedUtilFunctions(t *testing.T) {
	t.Parallel()

	key, err := GetRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed generating AES key [%s]", err)
	}

	for i := 1; i < 100; i++ {
		l, err := rand.Int(rand.Reader, big.NewInt(1024))
		if err != nil {
			t.Fatalf("Failed generating AES key [%s]", err)
		}
		msg, err := GetRandomBytes(int(l.Int64()) + 1)
		if err != nil {
			t.Fatalf("Failed generating AES key [%s]", err)
		}

		ct, err := AESCBCPKCS7Encrypt(key, msg)
		if err != nil {
			t.Fatalf("Failed encrypting [%s]", err)
		}

		msg2, err := AESCBCPKCS7Decrypt(key, ct)
		if err != nil {
			t.Fatalf("Failed decrypting [%s]", err)
		}

		if 0 != bytes.Compare(msg, msg2) {
			t.Fatalf("Wrong decryption output [%x][%x]", msg, msg2)
		}
	}
}

//测试变量眼睛编码测试一些aes<->pem转换
func TestVariousAESKeyEncoding(t *testing.T) {
	t.Parallel()

	key, err := GetRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed generating AES key [%s]", err)
	}

//PEM格式
	pem := utils.AEStoPEM(key)
	keyFromPEM, err := utils.PEMtoAES(pem, nil)
	if err != nil {
		t.Fatalf("Failed converting PEM to AES key [%s]", err)
	}
	if 0 != bytes.Compare(key, keyFromPEM) {
		t.Fatalf("Failed converting PEM to AES key. Keys are different [%x][%x]", key, keyFromPEM)
	}

//加密的PEM格式
	pem, err = utils.AEStoEncryptedPEM(key, []byte("passwd"))
	if err != nil {
		t.Fatalf("Failed converting AES key to Encrypted PEM [%s]", err)
	}
	keyFromPEM, err = utils.PEMtoAES(pem, []byte("passwd"))
	if err != nil {
		t.Fatalf("Failed converting encrypted PEM to AES key [%s]", err)
	}
	if 0 != bytes.Compare(key, keyFromPEM) {
		t.Fatalf("Failed converting encrypted PEM to AES key. Keys are different [%x][%x]", key, keyFromPEM)
	}
}

func TestPkcs7UnPaddingInvalidInputs(t *testing.T) {
	t.Parallel()

	_, err := pkcs7UnPadding([]byte{1, 2, 3, 4, 5, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	assert.Error(t, err)
	assert.Equal(t, "Invalid pkcs7 padding (pad[i] != unpadding)", err.Error())
}

func TestAESCBCEncryptInvalidInputs(t *testing.T) {
	t.Parallel()

	_, err := aesCBCEncrypt(nil, []byte{0, 1, 2, 3})
	assert.Error(t, err)
	assert.Equal(t, "Invalid plaintext. It must be a multiple of the block size", err.Error())

	_, err = aesCBCEncrypt([]byte{0}, []byte{1, 2, 3, 4, 5, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})
	assert.Error(t, err)
}

func TestAESCBCDecryptInvalidInputs(t *testing.T) {
	t.Parallel()

	_, err := aesCBCDecrypt([]byte{0}, []byte{1, 2, 3, 4, 5, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})
	assert.Error(t, err)

	_, err = aesCBCDecrypt([]byte{1, 2, 3, 4, 5, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, []byte{0})
	assert.Error(t, err)

	_, err = aesCBCDecrypt([]byte{1, 2, 3, 4, 5, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		[]byte{1, 2, 3, 4, 5, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	assert.Error(t, err)
}

//testeescbcpkcs7加密机ecrypt测试的集成
//AESCBCPKCS7加密机和AESCBCPKCS7解密机
func TestAESCBCPKCS7EncryptorDecrypt(t *testing.T) {
	t.Parallel()

	raw, err := GetRandomBytes(32)
	assert.NoError(t, err)

	k := &aesPrivateKey{privKey: raw, exportable: false}

	msg := []byte("Hello World")
	encryptor := &aescbcpkcs7Encryptor{}

	_, err = encryptor.Encrypt(k, msg, nil)
	assert.Error(t, err)

	_, err = encryptor.Encrypt(k, msg, &mocks.EncrypterOpts{})
	assert.Error(t, err)

	_, err = encryptor.Encrypt(k, msg, &bccsp.AESCBCPKCS7ModeOpts{IV: []byte{1}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid IV. It must have length the block size")

	_, err = encryptor.Encrypt(k, msg, &bccsp.AESCBCPKCS7ModeOpts{IV: []byte{1}, PRNG: rand.Reader})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid options. Either IV or PRNG should be different from nil, or both nil.")

	ct, err := encryptor.Encrypt(k, msg, bccsp.AESCBCPKCS7ModeOpts{})
	assert.NoError(t, err)

	ct, err = encryptor.Encrypt(k, msg, &bccsp.AESCBCPKCS7ModeOpts{})
	assert.NoError(t, err)

	decryptor := &aescbcpkcs7Decryptor{}

	_, err = decryptor.Decrypt(k, ct, nil)
	assert.Error(t, err)

	_, err = decryptor.Decrypt(k, ct, &mocks.EncrypterOpts{})
	assert.Error(t, err)

	msg2, err := decryptor.Decrypt(k, ct, &bccsp.AESCBCPKCS7ModeOpts{})
	assert.NoError(t, err)
	assert.Equal(t, msg, msg2)
}

func TestAESCBCPKCS7EncryptorWithIVSameCiphertext(t *testing.T) {
	t.Parallel()

	raw, err := GetRandomBytes(32)
	assert.NoError(t, err)

	k := &aesPrivateKey{privKey: raw, exportable: false}

	msg := []byte("Hello World")
	encryptor := &aescbcpkcs7Encryptor{}

	iv := make([]byte, aes.BlockSize)

	ct, err := encryptor.Encrypt(k, msg, &bccsp.AESCBCPKCS7ModeOpts{IV: iv})
	assert.NoError(t, err)
	assert.NotNil(t, ct)
	assert.Equal(t, iv, ct[:aes.BlockSize])

	ct2, err := encryptor.Encrypt(k, msg, &bccsp.AESCBCPKCS7ModeOpts{IV: iv})
	assert.NoError(t, err)
	assert.NotNil(t, ct2)
	assert.Equal(t, iv, ct2[:aes.BlockSize])

	assert.Equal(t, ct, ct2)
}

func TestAESCBCPKCS7EncryptorWithRandSameCiphertext(t *testing.T) {
	t.Parallel()

	raw, err := GetRandomBytes(32)
	assert.NoError(t, err)

	k := &aesPrivateKey{privKey: raw, exportable: false}

	msg := []byte("Hello World")
	encryptor := &aescbcpkcs7Encryptor{}

	r := mrand.New(mrand.NewSource(0))
	iv := make([]byte, aes.BlockSize)
	_, err = io.ReadFull(r, iv)
	assert.NoError(t, err)

	r = mrand.New(mrand.NewSource(0))
	ct, err := encryptor.Encrypt(k, msg, &bccsp.AESCBCPKCS7ModeOpts{PRNG: r})
	assert.NoError(t, err)
	assert.NotNil(t, ct)
	assert.Equal(t, iv, ct[:aes.BlockSize])

	r = mrand.New(mrand.NewSource(0))
	ct2, err := encryptor.Encrypt(k, msg, &bccsp.AESCBCPKCS7ModeOpts{PRNG: r})
	assert.NoError(t, err)
	assert.NotNil(t, ct2)
	assert.Equal(t, iv, ct2[:aes.BlockSize])

	assert.Equal(t, ct, ct2)
}
