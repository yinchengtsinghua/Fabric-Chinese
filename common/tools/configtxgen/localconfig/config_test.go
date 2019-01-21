
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package localconfig

import (
	"testing"

	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/hyperledger/fabric/protos/orderer/etcdraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProfile(t *testing.T) {
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	pNames := []string{
		SampleDevModeKafkaProfile,
		SampleDevModeSoloProfile,
		SampleSingleMSPChannelProfile,
		SampleSingleMSPKafkaProfile,
		SampleSingleMSPSoloProfile,
	}
	for _, pName := range pNames {
		t.Run(pName, func(t *testing.T) {
			p := Load(pName)
			assert.NotNil(t, p, "profile should not be nil")
		})
	}
}

func TestLoadProfileWithPath(t *testing.T) {
	devConfigDir, err := configtest.GetDevConfigDir()
	assert.NoError(t, err, "failed to get dev config dir")

	pNames := []string{
		SampleDevModeKafkaProfile,
		SampleDevModeSoloProfile,
		SampleSingleMSPChannelProfile,
		SampleSingleMSPKafkaProfile,
		SampleSingleMSPSoloProfile,
	}
	for _, pName := range pNames {
		t.Run(pName, func(t *testing.T) {
			p := Load(pName, devConfigDir)
			assert.NotNil(t, p, "profile should not be nil")
		})
	}
}

func TestLoadTopLevel(t *testing.T) {
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	topLevel := LoadTopLevel()
	assert.NotNil(t, topLevel.Application, "application should not be nil")
	assert.NotNil(t, topLevel.Capabilities, "capabilities should not be nil")
	assert.NotNil(t, topLevel.Orderer, "orderer should not be nil")
	assert.NotNil(t, topLevel.Organizations, "organizations should not be nil")
	assert.NotNil(t, topLevel.Profiles, "profiles should not be nil")
}

func TestLoadTopLevelWithPath(t *testing.T) {
	devConfigDir, err := configtest.GetDevConfigDir()
	require.NoError(t, err)

	topLevel := LoadTopLevel(devConfigDir)
	assert.NotNil(t, topLevel.Application, "application should not be nil")
	assert.NotNil(t, topLevel.Capabilities, "capabilities should not be nil")
	assert.NotNil(t, topLevel.Orderer, "orderer should not be nil")
	assert.NotNil(t, topLevel.Organizations, "organizations should not be nil")
	assert.NotNil(t, topLevel.Profiles, "profiles should not be nil")
}

func TestConsensusSpecificInit(t *testing.T) {
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	devConfigDir, err := configtest.GetDevConfigDir()
	require.NoError(t, err)

	t.Run("nil orderer type", func(t *testing.T) {
		profile := &Profile{
			Orderer: &Orderer{
				OrdererType: "",
			},
		}
		profile.completeInitialization(devConfigDir)

		assert.Equal(t, profile.Orderer.OrdererType, genesisDefaults.Orderer.OrdererType)
	})

	t.Run("unknown orderer type", func(t *testing.T) {
		profile := &Profile{
			Orderer: &Orderer{
				OrdererType: "unknown",
			},
		}

		assert.Panics(t, func() {
			profile.completeInitialization(devConfigDir)
		})
	})

	t.Run("solo", func(t *testing.T) {
		profile := &Profile{
			Orderer: &Orderer{
				OrdererType: "solo",
			},
		}
		profile.completeInitialization(devConfigDir)
		assert.Nil(t, profile.Orderer.Kafka.Brokers, "Kafka config settings should not be set")
	})

	t.Run("kafka", func(t *testing.T) {
		profile := &Profile{
			Orderer: &Orderer{
				OrdererType: "kafka",
			},
		}
		profile.completeInitialization(devConfigDir)
		assert.NotNil(t, profile.Orderer.Kafka.Brokers, "Kafka config settings should be set")
	})

	t.Run("raft", func(t *testing.T) {
		makeProfile := func(consenters []*etcdraft.Consenter, options *etcdraft.Options) *Profile {
			return &Profile{
				Orderer: &Orderer{
					OrdererType: "etcdraft",
					EtcdRaft: &etcdraft.Metadata{
						Consenters: consenters,
						Options:    options,
					},
				},
			}
		}
		t.Run("EtcdRaft section not specified in profile", func(t *testing.T) {
			profile := &Profile{
				Orderer: &Orderer{
					OrdererType: "etcdraft",
				},
			}

			assert.Panics(t, func() {
				profile.completeInitialization(devConfigDir)
			})
		})

t.Run("nil consenter set", func(t *testing.T) { //应该恐慌
			profile := makeProfile(nil, nil)

			assert.Panics(t, func() {
				profile.completeInitialization(devConfigDir)
			})
		})

		t.Run("single consenter", func(t *testing.T) {
			consenters := []*etcdraft.Consenter{
				{
					Host:          "node-1.example.com",
					Port:          7050,
					ClientTlsCert: []byte("path/to/client/cert"),
					ServerTlsCert: []byte("path/to/server/cert"),
				},
			}

			t.Run("invalid consenters specification", func(t *testing.T) {
				failingConsenterSpecifications := []*etcdraft.Consenter{
{ //遗失主机
						Port:          7050,
						ClientTlsCert: []byte("path/to/client/cert"),
						ServerTlsCert: []byte("path/to/server/cert"),
					},
{ //丢失端口
						Host:          "node-1.example.com",
						ClientTlsCert: []byte("path/to/client/cert"),
						ServerTlsCert: []byte("path/to/server/cert"),
					},
{ //
						Host:          "node-1.example.com",
						Port:          7050,
						ServerTlsCert: []byte("path/to/server/cert"),
					},
{ //
						Host:          "node-1.example.com",
						Port:          7050,
						ClientTlsCert: []byte("path/to/client/cert"),
					},
				}

				for _, consenter := range failingConsenterSpecifications {
					profile := makeProfile([]*etcdraft.Consenter{consenter}, nil)

					assert.Panics(t, func() {
						profile.completeInitialization(devConfigDir)
					})
				}
			})

			t.Run("nil Options", func(t *testing.T) {
				profile := makeProfile(consenters, nil)
				profile.completeInitialization(devConfigDir)

//无需在后续测试中进行测试
				assert.NotNil(t, profile.Orderer.EtcdRaft, "EtcdRaft config settings should be set")
				assert.Equal(t, profile.Orderer.EtcdRaft.Consenters[0].ClientTlsCert, consenters[0].ClientTlsCert,
					"Client TLS cert path should be correctly set")

//此测试上下文的特定断言
				assert.Equal(t, profile.Orderer.EtcdRaft.Options, genesisDefaults.Orderer.EtcdRaft.Options,
					"Options should be set to the default value")
			})

			t.Run("heartbeat tick specified in Options", func(t *testing.T) {
				heartbeatTick := uint32(2)
options := &etcdraft.Options{ //
					HeartbeatTick: heartbeatTick,
				}
				profile := makeProfile(consenters, options)
				profile.completeInitialization(devConfigDir)

//此测试上下文的特定断言
				assert.Equal(t, profile.Orderer.EtcdRaft.Options.HeartbeatTick, heartbeatTick,
					"HeartbeatTick should be set to the specified value")
				assert.Equal(t, profile.Orderer.EtcdRaft.Options.ElectionTick, genesisDefaults.Orderer.EtcdRaft.Options.ElectionTick,
					"ElectionTick should be set to the default value")
			})

			t.Run("election tick specified in Options", func(t *testing.T) {
				electionTick := uint32(20)
options := &etcdraft.Options{ //部分设置，以便检查其他成员是否设置为默认值
					ElectionTick: electionTick,
				}
				profile := makeProfile(consenters, options)
				profile.completeInitialization(devConfigDir)

//此测试上下文的特定断言
				assert.Equal(t, profile.Orderer.EtcdRaft.Options.ElectionTick, electionTick,
					"ElectionTick should be set to the specified value")
				assert.Equal(t, profile.Orderer.EtcdRaft.Options.HeartbeatTick, genesisDefaults.Orderer.EtcdRaft.Options.HeartbeatTick,
					"HeartbeatTick should be set to the default value")
			})

			t.Run("panic on invalid options", func(t *testing.T) {
				options := &etcdraft.Options{
					HeartbeatTick: 2,
					ElectionTick:  1,
				}
				profile := makeProfile(consenters, options)

				assert.Panics(t, func() {
					profile.completeInitialization(devConfigDir)
				})
			})
		})
	})
}
