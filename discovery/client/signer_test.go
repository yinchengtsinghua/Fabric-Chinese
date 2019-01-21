
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package discovery

import (
	"crypto/rand"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func init() {
	factory.InitFactories(nil)
}

func TestSameMessage(t *testing.T) {
	var signedInvokedCount int
	sign := func(msg []byte) ([]byte, error) {
		signedInvokedCount++
		return msg, nil
	}

	ms := NewMemoizeSigner(sign, 10)
	for i := 0; i < 5; i++ {
		sig, err := ms.Sign([]byte{1, 2, 3})
		assert.NoError(t, err)
		assert.Equal(t, []byte{1, 2, 3}, sig)
		assert.Equal(t, 1, signedInvokedCount)
	}
}

func TestDifferentMessages(t *testing.T) {
	var n uint = 5
	var signedInvokedCount uint32
	sign := func(msg []byte) ([]byte, error) {
		atomic.AddUint32(&signedInvokedCount, 1)
		return msg, nil
	}

	ms := NewMemoizeSigner(sign, n)
	parallelSignRange := func(start, end uint) {
		var wg sync.WaitGroup
		wg.Add((int)(end - start))
		for i := start; i < end; i++ {
			i := i
			go func() {
				defer wg.Done()
				sig, err := ms.Sign([]byte{byte(i)})
				assert.NoError(t, err)
				assert.Equal(t, []byte{byte(i)}, sig)
			}()
		}
		wg.Wait()
	}

//查询一次
	parallelSignRange(0, n)
	assert.Equal(t, uint32(n), atomic.LoadUint32(&signedInvokedCount))

//两次查询
	parallelSignRange(0, n)
	assert.Equal(t, uint32(n), atomic.LoadUint32(&signedInvokedCount))

//在不相交的范围上查询三次
	parallelSignRange(n+1, 2*n)
	oldSignedInvokedCount := atomic.LoadUint32(&signedInvokedCount)

//确保从内存中清除了一些早期消息0-n
	parallelSignRange(0, n)
	assert.True(t, oldSignedInvokedCount < atomic.LoadUint32(&signedInvokedCount))
}

func TestFailure(t *testing.T) {
	sign := func(_ []byte) ([]byte, error) {
		return nil, errors.New("something went wrong")
	}

	ms := NewMemoizeSigner(sign, 1)
	_, err := ms.Sign([]byte{1, 2, 3})
	assert.Equal(t, "something went wrong", err.Error())
}

func TestNotSavingInMem(t *testing.T) {
	sign := func(_ []byte) ([]byte, error) {
		b := make([]byte, 30)
		_, err := io.ReadFull(rand.Reader, b)
		assert.NoError(t, err)
		return b, nil
	}
	ms := NewMemoizeSigner(sign, 0)
	sig1, _ := ms.sign(([]byte)("aa"))
	sig2, _ := ms.sign(([]byte)("aa"))
	assert.NotEqual(t, sig1, sig2)

}
