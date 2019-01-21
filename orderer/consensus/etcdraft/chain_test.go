
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
	"bytes"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"sync"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"github.com/coreos/etcd/raft"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/common/flogging"
	mockconfig "github.com/hyperledger/fabric/common/mocks/config"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/consensus/etcdraft"
	"github.com/hyperledger/fabric/orderer/consensus/etcdraft/mocks"
	consensusmocks "github.com/hyperledger/fabric/orderer/consensus/mocks"
	mockblockcutter "github.com/hyperledger/fabric/orderer/mocks/common/blockcutter"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	raftprotos "github.com/hyperledger/fabric/protos/orderer/etcdraft"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const (
	interval            = time.Second
	LongEventualTimeout = 5 * time.Second
	ELECTION_TICK       = 2
	HEARTBEAT_TICK      = 1
)

func init() {
	factory.InitFactories(nil)
}

//对于某些测试案例，我们将chmod file/dir设置为测试由外来权限引起的失败。
//但是，如果测试作为根（即在容器中）运行，则这不起作用。
func skipIfRoot() {
	u, err := user.Current()
	Expect(err).NotTo(HaveOccurred())
	if u.Uid == "0" {
		Skip("you are running test as root, there's no way to make files unreadable")
	}
}

var _ = Describe("Chain", func() {
	var (
		env       *common.Envelope
		channelID string
		tlsCA     tlsgen.CA
		logger    *flogging.FabricLogger
	)

	BeforeEach(func() {
		tlsCA, _ = tlsgen.NewCA()
		channelID = "test-channel"
		logger = flogging.NewFabricLogger(zap.NewNop())
		env = &common.Envelope{
			Payload: marshalOrPanic(&common.Payload{
				Header: &common.Header{ChannelHeader: marshalOrPanic(&common.ChannelHeader{Type: int32(common.HeaderType_MESSAGE), ChannelId: channelID})},
				Data:   []byte("TEST_MESSAGE"),
			}),
		}
	})

	Describe("Single Raft node", func() {
		var (
			configurator      *mocks.Configurator
			consenterMetadata *raftprotos.Metadata
			clock             *fakeclock.FakeClock
			opts              etcdraft.Options
			support           *consensusmocks.FakeConsenterSupport
			cutter            *mockblockcutter.Receiver
			storage           *raft.MemoryStorage
			observeC          chan uint64
			chain             *etcdraft.Chain
			dataDir           string
			walDir            string
			snapDir           string
			err               error
		)

		BeforeEach(func() {
			configurator = &mocks.Configurator{}
			configurator.On("Configure", mock.Anything, mock.Anything)
			clock = fakeclock.NewFakeClock(time.Now())
			storage = raft.NewMemoryStorage()

			dataDir, err = ioutil.TempDir("", "wal-")
			Expect(err).NotTo(HaveOccurred())
			walDir = path.Join(dataDir, "wal")
			snapDir = path.Join(dataDir, "snapshot")

			observeC = make(chan uint64, 1)

			support = &consensusmocks.FakeConsenterSupport{}
			support.ChainIDReturns(channelID)
			consenterMetadata = createMetadata(1, tlsCA)
			support.SharedConfigReturns(&mockconfig.Orderer{
				BatchTimeoutVal:      time.Hour,
				ConsensusMetadataVal: marshalOrPanic(consenterMetadata),
			})
			cutter = mockblockcutter.NewReceiver()
			support.BlockCutterReturns(cutter)

//用于块创建器初始化
			support.HeightReturns(1)
			support.BlockReturns(getSeedBlock())

			meta := &raftprotos.RaftMetadata{
				Consenters:      map[uint64]*raftprotos.Consenter{},
				NextConsenterId: 1,
			}

			for _, c := range consenterMetadata.Consenters {
				meta.Consenters[meta.NextConsenterId] = c
				meta.NextConsenterId++
			}

			opts = etcdraft.Options{
				RaftID:          1,
				Clock:           clock,
				TickInterval:    interval,
				ElectionTick:    ELECTION_TICK,
				HeartbeatTick:   HEARTBEAT_TICK,
				MaxSizePerMsg:   1024 * 1024,
				MaxInflightMsgs: 256,
				RaftMetadata:    meta,
				Logger:          logger,
				MemoryStorage:   storage,
				WALDir:          walDir,
				SnapDir:         snapDir,
			}
		})

		campaign := func(clock *fakeclock.FakeClock, observeC <-chan uint64) {
			Eventually(func() bool {
				clock.Increment(interval)
				select {
				case <-observeC:
					return true
				default:
					return false
				}
			}, LongEventualTimeout).Should(BeTrue())
		}

		JustBeforeEach(func() {
			chain, err = etcdraft.NewChain(support, opts, configurator, nil, nil, observeC)
			Expect(err).NotTo(HaveOccurred())

			chain.Start()

//当raft节点启动时，它会生成confchange
//要添加自己，需要使用ready（）。
//如果筏板中有待定的配置更改，
//不管经过多少节拍，它都拒绝竞选。
//这在生产代码中不是问题，因为raft.ready
//最终会随着挂钟的前进而消耗掉。
//
//但是，在使用假时钟和
//人工扁虱。而不是无限期地挠木筏直到
//木筏。准备好了就被消耗了，这个检查被添加到间接保证
//第一个confchange实际上被消耗掉了，我们可以安全地
//继续勾选筏FSM。
			Eventually(func() error {
				_, err := storage.Entries(1, 1, 1)
				return err
			}, LongEventualTimeout).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			chain.Halt()
			os.RemoveAll(dataDir)
		})

		Context("when a node starts up", func() {
			It("properly configures the communication layer", func() {
				expectedNodeConfig := nodeConfigFromMetadata(consenterMetadata)
				configurator.AssertCalled(testingInstance, "Configure", channelID, expectedNodeConfig)
			})
		})

		Context("when no Raft leader is elected", func() {
			It("fails to order envelope", func() {
				err := chain.Order(env, 0)
				Expect(err).To(MatchError("no Raft leader"))
			})
		})

		Context("when Raft leader is elected", func() {
			JustBeforeEach(func() {
				campaign(clock, observeC)
			})

			It("fails to order envelope if chain is halted", func() {
				chain.Halt()
				err := chain.Order(env, 0)
				Expect(err).To(MatchError("chain is stopped"))
			})

			It("produces blocks following batch rules", func() {
				close(cutter.Block)

				By("cutting next batch directly")
				cutter.CutNext = true
				err := chain.Order(env, 0)
				Expect(err).NotTo(HaveOccurred())
				Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))

				By("respecting batch timeout")
				cutter.CutNext = false
				timeout := time.Second
				support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})
				err = chain.Order(env, 0)
				Expect(err).NotTo(HaveOccurred())

				clock.WaitForNWatchersAndIncrement(timeout, 2)
				Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
			})

			It("does not reset timer for every envelope", func() {
				close(cutter.Block)

				timeout := time.Second
				support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})

				err := chain.Order(env, 0)
				Expect(err).NotTo(HaveOccurred())
				Eventually(cutter.CurBatch, LongEventualTimeout).Should(HaveLen(1))

				clock.WaitForNWatchersAndIncrement(timeout/2, 2)

				err = chain.Order(env, 0)
				Expect(err).NotTo(HaveOccurred())
				Eventually(cutter.CurBatch, LongEventualTimeout).Should(HaveLen(2))

//第二个信封不应该重置计时器；它应该
//因此，如果我们只增加超时值，它将过期/2
				clock.Increment(timeout / 2)
				Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))
			})

			It("does not write a block if halted before timeout", func() {
				close(cutter.Block)
				timeout := time.Second
				support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})

				err := chain.Order(env, 0)
				Expect(err).NotTo(HaveOccurred())
				Eventually(cutter.CurBatch, LongEventualTimeout).Should(HaveLen(1))

//等待计时器启动
				Eventually(clock.WatcherCount, LongEventualTimeout).Should(Equal(2))

				chain.Halt()
				Consistently(support.WriteBlockCallCount).Should(Equal(0))
			})

			It("stops the timer if a batch is cut", func() {
				close(cutter.Block)

				timeout := time.Second
				support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})

				err := chain.Order(env, 0)
				Expect(err).NotTo(HaveOccurred())
				Eventually(cutter.CurBatch, LongEventualTimeout).Should(HaveLen(1))

				clock.WaitForNWatchersAndIncrement(timeout/2, 2)

				By("force a batch to be cut before timer expires")
				cutter.CutNext = true
				err = chain.Order(env, 0)
				Expect(err).NotTo(HaveOccurred())

				Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))
				b, _ := support.WriteBlockArgsForCall(0)
				Expect(b.Data.Data).To(HaveLen(2))
				Expect(cutter.CurBatch()).To(HaveLen(0))

//这应该会启动一个新的计时器
				cutter.CutNext = false
				err = chain.Order(env, 0)
				Expect(err).NotTo(HaveOccurred())
				Eventually(cutter.CurBatch, LongEventualTimeout).Should(HaveLen(1))

				clock.WaitForNWatchersAndIncrement(timeout/2, 2)
				Consistently(support.WriteBlockCallCount).Should(Equal(1))

				clock.Increment(timeout / 2)

				Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
				b, _ = support.WriteBlockArgsForCall(1)
				Expect(b.Data.Data).To(HaveLen(1))
			})

			It("cut two batches if incoming envelope does not fit into first batch", func() {
				close(cutter.Block)

				timeout := time.Second
				support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})

				err := chain.Order(env, 0)
				Expect(err).NotTo(HaveOccurred())
				Eventually(cutter.CurBatch, LongEventualTimeout).Should(HaveLen(1))

				cutter.IsolatedTx = true
				err = chain.Order(env, 0)
				Expect(err).NotTo(HaveOccurred())

				Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
			})

			Context("revalidation", func() {
				BeforeEach(func() {
					close(cutter.Block)

					timeout := time.Hour
					support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})
					support.SequenceReturns(1)
				})

				It("enqueue if envelope is still valid", func() {
					support.ProcessNormalMsgReturns(1, nil)

					err := chain.Order(env, 0)
					Expect(err).NotTo(HaveOccurred())
					Eventually(cutter.CurBatch, LongEventualTimeout).Should(HaveLen(1))
				})

				It("does not enqueue if envelope is not valid", func() {
					support.ProcessNormalMsgReturns(1, errors.Errorf("Envelope is invalid"))

					err := chain.Order(env, 0)
					Expect(err).NotTo(HaveOccurred())
					Consistently(cutter.CurBatch).Should(HaveLen(0))
				})
			})

			It("unblocks Errored if chain is halted", func() {
				Expect(chain.Errored()).NotTo(Receive())
				chain.Halt()
				Expect(chain.Errored()).Should(BeClosed())
			})

			Describe("Config updates", func() {
				var (
					configEnv *common.Envelope
					configSeq uint64
				)

				Context("when a config update with invalid header comes", func() {

					BeforeEach(func() {
						configEnv = newConfigEnv(channelID,
common.HeaderType_CONFIG_UPDATE, //无效的头；具有config_update头的信封永远不会到达链
							&common.ConfigUpdateEnvelope{ConfigUpdate: []byte("test invalid envelope")})
						configSeq = 0
					})

					It("should throw an error", func() {
						err := chain.Configure(configEnv, configSeq)
						Expect(err).To(MatchError("config transaction has unknown header type"))
					})
				})

				Context("when a type A config update comes", func() {

					Context("for existing channel", func() {

//用于准备医嘱者值
						BeforeEach(func() {
							values := map[string]*common.ConfigValue{
								"BatchTimeout": {
									Version: 1,
									Value: marshalOrPanic(&orderer.BatchTimeout{
										Timeout: "3ms",
									}),
								},
							}
							configEnv = newConfigEnv(channelID,
								common.HeaderType_CONFIG,
								newConfigUpdateEnv(channelID, values),
							)
							configSeq = 0
}) //每个街区之前

						Context("without revalidation (i.e. correct config sequence)", func() {

							Context("without pending normal envelope", func() {
								It("should create a config block and no normal block", func() {
									err := chain.Configure(configEnv, configSeq)
									Expect(err).NotTo(HaveOccurred())
									Eventually(support.WriteConfigBlockCallCount, LongEventualTimeout).Should(Equal(1))
									Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(0))
								})
							})

							Context("with pending normal envelope", func() {
								It("should create a normal block and a config block", func() {
//我们不需要阻止刀具在我们的测试用例中订购，因此关闭这个通道。
									close(cutter.Block)

									By("adding a normal envelope")
									err := chain.Order(env, 0)
									Expect(err).NotTo(HaveOccurred())
									Eventually(cutter.CurBatch, LongEventualTimeout).Should(HaveLen(1))

////clock.waitForNwatchersandCrement（超时，2）

									By("adding a config envelope")
									err = chain.Configure(configEnv, configSeq)
									Expect(err).NotTo(HaveOccurred())

									Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))
									Eventually(support.WriteConfigBlockCallCount, LongEventualTimeout).Should(Equal(1))
								})
							})
						})

						Context("with revalidation (i.e. incorrect config sequence)", func() {

							BeforeEach(func() {
support.SequenceReturns(1) //这导致了再验证
							})

							It("should create config block upon correct revalidation", func() {
support.ProcessConfigMsgReturns(configEnv, 1, nil) //零意味着正确的再验证

								err := chain.Configure(configEnv, configSeq)
								Expect(err).NotTo(HaveOccurred())
								Eventually(support.WriteConfigBlockCallCount, LongEventualTimeout).Should(Equal(1))
							})

							It("should not create config block upon incorrect revalidation", func() {
								support.ProcessConfigMsgReturns(configEnv, 1, errors.Errorf("Invalid config envelope at changed config sequence"))

								err := chain.Configure(configEnv, configSeq)
								Expect(err).NotTo(HaveOccurred())
Consistently(support.WriteConfigBlockCallCount).Should(Equal(0)) //没有调用WriteConfigBlock
							})
						})
					})

					Context("for creating a new channel", func() {

//用于准备医嘱者值
						BeforeEach(func() {
							chainID := "mychannel"
							configEnv = newConfigEnv(chainID,
								common.HeaderType_ORDERER_TRANSACTION,
								&common.ConfigUpdateEnvelope{ConfigUpdate: []byte("test channel creation envelope")})
							configSeq = 0
}) //每个街区之前

						It("should be able to create a channel", func() {
							err := chain.Configure(configEnv, configSeq)
							Expect(err).NotTo(HaveOccurred())
						})
					})
}) //类型A配置的上下文块

				Context("when a type B config update comes", func() {
					Context("updating protocol values", func() {
//用于准备医嘱者值
						BeforeEach(func() {
							values := map[string]*common.ConfigValue{
								"ConsensusType": {
									Version: 1,
									Value: marshalOrPanic(&orderer.ConsensusType{
										Metadata: marshalOrPanic(consenterMetadata),
									}),
								},
							}
							configEnv = newConfigEnv(channelID,
								common.HeaderType_CONFIG,
								newConfigUpdateEnv(channelID, values))
							configSeq = 0

}) //每个街区之前

						It("should be able to process config update of type B", func() {
							err := chain.Configure(configEnv, configSeq)
							Expect(err).NotTo(HaveOccurred())
						})
					})

					Context("updating consenters set by more than one node", func() {
//用于准备医嘱者值
						BeforeEach(func() {
							values := map[string]*common.ConfigValue{
								"ConsensusType": {
									Version: 1,
									Value: marshalOrPanic(&orderer.ConsensusType{
										Metadata: marshalOrPanic(createMetadata(3, tlsCA)),
									}),
								},
							}
							configEnv = newConfigEnv(channelID,
								common.HeaderType_CONFIG,
								newConfigUpdateEnv(channelID, values))
							configSeq = 0

}) //每个街区之前

						It("should fail, since consenters set change is not supported", func() {
							err := chain.Configure(configEnv, configSeq)
							Expect(err).To(MatchError("update of more than one consenters at a time is not supported"))
						})
					})

					Context("updating consenters set by exactly one node", func() {
						It("should be able to process config update adding single node", func() {
							metadata := proto.Clone(consenterMetadata).(*raftprotos.Metadata)
							metadata.Consenters = append(metadata.Consenters, &raftprotos.Consenter{
								Host:          "localhost",
								Port:          7050,
								ServerTlsCert: serverTLSCert(tlsCA),
								ClientTlsCert: clientTLSCert(tlsCA),
							})

							values := map[string]*common.ConfigValue{
								"ConsensusType": {
									Version: 1,
									Value: marshalOrPanic(&orderer.ConsensusType{
										Metadata: marshalOrPanic(metadata),
									}),
								},
							}
							configEnv = newConfigEnv(channelID,
								common.HeaderType_CONFIG,
								newConfigUpdateEnv(channelID, values))
							configSeq = 0

							err := chain.Configure(configEnv, configSeq)
							Expect(err).NotTo(HaveOccurred())
						})

						It("should be able to process config update removing single node", func() {
							metadata := proto.Clone(consenterMetadata).(*raftprotos.Metadata)
//删除其中一个同意者
							metadata.Consenters = metadata.Consenters[1:]
							values := map[string]*common.ConfigValue{
								"ConsensusType": {
									Version: 1,
									Value: marshalOrPanic(&orderer.ConsensusType{
										Metadata: marshalOrPanic(metadata),
									}),
								},
							}
							configEnv = newConfigEnv(channelID,
								common.HeaderType_CONFIG,
								newConfigUpdateEnv(channelID, values))
							configSeq = 0

							err := chain.Configure(configEnv, configSeq)
							Expect(err).NotTo(HaveOccurred())
						})

						It("fail since not allowed to add and remove node at same change", func() {
							metadata := proto.Clone(consenterMetadata).(*raftprotos.Metadata)
//删除其中一个同意者
							metadata.Consenters = metadata.Consenters[1:]
							metadata.Consenters = append(metadata.Consenters, &raftprotos.Consenter{
								Host:          "localhost",
								Port:          7050,
								ServerTlsCert: serverTLSCert(tlsCA),
								ClientTlsCert: clientTLSCert(tlsCA),
							})
							values := map[string]*common.ConfigValue{
								"ConsensusType": {
									Version: 1,
									Value: marshalOrPanic(&orderer.ConsensusType{
										Metadata: marshalOrPanic(metadata),
									}),
								},
							}
							configEnv = newConfigEnv(channelID,
								common.HeaderType_CONFIG,
								newConfigUpdateEnv(channelID, values))
							configSeq = 0

							err := chain.Configure(configEnv, configSeq)
							Expect(err).To(MatchError("update of more than one consenters at a time is not supported"))
						})
					})
				})
			})

			Describe("Crash Fault Tolerance", func() {
				var (
					raftMetadata *raftprotos.RaftMetadata
				)

				BeforeEach(func() {
					tlsCA, _ := tlsgen.NewCA()

					raftMetadata = &raftprotos.RaftMetadata{
						Consenters: map[uint64]*raftprotos.Consenter{
							1: {
								Host:          "localhost",
								Port:          7051,
								ClientTlsCert: clientTLSCert(tlsCA),
								ServerTlsCert: serverTLSCert(tlsCA),
							},
						},
						NextConsenterId: 2,
					}
				})

				Describe("when a chain is started with existing WAL", func() {
					var (
						m1 *raftprotos.RaftMetadata
						m2 *raftprotos.RaftMetadata
					)
					JustBeforeEach(func() {
//为了生成Wal数据，我们启动了一个链，
//订购几个信封，然后停止链条。
						close(cutter.Block)
						cutter.CutNext = true

//用raft将一些数据保存在磁盘上
						err := chain.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))

						_, metadata := support.WriteBlockArgsForCall(0)
						m1 = &raftprotos.RaftMetadata{}
						proto.Unmarshal(metadata, m1)

						err = chain.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))

						_, metadata = support.WriteBlockArgsForCall(1)
						m2 = &raftprotos.RaftMetadata{}
						proto.Unmarshal(metadata, m2)

						chain.Halt()
					})

					It("replays blocks from committed entries", func() {
						c := newChain(10*time.Second, channelID, dataDir, 1, raftMetadata)
						c.init()
						c.Start()
						defer c.Halt()

						Eventually(c.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))

						_, metadata := c.support.WriteBlockArgsForCall(0)
						m := &raftprotos.RaftMetadata{}
						proto.Unmarshal(metadata, m)
						Expect(m.RaftIndex).To(Equal(m1.RaftIndex))

						_, metadata = c.support.WriteBlockArgsForCall(1)
						m = &raftprotos.RaftMetadata{}
						proto.Unmarshal(metadata, m)
						Expect(m.RaftIndex).To(Equal(m2.RaftIndex))

//链条应保持功能正常。
						campaign(c.clock, c.observe)

						c.cutter.CutNext = true

						err := c.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(c.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(3))
					})

					It("only replays blocks after Applied index", func() {
						raftMetadata.RaftIndex = m1.RaftIndex
						c := newChain(10*time.Second, channelID, dataDir, 1, raftMetadata)
						c.init()
						c.Start()
						defer c.Halt()

						Eventually(c.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))

						_, metadata := c.support.WriteBlockArgsForCall(0)
						m := &raftprotos.RaftMetadata{}
						proto.Unmarshal(metadata, m)
						Expect(m.RaftIndex).To(Equal(m2.RaftIndex))

//链条应保持功能正常。
						campaign(c.clock, c.observe)

						c.cutter.CutNext = true

						err := c.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(c.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
					})

					It("does not replay any block if already in sync", func() {
						raftMetadata.RaftIndex = m2.RaftIndex
						c := newChain(10*time.Second, channelID, dataDir, 1, raftMetadata)
						c.init()
						c.Start()
						defer c.Halt()

						Consistently(c.support.WriteBlockCallCount).Should(Equal(0))

//链条应保持功能正常。
						campaign(c.clock, c.observe)

						c.cutter.CutNext = true

						err := c.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(c.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))
					})

					Context("WAL file is not readable", func() {
						It("fails to load wal", func() {
							skipIfRoot()

							files, err := ioutil.ReadDir(walDir)
							Expect(err).NotTo(HaveOccurred())
							for _, f := range files {
								os.Chmod(path.Join(walDir, f.Name()), 0300)
							}

							c, err := etcdraft.NewChain(support, opts, configurator, nil, nil, observeC)
							Expect(c).To(BeNil())
							Expect(err).To(MatchError(ContainSubstring("failed to open existing WAL")))
						})
					})
				})

				Describe("when snapshotting is enabled (snapshot interval is not zero)", func() {
					var (
						m *raftprotos.RaftMetadata

						ledgerLock sync.Mutex
						ledger     []*common.Block
					)

					countFiles := func() int {
						files, err := ioutil.ReadDir(snapDir)
						Expect(err).NotTo(HaveOccurred())
						return len(files)
					}

					BeforeEach(func() {
						opts.SnapInterval = 2
						opts.SnapshotCatchUpEntries = 2

						close(cutter.Block)
						cutter.CutNext = true

						support.WriteBlockStub = func(b *common.Block, meta []byte) {
							bytes, err := proto.Marshal(&common.Metadata{Value: meta})
							Expect(err).NotTo(HaveOccurred())
							b.Metadata.Metadata[common.BlockMetadataIndex_ORDERER] = bytes

							ledgerLock.Lock()
							defer ledgerLock.Unlock()
							ledger = append(ledger, b)
						}

						support.HeightStub = func() uint64 {
							ledgerLock.Lock()
							defer ledgerLock.Unlock()
							return uint64(len(ledger))
						}
					})

					JustBeforeEach(func() {
						err = chain.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))

						err = chain.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))

						_, metadata := support.WriteBlockArgsForCall(1)
						m = &raftprotos.RaftMetadata{}
						proto.Unmarshal(metadata, m)
					})

					It("writes snapshot file to snapDir", func() {
//场景：以snapinterval=1开始一个链，期望它
//订购3个数据块后提供一个快照。
//
//块号从0开始，我们确定快照是否应由以下人员拍摄：
//appliedblocknum-snapblocknum<snapinterval

						Eventually(countFiles, LongEventualTimeout).Should(Equal(1))
						Eventually(opts.MemoryStorage.FirstIndex, LongEventualTimeout).Should(BeNumerically(">", 1))

//
						err = chain.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(3))
					})

					It("pauses chain if sync is in progress", func() {
//脚本：
//拍摄快照后，重新启动raftindex=0的链
//链应在重新启动时尝试同步，并阻止
//“等待就绪”的API

//检查快照是否退出
						Eventually(countFiles, LongEventualTimeout).Should(Equal(1))

						chain.Halt()

						c := newChain(10*time.Second, channelID, dataDir, 1, raftMetadata)
						c.init()

						signal := make(chan struct{})

						c.puller.PullBlockStub = func(i uint64) *common.Block {
<-signal //阻止断言
							ledgerLock.Lock()
							defer ledgerLock.Unlock()
							if i >= uint64(len(ledger)) {
								return nil
							}

							return ledger[i]
						}

						err := c.WaitReady()
						Expect(err).To((MatchError("chain is not started")))

						c.Start()
						defer c.Halt()

//pull block被调用，所以链现在应该被捕获，waitready应该被阻止
						signal <- struct{}{}

						done := make(chan error)
						go func() {
							done <- c.WaitReady()
						}()

						Consistently(done).ShouldNot(Receive())
close(signal) //解锁拉块器

						Eventually(c.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
					})

					It("restores snapshot w/o extra entries", func() {
//脚本：
//拍摄快照后，不再追加任何条目。
//然后重新启动节点，加载快照，查找其术语
//和索引。当将wal重放到内存存储时，它应该
//没有附加任何条目，因为没有附加额外条目
//拍摄快照后。

						Eventually(countFiles, LongEventualTimeout).Should(Equal(1))
						Eventually(opts.MemoryStorage.FirstIndex, LongEventualTimeout).Should(BeNumerically(">", 1))
snapshot, err := opts.MemoryStorage.Snapshot() //获取刚刚创建的快照
						Expect(err).NotTo(HaveOccurred())
i, err := opts.MemoryStorage.FirstIndex() //获取内存中的第一个索引
						Expect(err).NotTo(HaveOccurred())

//希望存储在快照之前保留SnapshotChupentries项
						Expect(i).To(Equal(snapshot.Metadata.Index - opts.SnapshotCatchUpEntries + 1))

						chain.Halt()

						raftMetadata.RaftIndex = m.RaftIndex
						c := newChain(10*time.Second, channelID, dataDir, 1, raftMetadata)
//c.support.heightreturns（normalBlock.header.number+1）

						c.init()
						c.Start()
						defer c.Halt()

//下面的算法反映了ETCDraft内存存储是如何实现的。
//当快照加载后没有附加任何条目时。
						Eventually(c.opts.MemoryStorage.FirstIndex, LongEventualTimeout).Should(Equal(snapshot.Metadata.Index + 1))
						Eventually(c.opts.MemoryStorage.LastIndex, LongEventualTimeout).Should(Equal(snapshot.Metadata.Index))

//链条保持运转
						Eventually(func() bool {
							c.clock.Increment(interval)
							select {
							case <-c.observe:
								return true
							default:
								return false
							}
						}, LongEventualTimeout).Should(BeTrue())

						c.cutter.CutNext = true
						err = c.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(c.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))
					})

					It("restores snapshot w/ extra entries", func() {
//脚本：
//拍摄快照后，将追加更多条目。
//然后重新启动节点，加载快照，查找其术语
//和索引。当将wal重放到内存存储时，它应该
//附加一些条目。

//检查快照是否退出
						Eventually(countFiles, LongEventualTimeout).Should(Equal(1))
						Eventually(opts.MemoryStorage.FirstIndex, LongEventualTimeout).Should(BeNumerically(">", 1))
snapshot, err := opts.MemoryStorage.Snapshot() //获取刚刚创建的快照
						Expect(err).NotTo(HaveOccurred())
i, err := opts.MemoryStorage.FirstIndex() //获取内存中的第一个索引
						Expect(err).NotTo(HaveOccurred())

//希望存储在快照之前保留SnapshotChupentries项
						Expect(i).To(Equal(snapshot.Metadata.Index - opts.SnapshotCatchUpEntries + 1))

						err = chain.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(3))

						lasti, _ := opts.MemoryStorage.LastIndex()

						chain.Halt()

						raftMetadata.RaftIndex = m.RaftIndex
						c := newChain(10*time.Second, channelID, dataDir, 1, raftMetadata)
						c.support.HeightReturns(5)

						c.init()
						c.Start()
						defer c.Halt()

						Eventually(c.opts.MemoryStorage.FirstIndex, LongEventualTimeout).Should(Equal(snapshot.Metadata.Index + 1))
						Eventually(c.opts.MemoryStorage.LastIndex, LongEventualTimeout).Should(Equal(lasti))

//链条保持运转
						Eventually(func() bool {
							c.clock.Increment(interval)
							select {
							case <-c.observe:
								return true
							default:
								return false
							}
						}, LongEventualTimeout).Should(BeTrue())

						c.cutter.CutNext = true
						err = c.Order(env, uint64(0))
						Expect(err).NotTo(HaveOccurred())
						Eventually(c.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
					})

					When("local ledger is in sync with snapshot", func() {
						It("does not pull blocks and still respects snapshot interval", func() {
//脚本：
//-在块2拍摄快照
//-再订购一个信封（第3块）
//-在块2重新启动链
//-第3区应在WAL上重放。
//-订购另一个信封以触发快照，包含块3和4
//断言：
//-不应调用块拉具
//-重新启动后链应保持运行
//-链应考虑快照间隔以触发下一个快照

//检查快照是否退出
							Eventually(countFiles, LongEventualTimeout).Should(Equal(1))

//再订一个信封。这不应触发快照
							err = chain.Order(env, uint64(0))
							Expect(err).NotTo(HaveOccurred())
							Eventually(support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(3))

							chain.Halt()

							raftMetadata.RaftIndex = m.RaftIndex
							c := newChain(10*time.Second, channelID, dataDir, 1, raftMetadata)
//启动2号区块的链条（高度=3）
							c.support.HeightReturns(3)
							c.opts.SnapInterval = 2

							c.init()
							c.Start()
							defer c.Halt()

//选主
							Eventually(func() bool {
								c.clock.Increment(interval)
								select {
								case <-c.observe:
									return true
								default:
									return false
								}
							}, LongEventualTimeout).Should(BeTrue())

							c.cutter.CutNext = true
							err = c.Order(env, uint64(0))
							Expect(err).NotTo(HaveOccurred())

							Eventually(c.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
							Expect(c.puller.PullBlockCallCount()).Should(BeZero())
							Eventually(countFiles, LongEventualTimeout).Should(Equal(2))
						})
					})
				})
			})

			Context("Invalid WAL dir", func() {
				var support = &consensusmocks.FakeConsenterSupport{}
				BeforeEach(func() {
//用于块创建器初始化
					support.HeightReturns(1)
					support.BlockReturns(getSeedBlock())
				})

				When("WAL dir is a file", func() {
					It("replaces file with fresh WAL dir", func() {
						f, err := ioutil.TempFile("", "wal-")
						Expect(err).NotTo(HaveOccurred())
						defer os.RemoveAll(f.Name())

						chain, err := etcdraft.NewChain(
							support,
							etcdraft.Options{
								WALDir:        f.Name(),
								SnapDir:       snapDir,
								Logger:        logger,
								MemoryStorage: storage,
								RaftMetadata:  &raftprotos.RaftMetadata{},
							},
							configurator,
							nil,
							nil,
							observeC)
						Expect(chain).NotTo(BeNil())
						Expect(err).NotTo(HaveOccurred())

						info, err := os.Stat(f.Name())
						Expect(err).NotTo(HaveOccurred())
						Expect(info.IsDir()).To(BeTrue())
					})
				})

				When("WAL dir is not writeable", func() {
					It("replace it with fresh WAL dir", func() {
						d, err := ioutil.TempDir("", "wal-")
						Expect(err).NotTo(HaveOccurred())
						defer os.RemoveAll(d)

						err = os.Chmod(d, 0500)
						Expect(err).NotTo(HaveOccurred())

						chain, err := etcdraft.NewChain(
							support,
							etcdraft.Options{
								WALDir:        d,
								SnapDir:       snapDir,
								Logger:        logger,
								MemoryStorage: storage,
								RaftMetadata:  &raftprotos.RaftMetadata{},
							},
							nil,
							nil,
							nil,
							nil)
						Expect(chain).NotTo(BeNil())
						Expect(err).ToNot(HaveOccurred())
					})
				})

				When("WAL parent dir is not writeable", func() {
					It("fails to bootstrap fresh raft node", func() {
						skipIfRoot()

						d, err := ioutil.TempDir("", "wal-")
						Expect(err).NotTo(HaveOccurred())
						defer os.RemoveAll(d)

						err = os.Chmod(d, 0500)
						Expect(err).NotTo(HaveOccurred())

						chain, err := etcdraft.NewChain(
							support,
							etcdraft.Options{
								WALDir:       path.Join(d, "wal-dir"),
								SnapDir:      snapDir,
								Logger:       logger,
								RaftMetadata: &raftprotos.RaftMetadata{},
							},
							nil,
							nil,
							nil,
							nil)
						Expect(chain).To(BeNil())
						Expect(err).To(MatchError(ContainSubstring("failed to initialize WAL: mkdir")))
					})
				})
			})
		})

	})

	Describe("Multiple Raft nodes", func() {
		var (
			network      *network
			channelID    string
			timeout      time.Duration
			dataDir      string
			c1, c2, c3   *chain
			raftMetadata *raftprotos.RaftMetadata
		)

		BeforeEach(func() {
			var err error

			channelID = "multi-node-channel"
			timeout = 10 * time.Second

			dataDir, err = ioutil.TempDir("", "raft-test-")
			Expect(err).NotTo(HaveOccurred())

			raftMetadata = &raftprotos.RaftMetadata{
				Consenters: map[uint64]*raftprotos.Consenter{
					1: {
						Host:          "localhost",
						Port:          7051,
						ClientTlsCert: clientTLSCert(tlsCA),
						ServerTlsCert: serverTLSCert(tlsCA),
					},
					2: {
						Host:          "localhost",
						Port:          7051,
						ClientTlsCert: clientTLSCert(tlsCA),
						ServerTlsCert: serverTLSCert(tlsCA),
					},
					3: {
						Host:          "localhost",
						Port:          7051,
						ClientTlsCert: clientTLSCert(tlsCA),
						ServerTlsCert: serverTLSCert(tlsCA),
					},
				},
				NextConsenterId: 4,
			}

			network = createNetwork(timeout, channelID, dataDir, raftMetadata)
			c1 = network.chains[1]
			c2 = network.chains[2]
			c3 = network.chains[3]
		})

		AfterEach(func() {
			os.RemoveAll(dataDir)
		})

		When("2/3 nodes are running", func() {
			It("late node can catch up", func() {
				network.init()
				network.start(1, 2)
				network.elect(1)

				c1.cutter.CutNext = true
				err := c1.Order(env, 0)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() int { return c1.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
				Eventually(func() int { return c2.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
				Eventually(func() int { return c3.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(0))

				network.start(3)

				c1.clock.Increment(interval)
				Eventually(func() int { return c3.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))

				network.stop()
			})

			It("late node receives snapshot from leader", func() {
				c1.opts.SnapInterval = 1
				c1.opts.SnapshotCatchUpEntries = 1

				c1.cutter.CutNext = true

				var blocksLock sync.Mutex
blocks := make(map[uint64]*common.Block) //存放拉块器的书写块

				c1.support.WriteBlockStub = func(b *common.Block, meta []byte) {
					blocksLock.Lock()
					defer blocksLock.Unlock()
					bytes, err := proto.Marshal(&common.Metadata{Value: meta})
					Expect(err).NotTo(HaveOccurred())
					b.Metadata.Metadata[common.BlockMetadataIndex_ORDERER] = bytes
					blocks[b.Header.Number] = b
				}

				c3.puller.PullBlockStub = func(i uint64) *common.Block {
					blocksLock.Lock()
					defer blocksLock.Unlock()
					b, exist := blocks[i]
					if !exist {
						return nil
					}

					return b
				}

				network.init()
				network.start(1, 2)
				network.elect(1)

				err := c1.Order(env, 0)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() int { return c1.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
				Eventually(func() int { return c2.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
				Eventually(func() int { return c3.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(0))

				err = c1.Order(env, 0)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() int { return c1.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(2))
				Eventually(func() int { return c2.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(2))
				Eventually(func() int { return c3.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(0))

				network.start(3)

				c1.clock.Increment(interval)
				Eventually(func() int { return c3.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(2))

				network.stop()
			})
		})

		When("reconfiguring raft cluster", func() {
			const (
				defaultTimeout = 5 * time.Second
			)
			var (
				addConsenterConfigValue = func() map[string]*common.ConfigValue {
					metadata := &raftprotos.Metadata{}
					for _, consenter := range raftMetadata.Consenters {
						metadata.Consenters = append(metadata.Consenters, consenter)
					}

					newConsenter := &raftprotos.Consenter{
						Host:          "localhost",
						Port:          7050,
						ServerTlsCert: serverTLSCert(tlsCA),
						ClientTlsCert: clientTLSCert(tlsCA),
					}
					metadata.Consenters = append(metadata.Consenters, newConsenter)

					return map[string]*common.ConfigValue{
						"ConsensusType": {
							Version: 1,
							Value: marshalOrPanic(&orderer.ConsensusType{
								Metadata: marshalOrPanic(metadata),
							}),
						},
					}
				}
			)

			BeforeEach(func() {
				network.init()
				network.start()
				network.elect(1)

				By("Submitting first tx to cut the block")
				c1.cutter.CutNext = true
				err := c1.Order(env, 0)
				Expect(err).ToNot(HaveOccurred())

				c1.clock.Increment(interval)

				network.exec(
					func(c *chain) {
						Eventually(func() int { return c.support.WriteBlockCallCount() }, defaultTimeout).Should(Equal(1))
					})
			})

			AfterEach(func() {
				network.stop()
			})

			Context("reconfiguration", func() {
				It("trying to simultaneously add and remove nodes in one config update", func() {

					updatedRaftMetadata := proto.Clone(raftMetadata).(*raftprotos.RaftMetadata)
//删除第二个同意者
					delete(updatedRaftMetadata.Consenters, 2)

					metadata := &raftprotos.Metadata{}
					for _, consenter := range updatedRaftMetadata.Consenters {
						metadata.Consenters = append(metadata.Consenters, consenter)
					}

//添加新同意人
					newConsenter := &raftprotos.Consenter{
						Host:          "localhost",
						Port:          7050,
						ServerTlsCert: serverTLSCert(tlsCA),
						ClientTlsCert: clientTLSCert(tlsCA),
					}
					metadata.Consenters = append(metadata.Consenters, newConsenter)

					value := map[string]*common.ConfigValue{
						"ConsensusType": {
							Version: 1,
							Value: marshalOrPanic(&orderer.ConsensusType{
								Metadata: marshalOrPanic(metadata),
							}),
						},
					}

					By("adding new consenter into configuration")
					configEnv := newConfigEnv(channelID, common.HeaderType_CONFIG, newConfigUpdateEnv(channelID, value))
					c1.cutter.CutNext = true

					By("sending config transaction")
					err := c1.Configure(configEnv, 0)
					Expect(err).To(MatchError("update of more than one consenters at a time is not supported"))
				})

				It("adding node to the cluster", func() {

					c4 := newChain(timeout, channelID, dataDir, 4, &raftprotos.RaftMetadata{
						Consenters: map[uint64]*raftprotos.Consenter{},
					})
					c4.init()

					By("adding new node to the network")
					Eventually(func() int { return c4.support.WriteBlockCallCount() }, defaultTimeout).Should(Equal(0))
					Eventually(func() int { return c4.support.WriteConfigBlockCallCount() }, defaultTimeout).Should(Equal(0))

					configEnv := newConfigEnv(channelID, common.HeaderType_CONFIG, newConfigUpdateEnv(channelID, addConsenterConfigValue()))
					c1.cutter.CutNext = true

					By("sending config transaction")
					err := c1.Configure(configEnv, 0)
					Expect(err).ToNot(HaveOccurred())

					network.exec(func(c *chain) {
						Eventually(c.support.WriteConfigBlockCallCount, defaultTimeout).Should(Equal(1))
					})

					network.addChain(c4)
					c4.Start()

//confchange异步应用于etcd/raft，表示没有添加节点4
//立即到领导的节点列表。即时的滴答声不会触发心跳。
//正在发送到节点4。因此，我们重复地勾选引线，直到节点4加入。
//群集成功。
					Eventually(func() <-chan uint64 {
						c1.clock.Increment(interval)
						return c4.observe
					}, defaultTimeout).Should(Receive(Equal(uint64(1))))

					Eventually(c4.support.WriteBlockCallCount, defaultTimeout).Should(Equal(1))
					Eventually(c4.support.WriteConfigBlockCallCount, defaultTimeout).Should(Equal(1))

					By("submitting new transaction to follower")
					c1.cutter.CutNext = true
					err = c4.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())

					c1.clock.Increment(interval)

					network.exec(func(c *chain) {
						Eventually(c.support.WriteBlockCallCount, defaultTimeout).Should(Equal(2))
					})
				})

				It("adding node to the cluster of 2/3 available nodes", func() {
//场景：从副本集断开一个现有节点的连接
//添加新节点，重新连接旧节点并选择“新添加”作为
//引导，检查断开连接的节点何时将获得新配置
//将看到新加入的领导者

//断开第二个节点
					network.disconnect(2)

					c4 := newChain(timeout, channelID, dataDir, 4, &raftprotos.RaftMetadata{
						Consenters: map[uint64]*raftprotos.Consenter{},
					})
					c4.init()

					By("adding new node to the network")
					Eventually(c4.support.WriteBlockCallCount, defaultTimeout).Should(Equal(0))
					Eventually(c4.support.WriteConfigBlockCallCount, defaultTimeout).Should(Equal(0))

					configEnv := newConfigEnv(channelID, common.HeaderType_CONFIG, newConfigUpdateEnv(channelID, addConsenterConfigValue()))
					c1.cutter.CutNext = true

					By("sending config transaction")
					err := c1.Configure(configEnv, 0)
					Expect(err).ToNot(HaveOccurred())

					Eventually(c1.support.WriteConfigBlockCallCount, defaultTimeout).Should(Equal(1))
//第二个节点已断开连接，因此不能获取配置块
					Eventually(c2.support.WriteConfigBlockCallCount, defaultTimeout).Should(Equal(0))
					Eventually(c3.support.WriteConfigBlockCallCount, defaultTimeout).Should(Equal(1))

					network.addChain(c4)
					c4.Start()

//confchange异步应用于etcd/raft，表示没有添加节点4
//立即到领导的节点列表。即时的滴答声不会触发心跳。
//正在发送到节点4。因此，我们重复地勾选引线，直到节点4加入。
//群集成功。
					Eventually(func() <-chan uint64 {
						c1.clock.Increment(interval)
						return c4.observe
					}).Should(Receive(Equal(uint64(1))))

					Eventually(c4.support.WriteBlockCallCount, defaultTimeout).Should(Equal(1))
					Eventually(c4.support.WriteConfigBlockCallCount, defaultTimeout).Should(Equal(1))

					By("submitting new transaction to follower")
					c1.cutter.CutNext = true
					err = c4.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())

					c1.clock.Increment(interval)

//选择新添加的节点作为领队
					network.elect(4)

//重新连接节点，应该能够赶上并获得配置更新
					network.connect(2)

//确保第二个节点将第四个节点视为领导者
					Eventually(func() <-chan uint64 {
						c4.clock.Increment(interval)
						return c2.observe
					}, defaultTimeout).Should(Receive(Equal(uint64(4))))

					By("submitting new transaction to re-connected node")
					c4.cutter.CutNext = true
					err = c2.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())

					network.exec(func(c *chain) {
						Eventually(c.support.WriteBlockCallCount, defaultTimeout).Should(Equal(3))
					})
				})

				It("stop leader and continue reconfiguration failing over to new leader", func() {
//场景：启动3个raft节点的副本集，选择节点c1作为领队
//配置链支持模拟以在C1写入配置块后立即断开连接
//在分类帐中，这可以模拟故障转移。
//接下来，提升新节点C4以加入集群并创建配置事务，提交
//交给领导。一旦Leader写入配置块，它将失败并将Leadership转移到
//C2
//测试断言新节点C4将加入集群，C2将处理
//重新配置。稍后，我们将C1连接回去，确保它能够赶上
//新配置并成功地重新加入副本集。

					c4 := newChain(timeout, channelID, dataDir, 4, &raftprotos.RaftMetadata{
						Consenters: map[uint64]*raftprotos.Consenter{},
					})
					c4.init()

					By("adding new node to the network")
					Expect(c4.support.WriteBlockCallCount()).Should(Equal(0))
					Expect(c4.support.WriteConfigBlockCallCount()).Should(Equal(0))

					configEnv := newConfigEnv(channelID, common.HeaderType_CONFIG, newConfigUpdateEnv(channelID, addConsenterConfigValue()))
					c1.cutter.CutNext = true
					configBlock := &common.Block{
						Header: &common.BlockHeader{},
						Data:   &common.BlockData{Data: [][]byte{marshalOrPanic(configEnv)}}}

					c1.support.WriteConfigBlockStub = func(_ *common.Block, _ []byte) {
//提交块后断开引线
						network.disconnect(1)
//选举新领导
						network.elect(2)
					}

//返回最近配置块的模拟块方法
					c2.support.BlockReturns(configBlock)

					By("sending config transaction")
					err := c1.Configure(configEnv, 0)
					Expect(err).ToNot(HaveOccurred())

//每个节点都已将配置块写入OSN分类帐
					network.exec(
						func(c *chain) {
							Eventually(c.support.WriteConfigBlockCallCount, LongEventualTimeout).Should(Equal(1))
						})

					network.addChain(c4)
					c4.Start()
//confchange异步应用于etcd/raft，表示没有添加节点4
//立即到领导的节点列表。即时的滴答声不会触发心跳。
//正在发送到节点4。因此，我们重复地勾选引线，直到节点4加入。
//群集成功。
					Eventually(func() <-chan uint64 {
						c2.clock.Increment(interval)
						return c4.observe
					}, LongEventualTimeout).Should(Receive(Equal(uint64(2))))

					Eventually(c4.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))
					Eventually(c4.support.WriteConfigBlockCallCount, LongEventualTimeout).Should(Equal(1))

					By("submitting new transaction to follower")
					c2.cutter.CutNext = true
					err = c4.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())

					c2.clock.Increment(interval)

//REST节点是活动的，包括新添加的，因此应写入2个块
					Eventually(c2.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
					Eventually(c3.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
					Eventually(c4.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))

//节点1已停止不应写入任何块
					Consistently(c1.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))

					network.connect(1)

					c2.clock.Increment(interval)
//检查前领导没有被卡住，实际上收到了辞职信号，
//一旦连接，就可以与其他副本集通信
					Eventually(c1.observe, LongEventualTimeout).Should(Receive(Equal(uint64(2))))
					Eventually(c1.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
				})
			})
		})

		When("3/3 nodes are running", func() {
			JustBeforeEach(func() {
				network.init()
				network.start()
				network.elect(1)
			})

			AfterEach(func() {
				network.stop()
			})

			It("orders envelope on leader", func() {
				By("instructed to cut next block")
				c1.cutter.CutNext = true
				err := c1.Order(env, 0)
				Expect(err).ToNot(HaveOccurred())

				network.exec(
					func(c *chain) {
						Eventually(func() int { return c.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
					})

				By("respect batch timeout")
				c1.cutter.CutNext = false

				err = c1.Order(env, 0)
				Expect(err).ToNot(HaveOccurred())
				Eventually(c1.cutter.CurBatch, LongEventualTimeout).Should(HaveLen(1))

				c1.clock.WaitForNWatchersAndIncrement(timeout, 2)
				network.exec(
					func(c *chain) {
						Eventually(func() int { return c.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(2))
					})
			})

			It("orders envelope on follower", func() {
				By("instructed to cut next block")
				c1.cutter.CutNext = true
				err := c2.Order(env, 0)
				Expect(err).ToNot(HaveOccurred())

				network.exec(
					func(c *chain) {
						Eventually(func() int { return c.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
					})

				By("respect batch timeout")
				c1.cutter.CutNext = false

				err = c2.Order(env, 0)
				Expect(err).ToNot(HaveOccurred())
				Eventually(c1.cutter.CurBatch, LongEventualTimeout).Should(HaveLen(1))

				c1.clock.WaitForNWatchersAndIncrement(timeout, 2)
				network.exec(
					func(c *chain) {
						Eventually(func() int { return c.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(2))
					})
			})

			It("allows the leader to create multiple normal blocks without having to wait for them to be written out", func() {
//这样可以确保创建的块不会被写出
				network.disconnect(1)

				c1.cutter.CutNext = true
				for i := 0; i < 10; i++ {
					err := c1.Order(env, 0)
					Expect(err).NotTo(HaveOccurred())
				}

				Eventually(c1.BlockCreator.CreatedBlocks).Should(HaveLen(10))
			})

			It("calls BlockCreator.commitBlock on all the nodes' chains once a block is written", func() {
				normalBlock := &common.Block{
					Header:   &common.BlockHeader{},
					Data:     &common.BlockData{Data: [][]byte{[]byte("foo")}},
					Metadata: &common.BlockMetadata{Metadata: make([][]byte, 4)},
				}
//为了测试在c2（follower）上也调用了commitBlock；应该丢弃此块
//调用了commitBlock之后，因为它是一个分散的块
				c2.BlockCreator.CreatedBlocks <- normalBlock

				c1.cutter.CutNext = true
				err := c1.Order(env, 0)
				Expect(err).ToNot(HaveOccurred())

				network.exec(
					func(c *chain) {
						Eventually(func() int { return c.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
						b, _ := c.support.WriteBlockArgsForCall(0)
Eventually(c.BlockCreator.CreatedBlocks, LongEventualTimeout).Should(HaveLen(0)) //表示已调用BlockCreator.CommitBlock
//检查它是否也正确更新了lastcreatedblock
						Eventually(bytes.Equal(b.Header.Bytes(), c.BlockCreator.LastCreatedBlock.Header.Bytes()), LongEventualTimeout).Should(BeTrue())
					})
			})

			Context("handling config blocks", func() {
				var configEnv *common.Envelope
				BeforeEach(func() {
					values := map[string]*common.ConfigValue{
						"BatchTimeout": {
							Version: 1,
							Value: marshalOrPanic(&orderer.BatchTimeout{
								Timeout: "3ms",
							}),
						},
					}
					configEnv = newConfigEnv(channelID,
						common.HeaderType_CONFIG,
						newConfigUpdateEnv(channelID, values),
					)
				})

				It("holds up block creation on leader once a config block has been created and not written out", func() {
//这样可以确保创建的块不会被写出
					network.disconnect(1)

					c1.cutter.CutNext = true
//配置块
					err := c1.Order(configEnv, 0)
					Expect(err).NotTo(HaveOccurred())
					Eventually(c1.BlockCreator.CreatedBlocks).Should(HaveLen(1))

//避免数据竞争，因为我们在Goroutine中访问这些数据
					tempEnv := env
					tempC1 := c1

//正规块
					go func() {
						defer GinkgoRecover()
						err := tempC1.Order(tempEnv, 0)
//因为链条在以下持续测试通过后停止
						Expect(err).To(MatchError("chain is stopped"))
					}()

//确保只创建一个块，因为从未写出配置块
					Consistently(c1.BlockCreator.CreatedBlocks).Should(HaveLen(1))
				})

				It("continues creating blocks on leader after a config block has been successfully written out", func() {
					c1.cutter.CutNext = true
//配置块
					err := c1.Configure(configEnv, 0)
					Expect(err).NotTo(HaveOccurred())
					network.exec(
						func(c *chain) {
							Eventually(func() int { return c.support.WriteConfigBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
						})

//配置块后的普通块
					err = c1.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())
					network.exec(
						func(c *chain) {
							Eventually(func() int { return c.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
						})
				})
			})

			When("Snapshotting is enabled", func() {
				BeforeEach(func() {
					c1.opts.SnapInterval = 1
					c1.opts.SnapshotCatchUpEntries = 1
				})

				It("keeps running if some entries in memory are purged", func() {
//场景：在节点1上启用快照，它清除内存存储
//每个快照。群集应正常工作。

					i, err := c1.opts.MemoryStorage.FirstIndex()
					Expect(err).NotTo(HaveOccurred())
					Expect(i).To(Equal(uint64(1)))

					c1.cutter.CutNext = true

					err = c1.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())

					network.exec(
						func(c *chain) {
							Eventually(func() int { return c.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
						})

					Eventually(c1.opts.MemoryStorage.FirstIndex, LongEventualTimeout).Should(BeNumerically(">", i))
					i, err = c1.opts.MemoryStorage.FirstIndex()
					Expect(err).NotTo(HaveOccurred())

					err = c1.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())

					network.exec(
						func(c *chain) {
							Eventually(func() int { return c.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(2))
						})

					Eventually(c1.opts.MemoryStorage.FirstIndex, LongEventualTimeout).Should(BeNumerically(">", i))
					i, err = c1.opts.MemoryStorage.FirstIndex()
					Expect(err).NotTo(HaveOccurred())

					err = c1.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())

					network.exec(
						func(c *chain) {
							Eventually(func() int { return c.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(3))
						})

					Eventually(c1.opts.MemoryStorage.FirstIndex).Should(BeNumerically(">", i))
				})

				It("lagged node can catch up using snapshot", func() {
					network.disconnect(2)

					c1.cutter.CutNext = true

					for i := 1; i <= 10; i++ {
						err := c1.Order(env, 0)
						Expect(err).NotTo(HaveOccurred())
						Eventually(c1.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(i))
						Eventually(c3.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(i))
					}

					Eventually(c2.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(0))

					network.rejoin(2, false)

					Eventually(c2.puller.PullBlockCallCount, LongEventualTimeout).Should(Equal(10))
					Eventually(c2.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(10))

					files, err := ioutil.ReadDir(c2.opts.SnapDir)
					Expect(err).NotTo(HaveOccurred())
Expect(files).To(HaveLen(1)) //希望存储精确的1个快照

//链条应保持运转
					err = c1.Order(env, 0)
					Expect(err).NotTo(HaveOccurred())

					network.exec(
						func(c *chain) {
							Eventually(func() int { return c.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(11))
						})
				})
			})

			Context("failover", func() {
				It("follower should step up as leader upon failover", func() {
					network.stop(1)
					network.elect(2)

					By("order envelope on new leader")
					c2.cutter.CutNext = true
					err := c2.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())

//不应在链1上生成块
					Eventually(c1.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(0))

//应在2号和3号链条上生产砌块。
					Eventually(c2.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))
					Eventually(c3.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))

					By("order envelope on follower")
					err = c3.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())

//不应在链1上生成块
					Eventually(c1.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(0))

//应在2号和3号链条上生产砌块。
					Eventually(c2.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
					Eventually(c3.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(2))
				})

				It("follower cannot be elected if its log is not up-to-date", func() {
					network.disconnect(2)

					c1.cutter.CutNext = true
					err := c1.Order(env, 0)
					Expect(err).NotTo(HaveOccurred())

					Eventually(c1.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))
					Eventually(c2.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(0))
					Eventually(c3.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(1))

					network.disconnect(1)
					network.connect(2)

//节点2未赶上其他节点
					for tick := 0; tick < 2*ELECTION_TICK-1; tick++ {
						c2.clock.Increment(interval)
						Consistently(c2.observe).ShouldNot(Receive(Equal(2)))
					}

//启用prevote时，节点2将无法收集足够的数据
//因为它的索引不是最新的。因此，它
//不会导致其他节点上的引线更改。
					Consistently(c3.observe).ShouldNot(Receive())
network.elect(3) //节点3在2和3之间有最新的日志，因此可以选择
				})

				It("PreVote prevents reconnected node from disturbing network", func() {
					network.disconnect(2)

					c1.cutter.CutNext = true
					err := c1.Order(env, 0)
					Expect(err).NotTo(HaveOccurred())

					Eventually(c1.support.WriteBlockCallCount).Should(Equal(1))
					Eventually(c2.support.WriteBlockCallCount).Should(Equal(0))
					Eventually(c3.support.WriteBlockCallCount).Should(Equal(1))

					network.connect(2)

					for tick := 0; tick < 2*ELECTION_TICK-1; tick++ {
						c2.clock.Increment(interval)
						Consistently(c2.observe).ShouldNot(Receive(Equal(2)))
					}

					Consistently(c1.observe).ShouldNot(Receive())
					Consistently(c3.observe).ShouldNot(Receive())
				})

				It("follower can catch up and then campaign with success", func() {
					network.disconnect(2)

					c1.cutter.CutNext = true
					for i := 0; i < 10; i++ {
						err := c1.Order(env, 0)
						Expect(err).NotTo(HaveOccurred())
					}

					Eventually(c1.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(10))
					Eventually(c2.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(0))
					Eventually(c3.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(10))

					network.rejoin(2, false)
					Eventually(c2.support.WriteBlockCallCount, LongEventualTimeout).Should(Equal(10))

					network.disconnect(1)
					network.elect(2)
				})

				It("purges blockcutter, stops timer and discards created blocks if leadership is lost", func() {
//在链1上创建一个块以测试所创建块的重置
					network.disconnect(1)
					normalBlock := &common.Block{
						Header:   &common.BlockHeader{},
						Data:     &common.BlockData{Data: [][]byte{[]byte("foo")}},
						Metadata: &common.BlockMetadata{Metadata: make([][]byte, 4)},
					}
					c1.BlockCreator.CreatedBlocks <- normalBlock
					Expect(len(c1.BlockCreator.CreatedBlocks)).To(Equal(1))

//将一个事务排队到1的BlockCutter中，以测试是否清除BlockCutter
					c1.cutter.CutNext = false
					err := c1.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())
					Eventually(c1.cutter.CurBatch, LongEventualTimeout).Should(HaveLen(1))

//创建的块不应写入，因为领导者不应获得投票。
//由于网络断开而导致的块。
					Consistently(c1.support.WriteBlockCallCount).Should(Equal(0))

					network.elect(2)
					network.rejoin(1, true)

Eventually(c1.clock.WatcherCount, LongEventualTimeout).Should(Equal(1)) //切断机时间停止
					Eventually(c1.cutter.CurBatch, LongEventualTimeout).Should(HaveLen(0))
//创建的块应丢弃，因为存在领导层更改
					Eventually(c1.BlockCreator.CreatedBlocks).Should(HaveLen(0))
					Consistently(c1.support.WriteBlockCallCount).Should(Equal(0))

					network.disconnect(2)
n := network.elect(1) //以n个间隔前进1秒

					err = c1.Order(env, 0)
					Expect(err).ToNot(HaveOccurred())

//下面的断言组是多余的-这里是为了完整性。
//如果拦截器没有复位，将时钟快速转发到“超时”，应导致拦截器点火。
//如果BlockCutter已重置，快速转发将不起任何作用。
//
//换个说法：
//
//对的：
//停止启动火灾
//|--------------|---------------------------|
//n*间隔超时
//（在选举中领先）
//
//错误：
//不熄火
//--------------------------
//超时
//
//超时-n*间隔n*间隔
//---------_---------_
//^ ^ ^
//在这个时候，它应该开火
//此时不应触发计时器

					c1.clock.WaitForNWatchersAndIncrement(timeout-time.Duration(n*int(interval/time.Millisecond)), 2)
					Eventually(func() int { return c1.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(0))
					Eventually(func() int { return c3.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(0))

					c1.clock.Increment(time.Duration(n * int(interval/time.Millisecond)))
					Eventually(func() int { return c1.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
					Eventually(func() int { return c3.support.WriteBlockCallCount() }, LongEventualTimeout).Should(Equal(1))
				})

				It("stale leader should not be able to propose block because of lagged term", func() {
					network.disconnect(1)
					network.elect(2)
					network.connect(1)

					c1.cutter.CutNext = true
					err := c1.Order(env, 0)
					Expect(err).NotTo(HaveOccurred())

					network.exec(
						func(c *chain) {
							Consistently(c.support.WriteBlockCallCount).Should(Equal(0))
						})
				})

				It("aborts waiting for block to be committed upon leadership lost", func() {
					network.disconnect(1)

					c1.cutter.CutNext = true
					err := c1.Order(env, 0)
					Expect(err).NotTo(HaveOccurred())

					network.exec(
						func(c *chain) {
							Consistently(c.support.WriteBlockCallCount).Should(Equal(0))
						})

					network.elect(2)
					network.connect(1)

					c2.clock.Increment(interval)
//此检查可确保commitbatches方法使用resignc上的信号。
					Eventually(c1.observe, LongEventualTimeout).Should(Receive(Equal(uint64(2))))
				})
			})
		})
	})
})

func nodeConfigFromMetadata(consenterMetadata *raftprotos.Metadata) []cluster.RemoteNode {
	var nodes []cluster.RemoteNode
	for i, consenter := range consenterMetadata.Consenters {
//现在，跳过我们自己
		if i == 0 {
			continue
		}
		serverDER, _ := pem.Decode(consenter.ServerTlsCert)
		clientDER, _ := pem.Decode(consenter.ClientTlsCert)
		node := cluster.RemoteNode{
			ID:            uint64(i + 1),
			Endpoint:      "localhost:7050",
			ServerTLSCert: serverDER.Bytes,
			ClientTLSCert: clientDER.Bytes,
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func createMetadata(nodeCount int, tlsCA tlsgen.CA) *raftprotos.Metadata {
	md := &raftprotos.Metadata{}
	for i := 0; i < nodeCount; i++ {
		md.Consenters = append(md.Consenters, &raftprotos.Consenter{
			Host:          "localhost",
			Port:          7050,
			ServerTlsCert: serverTLSCert(tlsCA),
			ClientTlsCert: clientTLSCert(tlsCA),
		})
	}
	return md
}

func serverTLSCert(tlsCA tlsgen.CA) []byte {
	cert, err := tlsCA.NewServerCertKeyPair("localhost")
	if err != nil {
		panic(err)
	}
	return cert.Cert
}

func clientTLSCert(tlsCA tlsgen.CA) []byte {
	cert, err := tlsCA.NewClientCertKeyPair()
	if err != nil {
		panic(err)
	}
	return cert.Cert
}

//marshalorpanic序列化protobuf消息，如果此
//操作失败
func marshalOrPanic(pb proto.Message) []byte {
	data, err := proto.Marshal(pb)
	if err != nil {
		panic(err)
	}
	return data
}

//帮助进行测试
type chain struct {
	id uint64

	support      *consensusmocks.FakeConsenterSupport
	cutter       *mockblockcutter.Receiver
	configurator *mocks.Configurator
	rpc          *mocks.FakeRPC
	storage      *raft.MemoryStorage
	walDir       string
	clock        *fakeclock.FakeClock
	opts         etcdraft.Options
	puller       *mocks.FakeBlockPuller

	observe   chan uint64
	unstarted chan struct{}

	*etcdraft.Chain
}

func newChain(timeout time.Duration, channel string, dataDir string, id uint64, raftMetadata *raftprotos.RaftMetadata) *chain {
	rpc := &mocks.FakeRPC{}
	clock := fakeclock.NewFakeClock(time.Now())
	storage := raft.NewMemoryStorage()

	opts := etcdraft.Options{
		RaftID:          uint64(id),
		Clock:           clock,
		TickInterval:    interval,
		ElectionTick:    ELECTION_TICK,
		HeartbeatTick:   HEARTBEAT_TICK,
		MaxSizePerMsg:   1024 * 1024,
		MaxInflightMsgs: 256,
		RaftMetadata:    raftMetadata,
		Logger:          flogging.NewFabricLogger(zap.NewNop()),
		MemoryStorage:   storage,
		WALDir:          path.Join(dataDir, "wal"),
		SnapDir:         path.Join(dataDir, "snapshot"),
	}

	support := &consensusmocks.FakeConsenterSupport{}
	support.ChainIDReturns(channel)
	support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})

	cutter := mockblockcutter.NewReceiver()
	close(cutter.Block)
	support.BlockCutterReturns(cutter)

//用于块创建器初始化
	support.HeightReturns(1)
	support.BlockReturns(getSeedBlock())

//更改导程后，导程在设置为实际值之前重置为0。
//新领导，即1->0->2。因此，2个数字将是
//用这条裤子寄的，所以我们需要2码的
	observe := make(chan uint64, 2)

	configurator := &mocks.Configurator{}
	configurator.On("Configure", mock.Anything, mock.Anything)

	puller := &mocks.FakeBlockPuller{}

	ch := make(chan struct{})
	close(ch)
	return &chain{
		id:           id,
		support:      support,
		cutter:       cutter,
		rpc:          rpc,
		storage:      storage,
		observe:      observe,
		clock:        clock,
		opts:         opts,
		unstarted:    ch,
		configurator: configurator,
		puller:       puller,
	}
}

func (c *chain) init() {
	ch, err := etcdraft.NewChain(c.support, c.opts, c.configurator, c.rpc, c.puller, c.observe)
	Expect(err).NotTo(HaveOccurred())
	c.Chain = ch
}

type network struct {
	leader uint64
	chains map[uint64]*chain

//存储要由模拟块拉具返回的书面块
	ledger *sync.Map

//用于确定链的连接性。
//实际值类型为“chan struct”，因为
//它用于跳过“elect”中的断言，如果
//因此，节点与网络断开连接
//不应观察领导变化
	connLock     sync.RWMutex
	connectivity map[uint64]chan struct{}
}

func (n *network) appendChain(c *chain) {
	n.connLock.Lock()
	n.chains[c.id] = c
	n.connLock.Unlock()
}

func (n *network) addConnection(id uint64) {
	n.connLock.Lock()
	n.connectivity[id] = make(chan struct{})
	n.connLock.Unlock()
}

func (n *network) addChain(c *chain) {
	n.addConnection(c.id)

	c.rpc.StepStub = func(dest uint64, msg *orderer.StepRequest) (*orderer.StepResponse, error) {
		n.connLock.RLock()
		defer n.connLock.RUnlock()

		select {
		case <-n.connectivity[dest]:
		case <-n.connectivity[c.id]:
		default:
			go n.chains[dest].Step(msg, c.id)
		}

		return nil, nil
	}

	c.rpc.SendSubmitStub = func(dest uint64, msg *orderer.SubmitRequest) error {
		n.connLock.RLock()
		defer n.connLock.RUnlock()

		select {
		case <-n.connectivity[dest]:
		case <-n.connectivity[c.id]:
		default:
			go n.chains[dest].Submit(msg, c.id)
		}

		return nil
	}

	c.support.WriteBlockStub = func(b *common.Block, meta []byte) {
		bytes, err := proto.Marshal(&common.Metadata{Value: meta})
		Expect(err).NotTo(HaveOccurred())
		b.Metadata.Metadata[common.BlockMetadataIndex_ORDERER] = bytes
		n.ledger.Store(b.Header.Number, b)
	}

	c.puller.PullBlockStub = func(i uint64) *common.Block {
		b, exist := n.ledger.Load(i)
		if !exist {
			return nil
		}

		return b.(*common.Block)
	}

	n.appendChain(c)
}

func createNetwork(timeout time.Duration, channel string, dataDir string, raftMetadata *raftprotos.RaftMetadata) *network {
	n := &network{
		chains:       make(map[uint64]*chain),
		connectivity: make(map[uint64]chan struct{}),
		ledger:       &sync.Map{},
	}

	for nodeID := range raftMetadata.Consenters {
		dir, err := ioutil.TempDir(dataDir, fmt.Sprintf("node-%d-", nodeID))
		Expect(err).NotTo(HaveOccurred())

		m := proto.Clone(raftMetadata).(*raftprotos.RaftMetadata)
		n.addChain(newChain(timeout, channel, dir, nodeID, m))
	}

	return n
}

//测试可以在创建链之前更改链的配置
func (n *network) init() {
	n.exec(func(c *chain) { c.init() })
}

func (n *network) start(ids ...uint64) {
	nodes := ids
	if len(nodes) == 0 {
		for i := range n.chains {
			nodes = append(nodes, i)
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(nodes))
	for _, i := range nodes {
		go func(id uint64) {
			defer GinkgoRecover()
			n.chains[id].Start()
			n.chains[id].unstarted = nil

//当raft节点启动时，它会生成confchange
//要添加自己，需要使用ready（）。
//如果筏板中有待定的配置更改，
//不管供应多少虱子，它都拒绝竞选。
//这在生产代码中不是问题，因为最终
//木筏。准备就绪会随着时间的推移而消耗掉。
//
//然而，这在使用假时钟和人工时钟时是有问题的。
//蜱类而不是无限期地在木筏上打勾。准备好了。
//已消费，此支票被添加到间接担保中
//第一个confchange实际上被消耗掉了，我们可以安全地
//继续打勾筏。
			Eventually(func() error {
				_, err := n.chains[id].storage.Entries(1, 1, 1)
				return err
			}).ShouldNot(HaveOccurred())

			wg.Done()
		}(i)
	}
	wg.Wait()
}

func (n *network) stop(ids ...uint64) {
	nodes := ids
	if len(nodes) == 0 {
		for i := range n.chains {
			nodes = append(nodes, i)
		}
	}

	for _, c := range nodes {
		n.chains[c].Halt()
		<-n.chains[c].Errored()
	}
}

func (n *network) exec(f func(c *chain), ids ...uint64) {
	if len(ids) == 0 {
		for _, c := range n.chains {
			f(c)
		}

		return
	}

	for _, i := range ids {
		f(n.chains[i])
	}
}

//将节点连接到网络并勾选“引线”以触发
//一个新加入的节点可以检测到领队的心跳。
func (n *network) rejoin(id uint64, wasLeader bool) {
	n.connect(id)
	n.chains[n.leader].clock.Increment(interval)

	if wasLeader {
		Eventually(n.chains[id].observe).Should(Receive(Equal(n.leader)))
	} else {
		Consistently(n.chains[id].observe).ShouldNot(Receive())
	}

//等待新加入的节点赶上领队
	i, err := n.chains[n.leader].opts.MemoryStorage.LastIndex()
	Expect(err).NotTo(HaveOccurred())
	Eventually(n.chains[id].opts.MemoryStorage.LastIndex).Should(Equal(i))
}

//确定性地选择一个节点作为领队
//只需在该节点上勾选计时器。它返回
//测试需要的实际计时周期数。
func (n *network) elect(id uint64) (tick int) {
//另外，由于假时钟的实现方式，
//缓慢的消费者可能会跳过一个勾号，这可能
//导致不确定行为。因此
//我们每次都要等足够的时间
//打勾以便生效。
	t := 1000 * time.Millisecond

	n.connLock.RLock()
	c := n.chains[id]
	n.connLock.RUnlock()

	var elected bool
	for !elected {
		c.clock.Increment(interval)
		tick++

		select {
		case <-time.After(t):
//此勾号不会触发t内的导程更改，继续
case n := <-c.observe: //领导层发生变化
			if n == 0 {
//在ETCD/RAFT中，如果已经有一个领导者，
//SoftState中的铅通过X->0->Y。
//因此，我们可以先观察0。在这
//情况，不需要再打勾，因为
//领袖选举已经开始。
				Eventually(c.observe).Should(Receive(Equal(id)))
			} else {
//如果没有领导者（新集群），我们有0->Y
//因此我们应该直接观察y。
				Expect(n).To(Equal(id))
			}
			elected = true
			break
		}
	}

//现在观察其他节点上的引线更改

	n.connLock.RLock()
	for _, c := range n.chains {
		if c.id == id {
			continue
		}

		select {
case <-c.Errored(): //如果节点退出，则跳过
case <-n.connectivity[c.id]: //跳过检查节点n是否断开连接
case <-c.unstarted: //跳过检查节点是否尚未启动
		default:
			Eventually(c.observe).Should(Receive(Equal(id)))
		}
	}
	n.connLock.RUnlock()

	n.leader = id
	return tick
}

func (n *network) disconnect(i uint64) {
	close(n.connectivity[i])
}

func (n *network) connect(i uint64) {
	n.connLock.Lock()
	defer n.connLock.Unlock()
	n.connectivity[i] = make(chan struct{})
}

//设置上面声明的configenv var
func newConfigEnv(chainID string, headerType common.HeaderType, configUpdateEnv *common.ConfigUpdateEnvelope) *common.Envelope {
	return &common.Envelope{
		Payload: marshalOrPanic(&common.Payload{
			Header: &common.Header{
				ChannelHeader: marshalOrPanic(&common.ChannelHeader{
					Type:      int32(headerType),
					ChannelId: chainID,
				}),
			},
			Data: marshalOrPanic(&common.ConfigEnvelope{
				LastUpdate: &common.Envelope{
					Payload: marshalOrPanic(&common.Payload{
						Header: &common.Header{
							ChannelHeader: marshalOrPanic(&common.ChannelHeader{
								Type:      int32(common.HeaderType_CONFIG_UPDATE),
								ChannelId: chainID,
							}),
						},
						Data: marshalOrPanic(configUpdateEnv),
}), //共同有效载荷
}, //最后更新
			}),
		}),
	}
}

func newConfigUpdateEnv(chainID string, values map[string]*common.ConfigValue) *common.ConfigUpdateEnvelope {
	return &common.ConfigUpdateEnvelope{
		ConfigUpdate: marshalOrPanic(&common.ConfigUpdate{
			ChannelId: chainID,
			ReadSet:   &common.ConfigGroup{},
			WriteSet: &common.ConfigGroup{
				Groups: map[string]*common.ConfigGroup{
					"Orderer": {
						Values: values,
					},
				},
}, //文集
		}),
	}
}

func getSeedBlock() *common.Block {
	return &common.Block{
		Header:   &common.BlockHeader{},
		Data:     &common.BlockData{Data: [][]byte{[]byte("foo")}},
		Metadata: &common.BlockMetadata{Metadata: make([][]byte, 4)},
	}
}
