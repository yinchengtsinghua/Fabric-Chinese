
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
	"io"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/hyperledger/fabric/protos/token"
	"github.com/hyperledger/fabric/token/client"
	"github.com/hyperledger/fabric/token/client/mock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

//go：生成伪造者-o mock/prover_client.go-伪造名称prover client。代理人

type proverClient interface {
	token.ProverClient
}

var _ = Describe("TokenClient", func() {
	var (
		channelId            string
		commandHeader        *token.Header
		signedCommandResp    *token.SignedCommandResponse
		fakeIdentity         *mock.Identity
		fakeSigningIdentity  *mock.SigningIdentity
		fakeRandomnessReader io.Reader
		fakeProverClient     *mock.ProverClient

		prover client.Prover
	)

	BeforeEach(func() {
		channelId = "mychannel"

		nonce := make([]byte, 32)
		ts, _ := ptypes.TimestampProto(clock())
		commandHeader = &token.Header{
			Timestamp: ts,
			Nonce:     nonce,
			Creator:   []byte("Alice"),
			ChannelId: channelId,
		}

		fakeIdentity = &mock.Identity{}
		fakeSigningIdentity = &mock.SigningIdentity{}
		fakeRandomnessReader = strings.NewReader(string(nonce))
		fakeProverClient = &mock.ProverClient{}

		signedCommandResp = &token.SignedCommandResponse{
			Response:  []byte("command-response"),
			Signature: []byte("response-signature"),
		}

		fakeIdentity.SerializeReturns([]byte("Alice"), nil)
		fakeSigningIdentity.GetPublicVersionReturns(fakeIdentity)
		fakeSigningIdentity.SignReturns([]byte("pineapple"), nil)
		fakeProverClient.ProcessCommandReturns(signedCommandResp, nil)
		prover = &client.ProverPeer{RandomnessReader: fakeRandomnessReader, ProverClient: fakeProverClient, ChannelID: channelId, Time: clock}
	})

	Describe("RequestImport", func() {
		var (
			tokensToIssue     []*token.TokenToIssue
			marshalledCommand []byte
			signedCommand     *token.SignedCommand
		)

		BeforeEach(func() {
			tokensToIssue = []*token.TokenToIssue{{
				Type:      "type",
				Quantity:  10,
				Recipient: []byte("alice"),
			}}

			command := &token.Command{
				Header: commandHeader,
				Payload: &token.Command_ImportRequest{
					ImportRequest: &token.ImportRequest{
						TokensToIssue: tokensToIssue,
					},
				},
			}
			marshalledCommand = ProtoMarshal(command)
			signedCommand = &token.SignedCommand{
				Command:   marshalledCommand,
				Signature: []byte("pineapple"),
			}
		})

		It("returns serialized token transaction", func() {
			response, err := prover.RequestImport(tokensToIssue, fakeSigningIdentity)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(signedCommandResp.Response))

			Expect(fakeIdentity.SerializeCallCount()).To(Equal(1))
			Expect(fakeSigningIdentity.SignCallCount()).To(Equal(1))
			raw := fakeSigningIdentity.SignArgsForCall(0)
			Expect(raw).To(Equal(marshalledCommand))

			Expect(fakeProverClient.ProcessCommandCallCount()).To(Equal(1))
			_, sc, _ := fakeProverClient.ProcessCommandArgsForCall(0)
			Expect(sc).To(Equal(signedCommand))
		})

		Context("when SigningIdentity serialize fails", func() {
			BeforeEach(func() {
				fakeIdentity.SerializeReturns(nil, errors.New("wild-banana"))
			})

			It("returns an error", func() {
				_, err := prover.RequestImport(tokensToIssue, fakeSigningIdentity)
				Expect(err).To(MatchError("wild-banana"))
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(0))
				Expect(fakeProverClient.ProcessCommandCallCount()).To(Equal(0))
			})
		})

		Context("when SigningIdentity sign fails", func() {
			BeforeEach(func() {
				fakeSigningIdentity.SignReturns(nil, errors.New("wild-banana"))
			})

			It("returns an error", func() {
				_, err := prover.RequestImport(tokensToIssue, fakeSigningIdentity)
				Expect(err).To(MatchError("wild-banana"))

				Expect(fakeProverClient.ProcessCommandCallCount()).To(Equal(0))
				Expect(fakeIdentity.SerializeCallCount()).To(Equal(1))
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(1))
				raw := fakeSigningIdentity.SignArgsForCall(0)
				Expect(raw).To(Equal(marshalledCommand))
			})
		})

		Context("when processcommand fails", func() {
			BeforeEach(func() {
				fakeProverClient.ProcessCommandReturns(nil, errors.New("wild-banana"))
			})

			It("returns an error", func() {
				_, err := prover.RequestImport(tokensToIssue, fakeSigningIdentity)
				Expect(err).To(MatchError("wild-banana"))
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(1))
				Expect(fakeProverClient.ProcessCommandCallCount()).To(Equal(1))
			})
		})
	})

	Describe("RequestTransfer", func() {
		var (
			tokenIDs          [][]byte
			transferShares    []*token.RecipientTransferShare
			marshalledCommand []byte
			signedCommand     *token.SignedCommand
		)

		BeforeEach(func() {
//输入传输数据
			tokenIDs = [][]byte{[]byte("id1"), []byte("id2")}
			transferShares = []*token.RecipientTransferShare{
				{Recipient: []byte("alice"), Quantity: 100},
				{Recipient: []byte("Bob"), Quantity: 50},
			}

			command := &token.Command{
				Header: commandHeader,
				Payload: &token.Command_TransferRequest{
					TransferRequest: &token.TransferRequest{
						TokenIds: tokenIDs,
						Shares:   transferShares,
					},
				},
			}
			marshalledCommand = ProtoMarshal(command)
			signedCommand = &token.SignedCommand{
				Command:   marshalledCommand,
				Signature: []byte("pineapple"),
			}
		})

		It("returns serialized token transaction", func() {
			response, err := prover.RequestTransfer(tokenIDs, transferShares, fakeSigningIdentity)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(signedCommandResp.Response))

			Expect(fakeIdentity.SerializeCallCount()).To(Equal(1))
			Expect(fakeSigningIdentity.SignCallCount()).To(Equal(1))
			raw := fakeSigningIdentity.SignArgsForCall(0)
			Expect(raw).To(Equal(marshalledCommand))

			Expect(fakeProverClient.ProcessCommandCallCount()).To(Equal(1))
			_, sc, _ := fakeProverClient.ProcessCommandArgsForCall(0)
			Expect(sc).To(Equal(signedCommand))
		})

		Context("when Identity serialize fails", func() {
			BeforeEach(func() {
				fakeIdentity.SerializeReturns(nil, errors.New("wild-banana"))
			})

			It("returns an error", func() {
				_, err := prover.RequestTransfer(tokenIDs, transferShares, fakeSigningIdentity)
				Expect(err).To(MatchError("wild-banana"))
				Expect(fakeIdentity.SerializeCallCount()).To(Equal(1))
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(0))
				Expect(fakeProverClient.ProcessCommandCallCount()).To(Equal(0))
			})
		})

		Context("when SigningIdentity sign fails", func() {
			BeforeEach(func() {
				fakeSigningIdentity.SignReturns(nil, errors.New("wild-banana"))
			})

			It("returns an error", func() {
				_, err := prover.RequestTransfer(tokenIDs, transferShares, fakeSigningIdentity)
				Expect(err).To(MatchError("wild-banana"))

				Expect(fakeProverClient.ProcessCommandCallCount()).To(Equal(0))
				Expect(fakeIdentity.SerializeCallCount()).To(Equal(1))
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(1))
				raw := fakeSigningIdentity.SignArgsForCall(0)
				Expect(raw).To(Equal(marshalledCommand))
			})
		})

		Context("when processcommand fails", func() {
			BeforeEach(func() {
				fakeProverClient.ProcessCommandReturns(nil, errors.New("wild-banana"))
			})

			It("returns an error", func() {
				_, err := prover.RequestTransfer(tokenIDs, transferShares, fakeSigningIdentity)
				Expect(err).To(MatchError("wild-banana"))
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(1))
				Expect(fakeProverClient.ProcessCommandCallCount()).To(Equal(1))
			})
		})
	})
})

func clock() time.Time {
	return time.Time{}
}
