
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


package cluster_test

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/common/cluster/mocks"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestParallelStubActivation(t *testing.T) {
	t.Parallel()
//场景：并行激活来自不同goroutine的存根。
	stub := &cluster.Stub{}
	var wg sync.WaitGroup
	n := 100
	wg.Add(n)
	instance := &cluster.RemoteContext{}
	var activationCount int
	maybeCreateInstance := func() (*cluster.RemoteContext, error) {
		activationCount++
		return instance, nil
	}

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			stub.Activate(maybeCreateInstance)
		}()
	}
	wg.Wait()
	activatedStub := stub.RemoteContext
//确保实例是我们存储的引用
//也不是其他的参考资料，也就是说，它不是
//按值复制。
	assert.True(t, activatedStub == instance)
//确保该方法仅被调用一次。
	assert.Equal(t, activationCount, 1)
}

func TestDialerCustomKeepAliveOptions(t *testing.T) {
	t.Parallel()
	ca, err := tlsgen.NewCA()
	assert.NoError(t, err)

	clientKeyPair, err := ca.NewClientCertKeyPair()
	clientConfig := comm.ClientConfig{
		KaOpts: &comm.KeepaliveOptions{
			ClientTimeout: time.Second * 12345,
		},
		Timeout: time.Millisecond * 100,
		SecOpts: &comm.SecureOptions{
			RequireClientCert: true,
			Key:               clientKeyPair.Key,
			Certificate:       clientKeyPair.Cert,
			ServerRootCAs:     [][]byte{ca.CertBytes()},
			UseTLS:            true,
			ClientRootCAs:     [][]byte{ca.CertBytes()},
		},
	}

	dialer := cluster.NewTLSPinningDialer(clientConfig)
	timeout := dialer.Config.Load().(comm.ClientConfig).KaOpts.ClientTimeout
	assert.Equal(t, time.Second*12345, timeout)
}

func TestDialerBadConfig(t *testing.T) {
	t.Parallel()
	emptyCertificate := []byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----")
	dialer := cluster.NewTLSPinningDialer(comm.ClientConfig{SecOpts: &comm.SecureOptions{UseTLS: true, ServerRootCAs: [][]byte{emptyCertificate}}})
	_, err := dialer.Dial("127.0.0.1:8080", func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		return nil
	})
	assert.EqualError(t, err, "error adding root certificate: asn1: syntax error: sequence truncated")
}

func TestDERtoPEM(t *testing.T) {
	t.Parallel()
	ca, err := tlsgen.NewCA()
	assert.NoError(t, err)
	keyPair, err := ca.NewServerCertKeyPair("localhost")
	assert.NoError(t, err)
	assert.Equal(t, cluster.DERtoPEM(keyPair.TLSCert.Raw), string(keyPair.Cert))
}

func TestStandardDialerDialer(t *testing.T) {
	t.Parallel()
	emptyCertificate := []byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----")
	dialer := cluster.NewTLSPinningDialer(comm.ClientConfig{SecOpts: &comm.SecureOptions{UseTLS: true, ServerRootCAs: [][]byte{emptyCertificate}}})
	standardDialer := &cluster.StandardDialer{Dialer: dialer}
	_, err := standardDialer.Dial("127.0.0.1:8080")
	assert.EqualError(t, err, "error adding root certificate: asn1: syntax error: sequence truncated")
}

func TestVerifyBlockSignature(t *testing.T) {
	verifier := &mocks.BlockVerifier{}
	var nilConfigEnvelope *common.ConfigEnvelope
	verifier.On("VerifyBlockSignature", mock.Anything, nilConfigEnvelope).Return(nil)

	block := createBlockChain(3, 3)[0]

//块应具有有效的结构
	err := cluster.VerifyBlockSignature(block, verifier, nil)
	assert.NoError(t, err)

	for _, testCase := range []struct {
		name          string
		mutateBlock   func(*common.Block) *common.Block
		errorContains string
	}{
		{
			name:          "nil metadata",
			errorContains: "no metadata in block",
			mutateBlock: func(block *common.Block) *common.Block {
				block.Metadata = nil
				return block
			},
		},
		{
			name:          "zero metadata slice",
			errorContains: "no metadata in block",
			mutateBlock: func(block *common.Block) *common.Block {
				block.Metadata.Metadata = nil
				return block
			},
		},
		{
			name:          "nil metadata",
			errorContains: "failed unmarshaling medatata for signatures",
			mutateBlock: func(block *common.Block) *common.Block {
				block.Metadata.Metadata[0] = []byte{1, 2, 3}
				return block
			},
		},
		{
			name:          "bad signature header",
			errorContains: "failed unmarshaling signature header",
			mutateBlock: func(block *common.Block) *common.Block {
				metadata := utils.GetMetadataFromBlockOrPanic(block, common.BlockMetadataIndex_SIGNATURES)
				metadata.Signatures[0].SignatureHeader = []byte{1, 2, 3}
				block.Metadata.Metadata[common.BlockMetadataIndex_SIGNATURES] = utils.MarshalOrPanic(metadata)
				return block
			},
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
//创建块的副本
			blockCopy := &common.Block{}
			err := proto.Unmarshal(utils.MarshalOrPanic(block), blockCopy)
			assert.NoError(t, err)
//改变块以破坏它
			blockCopy = testCase.mutateBlock(blockCopy)
			err = cluster.VerifyBlockSignature(blockCopy, verifier, nil)
			assert.Contains(t, err.Error(), testCase.errorContains)
		})
	}
}

func TestVerifyBlockHash(t *testing.T) {
	var start uint64 = 3
	var end uint64 = 23

	verify := func(blockchain []*common.Block) error {
		for i := 0; i < len(blockchain); i++ {
			err := cluster.VerifyBlockHash(i, blockchain)
			if err != nil {
				return err
			}
		}
		return nil
	}

//验证createBlockchain（）是否创建有效的区块链
	assert.NoError(t, verify(createBlockChain(start, end)))

	twoBlocks := createBlockChain(2, 3)
	twoBlocks[0].Header = nil
	assert.EqualError(t, cluster.VerifyBlockHash(1, twoBlocks), "previous block header is nil")

//索引越界
	blockchain := createBlockChain(start, end)
	err := cluster.VerifyBlockHash(100, blockchain)
	assert.EqualError(t, err, "index 100 out of bounds (total 21 blocks)")

	for _, testCase := range []struct {
		name                string
		mutateBlockSequence func([]*common.Block) []*common.Block
		errorContains       string
	}{
		{
			name:          "non consecutive sequences",
			errorContains: "sequences 12 and 666 were received consecutively",
			mutateBlockSequence: func(blockSequence []*common.Block) []*common.Block {
				blockSequence[len(blockSequence)/2].Header.Number = 666
				assignHashes(blockSequence)
				return blockSequence
			},
		},
		{
			name: "data hash mismatch",
			errorContains: "computed hash of block (13) (dcb2ec1c5e482e4914cb953ff8eedd12774b244b12912afbe6001ba5de9ff800)" +
				" doesn't match claimed hash (07)",
			mutateBlockSequence: func(blockSequence []*common.Block) []*common.Block {
				blockSequence[len(blockSequence)/2].Header.DataHash = []byte{7}
				return blockSequence
			},
		},
		{
			name: "prev hash mismatch",
			errorContains: "block 13's hash " +
				"(866351705f1c2f13e10d52ead9d0ca3b80689ede8cc8bf70a6d60c67578323f4) " +
				"mismatches 12's prev block hash (07)",
			mutateBlockSequence: func(blockSequence []*common.Block) []*common.Block {
				blockSequence[len(blockSequence)/2].Header.PreviousHash = []byte{7}
				return blockSequence
			},
		},
		{
			name:          "nil block header",
			errorContains: "missing block header",
			mutateBlockSequence: func(blockSequence []*common.Block) []*common.Block {
				blockSequence[0].Header = nil
				return blockSequence
			},
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			blockchain := createBlockChain(start, end)
			blockchain = testCase.mutateBlockSequence(blockchain)
			err := verify(blockchain)
			assert.EqualError(t, err, testCase.errorContains)
		})
	}
}

func TestVerifyBlocks(t *testing.T) {
	var sigSet1 []*common.SignedData
	var sigSet2 []*common.SignedData

	configEnvelope1 := &common.ConfigEnvelope{
		Config: &common.Config{
			Sequence: 1,
		},
	}
	configEnvelope2 := &common.ConfigEnvelope{
		Config: &common.Config{
			Sequence: 2,
		},
	}
	configTransaction := func(envelope *common.ConfigEnvelope) *common.Envelope {
		return &common.Envelope{
			Payload: utils.MarshalOrPanic(&common.Payload{
				Data: utils.MarshalOrPanic(envelope),
				Header: &common.Header{
					ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
						Type: int32(common.HeaderType_CONFIG),
					}),
				},
			}),
		}
	}

	for _, testCase := range []struct {
		name                string
		configureVerifier   func(*mocks.BlockVerifier)
		mutateBlockSequence func([]*common.Block) []*common.Block
		expectedError       string
	}{
		{
			name: "empty sequence",
			mutateBlockSequence: func(blockSequence []*common.Block) []*common.Block {
				return nil
			},
			expectedError: "buffer is empty",
		},
		{
			name: "prev hash mismatch",
			mutateBlockSequence: func(blockSequence []*common.Block) []*common.Block {
				blockSequence[len(blockSequence)/2].Header.PreviousHash = []byte{7}
				return blockSequence
			},
			expectedError: "block 75's hash " +
				"(5cb4bd1b6a73f81afafd96387bb7ff4473c2425929d0862586f5fbfa12d762dd) " +
				"mismatches 74's prev block hash (07)",
		},
		{
			name: "bad signature",
			mutateBlockSequence: func(blockSequence []*common.Block) []*common.Block {
				return blockSequence
			},
			configureVerifier: func(verifier *mocks.BlockVerifier) {
				var nilEnvelope *common.ConfigEnvelope
				verifier.On("VerifyBlockSignature", mock.Anything, nilEnvelope).Return(errors.New("bad signature"))
			},
			expectedError: "bad signature",
		},
		{
			name: "block that its type cannot be classified",
			mutateBlockSequence: func(blockSequence []*common.Block) []*common.Block {
				blockSequence[len(blockSequence)/2].Data = &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{})},
				}
				blockSequence[len(blockSequence)/2].Header.DataHash = blockSequence[len(blockSequence)/2].Data.Hash()
				assignHashes(blockSequence)
				return blockSequence
			},
			expectedError: "nil header in payload",
		},
		{
			name: "config blocks in the sequence need to be verified and one of them is improperly signed",
			mutateBlockSequence: func(blockSequence []*common.Block) []*common.Block {
				var err error
//将配置事务放入块N/4
				blockSequence[len(blockSequence)/4].Data = &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(configTransaction(configEnvelope1))},
				}
				blockSequence[len(blockSequence)/4].Header.DataHash = blockSequence[len(blockSequence)/4].Data.Hash()

//将配置事务置于块N/2中
				blockSequence[len(blockSequence)/2].Data = &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(configTransaction(configEnvelope2))},
				}
				blockSequence[len(blockSequence)/2].Header.DataHash = blockSequence[len(blockSequence)/2].Data.Hash()

				assignHashes(blockSequence)

				sigSet1, err = cluster.SignatureSetFromBlock(blockSequence[len(blockSequence)/4])
				assert.NoError(t, err)
				sigSet2, err = cluster.SignatureSetFromBlock(blockSequence[len(blockSequence)/2])
				assert.NoError(t, err)

				return blockSequence
			},
			configureVerifier: func(verifier *mocks.BlockVerifier) {
				var nilEnvelope *common.ConfigEnvelope
//第一个配置块正确验证。
				verifier.On("VerifyBlockSignature", sigSet1, nilEnvelope).Return(nil).Once()
//但是，第二个配置块-验证不正确。
				confEnv1 := &common.ConfigEnvelope{}
				proto.Unmarshal(utils.MarshalOrPanic(configEnvelope1), confEnv1)
				verifier.On("VerifyBlockSignature", sigSet2, confEnv1).Return(errors.New("bad signature")).Once()
			},
			expectedError: "bad signature",
		},
		{
			name: "config block in the sequence needs to be verified, and it isproperly signed",
			mutateBlockSequence: func(blockSequence []*common.Block) []*common.Block {
				var err error
//将配置事务放入块N/4
				blockSequence[len(blockSequence)/4].Data = &common.BlockData{
					Data: [][]byte{utils.MarshalOrPanic(configTransaction(configEnvelope1))},
				}
				blockSequence[len(blockSequence)/4].Header.DataHash = blockSequence[len(blockSequence)/4].Data.Hash()

				assignHashes(blockSequence)

				sigSet1, err = cluster.SignatureSetFromBlock(blockSequence[len(blockSequence)/4])
				assert.NoError(t, err)

				sigSet2, err = cluster.SignatureSetFromBlock(blockSequence[len(blockSequence)-1])
				assert.NoError(t, err)

				return blockSequence
			},
			configureVerifier: func(verifier *mocks.BlockVerifier) {
				var nilEnvelope *common.ConfigEnvelope
				confEnv1 := &common.ConfigEnvelope{}
				proto.Unmarshal(utils.MarshalOrPanic(configEnvelope1), confEnv1)
				verifier.On("VerifyBlockSignature", sigSet1, nilEnvelope).Return(nil).Once()
				verifier.On("VerifyBlockSignature", sigSet2, confEnv1).Return(nil).Once()
			},
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			blockchain := createBlockChain(50, 100)
			blockchain = testCase.mutateBlockSequence(blockchain)
			verifier := &mocks.BlockVerifier{}
			if testCase.configureVerifier != nil {
				testCase.configureVerifier(verifier)
			}
			err := cluster.VerifyBlocks(blockchain, verifier)
			if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
			}
		})
	}
}

func assignHashes(blockchain []*common.Block) {
	for i := 1; i < len(blockchain); i++ {
		blockchain[i].Header.PreviousHash = blockchain[i-1].Header.Hash()
	}
}

func createBlockChain(start, end uint64) []*common.Block {
	newBlock := func(seq uint64) *common.Block {
		sHdr := &common.SignatureHeader{
			Creator: []byte{1, 2, 3},
			Nonce:   []byte{9, 5, 42, 66},
		}
		block := common.NewBlock(seq, nil)
		blockSignature := &common.MetadataSignature{
			SignatureHeader: utils.MarshalOrPanic(sHdr),
		}
		block.Metadata.Metadata[common.BlockMetadataIndex_SIGNATURES] = utils.MarshalOrPanic(&common.Metadata{
			Value: nil,
			Signatures: []*common.MetadataSignature{
				blockSignature,
			},
		})

		txn := utils.MarshalOrPanic(&common.Envelope{
			Payload: utils.MarshalOrPanic(&common.Payload{
				Header: &common.Header{},
			}),
		})
		block.Data.Data = append(block.Data.Data, txn)
		return block
	}
	var blockchain []*common.Block
	for seq := uint64(start); seq <= uint64(end); seq++ {
		block := newBlock(seq)
		block.Data.Data = append(block.Data.Data, make([]byte, 100))
		block.Header.DataHash = block.Data.Hash()
		blockchain = append(blockchain, block)
	}
	assignHashes(blockchain)
	return blockchain
}

func TestEndpointconfigFromConfigBlockGreenPath(t *testing.T) {
	blockBytes, err := ioutil.ReadFile("testdata/mychannel.block")
	assert.NoError(t, err)

	block := &common.Block{}
	assert.NoError(t, proto.Unmarshal(blockBytes, block))

	endpointConfig, err := cluster.EndpointconfigFromConfigBlock(block)
	assert.NoError(t, err)
	assert.Len(t, endpointConfig.TLSRootCAs, 1)
	assert.Equal(t, []string{"orderer.example.com:7050"}, endpointConfig.Endpoints)

	bl, _ := pem.Decode(endpointConfig.TLSRootCAs[0])
	cert, err := x509.ParseCertificate(bl.Bytes)
	assert.NoError(t, err)

	assert.True(t, cert.IsCA)
	assert.Equal(t, "tlsca.example.com", cert.Subject.CommonName)
}

func TestEndpointconfigFromConfigBlockFailures(t *testing.T) {
	t.Run("nil block", func(t *testing.T) {
		certs, err := cluster.EndpointconfigFromConfigBlock(nil)
		assert.Nil(t, certs)
		assert.EqualError(t, err, "nil block")
	})

	t.Run("nil block data", func(t *testing.T) {
		certs, err := cluster.EndpointconfigFromConfigBlock(&common.Block{})
		assert.Nil(t, certs)
		assert.EqualError(t, err, "block data is nil")
	})

	t.Run("no envelope", func(t *testing.T) {
		certs, err := cluster.EndpointconfigFromConfigBlock(&common.Block{
			Data: &common.BlockData{},
		})
		assert.Nil(t, certs)
		assert.EqualError(t, err, "envelope index out of bounds")
	})

	t.Run("bad envelope", func(t *testing.T) {
		certs, err := cluster.EndpointconfigFromConfigBlock(&common.Block{
			Data: &common.BlockData{
				Data: [][]byte{{}},
			},
		})
		assert.Nil(t, certs)
		assert.EqualError(t, err, "failed extracting bundle from envelope: envelope header cannot be nil")
	})
}

func TestClientConfig(t *testing.T) {
	t.Run("Uninitialized dialer", func(t *testing.T) {
		dialer := &cluster.PredicateDialer{}
		_, err := dialer.ClientConfig()
		assert.EqualError(t, err, "client config not initialized")
	})

	t.Run("Wrong type stored", func(t *testing.T) {
		dialer := &cluster.PredicateDialer{}
		dialer.Config.Store("foo")
		_, err := dialer.ClientConfig()
		assert.EqualError(t, err, "value stored is string, not comm.ClientConfig")
	})

	t.Run("Nil secure options", func(t *testing.T) {
		dialer := &cluster.PredicateDialer{}
		dialer.Config.Store(comm.ClientConfig{
			SecOpts: nil,
		})
		_, err := dialer.ClientConfig()
		assert.EqualError(t, err, "SecOpts is nil")
	})

	t.Run("Valid config", func(t *testing.T) {
		dialer := &cluster.PredicateDialer{}
		dialer.Config.Store(comm.ClientConfig{
			SecOpts: &comm.SecureOptions{
				Key: []byte{1, 2, 3},
			},
		})
		cc, err := dialer.ClientConfig()
		assert.NoError(t, err)
		assert.Equal(t, []byte{1, 2, 3}, cc.SecOpts.Key)
	})
}

func TestConfigFromBlockBadInput(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		block         *common.Block
		expectedError string
	}{
		{
			name:          "nil block",
			expectedError: "empty block",
			block:         nil,
		},
		{
			name:          "nil block data",
			expectedError: "empty block",
			block:         &common.Block{},
		},
		{
			name:          "no data in block",
			expectedError: "empty block",
			block:         &common.Block{Data: &common.BlockData{}},
		},
		{
			name:          "invalid envelope in block",
			expectedError: "error unmarshaling Envelope: proto: common.Envelope: illegal tag 0 (wire type 1)",
			block:         &common.Block{Data: &common.BlockData{Data: [][]byte{{1, 2, 3}}}},
		},
		{
			name:          "invalid payload in block envelope",
			expectedError: "error unmarshaling Payload: proto: common.Payload: illegal tag 0 (wire type 1)",
			block: &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
				Payload: []byte{1, 2, 3},
			})}}},
		},
		{
			name:          "nil header in payload",
			expectedError: "nil header in payload",
			block:         &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{})}}},
		},
		{
			name:          "invalid channel header",
			expectedError: "error unmarshaling ChannelHeader: proto: common.ChannelHeader: illegal tag 0 (wire type 1)",
			block: &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
				Payload: utils.MarshalOrPanic(&common.Payload{
					Header: &common.Header{
						ChannelHeader: []byte{1, 2, 3},
					},
				}),
			})}}},
		},
		{
			name:          "invalid config block",
			expectedError: "invalid config envelope: proto: common.ConfigEnvelope: illegal tag 0 (wire type 1)",
			block: &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{
				Payload: utils.MarshalOrPanic(&common.Payload{
					Data: []byte{1, 2, 3},
					Header: &common.Header{
						ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
							Type: int32(common.HeaderType_CONFIG),
						}),
					},
				}),
			})}}},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			conf, err := cluster.ConfigFromBlock(testCase.block)
			assert.Nil(t, conf)
			assert.EqualError(t, err, testCase.expectedError)
		})
	}
}
