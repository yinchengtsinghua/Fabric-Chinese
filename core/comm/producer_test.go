
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


package comm

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestEmptyEndpoints(t *testing.T) {
	t.Parallel()
	noopFactory := func(endpoint string) (*grpc.ClientConn, error) {
		return nil, nil
	}
	assert.Nil(t, NewConnectionProducer(noopFactory, []string{}))
}

func TestConnFailures(t *testing.T) {
	t.Parallel()
	conn2Endpoint := make(map[string]string)
	shouldConnFail := map[string]bool{
		"a": true,
		"b": false,
		"c": false,
	}
	connFactory := func(endpoint string) (*grpc.ClientConn, error) {
		conn := &grpc.ClientConn{}
		conn2Endpoint[fmt.Sprintf("%p", conn)] = endpoint
		if !shouldConnFail[endpoint] {
			return conn, nil
		}
		return nil, fmt.Errorf("Failed connecting to %s", endpoint)
	}
//用一些端点创建一个生产者，并让第一个端点失败，而所有其他端点都没有失败。
	producer := NewConnectionProducer(connFactory, []string{"a", "b", "c"})
	conn, _, err := producer.NewConnection()
	assert.NoError(t, err)
//我们不应该返回“a”，因为连接到“a”失败
	assert.NotEqual(t, "a", conn2Endpoint[fmt.Sprintf("%p", conn)])
//现在，复兴“A”
	shouldConnFail["a"] = false
//尝试获得1000次连接，以确保选择被洗牌
	selected := make(map[string]struct{})
	for i := 0; i < 1000; i++ {
		conn, _, err := producer.NewConnection()
		assert.NoError(t, err)
		selected[conn2Endpoint[fmt.Sprintf("%p", conn)]] = struct{}{}
	}
//A、B或C不被选中的概率很小
	_, isAselected := selected["a"]
	_, isBselected := selected["b"]
	_, isCselected := selected["c"]
	assert.True(t, isBselected)
	assert.True(t, isCselected)
	assert.True(t, isAselected)

//现在，让每个主机都失败
	shouldConnFail["a"] = true
	shouldConnFail["b"] = true
	shouldConnFail["c"] = true
	conn, _, err = producer.NewConnection()
	assert.Nil(t, conn)
	assert.Error(t, err)
}

func TestUpdateEndpoints(t *testing.T) {
	t.Parallel()
	conn2Endpoint := make(map[string]string)
	connFactory := func(endpoint string) (*grpc.ClientConn, error) {
		conn := &grpc.ClientConn{}
		conn2Endpoint[fmt.Sprintf("%p", conn)] = endpoint
		return conn, nil
	}
//使用单个终结点创建生产者
	producer := NewConnectionProducer(connFactory, []string{"a"})
	conn, a, err := producer.NewConnection()
	assert.NoError(t, err)
	assert.Equal(t, "a", conn2Endpoint[fmt.Sprintf("%p", conn)])
	assert.Equal(t, "a", a)
//Now update the endpoint and check that when we create a new connection,
//我们不连接到上一个端点
	producer.UpdateEndpoints([]string{"b"})
	conn, b, err := producer.NewConnection()
	assert.NoError(t, err)
	assert.NotEqual(t, "a", conn2Endpoint[fmt.Sprintf("%p", conn)])
	assert.Equal(t, "b", conn2Endpoint[fmt.Sprintf("%p", conn)])
	assert.Equal(t, "b", b)
//接下来，确保忽略空更新
	producer.UpdateEndpoints([]string{})
	conn, _, err = producer.NewConnection()
	assert.Equal(t, "b", conn2Endpoint[fmt.Sprintf("%p", conn)])
}

func TestDisableEndpoint(t *testing.T) {
	orgEndpointDisableInterval := EndpointDisableInterval
	EndpointDisableInterval = time.Millisecond * 100
	defer func() { EndpointDisableInterval = orgEndpointDisableInterval }()

	conn2Endpoint := make(map[string]string)
	connFactory := func(endpoint string) (*grpc.ClientConn, error) {
		conn := &grpc.ClientConn{}
		conn2Endpoint[fmt.Sprintf("%p", conn)] = endpoint
		return conn, nil
	}
//使用单端点创建生成器
	producer := NewConnectionProducer(connFactory, []string{"a"})
	conn, a, err := producer.NewConnection()
	assert.NoError(t, err)
	assert.Equal(t, "a", conn2Endpoint[fmt.Sprintf("%p", conn)])
	assert.Equal(t, "a", a)
//现在禁用端点100毫秒
	producer.DisableEndpoint("a")
	_, _, err = producer.NewConnection()
//确保如果只剩下1个端点，我们不会将其列入黑名单。
	assert.NoError(t, err)
//更新终结点-添加终结点“b”
	producer.UpdateEndpoints([]string{"a", "b"})
//再次禁用
	producer.DisableEndpoint("a")
	conn, a, err = producer.NewConnection()
	assert.NoError(t, err)
//确保仅返回B，因为“A”被禁用
	assert.Equal(t, "b", conn2Endpoint[fmt.Sprintf("%p", conn)])
	assert.Equal(t, "b", a)

}
