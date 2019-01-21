
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


package cc_test

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/common/flogging/floggingtest"
	"github.com/hyperledger/fabric/core/cclifecycle"
	"github.com/hyperledger/fabric/core/cclifecycle/mocks"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
	"github.com/hyperledger/fabric/protos/utils"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewQuery(t *testing.T) {
//这测试querycreatorfunc可以将下面的函数强制转换为接口类型
	var q cc.Query
	queryCreator := func() (cc.Query, error) {
		q := &mocks.Query{}
		q.On("Done")
		return q, nil
	}
	q, _ = cc.QueryCreatorFunc(queryCreator).NewQuery()
	q.Done()
}

func TestHandleMetadataUpdate(t *testing.T) {
	f := func(channel string, chaincodes chaincode.MetadataSet) {
		assert.Len(t, chaincodes, 2)
		assert.Equal(t, "mychannel", channel)
	}
	cc.HandleMetadataUpdate(f).LifeCycleChangeListener("mychannel", chaincode.MetadataSet{{}, {}})
}

func TestEnumerate(t *testing.T) {
	f := func() ([]chaincode.InstalledChaincode, error) {
		return []chaincode.InstalledChaincode{{}, {}}, nil
	}
	ccs, err := cc.Enumerate(f).Enumerate()
	assert.NoError(t, err)
	assert.Len(t, ccs, 2)
}

func TestLifecycleInitFailure(t *testing.T) {
	listCCs := &mocks.Enumerator{}
	listCCs.On("Enumerate").Return(nil, errors.New("failed accessing DB"))
	lc, err := cc.NewLifeCycle(listCCs)
	assert.Nil(t, lc)
	assert.Contains(t, err.Error(), "failed accessing DB")
}

func TestHandleChaincodeDeployGreenPath(t *testing.T) {
	recorder, restoreLogger := newLogRecorder(t)
	defer restoreLogger()

	cc1Bytes := utils.MarshalOrPanic(&ccprovider.ChaincodeData{
		Name:    "cc1",
		Version: "1.0",
		Id:      []byte{42},
		Policy:  []byte{1, 2, 3, 4, 5},
	})

	cc2Bytes := utils.MarshalOrPanic(&ccprovider.ChaincodeData{
		Name:    "cc2",
		Version: "1.0",
		Id:      []byte{42},
	})

	cc3Bytes := utils.MarshalOrPanic(&ccprovider.ChaincodeData{
		Name:    "cc3",
		Version: "1.0",
		Id:      []byte{42},
	})

	query := &mocks.Query{}
	query.On("GetState", "lscc", "cc1").Return(cc1Bytes, nil)
	query.On("GetState", "lscc", "cc2").Return(cc2Bytes, nil)
	query.On("GetState", "lscc", "cc3").Return(cc3Bytes, nil).Once()
	query.On("Done")
	queryCreator := &mocks.QueryCreator{}
	queryCreator.On("NewQuery").Return(query, nil)

	enum := &mocks.Enumerator{}
	enum.On("Enumerate").Return([]chaincode.InstalledChaincode{
		{
			Name:    "cc1",
			Version: "1.0",
			Id:      []byte{42},
		},
		{
//此链码安装的版本与实例化的版本不同
			Name:    "cc2",
			Version: "1.1",
			Id:      []byte{50},
		},
		{
//此链码未在通道上实例化（ID为50，但状态为42），但已安装
			Name:    "cc3",
			Version: "1.0",
			Id:      []byte{50},
		},
	}, nil)

	lc, err := cc.NewLifeCycle(enum)
	assert.NoError(t, err)

	lsnr := &mocks.LifeCycleChangeListener{}
	lsnr.On("LifeCycleChangeListener", mock.Anything, mock.Anything)
	lc.AddListener(lsnr)

	sub, err := lc.NewChannelSubscription("mychannel", queryCreator)
	assert.NoError(t, err)
	assert.NotNil(t, sub)

//
	assertLogged(t, recorder, "Listeners for channel mychannel invoked")
	lsnr.AssertCalled(t, "LifeCycleChangeListener", "mychannel", chaincode.MetadataSet{chaincode.Metadata{
		Name:    "cc1",
		Version: "1.0",
		Id:      []byte{42},
		Policy:  []byte{1, 2, 3, 4, 5},
	}})

//发出新链码部署的信号，并确保使用两个链码更新链码侦听器。
	cc3Bytes = utils.MarshalOrPanic(&ccprovider.ChaincodeData{
		Name:    "cc3",
		Version: "1.0",
		Id:      []byte{50},
	})
	query.On("GetState", "lscc", "cc3").Return(cc3Bytes, nil).Once()
	sub.HandleChaincodeDeploy(&cceventmgmt.ChaincodeDefinition{Name: "cc3", Version: "1.0", Hash: []byte{50}}, nil)
	sub.ChaincodeDeployDone(true)
//确保使用新的chaincode和旧的chaincode元数据调用侦听器。
	assertLogged(t, recorder, "Listeners for channel mychannel invoked")
	assert.Len(t, lsnr.Calls, 2)
	sortedMetadata := sortedMetadataSet(lsnr.Calls[1].Arguments.Get(1).(chaincode.MetadataSet)).sort()
	assert.Equal(t, sortedMetadata, chaincode.MetadataSet{{
		Name:    "cc1",
		Version: "1.0",
		Id:      []byte{42},
		Policy:  []byte{1, 2, 3, 4, 5},
	}, {
		Name:    "cc3",
		Version: "1.0",
		Id:      []byte{50},
	}})

//接下来，更新第二个chaincode的chaincode元数据，以确保使用更新的
//元数据，而不是旧的元数据。
	cc3Bytes = utils.MarshalOrPanic(&ccprovider.ChaincodeData{
		Name:    "cc3",
		Version: "1.1",
		Id:      []byte{50},
	})
	query.On("GetState", "lscc", "cc3").Return(cc3Bytes, nil).Once()
	sub.HandleChaincodeDeploy(&cceventmgmt.ChaincodeDefinition{Name: "cc3", Version: "1.1", Hash: []byte{50}}, nil)
	sub.ChaincodeDeployDone(true)
//确保使用新的chaincode和旧的chaincode元数据调用侦听器。
	assertLogged(t, recorder, "Listeners for channel mychannel invoked")
	assert.Len(t, lsnr.Calls, 3)
	sortedMetadata = sortedMetadataSet(lsnr.Calls[2].Arguments.Get(1).(chaincode.MetadataSet)).sort()
	assert.Equal(t, sortedMetadata, chaincode.MetadataSet{{
		Name:    "cc1",
		Version: "1.0",
		Id:      []byte{42},
		Policy:  []byte{1, 2, 3, 4, 5},
	}, {
		Name:    "cc3",
		Version: "1.1",
		Id:      []byte{50},
	}})
}

func TestHandleChaincodeDeployFailures(t *testing.T) {
	recorder, restoreLogger := newLogRecorder(t)
	defer restoreLogger()

	cc1Bytes := utils.MarshalOrPanic(&ccprovider.ChaincodeData{
		Name:    "cc1",
		Version: "1.0",
		Id:      []byte{42},
		Policy:  []byte{1, 2, 3, 4, 5},
	})

	query := &mocks.Query{}
	query.On("Done")
	queryCreator := &mocks.QueryCreator{}
	enum := &mocks.Enumerator{}
	enum.On("Enumerate").Return([]chaincode.InstalledChaincode{
		{
			Name:    "cc1",
			Version: "1.0",
			Id:      []byte{42},
		},
	}, nil)

	lc, err := cc.NewLifeCycle(enum)
	assert.NoError(t, err)

	lsnr := &mocks.LifeCycleChangeListener{}
	lsnr.On("LifeCycleChangeListener", mock.Anything, mock.Anything)
	lc.AddListener(lsnr)

//场景一：进行了通道订阅，但无法获取新查询。
	queryCreator.On("NewQuery").Return(nil, errors.New("failed accessing DB")).Once()
	sub, err := lc.NewChannelSubscription("mychannel", queryCreator)
	assert.Nil(t, sub)
	assert.Contains(t, err.Error(), "failed accessing DB")
	lsnr.AssertNumberOfCalls(t, "LifeCycleChangeListener", 0)

//场景二：进行了一个通道订阅，获得一个新的查询成功，但是-获得它一次
//部署通知发生-失败。
	queryCreator.On("NewQuery").Return(query, nil).Once()
	queryCreator.On("NewQuery").Return(nil, errors.New("failed accessing DB")).Once()
	query.On("GetState", "lscc", "cc1").Return(cc1Bytes, nil).Once()
	sub, err = lc.NewChannelSubscription("mychannel", queryCreator)
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	lsnr.AssertNumberOfCalls(t, "LifeCycleChangeListener", 1)
	sub.HandleChaincodeDeploy(&cceventmgmt.ChaincodeDefinition{Name: "cc1", Version: "1.0", Hash: []byte{42}}, nil)
	sub.ChaincodeDeployDone(true)
	assertLogged(t, recorder, "Failed creating a new query for channel mychannel: failed accessing DB")
	lsnr.AssertNumberOfCalls(t, "LifeCycleChangeListener", 1)

//场景三：进行通道订阅，在订阅初始化时获取新查询都会成功
//以及在部署通知时。但是-getstate返回一个错误。
//注意：因为我们订阅了两次同一个通道，所以没有从statedb加载信息，因为它已经有了。
	queryCreator.On("NewQuery").Return(query, nil).Once()
	query.On("GetState", "lscc", "cc1").Return(nil, errors.New("failed accessing DB")).Once()
	sub, err = lc.NewChannelSubscription("mychannel", queryCreator)
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	lsnr.AssertNumberOfCalls(t, "LifeCycleChangeListener", 2)
	sub.HandleChaincodeDeploy(&cceventmgmt.ChaincodeDefinition{Name: "cc1", Version: "1.0", Hash: []byte{42}}, nil)
	sub.ChaincodeDeployDone(true)
	assertLogged(t, recorder, "Query for channel mychannel for Name=cc1, Version=1.0, Hash=[]byte{0x2a} failed with error failed accessing DB")
	lsnr.AssertNumberOfCalls(t, "LifeCycleChangeListener", 2)

//场景四：成功进行通道订阅，订阅初始化成功获取新查询。
//但是-部署通知指示部署失败。
//因此，不应调用生命周期更改侦听器。
	sub, err = lc.NewChannelSubscription("mychannel", queryCreator)
	lsnr.AssertNumberOfCalls(t, "LifeCycleChangeListener", 3)
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	sub.HandleChaincodeDeploy(&cceventmgmt.ChaincodeDefinition{Name: "cc1", Version: "1.1", Hash: []byte{42}}, nil)
	sub.ChaincodeDeployDone(false)
	lsnr.AssertNumberOfCalls(t, "LifeCycleChangeListener", 3)
	assertLogged(t, recorder, "Chaincode deploy for cc1 failed")
}

func TestMetadata(t *testing.T) {
	recorder, restoreLogger := newLogRecorder(t)
	defer restoreLogger()

	cc1Bytes := utils.MarshalOrPanic(&ccprovider.ChaincodeData{
		Name:    "cc1",
		Version: "1.0",
		Id:      []byte{42},
		Policy:  []byte{1, 2, 3, 4, 5},
	})

	cc2Bytes := utils.MarshalOrPanic(&ccprovider.ChaincodeData{
		Name:    "cc2",
		Version: "1.0",
		Id:      []byte{42},
	})

	query := &mocks.Query{}
	query.On("GetState", "lscc", "cc3").Return(cc1Bytes, nil)
	query.On("Done")
	queryCreator := &mocks.QueryCreator{}

	enum := &mocks.Enumerator{}
	enum.On("Enumerate").Return([]chaincode.InstalledChaincode{
		{
			Name:    "cc1",
			Version: "1.0",
			Id:      []byte{42},
		},
	}, nil)

	lc, err := cc.NewLifeCycle(enum)
	assert.NoError(t, err)

//场景一：生命周期中没有调用订阅
	md := lc.Metadata("mychannel", "cc1", false)
	assert.Nil(t, md)
	assertLogged(t, recorder, "Requested Metadata for non-existent channel mychannel")

//
//因为chaincode是在订阅之前安装的，所以它是在订阅期间加载的。
	query.On("GetState", "lscc", "cc1").Return(cc1Bytes, nil).Once()
	queryCreator.On("NewQuery").Return(query, nil).Once()
	sub, err := lc.NewChannelSubscription("mychannel", queryCreator)
	defer sub.ChaincodeDeployDone(true)
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	md = lc.Metadata("mychannel", "cc1", false)
	assert.Equal(t, &chaincode.Metadata{
		Name:    "cc1",
		Version: "1.0",
		Id:      []byte{42},
		Policy:  []byte{1, 2, 3, 4, 5},
	}, md)
	assertLogged(t, recorder, "Returning metadata for channel mychannel , chaincode cc1")

//场景三：进行元数据检索，链码还不在内存中，
//
	queryCreator.On("NewQuery").Return(nil, errors.New("failed obtaining query executor")).Once()
	md = lc.Metadata("mychannel", "cc2", false)
	assert.Nil(t, md)
	assertLogged(t, recorder, "Failed obtaining new query for channel mychannel : failed obtaining query executor")

//场景四：进行元数据检索，链码还不在内存中，
//
	queryCreator.On("NewQuery").Return(query, nil).Once()
	query.On("GetState", "lscc", "cc2").Return(nil, errors.New("GetState failed")).Once()
	md = lc.Metadata("mychannel", "cc2", false)
	assert.Nil(t, md)
	assertLogged(t, recorder, "Failed querying LSCC for channel mychannel : GetState failed")

//场景五：进行元数据检索，链码还不在内存中，
//查询和getstate都成功，但是-getstate返回nil
	queryCreator.On("NewQuery").Return(query, nil).Once()
	query.On("GetState", "lscc", "cc2").Return(nil, nil).Once()
	md = lc.Metadata("mychannel", "cc2", false)
	assert.Nil(t, md)
	assertLogged(t, recorder, "Chaincode cc2 isn't defined in channel mychannel")

//场景六：进行元数据检索，链码还不在内存中，
//查询和getstate都成功，但是-getstate返回有效的元数据
	queryCreator.On("NewQuery").Return(query, nil).Once()
	query.On("GetState", "lscc", "cc2").Return(cc2Bytes, nil).Once()
	md = lc.Metadata("mychannel", "cc2", false)
	assert.Equal(t, &chaincode.Metadata{
		Name:    "cc2",
		Version: "1.0",
		Id:      []byte{42},
	}, md)

//场景七：进行元数据检索，链码在内存中，
//但是也指定了一个集合，因此-检索应该绕过内存缓存
//直接进入StateDB。
	queryCreator.On("NewQuery").Return(query, nil).Once()
	query.On("GetState", "lscc", "cc1").Return(cc1Bytes, nil).Once()
	query.On("GetState", "lscc", privdata.BuildCollectionKVSKey("cc1")).Return([]byte{10, 10, 10}, nil).Once()
	md = lc.Metadata("mychannel", "cc1", true)
	assert.Equal(t, &chaincode.Metadata{
		Name:              "cc1",
		Version:           "1.0",
		Id:                []byte{42},
		Policy:            []byte{1, 2, 3, 4, 5},
		CollectionsConfig: []byte{10, 10, 10},
	}, md)
	assertLogged(t, recorder, "Retrieved collection config for cc1 from cc1~collection")

//场景八：进行元数据检索，链码在内存中，
//但是也指定了一个集合，因此-检索应该绕过内存缓存
//直接进入StateDB。但是-检索失败
	queryCreator.On("NewQuery").Return(query, nil).Once()
	query.On("GetState", "lscc", "cc1").Return(cc1Bytes, nil).Once()
	query.On("GetState", "lscc", privdata.BuildCollectionKVSKey("cc1")).Return(nil, errors.New("foo")).Once()
	md = lc.Metadata("mychannel", "cc1", true)
	assert.Nil(t, md)
	assertLogged(t, recorder, "Failed querying lscc namespace for cc1~collection: foo")
}

func newLogRecorder(t *testing.T) (*floggingtest.Recorder, func()) {
	oldLogger := cc.Logger

	logger, recorder := floggingtest.NewTestLogger(t)
	cc.Logger = logger

	return recorder, func() { cc.Logger = oldLogger }
}

func assertLogged(t *testing.T, r *floggingtest.Recorder, msg string) {
	gt := NewGomegaWithT(t)
	gt.Eventually(r).Should(gbytes.Say(regexp.QuoteMeta(msg)))
}

type sortedMetadataSet chaincode.MetadataSet

func (mds sortedMetadataSet) Len() int {
	return len(mds)
}

func (mds sortedMetadataSet) Less(i, j int) bool {
	eI := strings.Replace(mds[i].Name, "cc", "", -1)
	eJ := strings.Replace(mds[j].Name, "cc", "", -1)
	nI, _ := strconv.ParseInt(eI, 10, 32)
	nJ, _ := strconv.ParseInt(eJ, 10, 32)
	return nI < nJ
}

func (mds sortedMetadataSet) Swap(i, j int) {
	mds[i], mds[j] = mds[j], mds[i]
}

func (mds sortedMetadataSet) sort() chaincode.MetadataSet {
	sort.Sort(mds)
	return chaincode.MetadataSet(mds)
}
