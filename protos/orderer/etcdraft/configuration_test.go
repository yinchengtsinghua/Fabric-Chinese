
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


package etcdraft_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/orderer/etcdraft"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	md := &etcdraft.Metadata{
		Consenters: []*etcdraft.Consenter{
			{
				Host:          "node-1.example.com",
				Port:          7050,
				ClientTlsCert: []byte("testdata/tls-client-1.pem"),
				ServerTlsCert: []byte("testdata/tls-server-1.pem"),
			},
			{
				Host:          "node-2.example.com",
				Port:          7050,
				ClientTlsCert: []byte("testdata/tls-client-2.pem"),
				ServerTlsCert: []byte("testdata/tls-server-2.pem"),
			},
			{
				Host:          "node-3.example.com",
				Port:          7050,
				ClientTlsCert: []byte("testdata/tls-client-3.pem"),
				ServerTlsCert: []byte("testdata/tls-server-3.pem"),
			},
		},
	}
	packed, err := etcdraft.Marshal(md)
	require.Nil(t, err, "marshalling should succeed")

	unpacked := &etcdraft.Metadata{}
	require.Nil(t, proto.Unmarshal(packed, unpacked), "unmarshalling should succeed")

	var outputCerts, inputCerts [3][]byte
	for i := range unpacked.GetConsenters() {
		outputCerts[i] = []byte(unpacked.GetConsenters()[i].GetClientTlsCert())
		inputCerts[i], _ = ioutil.ReadFile(fmt.Sprintf("testdata/tls-client-%d.pem", i+1))

	}

	for i := 0; i < len(inputCerts)-1; i++ {
		require.NotEqual(t, outputCerts[i+1], outputCerts[i], "expected extracted certs to differ from each other")
	}
}
