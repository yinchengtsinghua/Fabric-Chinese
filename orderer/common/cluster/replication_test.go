
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


package cluster_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/configtx"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/mocks/crypto"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/common/cluster/mocks"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestIsReplicationNeeded(t *testing.T) {
	for _, testCase := range []struct {
		name                string
		bootBlock           *common.Block
		systemChannelHeight uint64
		systemChannelError  error
		expectedError       string
		replicationNeeded   bool
	}{
		{
			name:                "no replication needed",
			systemChannelHeight: 100,
			bootBlock:           &common.Block{Header: &common.BlockHeader{Number: 99}},
		},
		{
			name:                "replication is needed - bootstrap block's index equal to height",
			systemChannelHeight: 99,
			bootBlock:           &common.Block{Header: &common.BlockHeader{Number: 99}},
			replicationNeeded:   true,
		},
		{
			name:                "replication is needed - no ledger",
			systemChannelHeight: 0,
			bootBlock:           &common.Block{Header: &common.BlockHeader{Number: 99}},
			replicationNeeded:   true,
		},
		{
			name:               "IO error",
			systemChannelError: errors.New("IO error"),
			expectedError:      "IO error",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ledgerWriter := &mocks.LedgerWriter{}
			ledgerWriter.On("Height").Return(testCase.systemChannelHeight)

			ledgerFactory := &mocks.LedgerFactory{}
			ledgerFactory.On("GetOrCreate", "system").Return(ledgerWriter, testCase.systemChannelError)

			r := cluster.Replicator{
				Logger:        flogging.MustGetLogger("test"),
				BootBlock:     testCase.bootBlock,
				SystemChannel: "system",
				LedgerFactory: ledgerFactory,
			}

			ok, err := r.IsReplicationNeeded()
			if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.replicationNeeded, ok)
			}
		})
	}
}

func TestReplicateChainsFailures(t *testing.T) {
	for _, testCase := range []struct {
		name                    string
		isProbeResponseDelayed  bool
		latestBlockSeqInOrderer uint64
		ledgerFactoryError      error
		appendBlockError        error
		expectedPanic           string
		mutateBlocks            func([]*common.Block)
	}{
		{
			name: "no block received",
			expectedPanic: "Failed pulling system channel: " +
				"failed obtaining the latest block for channel system",
		},
		{
			name: "latest block seq is less than boot block seq",
			expectedPanic: "Failed pulling system channel: " +
				"latest height found among system channel(system) orderers is 19," +
				" but the boot block's sequence is 21",
			latestBlockSeqInOrderer: 18,
		},
		{
			name: "hash chain mismatch",
			expectedPanic: "Failed pulling system channel: " +
				"block header mismatch on sequence 11, " +
				"expected 9cd61b7e9a5ea2d128cc877e5304e7205888175a8032d40b97db7412dca41d9e, got 010203",
			latestBlockSeqInOrderer: 21,
			mutateBlocks: func(systemChannelBlocks []*common.Block) {
				systemChannelBlocks[len(systemChannelBlocks)/2].Header.PreviousHash = []byte{1, 2, 3}
			},
		},
		{
			name: "last pulled block doesn't match the boot block",
			expectedPanic: "Block header mismatch on last system channel block," +
				" expected 8ec93b2ef5ffdc302f0c0e24611be04ad2b17b099a1aeafd7cfb76a95923f146," +
				" got e428decfc78f8e4c97b26da9c16f9d0b73f886dafa80477a0dd9bac7eb14fe7a",
			latestBlockSeqInOrderer: 21,
			mutateBlocks: func(systemChannelBlocks []*common.Block) {
				systemChannelBlocks[21].Header.DataHash = nil
			},
		},
		{
			name:                    "failure in creating ledger",
			latestBlockSeqInOrderer: 21,
			ledgerFactoryError:      errors.New("IO error"),
			expectedPanic:           "Failed to create a ledger for channel system: IO error",
		},
		{
			name:                    "failure in appending a block to the ledger",
			latestBlockSeqInOrderer: 21,
			appendBlockError:        errors.New("IO error"),
			expectedPanic:           "Failed to write block 0: IO error",
		},
		{
			name:                    "failure pulling the system chain",
			latestBlockSeqInOrderer: 21,
			expectedPanic: "Failed pulling system channel: " +
				"failed obtaining the latest block for channel system",
			isProbeResponseDelayed: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			systemChannelBlocks := createBlockChain(0, 21)
			if testCase.mutateBlocks != nil {
				testCase.mutateBlocks(systemChannelBlocks)
			}

			lw := &mocks.LedgerWriter{}
			lw.On("Append", mock.Anything).Return(testCase.appendBlockError)

			lf := &mocks.LedgerFactory{}
			lf.On("GetOrCreate", "system").Return(lw, testCase.ledgerFactoryError)

			osn := newClusterNode(t)
			defer osn.stop()

			dialer := newCountingDialer()
			bp := newBlockPuller(dialer, osn.srv.Address())
			bp.FetchTimeout = time.Millisecond * 100

			cl := &mocks.ChannelLister{}
			cl.On("Channels").Return(nil)
			cl.On("Close")

			r := cluster.Replicator{
				Logger:        flogging.MustGetLogger("test"),
				BootBlock:     systemChannelBlocks[21],
				SystemChannel: "system",
				LedgerFactory: lf,
				Puller:        bp,
				ChannelLister: cl,
			}

			if !testCase.isProbeResponseDelayed {
				osn.enqueueResponse(testCase.latestBlockSeqInOrderer)
				osn.enqueueResponse(testCase.latestBlockSeqInOrderer)
			}
			osn.addExpectProbeAssert()
			osn.addExpectProbeAssert()
			osn.addExpectPullAssert(0)
			for _, block := range systemChannelBlocks {
				osn.blockResponses <- &orderer.DeliverResponse{
					Type: &orderer.DeliverResponse_Block{Block: block},
				}
			}

			assert.PanicsWithValue(t, testCase.expectedPanic, r.ReplicateChains)
			bp.Close()
			dialer.assertAllConnectionsClosed(t)
		})
	}
}

func TestPullerConfigFromTopLevelConfig(t *testing.T) {
	signer := &crypto.LocalSigner{}
	expected := cluster.PullerConfig{
		Channel:             "system",
		MaxTotalBufferBytes: 100,
		Signer:              signer,
		TLSCert:             []byte{3, 2, 1},
		TLSKey:              []byte{1, 2, 3},
		Timeout:             time.Hour,
	}

	topLevelConfig := &localconfig.TopLevel{
		General: localconfig.General{
			Cluster: localconfig.Cluster{
				ReplicationBufferSize: 100,
				RPCTimeout:            time.Hour,
			},
		},
	}

	config := cluster.PullerConfigFromTopLevelConfig("system", topLevelConfig, []byte{1, 2, 3}, []byte{3, 2, 1}, signer)
	assert.Equal(t, expected, config)
}

func TestReplicateChainsChannelClassificationFailure(t *testing.T) {
//情景：我们无法分类我们是否是渠道的一部分，
//所以我们崩溃了，因为这是一个编程错误。

	block30WithConfigBlockOf21 := common.NewBlock(30, nil)
	block30WithConfigBlockOf21.Metadata.Metadata[common.BlockMetadataIndex_LAST_CONFIG] = utils.MarshalOrPanic(&common.Metadata{
		Value: utils.MarshalOrPanic(&common.LastConfig{Index: 21}),
	})

	osn := newClusterNode(t)
	defer osn.stop()
	osn.blockResponses = make(chan *orderer.DeliverResponse, 1000)

	dialer := newCountingDialer()
	bp := newBlockPuller(dialer, osn.srv.Address())
	bp.FetchTimeout = time.Hour

	channelLister := &mocks.ChannelLister{}
	channelLister.On("Channels").Return([]string{"A"})
	channelLister.On("Close")

//我们调查订购者的最新信息块
	osn.addExpectProbeAssert()
	osn.enqueueResponse(30)

//现在再拉一次（先投票，然后真正拉）。
	osn.addExpectProbeAssert()
	osn.enqueueResponse(30)
	osn.addExpectPullAssert(30)
	osn.blockResponses <- &orderer.DeliverResponse{
		Type: &orderer.DeliverResponse_Block{Block: block30WithConfigBlockOf21},
	}
//现在，我们从上一个提取的块中提取最新的配置块。
//事先我们重新连接到订购方，所以我们放置了一个人工信号来关闭服务器端的流，
//以便建立新的流。
	osn.blockResponses <- nil
//订购者最后一个块的序列是30，
	osn.addExpectProbeAssert()
	osn.enqueueResponse(30)
//复制者现在要求21号区块。
	osn.enqueueResponse(21)
	osn.addExpectPullAssert(21)

	r := cluster.Replicator{
		AmIPartOfChannel: func(configBlock *common.Block) error {
			return errors.New("oops")
		},
		Logger:        flogging.MustGetLogger("test"),
		SystemChannel: "system",
		ChannelLister: channelLister,
		Puller:        bp,
	}

	assert.PanicsWithValue(t, "Failed classifying whether I belong to channel A: oops, skipping chain retrieval", func() {
		r.ReplicateChains()
	})

	bp.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestReplicateChainsGreenPath(t *testing.T) {
//场景：系统中有两个通道：A和B。
//因此，我们在A频道，而不是B频道
//我们应该先拉通道A，然后再拉系统通道。

	systemChannelBlocks := createBlockChain(0, 21)
	block30WithConfigBlockOf21 := common.NewBlock(30, nil)
	block30WithConfigBlockOf21.Metadata.Metadata[common.BlockMetadataIndex_LAST_CONFIG] = utils.MarshalOrPanic(&common.Metadata{
		Value: utils.MarshalOrPanic(&common.LastConfig{Index: 21}),
	})

	osn := newClusterNode(t)
	defer osn.stop()
	osn.blockResponses = make(chan *orderer.DeliverResponse, 1000)

	dialer := newCountingDialer()
	bp := newBlockPuller(dialer, osn.srv.Address())
	bp.FetchTimeout = time.Hour

	channelLister := &mocks.ChannelLister{}
	channelLister.On("Channels").Return([]string{"A", "B"})
	channelLister.On("Close")

	amIPartOfChannelMock := &mock.Mock{}
//对于通道A
	amIPartOfChannelMock.On("func2").Return(nil).Once()
//对于B频道
	amIPartOfChannelMock.On("func2").Return(cluster.ErrNotInChannel).Once()

//22表示系统信道，31表示信道A
	blocksCommittedToLedger := make(chan *common.Block, 22+31)

	lw := &mocks.LedgerWriter{}
	lw.On("Append", mock.Anything).Return(nil).Run(func(arg mock.Arguments) {
		blocksCommittedToLedger <- arg.Get(0).(*common.Block)
	})

	lf := &mocks.LedgerFactory{}
	lf.On("Close")
	lf.On("GetOrCreate", "A").Return(lw, nil)
	lf.On("GetOrCreate", "B").Return(lw, nil)
	lf.On("GetOrCreate", "system").Return(lw, nil)

	r := cluster.Replicator{
		LedgerFactory: lf,
		AmIPartOfChannel: func(configBlock *common.Block) error {
			return amIPartOfChannelMock.Called().Error(0)
		},
		Logger:        flogging.MustGetLogger("test"),
		SystemChannel: "system",
		ChannelLister: channelLister,
		Puller:        bp,
		BootBlock:     systemChannelBlocks[21],
	}

	for _, channel := range []string{"A", "B"} {
		channel := channel
//首先，订购者需要弄清楚它是否在通道中，
//所以它找到所有订购者要获取的最新块
//最新的配置块，看看它是否在同意者中。

//订购方要求对当前通道的最后一个块进行轮询
		osn.seekAssertions <- func(info *orderer.SeekInfo, actualChannel string) {
//确保搜索到正确的频道
			assert.NotNil(osn.t, info.GetStart().GetNewest())
			assert.Equal(t, channel, actualChannel)
		}

//排序器返回其最后一个块为30。
//这需要通过比较所有订购者来获得最新的高度。
		osn.enqueueResponse(30)

//首先，我们从某个订购者那里对先前得到的块序列进行轮询。
		osn.addExpectProbeAssert()
		osn.enqueueResponse(30)

//然后从第一个订购者那里拉出块。
		osn.addExpectPullAssert(30)
		osn.blockResponses <- &orderer.DeliverResponse{
			Type: &orderer.DeliverResponse_Block{Block: block30WithConfigBlockOf21},
		}
//最后一个配置块通过重新连接到医嘱者而被拉出。
		osn.blockResponses <- nil
//订购者最后一个块的序列是30，
		osn.addExpectProbeAssert()
		osn.enqueueResponse(30)
//复制者现在要求21号区块。
		osn.enqueueResponse(21)
		osn.addExpectPullAssert(21)
//在尝试拉下一个块之前，我们总是先关闭连接
		osn.blockResponses <- nil
	}

//接下来，复制者计算出链的最新块序列
//知道什么时候拉

//我们只希望对通道A进行探测，因为通道B不在通道中
	osn.seekAssertions <- func(info *orderer.SeekInfo, actualChannel string) {
//确保搜索到正确的频道
		assert.NotNil(osn.t, info.GetStart().GetNewest())
		assert.Equal(t, "A", actualChannel)
	}
	osn.enqueueResponse(30)
//从这一点开始，我们拉动链条的滑轮。
	osn.enqueueResponse(30)
	osn.addExpectProbeAssert()
	osn.addExpectPullAssert(0)
//在腹部排31个街区
	for _, block := range createBlockChain(0, 30) {
		osn.blockResponses <- &orderer.DeliverResponse{
			Type: &orderer.DeliverResponse_Block{Block: block},
		}
	}
//通知订购方停止向我们发送数据块，因为我们要重新连接
//去问下一个频道
	osn.blockResponses <- nil

//现在我们定义系统通道的断言
//系统通道的拉断言
	osn.seekAssertions <- func(info *orderer.SeekInfo, actualChannel string) {
//确保搜索进入系统通道。
		assert.NotNil(osn.t, info.GetStart().GetNewest())
		assert.Equal(t, "system", actualChannel)
	}
	osn.blockResponses <- &orderer.DeliverResponse{
		Type: &orderer.DeliverResponse_Block{Block: systemChannelBlocks[21]},
	}
	osn.addExpectProbeAssert()
	osn.enqueueResponse(21)
	osn.addExpectPullAssert(0)
	for _, block := range systemChannelBlocks {
		osn.blockResponses <- &orderer.DeliverResponse{
			Type: &orderer.DeliverResponse_Block{Block: block},
		}
	}

//所有的工作都是在这里完成的。
//以上几行都是断言和准备
//测试的预期流程。
	r.ReplicateChains()

//我们复制了链条，所以剩下的就是确保
//块已按顺序提交，我们期望的所有块
//提交（针对通道A和系统通道）。
	close(blocksCommittedToLedger)
	assert.Len(t, blocksCommittedToLedger, cap(blocksCommittedToLedger))
//计算通道A的块数
	var expectedSequence uint64
	for block := range blocksCommittedToLedger {
		assert.Equal(t, expectedSequence, block.Header.Number)
		expectedSequence++
		if expectedSequence == 31 {
			break
		}
	}

//统计系统通道的块数
	expectedSequence = uint64(0)
	for block := range blocksCommittedToLedger {
		assert.Equal(t, expectedSequence, block.Header.Number)
		expectedSequence++
	}

	bp.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestParticipant(t *testing.T) {
	for _, testCase := range []struct {
		name                      string
		heightsByEndpointsReturns map[string]uint64
		latestBlockSeq            uint64
		latestBlock               *common.Block
		latestConfigBlockSeq      uint64
		latestConfigBlock         *common.Block
		expectedError             string
		predicateReturns          error
	}{
		{
			name:          "No available orderer",
			expectedError: "no available orderer",
		},
		{
			name: "Pulled block has no metadata",
			heightsByEndpointsReturns: map[string]uint64{
				"orderer.example.com:7050": 100,
			},
			latestBlockSeq: uint64(99),
			latestBlock:    &common.Block{},
			expectedError:  "no metadata in block",
		},
		{
			name: "Pulled block has no last config sequence in metadata",
			heightsByEndpointsReturns: map[string]uint64{
				"orderer.example.com:7050": 100,
			},
			latestBlockSeq: uint64(99),
			latestBlock: &common.Block{
				Metadata: &common.BlockMetadata{
					Metadata: [][]byte{{1, 2, 3}},
				},
			},
			expectedError: "no metadata in block",
		},
		{
			name: "Pulled block's metadata is malformed",
			heightsByEndpointsReturns: map[string]uint64{
				"orderer.example.com:7050": 100,
			},
			latestBlockSeq: uint64(99),
			latestBlock: &common.Block{
				Metadata: &common.BlockMetadata{
					Metadata: [][]byte{{1, 2, 3}, {1, 2, 3}},
				},
			},
			expectedError: "error unmarshaling metadata from" +
				" block at index [LAST_CONFIG]: proto: common.Metadata: illegal tag 0 (wire type 1)",
		},
		{
			name: "Pulled block's metadata is valid and has a last config",
			heightsByEndpointsReturns: map[string]uint64{
				"orderer.example.com:7050": 100,
			},
			latestBlockSeq: uint64(99),
			latestBlock: &common.Block{
				Metadata: &common.BlockMetadata{
					Metadata: [][]byte{{1, 2, 3}, utils.MarshalOrPanic(&common.Metadata{
						Value: utils.MarshalOrPanic(&common.LastConfig{
							Index: 42,
						}),
					})},
				},
			},
			latestConfigBlockSeq: 42,
			latestConfigBlock:    &common.Block{Header: &common.BlockHeader{Number: 42}},
			predicateReturns:     cluster.ErrNotInChannel,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			configBlocks := make(chan *common.Block, 1)
			predicate := func(configBlock *common.Block) error {
				configBlocks <- configBlock
				return testCase.predicateReturns
			}
			puller := &mocks.ChainPuller{}
			puller.On("HeightsByEndpoints").Return(testCase.heightsByEndpointsReturns)
			puller.On("PullBlock", testCase.latestBlockSeq).Return(testCase.latestBlock)
			puller.On("PullBlock", testCase.latestConfigBlockSeq).Return(testCase.latestConfigBlock)
			puller.On("Close")

			err := cluster.Participant(puller, predicate)
			if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
				assert.Len(t, configBlocks, 0)
			} else {
				assert.Len(t, configBlocks, 1)
				assert.Equal(t, err, testCase.predicateReturns)
			}
		})
	}
}

func TestBlockPullerFromConfigBlockFailures(t *testing.T) {
	blockBytes, err := ioutil.ReadFile("testdata/mychannel.block")
	assert.NoError(t, err)

	validBlock := &common.Block{}
	assert.NoError(t, proto.Unmarshal(blockBytes, validBlock))

	for _, testCase := range []struct {
		name         string
		expectedErr  string
		pullerConfig cluster.PullerConfig
		block        *common.Block
	}{
		{
			name:        "nil block",
			expectedErr: "nil block",
		},
		{
			name:        "invalid block",
			expectedErr: "block data is nil",
			block:       &common.Block{},
		},
		{
			name: "bad envelope inside block",
			expectedErr: "failed extracting bundle from envelope: " +
				"failed to unmarshal payload from envelope: " +
				"error unmarshaling Payload: " +
				"proto: common.Payload: illegal tag 0 (wire type 1)",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: []byte{1, 2, 3},
					})},
				},
			},
		},
		{
			name:        "invalid TLS certificate",
expectedErr: "unable to decode TLS certificate PEM: ///
			block:       validBlock,
			pullerConfig: cluster.PullerConfig{
				TLSCert: []byte{255, 255, 255},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			bp, err := cluster.BlockPullerFromConfigBlock(testCase.pullerConfig, testCase.block)
			assert.EqualError(t, err, testCase.expectedErr)
			assert.Nil(t, bp)
		})
	}
}

func TestBlockPullerFromConfigBlockGreenPath(t *testing.T) {
	caCert, err := ioutil.ReadFile(filepath.Join("testdata", "ca.crt"))
	assert.NoError(t, err)

	tlsCert, err := ioutil.ReadFile(filepath.Join("testdata", "server.crt"))
	assert.NoError(t, err)

	tlsKey, err := ioutil.ReadFile(filepath.Join("testdata", "server.key"))
	assert.NoError(t, err)

	osn := newClusterNode(t)
	osn.srv.Stop()
//用TLS服务器替换GRPC服务器
	osn.srv, err = comm.NewGRPCServer("127.0.0.1:0", comm.ServerConfig{
		SecOpts: &comm.SecureOptions{
			Key:               tlsKey,
			RequireClientCert: true,
			Certificate:       tlsCert,
			ClientRootCAs:     [][]byte{caCert},
			UseTLS:            true,
		},
	})
	assert.NoError(t, err)
	orderer.RegisterAtomicBroadcastServer(osn.srv.Server(), osn)
//然后开始
	go osn.srv.Start()
	defer osn.stop()

//从有效的配置块开始
	blockBytes, err := ioutil.ReadFile(filepath.Join("testdata", "mychannel.block"))
	assert.NoError(t, err)

	validBlock := &common.Block{}
	assert.NoError(t, proto.Unmarshal(blockBytes, validBlock))

//并向其中注入一个127.0.0.1订购方端点端点和一个新的TLS CA证书。
	injectTLSCACert(t, validBlock, caCert)
	injectOrdererEndpoint(t, validBlock, osn.srv.Address())
	validBlock.Header.DataHash = validBlock.Data.Hash()

	blockMsg := &orderer.DeliverResponse_Block{
		Block: validBlock,
	}

	osn.blockResponses <- &orderer.DeliverResponse{
		Type: blockMsg,
	}

	osn.blockResponses <- &orderer.DeliverResponse{
		Type: blockMsg,
	}

	bp, err := cluster.BlockPullerFromConfigBlock(cluster.PullerConfig{
		TLSCert:             tlsCert,
		TLSKey:              tlsKey,
		MaxTotalBufferBytes: 1,
		Channel:             "mychannel",
		Signer:              &crypto.LocalSigner{},
		Timeout:             time.Second,
	}, validBlock)
	assert.NoError(t, err)
	defer bp.Close()

	osn.addExpectProbeAssert()
	osn.addExpectPullAssert(0)

	block := bp.PullBlock(0)
	assert.Equal(t, uint64(0), block.Header.Number)
}

func TestNoopBlockVerifier(t *testing.T) {
	v := &cluster.NoopBlockVerifier{}
	assert.Nil(t, v.VerifyBlockSignature(nil, nil))
}

func injectOrdererEndpoint(t *testing.T, block *common.Block, endpoint string) {
	ordererAddresses := channelconfig.OrdererAddressesValue([]string{endpoint})
//展开层直到到达订购者地址
	env, err := utils.ExtractEnvelope(block, 0)
	assert.NoError(t, err)
	payload, err := utils.ExtractPayload(env)
	assert.NoError(t, err)
	confEnv, err := configtx.UnmarshalConfigEnvelope(payload.Data)
	assert.NoError(t, err)
//替换订购者地址
	confEnv.Config.ChannelGroup.Values[ordererAddresses.Key()].Value = utils.MarshalOrPanic(ordererAddresses.Value())
//把它放回街区
	payload.Data = utils.MarshalOrPanic(confEnv)
	env.Payload = utils.MarshalOrPanic(payload)
	block.Data.Data[0] = utils.MarshalOrPanic(env)
}

func injectTLSCACert(t *testing.T, block *common.Block, tlsCA []byte) {
//展开层，直到到达TLS CA证书
	env, err := utils.ExtractEnvelope(block, 0)
	assert.NoError(t, err)
	payload, err := utils.ExtractPayload(env)
	assert.NoError(t, err)
	confEnv, err := configtx.UnmarshalConfigEnvelope(payload.Data)
	assert.NoError(t, err)
	mspKey := confEnv.Config.ChannelGroup.Groups[channelconfig.OrdererGroupKey].Groups["OrdererOrg"].Values[channelconfig.MSPKey]
	rawMSPConfig := mspKey.Value
	mspConf := &msp.MSPConfig{}
	proto.Unmarshal(rawMSPConfig, mspConf)
	fabricMSPConf := &msp.FabricMSPConfig{}
	proto.Unmarshal(mspConf.Config, fabricMSPConf)
//将tls根证书替换为给定的根证书
	fabricMSPConf.TlsRootCerts = [][]byte{tlsCA}
//把它放回街区
	mspConf.Config = utils.MarshalOrPanic(fabricMSPConf)
	mspKey.Value = utils.MarshalOrPanic(mspConf)
	payload.Data = utils.MarshalOrPanic(confEnv)
	env.Payload = utils.MarshalOrPanic(payload)
	block.Data.Data[0] = utils.MarshalOrPanic(env)
}

func TestIsNewChannelBlock(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		expectedErr  string
		returnedName string
		block        *common.Block
	}{
		{
			name:        "nil block",
			expectedErr: "nil block",
		},
		{
			name:        "no data section in block",
			expectedErr: "block data is nil",
			block:       &common.Block{},
		},
		{
			name: "corrupt envelope in block",
			expectedErr: "block data does not carry an" +
				" envelope at index 0: error unmarshaling Envelope: proto: common.Envelope: illegal tag 0 (wire type 1)",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{{1, 2, 3}},
				},
			},
		},
		{
			name:        "corrupt payload in envelope",
			expectedErr: "no payload in envelope: proto: common.Payload: illegal tag 0 (wire type 1)",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: []byte{1, 2, 3},
					})},
				},
			},
		},
		{
			name:        "no header in block",
			expectedErr: "nil header in payload",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: utils.MarshalOrPanic(&common.Payload{}),
					})},
				},
			},
		},
		{
			name: "corrupt channel header",
			expectedErr: "error unmarshaling ChannelHeader:" +
				" proto: common.ChannelHeader: illegal tag 0 (wire type 1)",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: utils.MarshalOrPanic(&common.Payload{
							Header: &common.Header{
								ChannelHeader: []byte{1, 2, 3},
							},
						}),
					})},
				},
			},
		},
		{
			name:        "not an orderer transaction",
			expectedErr: "",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: utils.MarshalOrPanic(&common.Payload{
							Header: &common.Header{
								ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
									Type: int32(common.HeaderType_CONFIG_UPDATE),
								}),
							},
						}),
					})},
				},
			},
		},
		{
			name:        "orderer transaction with corrupt inner envelope",
			expectedErr: "error unmarshaling Envelope: proto: common.Envelope: illegal tag 0 (wire type 1)",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: utils.MarshalOrPanic(&common.Payload{
							Header: &common.Header{
								ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
									Type: int32(common.HeaderType_ORDERER_TRANSACTION),
								}),
							},
							Data: []byte{1, 2, 3},
						}),
					})},
				},
			},
		},
		{
			name:        "orderer transaction with corrupt inner payload",
			expectedErr: "error unmarshaling Payload: proto: common.Payload: illegal tag 0 (wire type 1)",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: utils.MarshalOrPanic(&common.Payload{
							Header: &common.Header{
								ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
									Type: int32(common.HeaderType_ORDERER_TRANSACTION),
								}),
							},
							Data: utils.MarshalOrPanic(&common.Envelope{
								Payload: []byte{1, 2, 3},
							}),
						}),
					})},
				},
			},
		},
		{
			name:        "orderer transaction with nil inner header",
			expectedErr: "inner payload's header is nil",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: utils.MarshalOrPanic(&common.Payload{
							Header: &common.Header{
								ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
									Type: int32(common.HeaderType_ORDERER_TRANSACTION),
								}),
							},
							Data: utils.MarshalOrPanic(&common.Envelope{
								Payload: utils.MarshalOrPanic(&common.Payload{}),
							}),
						}),
					})},
				},
			},
		},
		{
			name:        "orderer transaction with corrupt inner channel header",
			expectedErr: "error unmarshaling ChannelHeader: proto: common.ChannelHeader: illegal tag 0 (wire type 1)",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: utils.MarshalOrPanic(&common.Payload{
							Header: &common.Header{
								ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
									Type: int32(common.HeaderType_ORDERER_TRANSACTION),
								}),
							},
							Data: utils.MarshalOrPanic(&common.Envelope{
								Payload: utils.MarshalOrPanic(&common.Payload{
									Header: &common.Header{
										ChannelHeader: []byte{1, 2, 3},
									},
								}),
							}),
						}),
					})},
				},
			},
		},
		{
			name:        "orderer transaction that is not a config, but a config update",
			expectedErr: "",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: utils.MarshalOrPanic(&common.Payload{
							Header: &common.Header{
								ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
									Type: int32(common.HeaderType_ORDERER_TRANSACTION),
								}),
							},
							Data: utils.MarshalOrPanic(&common.Envelope{
								Payload: utils.MarshalOrPanic(&common.Payload{
									Header: &common.Header{
										ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
											Type: int32(common.HeaderType_CONFIG_UPDATE),
										}),
									},
								}),
							}),
						}),
					})},
				},
			},
		},
		{
			expectedErr: "",
			name:        "orderer transaction that is a system channel config block",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: utils.MarshalOrPanic(&common.Payload{
							Header: &common.Header{
								ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
									ChannelId: "systemChannel",
									Type:      int32(common.HeaderType_ORDERER_TRANSACTION),
								}),
							},
							Data: utils.MarshalOrPanic(&common.Envelope{
								Payload: utils.MarshalOrPanic(&common.Payload{
									Header: &common.Header{
										ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
											Type:      int32(common.HeaderType_CONFIG),
											ChannelId: "systemChannel",
										}),
									},
								}),
							}),
						}),
					})},
				},
			},
		},
		{
			name:         "orderer transaction that creates a new application channel",
			expectedErr:  "",
			returnedName: "notSystemChannel",
			block: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
						Payload: utils.MarshalOrPanic(&common.Payload{
							Header: &common.Header{
								ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
									ChannelId: "systemChannel",
									Type:      int32(common.HeaderType_ORDERER_TRANSACTION),
								}),
							},
							Data: utils.MarshalOrPanic(&common.Envelope{
								Payload: utils.MarshalOrPanic(&common.Payload{
									Header: &common.Header{
										ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
											Type:      int32(common.HeaderType_CONFIG),
											ChannelId: "notSystemChannel",
										}),
									},
								}),
							}),
						}),
					})},
				},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			channelName, err := cluster.IsNewChannelBlock(testCase.block)
			if testCase.expectedErr != "" {
				assert.EqualError(t, err, testCase.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.returnedName, channelName)
		})
	}
}

func TestChannels(t *testing.T) {
	makeBlock := func(outerChannelName, innerChannelName string) *common.Block {
		return &common.Block{
			Header: &common.BlockHeader{},
			Data: &common.BlockData{
				Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
					Payload: utils.MarshalOrPanic(&common.Payload{
						Header: &common.Header{
							ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
								ChannelId: outerChannelName,
								Type:      int32(common.HeaderType_ORDERER_TRANSACTION),
							}),
						},
						Data: utils.MarshalOrPanic(&common.Envelope{
							Payload: utils.MarshalOrPanic(&common.Payload{
								Header: &common.Header{
									ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
										Type:      int32(common.HeaderType_CONFIG),
										ChannelId: innerChannelName,
									}),
								},
							}),
						}),
					}),
				})},
			},
		}
	}

	for _, testCase := range []struct {
		name               string
		prepareSystemChain func(systemChain []*common.Block)
		assertion          func(t *testing.T, ci *cluster.ChainInspector)
	}{
		{
			name: "happy path - artificial blocks",
			prepareSystemChain: func(systemChain []*common.Block) {
				assignHashes(systemChain)
			},
			assertion: func(t *testing.T, ci *cluster.ChainInspector) {
				actual := ci.Channels()
//断言以任何顺序返回返回的通道
				assert.Contains(t, [][]string{{"mychannel", "mychannel2"}, {"mychannel2", "mychannel"}}, actual)
			},
		},
		{
			name: "happy path - one block is not artificial but real",
			prepareSystemChain: func(systemChain []*common.Block) {
				blockbytes, err := ioutil.ReadFile(filepath.Join("testdata", "block3.pb"))
				assert.NoError(t, err)
				block := &common.Block{}
				err = proto.Unmarshal(blockbytes, block)
				assert.NoError(t, err)

				systemChain[len(systemChain)/2] = block
				assignHashes(systemChain)
			},
			assertion: func(t *testing.T, ci *cluster.ChainInspector) {
				actual := ci.Channels()
//断言以任何顺序返回返回的通道
				assert.Contains(t, [][]string{{"mychannel", "bar"}, {"bar", "mychannel"}}, actual)
			},
		},
		{
			name: "bad path - pulled chain's last block hash doesn't match the last config block",
			prepareSystemChain: func(systemChain []*common.Block) {
				assignHashes(systemChain)
				systemChain[len(systemChain)-1].Header.PreviousHash = nil
			},
			assertion: func(t *testing.T, ci *cluster.ChainInspector) {
				panicValue := "System channel pulled doesn't match the boot last config block:" +
					" block 4's hash (34762d9deefdea2514a85663856e92b5c7e1ae4669e6265b27b079d1f320e741)" +
					" mismatches 3's prev block hash ()"
				assert.PanicsWithValue(t, panicValue, func() {
					ci.Channels()
				})
			},
		},
		{
			name: "bad path - hash chain mismatch",
			prepareSystemChain: func(systemChain []*common.Block) {
				assignHashes(systemChain)
				systemChain[len(systemChain)/2].Header.PreviousHash = nil
			},
			assertion: func(t *testing.T, ci *cluster.ChainInspector) {
				panicValue := "Claimed previous hash of block 3 is  but actual previous " +
					"hash is ab6be2effec106c0324f9d6b1af2cf115c60c3f60e250658362991cb8e195a50"
				assert.PanicsWithValue(t, panicValue, func() {
					ci.Channels()
				})
			},
		},
		{
			name: "bad path - a block cannot be classified",
			prepareSystemChain: func(systemChain []*common.Block) {
				assignHashes(systemChain)
				systemChain[len(systemChain)-2].Data.Data = [][]byte{{1, 2, 3}}
			},
			assertion: func(t *testing.T, ci *cluster.ChainInspector) {
				panicValue := "Failed classifying block 3 : block data does not carry" +
					" an envelope at index 0: error unmarshaling Envelope: " +
					"proto: common.Envelope: illegal tag 0 (wire type 1)"
				assert.PanicsWithValue(t, panicValue, func() {
					ci.Channels()
				})
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			systemChain := []*common.Block{
				makeBlock("systemChannel", "systemChannel"),
				makeBlock("systemChannel", "mychannel"),
				makeBlock("systemChannel", "mychannel2"),
				makeBlock("systemChannel", "systemChannel"),
			}

			for i := 0; i < len(systemChain); i++ {
				systemChain[i].Header.DataHash = systemChain[i].Data.Hash()
				systemChain[i].Header.Number = uint64(i + 1)
			}
			testCase.prepareSystemChain(systemChain)
			puller := &mocks.ChainPuller{}
			puller.On("Close")
			for seq := uint64(1); int(seq) <= len(systemChain); seq++ {
				puller.On("PullBlock", seq).Return(systemChain[int(seq)-1])
			}

			ci := &cluster.ChainInspector{
				Logger:          flogging.MustGetLogger("test"),
				Puller:          puller,
				LastConfigBlock: systemChain[len(systemChain)-1],
			}
			defer puller.AssertNumberOfCalls(t, "Close", 1)
			defer ci.Close()
			testCase.assertion(t, ci)
		})
	}
}
