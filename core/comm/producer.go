
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
	"math/rand"
	"sync"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"google.golang.org/grpc"
)

var logger = flogging.MustGetLogger("ConnProducer")

var EndpointDisableInterval = time.Second * 10

//ConnectionFactory创建到某个端点的连接
type ConnectionFactory func(endpoint string) (*grpc.ClientConn, error)

//ConnectionProducer从一组预定义的
//端点
type ConnectionProducer interface {
//NewConnection创建新连接。
//返回连接，所选端点，成功时为零。
//返回nil，“”，失败时出错
	NewConnection() (*grpc.ClientConn, string, error)
//updateEndpoints更新ConnectionProducer的端点
//作为给定的端点
	UpdateEndpoints(endpoints []string)
//禁用端点从端点删除端点一段时间
	DisableEndpoint(endpoint string)
//GetEndpoints返回对服务终结点的排序
	GetEndpoints() []string
}

type connProducer struct {
	sync.RWMutex
	endpoints         []string
	disabledEndpoints map[string]time.Time
	connect           ConnectionFactory
}

//NewConnectionProducer创建具有给定端点和连接工厂的新ConnectionProducer。
//如果给定的端点切片为空，则返回nil。
func NewConnectionProducer(factory ConnectionFactory, endpoints []string) ConnectionProducer {
	if len(endpoints) == 0 {
		return nil
	}
	return &connProducer{endpoints: endpoints, connect: factory, disabledEndpoints: make(map[string]time.Time)}
}

//NewConnection创建新连接。
//返回连接，所选端点，成功时为零。
//返回nil，“”，失败时出错
func (cp *connProducer) NewConnection() (*grpc.ClientConn, string, error) {
	cp.Lock()
	defer cp.Unlock()

	for endpoint, timeout := range cp.disabledEndpoints {
		if time.Since(timeout) >= EndpointDisableInterval {
			delete(cp.disabledEndpoints, endpoint)
		}
	}

	endpoints := shuffle(cp.endpoints)
	checkedEndpoints := make([]string, 0)
	for _, endpoint := range endpoints {
		if _, ok := cp.disabledEndpoints[endpoint]; !ok {
			checkedEndpoints = append(checkedEndpoints, endpoint)
			conn, err := cp.connect(endpoint)
			if err != nil {
				logger.Error("Failed connecting to", endpoint, ", error:", err)
				continue
			}
			return conn, endpoint, nil
		}
	}
	return nil, "", fmt.Errorf("Could not connect to any of the endpoints: %v", checkedEndpoints)
}

//updateEndpoints更新ConnectionProducer的端点
//作为给定的端点
func (cp *connProducer) UpdateEndpoints(endpoints []string) {
	if len(endpoints) == 0 {
//忽略端点为空的更新
		return
	}
	cp.Lock()
	defer cp.Unlock()

	newDisabled := make(map[string]time.Time)
	for i := range endpoints {
		if startTime, ok := cp.disabledEndpoints[endpoints[i]]; ok {
			newDisabled[endpoints[i]] = startTime
		}
	}
	cp.endpoints = endpoints
	cp.disabledEndpoints = newDisabled
}

func (cp *connProducer) DisableEndpoint(endpoint string) {
	cp.Lock()
	defer cp.Unlock()

	if len(cp.endpoints)-len(cp.disabledEndpoints) == 1 {
		logger.Warning("Only 1 endpoint remained, will not black-list it")
		return
	}

	for _, currEndpoint := range cp.endpoints {
		if currEndpoint == endpoint {
			cp.disabledEndpoints[endpoint] = time.Now()
			break
		}
	}
}

func shuffle(a []string) []string {
	n := len(a)
	returnedSlice := make([]string, n)
	rand.Seed(time.Now().UnixNano())
	indices := rand.Perm(n)
	for i, idx := range indices {
		returnedSlice[i] = a[idx]
	}
	return returnedSlice
}

//GetEndpoints返回用于排序服务的已配置终结点
func (cp *connProducer) GetEndpoints() []string {
	cp.RLock()
	defer cp.RUnlock()
	return cp.endpoints
}
