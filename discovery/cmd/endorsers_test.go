
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


package discovery_test

import (
	"bytes"
	"testing"

	"github.com/hyperledger/fabric/cmd/common"
	. "github.com/hyperledger/fabric/discovery/client"
	"github.com/hyperledger/fabric/discovery/cmd"
	"github.com/hyperledger/fabric/discovery/cmd/mocks"
	discprotos "github.com/hyperledger/fabric/protos/discovery"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEndorserCmd(t *testing.T) {
	server := "peer0"
	channel := "mychannel"
	stub := &mocks.Stub{}
	parser := &mocks.ResponseParser{}

	t.Run("no server supplied", func(t *testing.T) {
		cmd := discovery.NewEndorsersCmd(stub, parser)
		cmd.SetChannel(&channel)

		err := cmd.Execute(common.Config{})
		assert.Equal(t, err.Error(), "no server specified")
	})

	t.Run("no channel supplied", func(t *testing.T) {
		cmd := discovery.NewEndorsersCmd(stub, parser)
		cmd.SetServer(&server)

		err := cmd.Execute(common.Config{})
		assert.Equal(t, err.Error(), "no channel specified")
	})

	t.Run("Endorsement query with no chaincodes", func(t *testing.T) {
		cmd := discovery.NewEndorsersCmd(stub, parser)
		cmd.SetServer(&server)
		cmd.SetChannel(&channel)

		err := cmd.Execute(common.Config{})
		assert.Contains(t, err.Error(), "invocation chain should not be empty")
	})

	t.Run("Server return error", func(t *testing.T) {
		chaincodes := []string{"mycc"}
		cmd := discovery.NewEndorsersCmd(stub, parser)
		cmd.SetChannel(&channel)
		cmd.SetServer(&server)
		cmd.SetChaincodes(&chaincodes)
		stub.On("Send", server, mock.Anything, mock.Anything).Return(nil, errors.New("deadline exceeded")).Once()

		err := cmd.Execute(common.Config{})
		assert.Contains(t, err.Error(), "deadline exceeded")
	})

	t.Run("Endorsement query with no collections succeeds", func(t *testing.T) {
		chaincodes := []string{"mycc"}
		cmd := discovery.NewEndorsersCmd(stub, parser)
		cmd.SetChannel(&channel)
		cmd.SetServer(&server)
		cmd.SetChaincodes(&chaincodes)
		parser.On("ParseResponse", channel, mock.Anything).Return(nil).Once()
		stub.On("Send", server, mock.Anything, mock.Anything).Return(nil, nil).Once()

		err := cmd.Execute(common.Config{})
		assert.NoError(t, err)
	})

	t.Run("Endorsement query with collections succeeds", func(t *testing.T) {
		chaincodes := []string{"mycc", "yourcc"}
		collections := map[string]string{
			"mycc": "col1,col2",
		}

		stub := &mocks.Stub{}
		cmd := discovery.NewEndorsersCmd(stub, parser)
		cmd.SetChannel(&channel)
		cmd.SetServer(&server)
		cmd.SetChaincodes(&chaincodes)
		cmd.SetCollections(&collections)
		parser.On("ParseResponse", channel, mock.Anything).Return(nil).Once()
		stub.On("Send", server, mock.Anything, mock.Anything).Return(nil, nil).Once().Run(func(arg mock.Arguments) {
//确保存根得到与CLI传入内容对应的请求
			req := arg.Get(2).(*Request)
//确保调用链中的链代码名称匹配
			assert.Equal(t, "mycc", req.Queries[0].GetCcQuery().Interests[0].Chaincodes[0].Name)
//确保调用链中的集合名称匹配
			assert.Equal(t, []string{"col1", "col2"}, req.Queries[0].GetCcQuery().Interests[0].Chaincodes[0].CollectionNames)
		})

		err := cmd.Execute(common.Config{})
		assert.NoError(t, err)
		stub.AssertNumberOfCalls(t, "Send", 1)
	})

	t.Run("Endorsement query with collections that aren't mapped to any chaincode(s)", func(t *testing.T) {
		chaincodes := []string{"mycc", "yourcc"}
		collections := map[string]string{
			"mycc":  "col1,col2",
			"ourcc": "col3",
		}

		stub := &mocks.Stub{}
		cmd := discovery.NewEndorsersCmd(stub, parser)
		cmd.SetChannel(&channel)
		cmd.SetServer(&server)
		cmd.SetChaincodes(&chaincodes)
		cmd.SetCollections(&collections)
		stub.On("Send", server, mock.Anything, mock.Anything).Return(nil, nil).Once()

		err := cmd.Execute(common.Config{})
		assert.Contains(t, err.Error(), "a collection specified chaincode ourcc but it wasn't specified with a chaincode flag")
	})

	t.Run("Endorsement query with collections that aren't mapped to any chaincode(s)", func(t *testing.T) {
		chaincodes := []string{"mycc", "yourcc"}
		collections := map[string]string{
			"mycc":  "col1,col2",
			"ourcc": "col3",
		}

		stub := &mocks.Stub{}
		cmd := discovery.NewEndorsersCmd(stub, parser)
		cmd.SetChannel(&channel)
		cmd.SetServer(&server)
		cmd.SetChaincodes(&chaincodes)
		cmd.SetCollections(&collections)
		stub.On("Send", server, mock.Anything, mock.Anything).Return(nil, nil).Once()

		err := cmd.Execute(common.Config{})
		assert.Contains(t, err.Error(), "a collection specified chaincode ourcc but it wasn't specified with a chaincode flag")
	})
}

func TestParseEndorsementResponse(t *testing.T) {
	buff := &bytes.Buffer{}
	parser := &discovery.EndorserResponseParser{Writer: buff}
	res := &mocks.ServiceResponse{}

	t.Run("Server returns empty response", func(t *testing.T) {
		defer buff.Reset()
		res.On("Raw").Return(&discprotos.Response{}).Once()
		err := parser.ParseResponse("mychannel", res)
		assert.Contains(t, err.Error(), "empty results")
	})

	t.Run("Server returns an error", func(t *testing.T) {
		defer buff.Reset()
		res.On("Raw").Return(&discprotos.Response{
			Results: []*discprotos.QueryResult{
				{
					Result: &discprotos.QueryResult_Error{
						Error: &discprotos.Error{
							Content: "internal error",
						},
					},
				},
			},
		}).Once()
		err := parser.ParseResponse("mychannel", res)
		assert.Contains(t, err.Error(), "internal error")
	})

	t.Run("Server returns a response with the wrong type", func(t *testing.T) {
		defer buff.Reset()
		res.On("Raw").Return(&discprotos.Response{
			Results: []*discprotos.QueryResult{
				{
					Result: &discprotos.QueryResult_Members{
						Members: &discprotos.PeerMembershipResult{PeersByOrg: map[string]*discprotos.Peers{
							"Org1MSP": {},
						}},
					},
				},
			},
		}).Once()
		err := parser.ParseResponse("mychannel", res)
		assert.Contains(t, err.Error(), "server returned response of unexpected type: *discovery.QueryResult")
	})

	t.Run("Server returns a proper response", func(t *testing.T) {
		defer buff.Reset()
		res.On("Raw").Return(&discprotos.Response{
			Results: []*discprotos.QueryResult{
				{
					Result: endorsersResponse,
				},
			},
		}).Once()
		err := parser.ParseResponse("mychannel", res)
		assert.NoError(t, err)
		assert.Equal(t, expectedEndorsersOutput, buff.String())
	})
}

var endorsersResponse = &discprotos.QueryResult_CcQueryRes{
	CcQueryRes: &discprotos.ChaincodeQueryResult{
		Content: []*discprotos.EndorsementDescriptor{
			{
				Chaincode: "mycc",
				EndorsersByGroups: map[string]*discprotos.Peers{
					"Org1MSP": {
						Peers: []*discprotos.Peer{
							{
								Identity: utils.MarshalOrPanic(&msp.SerializedIdentity{
									Mspid:   "Org1MSP",
									IdBytes: []byte("identity"),
								}),
								StateInfo:      stateInfoMessage(100).Envelope,
								MembershipInfo: aliveMessage(0).Envelope,
							},
						},
					},
				},
				Layouts: []*discprotos.Layout{
					{
						QuantitiesByGroup: map[string]uint32{
							"Org1MSP": 2,
						},
					},
				},
			},
		},
	},
}

const expectedEndorsersOutput = `[
	{
		"Chaincode": "mycc",
		"EndorsersByGroups": {
			"Org1MSP": [
				{
					"MSPID": "Org1MSP",
					"LedgerHeight": 100,
					"Endpoint": "p0",
					"Identity": "identity"
				}
			]
		},
		"Layouts": [
			{
				"quantities_by_group": {
					"Org1MSP": 2
				}
			}
		]
	}
]
`
