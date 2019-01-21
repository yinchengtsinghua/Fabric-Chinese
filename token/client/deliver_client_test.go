
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

package client_test

import (
	"context"
	"math"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/token/client"
	"github.com/hyperledger/fabric/token/client/mock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("OrdererClient", func() {
	var (
		channelId             string
		creator               []byte
		expectedChannelHeader *common.ChannelHeader
		expectedPayloadData   []byte

		deliverResp *pb.DeliverResponse
		fakeTxid    string

		fakeSigner          *mock.SignerIdentity
		fakeDeliverFiltered *mock.DeliverFiltered
	)

	BeforeEach(func() {
		channelId = "test-channel"
		creator = []byte("creator")

//通道头的预期字段-排除动态生成的字段
		expectedChannelHeader = &common.ChannelHeader{
			Type:      int32(common.HeaderType_DELIVER_SEEK_INFO),
			ChannelId: channelId,
			Epoch:     uint64(0),
			TxId:      "dynamically generated",
		}

		seekInfo := &ab.SeekInfo{
			Start: &ab.SeekPosition{
				Type: &ab.SeekPosition_Newest{
					Newest: &ab.SeekNewest{},
				},
			},
			Stop: &ab.SeekPosition{
				Type: &ab.SeekPosition_Specified{
					Specified: &ab.SeekSpecified{
						Number: math.MaxUint64,
					},
				},
			},
			Behavior: ab.SeekInfo_BLOCK_UNTIL_READY,
		}
		expectedPayloadData = ProtoMarshal(seekInfo)

		fakeSigner = &mock.SignerIdentity{}
		fakeSigner.SignReturns([]byte("envelop-signature"), nil)

		fakeTxid = "test_txid_123"
		deliverResp = &pb.DeliverResponse{
			Type: &pb.DeliverResponse_FilteredBlock{
				FilteredBlock: createFilteredBlock(channelId, fakeTxid),
			},
		}
		fakeDeliverFiltered = &mock.DeliverFiltered{}
		fakeDeliverFiltered.SendReturns(nil)
		fakeDeliverFiltered.CloseSendReturns(nil)
		fakeDeliverFiltered.RecvReturns(deliverResp, nil)
	})

	Describe("CreateDeliverEnvelope", func() {
		It("returns expected envelope", func() {
			envelope, err := client.CreateDeliverEnvelope(channelId, creator, fakeSigner, nil)
			Expect(err).NotTo(HaveOccurred())

			payload := common.Payload{}
			err = proto.Unmarshal(envelope.Payload, &payload)
			Expect(err).NotTo(HaveOccurred())

//验证有效载荷数据
			Expect(payload.Data).To(Equal(expectedPayloadData))

//验证通道标题
			channelHeader := common.ChannelHeader{}
			err = proto.Unmarshal(payload.Header.ChannelHeader, &channelHeader)
			Expect(err).NotTo(HaveOccurred())
			Expect(channelHeader.ChannelId).To(Equal(expectedChannelHeader.ChannelId))
			Expect(channelHeader.Type).To(Equal(expectedChannelHeader.Type))
			Expect(channelHeader.Epoch).To(Equal(expectedChannelHeader.Epoch))

//验证签名头
			signatureHeader := common.SignatureHeader{}
			err = proto.Unmarshal(payload.Header.SignatureHeader, &signatureHeader)
			Expect(err).NotTo(HaveOccurred())
			Expect(signatureHeader.Creator).To(Equal(creator))

			Expect(fakeSigner.SignCallCount()).To(Equal(1))
			raw := fakeSigner.SignArgsForCall(0)
			Expect(raw).To(Equal(envelope.Payload))
		})

		Context("when SignerIdentity returns error", func() {
			BeforeEach(func() {
				fakeSigner.SignReturns(nil, errors.New("flying-pineapple"))
			})

			It("returns an error", func() {
				_, err := client.CreateDeliverEnvelope(channelId, creator, fakeSigner, nil)
				Expect(err).To(MatchError("flying-pineapple"))
			})
		})
	})

	Describe("DeliverSend", func() {
		var envelope *common.Envelope

		BeforeEach(func() {
			envelope = &common.Envelope{
				Payload:   []byte("envelope-payload"),
				Signature: []byte("envelop-signature"),
			}
		})

		It("returns without error", func() {
			err := client.DeliverSend(fakeDeliverFiltered, "dummyAddress", envelope)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when Deliver.Send returns error", func() {
			BeforeEach(func() {
				fakeDeliverFiltered.SendReturns(errors.New("flying-pineapple"))
			})

			It("returns an error", func() {
				err := client.DeliverSend(fakeDeliverFiltered, "dummyAddress", envelope)
				Expect(err.Error()).To(ContainSubstring("flying-pineapple"))
			})
		})
	})

	Describe("DeliverReceive", func() {
		var (
			eventCh chan client.TxEvent
		)

		BeforeEach(func() {
//eventch缓冲区大小必须大于或等于1
			eventCh = make(chan client.TxEvent, 1)
		})

		It("returns with success status", func() {
			ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
			defer cancelFunc()
			go client.DeliverReceive(fakeDeliverFiltered, "dummyAddress", fakeTxid, eventCh)
			committed, err := client.DeliverWaitForResponse(ctx, eventCh, fakeTxid)
			Expect(err).NotTo(HaveOccurred())
			Expect(committed).To(Equal(true))
		})

		Context("when Deliver.Recv returns error", func() {
			BeforeEach(func() {
				fakeDeliverFiltered.RecvReturns(nil, errors.New("flying-banana"))
			})

			It("returns an error", func() {
				ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
				defer cancelFunc()
				go client.DeliverReceive(fakeDeliverFiltered, "dummyAddress", fakeTxid, eventCh)
				_, err := client.DeliverWaitForResponse(ctx, eventCh, fakeTxid)
				Expect(err.Error()).To(ContainSubstring("flying-banana"))
			})
		})

		Context("when Deliver.Recv doesn't receive response for the txid", func() {
			BeforeEach(func() {
				fakeTxid = "another-txid"
			})

			It("returns an error", func() {
				ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
				defer cancelFunc()
//接收另一个txid的响应
				go client.DeliverReceive(fakeDeliverFiltered, "dummyAddress", fakeTxid, eventCh)
				_, err := client.DeliverWaitForResponse(ctx, eventCh, fakeTxid)
				Expect(err.Error()).To(ContainSubstring("timed out waiting for committing txid " + fakeTxid))
			})
		})
	})
})
