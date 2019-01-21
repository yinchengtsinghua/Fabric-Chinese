
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


package privacyenabledstate_test

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/stateleveldb"
	"github.com/hyperledger/fabric/core/ledger/mock"
	. "github.com/onsi/gomega"
)

func TestHealthCheckRegister(t *testing.T) {
	gt := NewGomegaWithT(t)
	fakeHealthCheckRegistry := &mock.HealthCheckRegistry{}

	dbProvider := &privacyenabledstate.CommonStorageDBProvider{
		VersionedDBProvider: &stateleveldb.VersionedDBProvider{},
		HealthCheckRegistry: fakeHealthCheckRegistry,
	}

	err := dbProvider.RegisterHealthChecker()
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(fakeHealthCheckRegistry.RegisterCheckerCallCount()).To(Equal(0))

	dbProvider.VersionedDBProvider = &statecouchdb.VersionedDBProvider{}
	err = dbProvider.RegisterHealthChecker()
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(fakeHealthCheckRegistry.RegisterCheckerCallCount()).To(Equal(1))

	arg1, arg2 := fakeHealthCheckRegistry.RegisterCheckerArgsForCall(0)
	gt.Expect(arg1).To(Equal("couchdb"))
	gt.Expect(arg2).NotTo(Equal(nil))
}
