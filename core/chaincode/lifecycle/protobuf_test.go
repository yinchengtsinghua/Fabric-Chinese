
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


package lifecycle_test

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/lifecycle"
	lc "github.com/hyperledger/fabric/protos/peer/lifecycle"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProtobufImpl", func() {
	var (
		pi        *lifecycle.ProtobufImpl
		sampleMsg *lc.InstallChaincodeArgs
	)

	BeforeEach(func() {
		pi = &lifecycle.ProtobufImpl{}
		sampleMsg = &lc.InstallChaincodeArgs{
			Name:                    "name",
			Version:                 "version",
			ChaincodeInstallPackage: []byte("install-package"),
		}
	})

	Describe("Marshal", func() {
		It("passes through to the proto implementation", func() {
			res, err := pi.Marshal(sampleMsg)
			Expect(err).NotTo(HaveOccurred())

			msg := &lc.InstallChaincodeArgs{}
			err = proto.Unmarshal(res, msg)
			Expect(err).NotTo(HaveOccurred())
			Expect(proto.Equal(msg, sampleMsg)).To(BeTrue())
		})
	})

	Describe("Unmarshal", func() {
		It("passes through to the proto implementation", func() {
			res, err := proto.Marshal(sampleMsg)
			Expect(err).NotTo(HaveOccurred())

			msg := &lc.InstallChaincodeArgs{}
			err = pi.Unmarshal(res, msg)
			Expect(err).NotTo(HaveOccurred())
			Expect(proto.Equal(msg, sampleMsg)).To(BeTrue())
		})
	})
})
