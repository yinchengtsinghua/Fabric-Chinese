
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


package util

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testHappyPath(t *testing.T) {
	n1 := RandomInt(10000)
	n2 := RandomInt(10000)
	assert.NotEqual(t, n1, n2)
	n3 := RandomUInt64()
	n4 := RandomUInt64()
	assert.NotEqual(t, n3, n4)
}

func TestContains(t *testing.T) {
	assert.True(t, Contains("foo", []string{"bar", "foo", "baz"}))
	assert.False(t, Contains("foo", []string{"bar", "baz"}))
}

func TestGetRandomInt(t *testing.T) {
	testHappyPath(t)
}

func TestNonNegativeValues(t *testing.T) {
	assert.True(t, RandomInt(1000000) >= 0)
}

func TestGetRandomIntBadInput(t *testing.T) {
	f1 := func() {
		RandomInt(0)
	}
	f2 := func() {
		RandomInt(-500)
	}
	assert.Panics(t, f1)
	assert.Panics(t, f2)
}

type reader struct {
	mock.Mock
}

func (r *reader) Read(p []byte) (int, error) {
	args := r.Mock.Called(p)
	n := args.Get(0).(int)
	err := args.Get(1)
	if err == nil {
		return n, nil
	}
	return n, err.(error)
}

func TestGetRandomIntNoEntropy(t *testing.T) {
	rr := rand.Reader
	defer func() {
		rand.Reader = rr
	}()
	r := &reader{}
	r.On("Read", mock.Anything).Return(0, errors.New("Not enough entropy"))
	rand.Reader = r
//确保随机性仍然有效，即使我们没有熵
	testHappyPath(t)
}

func TestRandomIndices(t *testing.T) {
	assert.Nil(t, GetRandomIndices(10, 5))
	GetRandomIndices(10, 9)
	GetRandomIndices(10, 12)
}

func TestGetIntOrDefault(t *testing.T) {
	viper.Set("N", 100)
	n := GetIntOrDefault("N", 100)
	assert.Equal(t, 100, n)
	m := GetIntOrDefault("M", 101)
	assert.Equal(t, 101, m)
}

func TestGetDurationOrDefault(t *testing.T) {
	viper.Set("foo", time.Second)
	foo := GetDurationOrDefault("foo", time.Second*2)
	assert.Equal(t, time.Second, foo)
	bar := GetDurationOrDefault("bar", time.Second*2)
	assert.Equal(t, time.Second*2, bar)
}

func TestPrintStackTrace(t *testing.T) {
	PrintStackTrace()
}

func TestGetLogger(t *testing.T) {
	l1 := GetLogger("foo", "bar")
	l2 := GetLogger("foo", "bar")
	assert.Equal(t, l1, l2)
}

func TestSet(t *testing.T) {
	s := NewSet()
	assert.Len(t, s.ToArray(), 0)
	assert.Equal(t, s.Size(), 0)
	assert.False(t, s.Exists(42))
	s.Add(42)
	assert.True(t, s.Exists(42))
	assert.Len(t, s.ToArray(), 1)
	assert.Equal(t, s.Size(), 1)
	s.Remove(42)
	assert.False(t, s.Exists(42))
	s.Add(42)
	assert.True(t, s.Exists(42))
	s.Clear()
	assert.False(t, s.Exists(42))
}

func TestStringsToBytesToStrings(t *testing.T) {
	strings := []string{"foo", "bar"}
	assert.Equal(t, strings, BytesToStrings(StringsToBytes(strings)))
}
