
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


package bccsp

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAESOpts(t *testing.T) {
	test := func(ephemeral bool) {
		for _, opts := range []KeyGenOpts{
			&AES128KeyGenOpts{ephemeral},
			&AES192KeyGenOpts{ephemeral},
			&AES256KeyGenOpts{ephemeral},
		} {
			expectedAlgorithm := reflect.TypeOf(opts).String()[7:13]
			assert.Equal(t, expectedAlgorithm, opts.Algorithm())
			assert.Equal(t, ephemeral, opts.Ephemeral())
		}
	}
	test(true)
	test(false)

	opts := &AESKeyGenOpts{true}
	assert.Equal(t, "AES", opts.Algorithm())
	assert.True(t, opts.Ephemeral())
	opts.Temporary = false
	assert.False(t, opts.Ephemeral())
}

func TestRSAOpts(t *testing.T) {
	test := func(ephemeral bool) {
		for _, opts := range []KeyGenOpts{
			&RSA1024KeyGenOpts{ephemeral},
			&RSA2048KeyGenOpts{ephemeral},
			&RSA3072KeyGenOpts{ephemeral},
			&RSA4096KeyGenOpts{ephemeral},
		} {
			expectedAlgorithm := reflect.TypeOf(opts).String()[7:14]
			assert.Equal(t, expectedAlgorithm, opts.Algorithm())
			assert.Equal(t, ephemeral, opts.Ephemeral())
		}
	}
	test(true)
	test(false)
}

func TestECDSAOpts(t *testing.T) {
	test := func(ephemeral bool) {
		for _, opts := range []KeyGenOpts{
			&ECDSAP256KeyGenOpts{ephemeral},
			&ECDSAP384KeyGenOpts{ephemeral},
		} {
			expectedAlgorithm := reflect.TypeOf(opts).String()[7:16]
			assert.Equal(t, expectedAlgorithm, opts.Algorithm())
			assert.Equal(t, ephemeral, opts.Ephemeral())
		}
	}
	test(true)
	test(false)

	test = func(ephemeral bool) {
		for _, opts := range []KeyGenOpts{
			&ECDSAKeyGenOpts{ephemeral},
			&ECDSAPKIXPublicKeyImportOpts{ephemeral},
			&ECDSAPrivateKeyImportOpts{ephemeral},
			&ECDSAGoPublicKeyImportOpts{ephemeral},
		} {
			assert.Equal(t, "ECDSA", opts.Algorithm())
			assert.Equal(t, ephemeral, opts.Ephemeral())
		}
	}
	test(true)
	test(false)

	opts := &ECDSAReRandKeyOpts{Temporary: true}
	assert.True(t, opts.Ephemeral())
	opts.Temporary = false
	assert.False(t, opts.Ephemeral())
	assert.Equal(t, "ECDSA_RERAND", opts.Algorithm())
	assert.Empty(t, opts.ExpansionValue())
}

func TestHashOpts(t *testing.T) {
	for _, ho := range []HashOpts{&SHA256Opts{}, &SHA384Opts{}, &SHA3_256Opts{}, &SHA3_384Opts{}} {
		s := strings.Replace(reflect.TypeOf(ho).String(), "*bccsp.", "", -1)
		algorithm := strings.Replace(s, "Opts", "", -1)
		assert.Equal(t, algorithm, ho.Algorithm())
		ho2, err := GetHashOpt(algorithm)
		assert.NoError(t, err)
		assert.Equal(t, ho.Algorithm(), ho2.Algorithm())
	}
	_, err := GetHashOpt("foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hash function not recognized")

	assert.Equal(t, "SHA", (&SHAOpts{}).Algorithm())
}

func TestHMAC(t *testing.T) {
	opts := &HMACTruncated256AESDeriveKeyOpts{Arg: []byte("arg")}
	assert.False(t, opts.Ephemeral())
	opts.Temporary = true
	assert.True(t, opts.Ephemeral())
	assert.Equal(t, "HMAC_TRUNCATED_256", opts.Algorithm())
	assert.Equal(t, []byte("arg"), opts.Argument())

	opts2 := &HMACDeriveKeyOpts{Arg: []byte("arg")}
	assert.False(t, opts2.Ephemeral())
	opts2.Temporary = true
	assert.True(t, opts2.Ephemeral())
	assert.Equal(t, "HMAC", opts2.Algorithm())
	assert.Equal(t, []byte("arg"), opts2.Argument())
}

func TestKeyGenOpts(t *testing.T) {
	expectedAlgorithms := map[reflect.Type]string{
		reflect.TypeOf(&HMACImportKeyOpts{}):        "HMAC",
		reflect.TypeOf(&RSAKeyGenOpts{}):            "RSA",
		reflect.TypeOf(&RSAGoPublicKeyImportOpts{}): "RSA",
		reflect.TypeOf(&X509PublicKeyImportOpts{}):  "X509Certificate",
		reflect.TypeOf(&AES256ImportKeyOpts{}):      "AES",
	}
	test := func(ephemeral bool) {
		for _, opts := range []KeyGenOpts{
			&HMACImportKeyOpts{ephemeral},
			&RSAKeyGenOpts{ephemeral},
			&RSAGoPublicKeyImportOpts{ephemeral},
			&X509PublicKeyImportOpts{ephemeral},
			&AES256ImportKeyOpts{ephemeral},
		} {
			expectedAlgorithm := expectedAlgorithms[reflect.TypeOf(opts)]
			assert.Equal(t, expectedAlgorithm, opts.Algorithm())
			assert.Equal(t, ephemeral, opts.Ephemeral())
		}
	}
	test(true)
	test(false)
}
