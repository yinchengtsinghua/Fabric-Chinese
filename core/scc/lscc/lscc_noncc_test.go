
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


package lscc_test

import (
	"errors"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/scc/lscc"
	"github.com/hyperledger/fabric/core/scc/lscc/mock"
	pb "github.com/hyperledger/fabric/protos/peer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LSCC", func() {

	var (
		l                 *lscc.LifeCycleSysCC
		fakeSupport       *mock.FileSystemSupport
		fakeSCCProvider   *mock.SystemChaincodeProvider
		fakeQueryExecutor *mock.QueryExecutor
		ccData            *ccprovider.ChaincodeData
		ccDataBytes       []byte
		err               error
	)

	BeforeEach(func() {
		fakeSupport = &mock.FileSystemSupport{}
		fakeSCCProvider = &mock.SystemChaincodeProvider{}
		fakeQueryExecutor = &mock.QueryExecutor{}

		l = &lscc.LifeCycleSysCC{
			Support:     fakeSupport,
			SCCProvider: fakeSCCProvider,
		}

		ccData = &ccprovider.ChaincodeData{
			Name:                "chaincode-data-name",
			Version:             "version",
			Escc:                "escc",
			Vscc:                "vscc",
			Policy:              []byte("policy"),
			Data:                []byte("data"),
			Id:                  []byte("id"),
			InstantiationPolicy: []byte("instantiation-policy"),
		}

		ccDataBytes, err = proto.Marshal(ccData)
		Expect(err).NotTo(HaveOccurred())

		fakeQueryExecutor = &mock.QueryExecutor{}
		fakeQueryExecutor.GetStateReturns(ccDataBytes, nil)
	})

	Describe("GetChaincodeDeploymentSpec", func() {
		var (
			fakeCCPackage  *mock.CCPackage
			deploymentSpec *pb.ChaincodeDeploymentSpec
		)

		BeforeEach(func() {
			fakeSCCProvider.GetQueryExecutorForLedgerReturns(fakeQueryExecutor, nil)

			deploymentSpec = &pb.ChaincodeDeploymentSpec{
				ChaincodeSpec: &pb.ChaincodeSpec{
					ChaincodeId: &pb.ChaincodeID{
						Name: "chaincode-name",
					},
				},
			}

			fakeCCPackage = &mock.CCPackage{}
			fakeCCPackage.GetDepSpecReturns(deploymentSpec)

			fakeSupport.GetChaincodeFromLocalStorageReturns(fakeCCPackage, nil)
		})

		It("returns the chaincode deployment spec for a valid chaincode", func() {
			ccci, err := l.ChaincodeContainerInfo("chaincode-data-name", fakeQueryExecutor)
			Expect(err).NotTo(HaveOccurred())
			Expect(ccci).To(Equal(ccprovider.DeploymentSpecToChaincodeContainerInfo(deploymentSpec)))

			Expect(fakeQueryExecutor.GetStateCallCount()).To(Equal(1))
			getStateNamespace, getStateCCName := fakeQueryExecutor.GetStateArgsForCall(0)
			Expect(getStateNamespace).To(Equal("lscc"))
			Expect(getStateCCName).To(Equal("chaincode-data-name"))
		})

		Context("when the get state query fails", func() {
			BeforeEach(func() {
				fakeQueryExecutor.GetStateReturns(nil, errors.New("fake-error"))
			})

			It("wraps and returns the error", func() {
				_, err := l.ChaincodeContainerInfo("chaincode-data-name", fakeQueryExecutor)
				Expect(err).To(MatchError("could not retrieve state for chaincode chaincode-data-name: fake-error"))
			})
		})

		Context("when the chaincode is not found in the table", func() {
			BeforeEach(func() {
				fakeQueryExecutor.GetStateReturns(nil, nil)
			})

			It("returns an error", func() {
				_, err := l.ChaincodeContainerInfo("chaincode-data-name", fakeQueryExecutor)
				Expect(err).To(MatchError("chaincode chaincode-data-name not found"))
			})
		})
	})

	Describe("ChaincodeDefinition", func() {
		BeforeEach(func() {
		})

		It("retrieves the chaincode data from the state", func() {
			chaincodeDefinition, err := l.ChaincodeDefinition("cc-name", fakeQueryExecutor)
			Expect(err).NotTo(HaveOccurred())
			returnedChaincodeData, ok := chaincodeDefinition.(*ccprovider.ChaincodeData)
			Expect(ok).To(BeTrue())
			Expect(returnedChaincodeData).To(Equal(ccData))

			Expect(fakeQueryExecutor.GetStateCallCount()).To(Equal(1))
			namespace, key := fakeQueryExecutor.GetStateArgsForCall(0)
			Expect(namespace).To(Equal("lscc"))
			Expect(key).To(Equal("cc-name"))
		})

		Context("when the state getter fails", func() {
			BeforeEach(func() {
				fakeQueryExecutor.GetStateReturns(nil, errors.New("fake-error"))
			})

			It("returns the wrapped error", func() {
				_, err := l.ChaincodeDefinition("cc-name", fakeQueryExecutor)
				Expect(err).To(MatchError("could not retrieve state for chaincode cc-name: fake-error"))
			})
		})

		Context("when the state getter does not find the key", func() {
			BeforeEach(func() {
				fakeQueryExecutor.GetStateReturns(nil, nil)
			})

			It("returns an error", func() {
				_, err := l.ChaincodeDefinition("cc-name", fakeQueryExecutor)
				Expect(err).To(MatchError("chaincode cc-name not found"))
			})
		})

		Context("when the state getter returns invalid data", func() {
			BeforeEach(func() {
				fakeQueryExecutor.GetStateReturns([]byte("garbage"), nil)
			})

			It("wraps and returns the error", func() {
				_, err := l.ChaincodeDefinition("cc-name", fakeQueryExecutor)
				Expect(err).To(MatchError(MatchRegexp("chaincode cc-name has bad definition: proto:.*")))
			})
		})
	})
})
