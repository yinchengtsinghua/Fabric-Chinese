
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


package util

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputeSHA256(t *testing.T) {
	if bytes.Compare(ComputeSHA256([]byte("foobar")), ComputeSHA256([]byte("foobar"))) != 0 {
		t.Fatalf("Expected hashes to match, but they did not match")
	}
	if bytes.Compare(ComputeSHA256([]byte("foobar1")), ComputeSHA256([]byte("foobar2"))) == 0 {
		t.Fatalf("Expected hashes to be different, but they match")
	}
}

func TestComputeSHA3256(t *testing.T) {
	if bytes.Compare(ComputeSHA3256([]byte("foobar")), ComputeSHA3256([]byte("foobar"))) != 0 {
		t.Fatalf("Expected hashes to match, but they did not match")
	}
	if bytes.Compare(ComputeSHA3256([]byte("foobar1")), ComputeSHA3256([]byte("foobar2"))) == 0 {
		t.Fatalf("Expected hashed to be different, but they match")
	}
}

func TestUUIDGeneration(t *testing.T) {
	uuid := GenerateUUID()
	if len(uuid) != 36 {
		t.Fatalf("UUID length is not correct. Expected = 36, Got = %d", len(uuid))
	}
	uuid2 := GenerateUUID()
	if uuid == uuid2 {
		t.Fatalf("Two UUIDs are equal. This should never occur")
	}
}

func TestIntUUIDGeneration(t *testing.T) {
	uuid := GenerateIntUUID()

	uuid2 := GenerateIntUUID()
	if uuid == uuid2 {
		t.Fatalf("Two UUIDs are equal. This should never occur")
	}
}
func TestTimestamp(t *testing.T) {
	for i := 0; i < 10; i++ {
		t.Logf("timestamp now: %v", CreateUtcTimestamp())
		time.Sleep(200 * time.Millisecond)
	}
}

func TestGenerateHashFromSignature(t *testing.T) {
	if bytes.Compare(GenerateHashFromSignature("aPath", []byte("aCtor12")),
		GenerateHashFromSignature("aPath", []byte("aCtor12"))) != 0 {
		t.Fatalf("Expected hashes to match, but they did not match")
	}
	if bytes.Compare(GenerateHashFromSignature("aPath", []byte("aCtor12")),
		GenerateHashFromSignature("bPath", []byte("bCtor34"))) == 0 {
		t.Fatalf("Expected hashes to be different, but they match")
	}
}

func TestGeneratIDfromTxSHAHash(t *testing.T) {
	txid := GenerateIDfromTxSHAHash([]byte("foobar"))
	txid2 := GenerateIDfromTxSHAHash([]byte("foobar1"))
	if txid == txid2 {
		t.Fatalf("Two TxIDs are equal. This should never occur")
	}
}

func TestGenerateIDWithAlg(t *testing.T) {
	_, err := GenerateIDWithAlg("sha256", []byte{1, 1, 1, 1})
	if err != nil {
		t.Fatalf("Generator failure: %v", err)
	}
}

func TestGenerateIDWithDefaultAlg(t *testing.T) {
	_, err := GenerateIDWithAlg("", []byte{1, 1, 1, 1})
	if err != nil {
		t.Fatalf("Generator failure: %v", err)
	}
}

func TestGenerateIDWithWrongAlg(t *testing.T) {
	_, err := GenerateIDWithAlg("foobar", []byte{1, 1, 1, 1})
	if err == nil {
		t.Fatalf("Expected error")
	}
}

func TestFindMissingElements(t *testing.T) {
	all := []string{"a", "b", "c", "d"}
	some := []string{"b", "c"}
	expectedDelta := []string{"a", "d"}
	actualDelta := FindMissingElements(all, some)
	if len(expectedDelta) != len(actualDelta) {
		t.Fatalf("Got %v, expected %v", actualDelta, expectedDelta)
	}
	for i := range expectedDelta {
		if strings.Compare(expectedDelta[i], actualDelta[i]) != 0 {
			t.Fatalf("Got %v, expected %v", actualDelta, expectedDelta)
		}
	}
}

func TestToChaincodeArgs(t *testing.T) {
	expected := [][]byte{[]byte("foo"), []byte("bar")}
	actual := ToChaincodeArgs("foo", "bar")
	if len(expected) != len(actual) {
		t.Fatalf("Got %v, expected %v", actual, expected)
	}
	for i := range expected {
		if bytes.Compare(expected[i], actual[i]) != 0 {
			t.Fatalf("Got %v, expected %v", actual, expected)
		}
	}
}

func TestArrayToChaincodeArgs(t *testing.T) {
	expected := [][]byte{[]byte("foo"), []byte("bar")}
	actual := ArrayToChaincodeArgs([]string{"foo", "bar"})
	if len(expected) != len(actual) {
		t.Fatalf("Got %v, expected %v", actual, expected)
	}
	for i := range expected {
		if bytes.Compare(expected[i], actual[i]) != 0 {
			t.Fatalf("Got %v, expected %v", actual, expected)
		}
	}
}

func TestMetadataSignatureBytesNormal(t *testing.T) {
	first := []byte("first")
	second := []byte("second")
	third := []byte("third")

	result := ConcatenateBytes(first, second, third)
	expected := []byte("firstsecondthird")
	if !bytes.Equal(result, expected) {
		t.Errorf("Did not concatenate bytes correctly, expected %s, got %s", expected, result)
	}
}

func TestMetadataSignatureBytesNil(t *testing.T) {
	first := []byte("first")
	second := []byte(nil)
	third := []byte("third")

	result := ConcatenateBytes(first, second, third)
	expected := []byte("firstthird")
	if !bytes.Equal(result, expected) {
		t.Errorf("Did not concatenate bytes correctly, expected %s, got %s", expected, result)
	}
}

type A struct {
	s string
}

type B struct {
	A A
	i int
	X string
}

type C struct{}

type D struct {
	B B
	c *C
}

func (a A) String() string {
	return fmt.Sprintf("I'm '%s'", a.s)
}

func TestFlattenStruct(t *testing.T) {
	d := &D{
		B: B{
			A: A{
				s: "foo",
			},
			i: 42,
			X: "bar ",
		},
		c: nil,
	}

	var x []string
	flatten("", &x, reflect.ValueOf(d))
	assert.Equal(t, 4, len(x), "expect 3 items")
	assert.Equal(t, x[0], "B.A = I'm 'foo'")
	assert.Equal(t, x[1], "B.i = 42")
	assert.Equal(t, x[2], "B.X = \"bar \"")
	assert.Equal(t, x[3], "c =")
}
