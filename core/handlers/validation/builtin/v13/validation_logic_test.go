
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

package v13

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/capabilities"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/channelconfig"
	commonerrors "github.com/hyperledger/fabric/common/errors"
	mc "github.com/hyperledger/fabric/common/mocks/config"
	lm "github.com/hyperledger/fabric/common/mocks/ledger"
	"github.com/hyperledger/fabric/common/mocks/scc"
	"github.com/hyperledger/fabric/common/util"
	aclmocks "github.com/hyperledger/fabric/core/aclmgmt/mocks"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	mocks2 "github.com/hyperledger/fabric/core/committer/txvalidator/mocks"
	"github.com/hyperledger/fabric/core/common/ccpackage"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	cutils "github.com/hyperledger/fabric/core/container/util"
	"github.com/hyperledger/fabric/core/handlers/validation/api/capabilities"
	"github.com/hyperledger/fabric/core/handlers/validation/builtin/v13/mocks"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	corepeer "github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	"github.com/hyperledger/fabric/core/scc/lscc"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createTx(endorsedByDuplicatedIdentity bool) (*common.Envelope, error) {
	ccid := &peer.ChaincodeID{Name: "foo", Version: "v1"}
	cis := &peer.ChaincodeInvocationSpec{ChaincodeSpec: &peer.ChaincodeSpec{ChaincodeId: ccid}}

	prop, _, err := utils.CreateProposalFromCIS(common.HeaderType_ENDORSER_TRANSACTION, util.GetTestChainID(), cis, sid)
	if err != nil {
		return nil, err
	}

	presp, err := utils.CreateProposalResponse(prop.Header, prop.Payload, &peer.Response{Status: 200}, []byte("res"), nil, ccid, nil, id)
	if err != nil {
		return nil, err
	}

	var env *common.Envelope
	if endorsedByDuplicatedIdentity {
		env, err = utils.CreateSignedTx(prop, id, presp, presp)
	} else {
		env, err = utils.CreateSignedTx(prop, id, presp)
	}
	if err != nil {
		return nil, err
	}
	return env, err
}

func processSignedCDS(cds *peer.ChaincodeDeploymentSpec, policy *common.SignaturePolicyEnvelope) ([]byte, error) {
	env, err := ccpackage.OwnerCreateSignedCCDepSpec(cds, policy, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create package %s", err)
	}

	b := utils.MarshalOrPanic(env)

	ccpack := &ccprovider.SignedCDSPackage{}
	cd, err := ccpack.InitFromBuffer(b)
	if err != nil {
		return nil, fmt.Errorf("error owner creating package %s", err)
	}

	if err = ccpack.PutChaincodeToFS(); err != nil {
		return nil, fmt.Errorf("error putting package on the FS %s", err)
	}

	cd.InstantiationPolicy = utils.MarshalOrPanic(policy)

	return utils.MarshalOrPanic(cd), nil
}

func constructDeploymentSpec(name string, path string, version string, initArgs [][]byte, createFS bool) (*peer.ChaincodeDeploymentSpec, error) {
	spec := &peer.ChaincodeSpec{Type: 1, ChaincodeId: &peer.ChaincodeID{Name: name, Path: path, Version: version}, Input: &peer.ChaincodeInput{Args: initArgs}}

	codePackageBytes := bytes.NewBuffer(nil)
	gz := gzip.NewWriter(codePackageBytes)
	tw := tar.NewWriter(gz)

	err := cutils.WriteBytesToPackage("src/garbage.go", []byte(name+path+version), tw)
	if err != nil {
		return nil, err
	}

	tw.Close()
	gz.Close()

	chaincodeDeploymentSpec := &peer.ChaincodeDeploymentSpec{ChaincodeSpec: spec, CodePackage: codePackageBytes.Bytes()}

	if createFS {
		err := ccprovider.PutChaincodeIntoFS(chaincodeDeploymentSpec)
		if err != nil {
			return nil, err
		}
	}

	return chaincodeDeploymentSpec, nil
}

func createCCDataRWsetWithCollection(nameK, nameV, version string, policy []byte, collectionConfigPackage []byte) ([]byte, error) {
	cd := &ccprovider.ChaincodeData{
		Name:                nameV,
		Version:             version,
		InstantiationPolicy: policy,
	}

	cdbytes := utils.MarshalOrPanic(cd)

	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	rwsetBuilder.AddToWriteSet("lscc", nameK, cdbytes)
	rwsetBuilder.AddToWriteSet("lscc", privdata.BuildCollectionKVSKey(nameK), collectionConfigPackage)
	sr, err := rwsetBuilder.GetTxSimulationResults()
	if err != nil {
		return nil, err
	}
	return sr.GetPubSimulationBytes()
}

func createCCDataRWset(nameK, nameV, version string, policy []byte) ([]byte, error) {
	cd := &ccprovider.ChaincodeData{
		Name:                nameV,
		Version:             version,
		InstantiationPolicy: policy,
	}

	cdbytes := utils.MarshalOrPanic(cd)

	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	rwsetBuilder.AddToWriteSet("lscc", nameK, cdbytes)
	sr, err := rwsetBuilder.GetTxSimulationResults()
	if err != nil {
		return nil, err
	}
	return sr.GetPubSimulationBytes()
}

func createLSCCTxWithCollection(ccname, ccver, f string, res []byte, policy []byte, ccpBytes []byte) (*common.Envelope, error) {
	return createLSCCTxPutCdsWithCollection(ccname, ccver, f, res, nil, true, policy, ccpBytes)
}

func createLSCCTx(ccname, ccver, f string, res []byte) (*common.Envelope, error) {
	return createLSCCTxPutCds(ccname, ccver, f, res, nil, true)
}

func createLSCCTxPutCdsWithCollection(ccname, ccver, f string, res, cdsbytes []byte, putcds bool, policy []byte, ccpBytes []byte) (*common.Envelope, error) {
	cds := &peer.ChaincodeDeploymentSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{
				Name:    ccname,
				Version: ccver,
			},
			Type: peer.ChaincodeSpec_GOLANG,
		},
	}

	cdsBytes, err := proto.Marshal(cds)
	if err != nil {
		return nil, err
	}

	var cis *peer.ChaincodeInvocationSpec
	if putcds {
		if cdsbytes != nil {
			cdsBytes = cdsbytes
		}
		cis = &peer.ChaincodeInvocationSpec{
			ChaincodeSpec: &peer.ChaincodeSpec{
				ChaincodeId: &peer.ChaincodeID{Name: "lscc"},
				Input: &peer.ChaincodeInput{
					Args: [][]byte{[]byte(f), []byte("barf"), cdsBytes, []byte("escc"), []byte("vscc"), policy, ccpBytes},
				},
				Type: peer.ChaincodeSpec_GOLANG,
			},
		}
	} else {
		cis = &peer.ChaincodeInvocationSpec{
			ChaincodeSpec: &peer.ChaincodeSpec{
				ChaincodeId: &peer.ChaincodeID{Name: "lscc"},
				Input: &peer.ChaincodeInput{
					Args: [][]byte{[]byte(f), []byte("barf")},
				},
				Type: peer.ChaincodeSpec_GOLANG,
			},
		}
	}

	prop, _, err := utils.CreateProposalFromCIS(common.HeaderType_ENDORSER_TRANSACTION, util.GetTestChainID(), cis, sid)
	if err != nil {
		return nil, err
	}

	ccid := &peer.ChaincodeID{Name: ccname, Version: ccver}

	presp, err := utils.CreateProposalResponse(prop.Header, prop.Payload, &peer.Response{Status: 200}, res, nil, ccid, nil, id)
	if err != nil {
		return nil, err
	}

	return utils.CreateSignedTx(prop, id, presp)
}

func createLSCCTxPutCds(ccname, ccver, f string, res, cdsbytes []byte, putcds bool) (*common.Envelope, error) {
	cds := &peer.ChaincodeDeploymentSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{
				Name:    ccname,
				Version: ccver,
			},
			Type: peer.ChaincodeSpec_GOLANG,
		},
	}

	cdsBytes, err := proto.Marshal(cds)
	if err != nil {
		return nil, err
	}

	var cis *peer.ChaincodeInvocationSpec
	if putcds {
		if cdsbytes != nil {
			cdsBytes = cdsbytes
		}
		cis = &peer.ChaincodeInvocationSpec{
			ChaincodeSpec: &peer.ChaincodeSpec{
				ChaincodeId: &peer.ChaincodeID{Name: "lscc"},
				Input: &peer.ChaincodeInput{
					Args: [][]byte{[]byte(f), []byte("barf"), cdsBytes},
				},
				Type: peer.ChaincodeSpec_GOLANG,
			},
		}
	} else {
		cis = &peer.ChaincodeInvocationSpec{
			ChaincodeSpec: &peer.ChaincodeSpec{
				ChaincodeId: &peer.ChaincodeID{Name: "lscc"},
				Input: &peer.ChaincodeInput{
					Args: [][]byte{[]byte(f), []byte(ccname)},
				},
				Type: peer.ChaincodeSpec_GOLANG,
			},
		}
	}

	prop, _, err := utils.CreateProposalFromCIS(common.HeaderType_ENDORSER_TRANSACTION, util.GetTestChainID(), cis, sid)
	if err != nil {
		return nil, err
	}

	ccid := &peer.ChaincodeID{Name: ccname, Version: ccver}

	presp, err := utils.CreateProposalResponse(prop.Header, prop.Payload, &peer.Response{Status: 200}, res, nil, ccid, nil, id)
	if err != nil {
		return nil, err
	}

	return utils.CreateSignedTx(prop, id, presp)
}

func getSignedByMSPMemberPolicy(mspID string) ([]byte, error) {
	p := cauthdsl.SignedByMspMember(mspID)

	b, err := utils.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("Could not marshal policy, err %s", err)
	}

	return b, err
}

func getSignedByMSPAdminPolicy(mspID string) ([]byte, error) {
	p := cauthdsl.SignedByMspAdmin(mspID)

	b, err := utils.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("Could not marshal policy, err %s", err)
	}

	return b, err
}

func newValidationInstance(state map[string]map[string][]byte) *Validator {
	qec := &mocks2.QueryExecutorCreator{}
	qec.On("NewQueryExecutor").Return(lm.NewMockQueryExecutor(state), nil)
	return newCustomValidationInstance(qec, &mc.MockApplicationCapabilities{})
}

func newCustomValidationInstance(qec txvalidator.QueryExecutorCreator, c validation.Capabilities) *Validator {
	sbvm := &mocks.StateBasedValidator{}
	sbvm.On("PreValidate", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	sbvm.On("PostValidate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	sbvm.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	sf := &txvalidator.StateFetcherImpl{QueryExecutorCreator: qec}
	is := &mocks.IdentityDeserializer{}
	pe := &txvalidator.PolicyEvaluator{
		IdentityDeserializer: mspmgmt.GetManagerForChain(util.GetTestChainID()),
	}
	v := New(c, sf, is, pe)

	v.stateBasedValidator = sbvm
	return v
}

func TestStateBasedValidationFailure(t *testing.T) {
	qec := &mocks2.QueryExecutorCreator{}
	qec.On("NewQueryExecutor").Return(lm.NewMockQueryExecutor(make(map[string]map[string][]byte)), nil)

	sbvm := &mocks.StateBasedValidator{}
	sbvm.On("PreValidate", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	sbvm.On("PostValidate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	sf := &txvalidator.StateFetcherImpl{QueryExecutorCreator: qec}
	is := &mocks.IdentityDeserializer{}
	pe := &txvalidator.PolicyEvaluator{
		IdentityDeserializer: mspmgmt.GetManagerForChain(util.GetTestChainID()),
	}
	v := New(&mc.MockApplicationCapabilities{}, sf, is, pe)
	v.stateBasedValidator = sbvm

	tx, err := createTx(false)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}

//错误路径：策略验证错误
	sbvm.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&commonerrors.VSCCEndorsementPolicyError{Err: fmt.Errorf("some sbe validation err")}).Once()
	err = v.Validate(b, "foo", 0, 0, policy)
	assert.Error(t, err)
	assert.IsType(t, &commonerrors.VSCCEndorsementPolicyError{}, err)

//错误路径：执行错误
	sbvm.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&commonerrors.VSCCExecutionFailureError{Err: fmt.Errorf("some sbe validation err")}).Once()
	err = v.Validate(b, "foo", 0, 0, policy)
	assert.Error(t, err)
	assert.IsType(t, &commonerrors.VSCCExecutionFailureError{}, err)

//好路径：由正确的MSP签名
	sbvm.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	err = v.Validate(b, "foo", 0, 0, policy)
	assert.NoError(t, err)
}

func TestInvoke(t *testing.T) {
	v := newValidationInstance(make(map[string]map[string][]byte))

//破损信封
	var err error
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{[]byte("a")}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, []byte("a"))
	assert.Error(t, err)

//（仍然）破信封
	b = &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(&common.Envelope{Payload: []byte("barf")})}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, []byte("a"))
	assert.Error(t, err)

//（仍然）破信封
	e := utils.MarshalOrPanic(&common.Envelope{Payload: utils.MarshalOrPanic(&common.Payload{Header: &common.Header{ChannelHeader: []byte("barf")}})})
	b = &common.Block{Data: &common.BlockData{Data: [][]byte{e}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, []byte("a"))
	assert.Error(t, err)

	tx, err := createTx(false)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

//破碎型
	e = utils.MarshalOrPanic(&common.Envelope{Payload: utils.MarshalOrPanic(&common.Payload{Header: &common.Header{ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{Type: int32(common.HeaderType_ORDERER_TRANSACTION)})}})})
	b = &common.Block{Data: &common.BlockData{Data: [][]byte{e}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, policy)
	assert.Error(t, err)

//损坏的Tx有效载荷
	e = utils.MarshalOrPanic(&common.Envelope{Payload: utils.MarshalOrPanic(&common.Payload{Header: &common.Header{ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{Type: int32(common.HeaderType_ORDERER_TRANSACTION)})}})})
	b = &common.Block{Data: &common.BlockData{Data: [][]byte{e}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, policy)
	assert.Error(t, err)

//好路径：由正确的MSP签名
	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, policy)
	assert.NoError(t, err)
}

func TestRWSetTooBig(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v := newValidationInstance(state)

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	r := stublccc.MockInit("1", [][]byte{})
	if r.Status != shim.OK {
		fmt.Println("Init failed", string(r.Message))
		t.FailNow()
	}

	ccname := "mycc"
	ccver := "1"

	cd := &ccprovider.ChaincodeData{
		Name:                ccname,
		Version:             ccver,
		InstantiationPolicy: nil,
	}

	cdbytes := utils.MarshalOrPanic(cd)

	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	rwsetBuilder.AddToWriteSet("lscc", ccname, cdbytes)
	rwsetBuilder.AddToWriteSet("lscc", "spurious", []byte("spurious"))

	sr, err := rwsetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	srBytes, err := sr.GetPubSimulationBytes()
	assert.NoError(t, err)
	tx, err := createLSCCTx(ccname, ccver, lscc.DEPLOY, srBytes)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)
}

func TestValidateDeployFail(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v := newValidationInstance(state)
	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	ccname := "mycc"
	ccver := "1"

 /**************/
 /*测试无写入集*/
 /**************/

	tx, err := createLSCCTx(ccname, ccver, lscc.DEPLOY, nil)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)

 /*******************/
 /*测试伪写集*/
 /*******************/

	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	rwsetBuilder.AddToWriteSet("lscc", ccname, []byte("barf"))
	sr, err := rwsetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	resBogusBytes, err := sr.GetPubSimulationBytes()
	assert.NoError(t, err)
	tx, err = createLSCCTx(ccname, ccver, lscc.DEPLOY, resBogusBytes)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err = utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err = getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)

 /*******************/
 /*测试错误的lscc参数*/
 /*******************/

	res, err := createCCDataRWset(ccname, ccname, ccver, nil)
	assert.NoError(t, err)

	tx, err = createLSCCTxPutCds(ccname, ccver, lscc.DEPLOY, res, nil, false)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err = utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err = getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)

 /*******************/
 /*测试错误的lscc参数*/
 /*******************/

	res, err = createCCDataRWset(ccname, ccname, ccver, nil)
	assert.NoError(t, err)

	tx, err = createLSCCTxPutCds(ccname, ccver, lscc.DEPLOY, res, []byte("barf"), true)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err = utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err = getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)

 /*******************/
 /*测试错误的CC版本*/
 /*******************/

	res, err = createCCDataRWset(ccname, ccname, ccver+".1", nil)
	assert.NoError(t, err)

	tx, err = createLSCCTx(ccname, ccver, lscc.DEPLOY, res)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err = utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err = getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)

 /*****/
 /*坏RWSET*/
 /*****/

	tx, err = createLSCCTx(ccname, ccver, lscc.DEPLOY, []byte("barf"))
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err = utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err = getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)

 /**************/
 /*测试错误的cc名称*/
 /**************/

	res, err = createCCDataRWset(ccname+".badbad", ccname, ccver, nil)
	assert.NoError(t, err)

	tx, err = createLSCCTx(ccname, ccver, lscc.DEPLOY, res)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err = utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

	policy, err = getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)

 /*******************/
 /*测试错误的CC名称2*/
 /*******************/

	res, err = createCCDataRWset(ccname, ccname+".badbad", ccver, nil)
	assert.NoError(t, err)

	tx, err = createLSCCTx(ccname, ccver, lscc.DEPLOY, res)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err = utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

	policy, err = getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)

 /*******************/
 /*测试suprious写入*/
 /*******************/

	cd := &ccprovider.ChaincodeData{
		Name:                ccname,
		Version:             ccver,
		InstantiationPolicy: nil,
	}

	cdbytes := utils.MarshalOrPanic(cd)
	rwsetBuilder = rwsetutil.NewRWSetBuilder()
	rwsetBuilder.AddToWriteSet("lscc", ccname, cdbytes)
	rwsetBuilder.AddToWriteSet("bogusbogus", "key", []byte("val"))
	sr, err = rwsetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	srBytes, err := sr.GetPubSimulationBytes()
	assert.NoError(t, err)
	tx, err = createLSCCTx(ccname, ccver, lscc.DEPLOY, srBytes)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err = utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

	policy, err = getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)

}

func TestAlreadyDeployed(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v := newValidationInstance(state)
	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	ccname := "mycc"
	ccver := "alreadydeployed"
	path := "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"

	cds, err := constructDeploymentSpec(ccname, path, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")}, true)
	if err != nil {
		fmt.Printf("%s\n", err)
		t.FailNow()
	}

	var b []byte
	if b, err = proto.Marshal(cds); err != nil || b == nil {
		t.FailNow()
	}

	sProp2, _ := utils.MockSignedEndorserProposal2OrPanic(chainId, &peer.ChaincodeSpec{}, id)
	args := [][]byte{[]byte("deploy"), []byte(ccname), b}
	if res := stublccc.MockInvokeWithSignedProposal("1", args, sProp2); res.Status != shim.OK {
		fmt.Printf("%#v\n", res)
		t.FailNow()
	}

	simresres, err := createCCDataRWset(ccname, ccname, ccver, nil)
	assert.NoError(t, err)

	tx, err := createLSCCTx(ccname, ccver, lscc.DEPLOY, simresres)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	bl := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(bl, "lscc", 0, 0, policy)
	assert.Error(t, err)
}

func TestValidateDeployNoLedger(t *testing.T) {
	qec := &mocks2.QueryExecutorCreator{}
	qec.On("NewQueryExecutor").Return(nil, errors.New("failed obtaining query executor"))
	v := newCustomValidationInstance(qec, &mc.MockApplicationCapabilities{})

	ccname := "mycc"
	ccver := "1"

	defaultPolicy, err := getSignedByMSPAdminPolicy(mspid)
	assert.NoError(t, err)
	res, err := createCCDataRWset(ccname, ccname, ccver, defaultPolicy)
	assert.NoError(t, err)

	tx, err := createLSCCTx(ccname, ccver, lscc.DEPLOY, res)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)
}

func TestValidateDeployOK(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v := newValidationInstance(state)

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	ccname := "mycc"
	ccver := "1"

	defaultPolicy, err := getSignedByMSPAdminPolicy(mspid)
	assert.NoError(t, err)
	res, err := createCCDataRWset(ccname, ccname, ccver, defaultPolicy)
	assert.NoError(t, err)

	tx, err := createLSCCTx(ccname, ccver, lscc.DEPLOY, res)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, policy)
	assert.NoError(t, err)
}

func TestValidateDeployWithCollection(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv: &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{
			PrivateChannelDataRv: true,
		}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	qec := &mocks2.QueryExecutorCreator{}
	qec.On("NewQueryExecutor").Return(lm.NewMockQueryExecutor(state), nil)
	v := newCustomValidationInstance(qec, &mc.MockApplicationCapabilities{
		PrivateChannelDataRv: true,
	})

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	r := stublccc.MockInit("1", [][]byte{})
	if r.Status != shim.OK {
		fmt.Println("Init failed", string(r.Message))
		t.FailNow()
	}

	ccname := "mycc"
	ccver := "1"

	collName1 := "mycollection1"
	collName2 := "mycollection2"
	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}
	policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
	var requiredPeerCount, maximumPeerCount int32
	var blockToLive uint64
	requiredPeerCount = 1
	maximumPeerCount = 2
	blockToLive = 1000
	coll1 := createCollectionConfig(collName1, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)
	coll2 := createCollectionConfig(collName2, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)

//测试1:使用有效的集合配置部署链代码——>成功
	ccp := &common.CollectionConfigPackage{Config: []*common.CollectionConfig{coll1, coll2}}
	ccpBytes, err := proto.Marshal(ccp)
	assert.NoError(t, err)
	assert.NotNil(t, ccpBytes)

	defaultPolicy, err := getSignedByMSPAdminPolicy(mspid)
	assert.NoError(t, err)
	res, err := createCCDataRWsetWithCollection(ccname, ccname, ccver, defaultPolicy, ccpBytes)
	assert.NoError(t, err)

	tx, err := createLSCCTxWithCollection(ccname, ccver, lscc.DEPLOY, res, defaultPolicy, ccpBytes)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, policy)
	assert.NoError(t, err)

//测试2：使用重复的集合配置部署链代码——>无错误
//对等机未处于v12validation模式
	ccp = &common.CollectionConfigPackage{Config: []*common.CollectionConfig{coll1, coll2, coll1}}
	ccpBytes, err = proto.Marshal(ccp)
	assert.NoError(t, err)
	assert.NotNil(t, ccpBytes)

	res, err = createCCDataRWsetWithCollection(ccname, ccname, ccver, defaultPolicy, ccpBytes)
	assert.NoError(t, err)

	tx, err = createLSCCTxWithCollection(ccname, ccver, lscc.DEPLOY, res, defaultPolicy, ccpBytes)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err = utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, policy)
	assert.NoError(t, err)

//测试3：启用v12validation后，由于重复的集合配置，验证应失败。
	state = make(map[string]map[string][]byte)
	mp = (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv: &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{
			PrivateChannelDataRv: true,
			V1_2ValidationRv:     true,
		}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v = newValidationInstance(state)

	lccc = lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc = shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	r = stublccc.MockInit("1", [][]byte{})
	if r.Status != shim.OK {
		fmt.Println("Init failed", string(r.Message))
		t.FailNow()
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, policy)
}

func TestValidateDeployWithPolicies(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v := newValidationInstance(state)

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	ccname := "mycc"
	ccver := "1"

 /***************************************/
 /*测试1：接受所有策略成功*/
 /***************************************/

	res, err := createCCDataRWset(ccname, ccname, ccver, cauthdsl.MarshaledAcceptAllPolicy)
	assert.NoError(t, err)

	tx, err := createLSCCTx(ccname, ccver, lscc.DEPLOY, res)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "foo", 0, 0, policy)
	assert.NoError(t, err)

 /***************************************/
 /*测试2：拒绝所有策略失败*/
 /***************************************/

	res, err = createCCDataRWset(ccname, ccname, ccver, cauthdsl.MarshaledRejectAllPolicy)
	assert.NoError(t, err)

	tx, err = createLSCCTx(ccname, ccver, lscc.DEPLOY, res)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err = utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err = getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)
}

func TestInvalidUpgrade(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v := newValidationInstance(state)

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	ccname := "mycc"
	ccver := "2"

	simresres, err := createCCDataRWset(ccname, ccname, ccver, nil)
	assert.NoError(t, err)

	tx, err := createLSCCTx(ccname, ccver, lscc.UPGRADE, simresres)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	b := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(b, "lscc", 0, 0, policy)
	assert.Error(t, err)
}

func TestValidateUpgradeOK(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v := newValidationInstance(state)

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	ccname := "mycc"
	ccver := "upgradeok"
	path := "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"

	cds, err := constructDeploymentSpec(ccname, path, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")}, true)
	if err != nil {
		fmt.Printf("%s\n", err)
		t.FailNow()
	}

	var b []byte
	if b, err = proto.Marshal(cds); err != nil || b == nil {
		t.FailNow()
	}

	sProp2, _ := utils.MockSignedEndorserProposal2OrPanic(chainId, &peer.ChaincodeSpec{}, id)
	args := [][]byte{[]byte("deploy"), []byte(ccname), b}
	if res := stublccc.MockInvokeWithSignedProposal("1", args, sProp2); res.Status != shim.OK {
		fmt.Printf("%#v\n", res)
		t.FailNow()
	}

	ccver = "2"

	simresres, err := createCCDataRWset(ccname, ccname, ccver, nil)
	assert.NoError(t, err)

	tx, err := createLSCCTx(ccname, ccver, lscc.UPGRADE, simresres)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	bl := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(bl, "foo", 0, 0, policy)
	assert.NoError(t, err)
}

func TestInvalidateUpgradeBadVersion(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v := newValidationInstance(state)

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	ccname := "mycc"
	ccver := "upgradebadversion"
	path := "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"

	cds, err := constructDeploymentSpec(ccname, path, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")}, true)
	if err != nil {
		fmt.Printf("%s\n", err)
		t.FailNow()
	}

	var b []byte
	if b, err = proto.Marshal(cds); err != nil || b == nil {
		t.FailNow()
	}

	sProp2, _ := utils.MockSignedEndorserProposal2OrPanic(chainId, &peer.ChaincodeSpec{}, id)
	args := [][]byte{[]byte("deploy"), []byte(ccname), b}
	if res := stublccc.MockInvokeWithSignedProposal("1", args, sProp2); res.Status != shim.OK {
		fmt.Printf("%#v\n", res)
		t.FailNow()
	}

	simresres, err := createCCDataRWset(ccname, ccname, ccver, nil)
	assert.NoError(t, err)

	tx, err := createLSCCTx(ccname, ccver, lscc.UPGRADE, simresres)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	bl := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(bl, "lscc", 0, 0, policy)
	assert.Error(t, err)
}

func validateUpgradeWithCollection(t *testing.T, ccver string, V1_2Validation bool) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv: &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{
			PrivateChannelDataRv: true,
			V1_2ValidationRv:     V1_2Validation,
		}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	qec := &mocks2.QueryExecutorCreator{}
	qec.On("NewQueryExecutor").Return(lm.NewMockQueryExecutor(state), nil)
	v := newCustomValidationInstance(qec, &mc.MockApplicationCapabilities{
		PrivateChannelDataRv: true,
		V1_2ValidationRv:     V1_2Validation,
	})

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	r := stublccc.MockInit("1", [][]byte{})
	if r.Status != shim.OK {
		fmt.Println("Init failed", string(r.Message))
		t.FailNow()
	}

	ccname := "mycc"
	path := "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"

	cds, err := constructDeploymentSpec(ccname, path, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")}, true)
	if err != nil {
		fmt.Printf("%s\n", err)
		t.FailNow()
	}

	var b []byte
	if b, err = proto.Marshal(cds); err != nil || b == nil {
		t.FailNow()
	}

	sProp2, _ := utils.MockSignedEndorserProposal2OrPanic(chainId, &peer.ChaincodeSpec{}, id)
	args := [][]byte{[]byte("deploy"), []byte(ccname), b}
	if res := stublccc.MockInvokeWithSignedProposal("1", args, sProp2); res.Status != shim.OK {
		fmt.Printf("%#v\n", res)
		t.FailNow()
	}

	ccver = "2"

	collName1 := "mycollection1"
	collName2 := "mycollection2"
	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}
	policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
	var requiredPeerCount, maximumPeerCount int32
	var blockToLive uint64
	requiredPeerCount = 1
	maximumPeerCount = 2
	blockToLive = 1000
	coll1 := createCollectionConfig(collName1, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)
	coll2 := createCollectionConfig(collName2, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)

//测试1:升级中的有效集合配置。
//已启用验证：成功
//v12validation disable:失败（不允许收集更新）
//注意：我们可能会使用CollectionUpdate功能更改v12Validation
	ccp := &common.CollectionConfigPackage{Config: []*common.CollectionConfig{coll1, coll2}}
	ccpBytes, err := proto.Marshal(ccp)
	assert.NoError(t, err)
	assert.NotNil(t, ccpBytes)

	defaultPolicy, err := getSignedByMSPAdminPolicy(mspid)
	assert.NoError(t, err)
	res, err := createCCDataRWsetWithCollection(ccname, ccname, ccver, defaultPolicy, ccpBytes)
	assert.NoError(t, err)

	tx, err := createLSCCTxWithCollection(ccname, ccver, lscc.UPGRADE, res, defaultPolicy, ccpBytes)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	bl := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(bl, "lscc", 0, 0, policy)
	if V1_2Validation {
		assert.NoError(t, err)
	} else {
		assert.Error(t, err, "LSCC can only issue a single putState upon deploy/upgrade")
	}

	state["lscc"][privdata.BuildCollectionKVSKey(ccname)] = ccpBytes

	if V1_2Validation {
		ccver = "3"

		collName3 := "mycollection3"
		coll3 := createCollectionConfig(collName3, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)

//测试2:更新的配置中缺少某些现有集合，而在中缺少对等集合
//v12Validation Mode（验证模式）->错误
		ccp = &common.CollectionConfigPackage{Config: []*common.CollectionConfig{coll3}}
		ccpBytes, err = proto.Marshal(ccp)
		assert.NoError(t, err)
		assert.NotNil(t, ccpBytes)

		res, err = createCCDataRWsetWithCollection(ccname, ccname, ccver, defaultPolicy, ccpBytes)
		assert.NoError(t, err)

		tx, err = createLSCCTxWithCollection(ccname, ccver, lscc.UPGRADE, res, defaultPolicy, ccpBytes)
		if err != nil {
			t.Fatalf("createTx returned err %s", err)
		}

		envBytes, err = utils.GetBytesEnvelope(tx)
		if err != nil {
			t.Fatalf("GetBytesEnvelope returned err %s", err)
		}

		bl = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
		err = v.Validate(bl, "lscc", 0, 0, policy)
		assert.Error(t, err, "Some existing collection configurations are missing in the new collection configuration package")

		ccver = "3"

//测试3:更新的配置中缺少某些现有集合，而在中缺少对等集合
//v12Validation Mode（验证模式）->错误
		ccp = &common.CollectionConfigPackage{Config: []*common.CollectionConfig{coll1, coll3}}
		ccpBytes, err = proto.Marshal(ccp)
		assert.NoError(t, err)
		assert.NotNil(t, ccpBytes)

		res, err = createCCDataRWsetWithCollection(ccname, ccname, ccver, defaultPolicy, ccpBytes)
		assert.NoError(t, err)

		tx, err = createLSCCTxWithCollection(ccname, ccver, lscc.UPGRADE, res, defaultPolicy, ccpBytes)
		if err != nil {
			t.Fatalf("createTx returned err %s", err)
		}

		envBytes, err = utils.GetBytesEnvelope(tx)
		if err != nil {
			t.Fatalf("GetBytesEnvelope returned err %s", err)
		}

		args = [][]byte{[]byte("dv"), envBytes, policy, ccpBytes}
		bl = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
		err = v.Validate(bl, "lscc", 0, 0, policy)
		assert.Error(t, err, "existing collection named mycollection2 is missing in the new collection configuration package")

		ccver = "3"

//测试4:v12validation模式下的有效收集配置配置和对等机——>成功
		ccp = &common.CollectionConfigPackage{Config: []*common.CollectionConfig{coll1, coll2, coll3}}
		ccpBytes, err = proto.Marshal(ccp)
		assert.NoError(t, err)
		assert.NotNil(t, ccpBytes)

		res, err = createCCDataRWsetWithCollection(ccname, ccname, ccver, defaultPolicy, ccpBytes)
		assert.NoError(t, err)

		tx, err = createLSCCTxWithCollection(ccname, ccver, lscc.UPGRADE, res, defaultPolicy, ccpBytes)
		if err != nil {
			t.Fatalf("createTx returned err %s", err)
		}

		envBytes, err = utils.GetBytesEnvelope(tx)
		if err != nil {
			t.Fatalf("GetBytesEnvelope returned err %s", err)
		}

		bl = &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
		err = v.Validate(bl, "lscc", 0, 0, policy)
		assert.NoError(t, err)
	}
}

func TestValidateUpgradeWithCollection(t *testing.T) {
//启用v12validation
	validateUpgradeWithCollection(t, "v12-validation-enabled", true)
//禁用v12validation
	validateUpgradeWithCollection(t, "v12-validation-disabled", false)
}

func TestValidateUpgradeWithPoliciesOK(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v := newValidationInstance(state)

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	ccname := "mycc"
	ccver := "upgradewithpoliciesok"
	path := "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"

	cds, err := constructDeploymentSpec(ccname, path, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")}, false)
	if err != nil {
		fmt.Printf("%s\n", err)
		t.FailNow()
	}
	_, err = processSignedCDS(cds, cauthdsl.AcceptAllPolicy)
	assert.NoError(t, err)

	var b []byte
	if b, err = proto.Marshal(cds); err != nil || b == nil {
		t.FailNow()
	}

	sProp2, _ := utils.MockSignedEndorserProposal2OrPanic(chainId, &peer.ChaincodeSpec{}, id)
	args := [][]byte{[]byte("deploy"), []byte(ccname), b}
	if res := stublccc.MockInvokeWithSignedProposal("1", args, sProp2); res.Status != shim.OK {
		fmt.Printf("%#v\n", res)
		t.FailNow()
	}

	ccver = "2"

	simresres, err := createCCDataRWset(ccname, ccname, ccver, nil)
	assert.NoError(t, err)

	tx, err := createLSCCTx(ccname, ccver, lscc.UPGRADE, simresres)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	args = [][]byte{[]byte("dv"), envBytes, policy}
	bl := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(bl, "foo", 0, 0, policy)
	assert.NoError(t, err)
}

func TestValidateUpgradeWithNewFailAllIP(t *testing.T) {
//我们正在测试升级。
//特别是，我们要测试升级
//符合当前版本的实例化策略
//但不是新版本的实例化策略。为此
//为什么我们首先部署一个IP等于可接受策略的CC
//然后尝试使用带有rejectallpolicy的cc进行升级。
//我们运行这个测试两次，一次使用v11功能（并期望
//一次失败，一次失败（我们期待成功）。

	validateUpgradeWithNewFailAllIP(t, "v11-capabilityenabled", true, true)
	validateUpgradeWithNewFailAllIP(t, "v11-capabilitydisabled", false, false)
}

func validateUpgradeWithNewFailAllIP(t *testing.T, ccver string, v11capability, expecterr bool) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{V1_1ValidationRv: v11capability}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	qec := &mocks2.QueryExecutorCreator{}
	capabilities := &mc.MockApplicationCapabilities{}
	if v11capability {
		capabilities.V1_1ValidationRv = true
	}
	qec.On("NewQueryExecutor").Return(lm.NewMockQueryExecutor(state), nil)
	v := newCustomValidationInstance(qec, capabilities)

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

//使用“全部接受”策略部署链代码

	ccname := "mycc"
	path := "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"

	cds, err := constructDeploymentSpec(ccname, path, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")}, false)
	if err != nil {
		fmt.Printf("%s\n", err)
		t.FailNow()
	}
	_, err = processSignedCDS(cds, cauthdsl.AcceptAllPolicy)
	assert.NoError(t, err)

	var b []byte
	if b, err = proto.Marshal(cds); err != nil || b == nil {
		t.FailNow()
	}

	sProp2, _ := utils.MockSignedEndorserProposal2OrPanic(chainId, &peer.ChaincodeSpec{}, id)
	args := [][]byte{[]byte("deploy"), []byte(ccname), b}
	if res := stublccc.MockInvokeWithSignedProposal("1", args, sProp2); res.Status != shim.OK {
		fmt.Printf("%#v\n", res)
		t.FailNow()
	}

//如果我们在这里，我们部署了一个带有接受所有IP的CC

//现在我们升级了同一个CC的V 2，关键的区别是它有一个拒绝所有IP

	ccver = ccver + ".2"

	simresres, err := createCCDataRWset(ccname, ccname, ccver,
cauthdsl.MarshaledRejectAllPolicy, //在这里我们指定升级后的CC的IP
	)
	assert.NoError(t, err)

	tx, err := createLSCCTx(ccname, ccver, lscc.UPGRADE, simresres)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

//执行升级tx
	if expecterr {
		bl := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
		err = v.Validate(bl, "lscc", 0, 0, policy)
		assert.Error(t, err)
	} else {
		bl := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
		err = v.Validate(bl, "lscc", 0, 0, policy)
		assert.NoError(t, err)
	}
}

func TestValidateUpgradeWithPoliciesFail(t *testing.T) {
	state := make(map[string]map[string][]byte)
	mp := (&scc.MocksccProviderFactory{
		Qe:                    lm.NewMockQueryExecutor(state),
		ApplicationConfigBool: true,
		ApplicationConfigRv:   &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
	}).NewSystemChaincodeProvider().(*scc.MocksccProviderImpl)

	v := newValidationInstance(state)

	mockAclProvider := &aclmocks.MockACLProvider{}
	lccc := lscc.New(mp, mockAclProvider, platforms.NewRegistry(&golang.Platform{}))
	stublccc := shim.NewMockStub("lscc", lccc)
	state["lscc"] = stublccc.State

	ccname := "mycc"
	ccver := "upgradewithpoliciesfail"
	path := "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"

	cds, err := constructDeploymentSpec(ccname, path, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")}, false)
	if err != nil {
		fmt.Printf("%s\n", err)
		t.FailNow()
	}
	cdbytes, err := processSignedCDS(cds, cauthdsl.RejectAllPolicy)
	assert.NoError(t, err)

	var b []byte
	if b, err = proto.Marshal(cds); err != nil || b == nil {
		t.FailNow()
	}

//在跳过策略验证的同时模拟LSCC调用，
//否则，我们将无法使用拒绝所有策略来DEPLY链式代码。
	stublccc.MockTransactionStart("barf")
	err = stublccc.PutState(ccname, cdbytes)
	assert.NoError(t, err)
	stublccc.MockTransactionEnd("barf")

	ccver = "2"

	simresres, err := createCCDataRWset(ccname, ccname, ccver, nil)
	assert.NoError(t, err)

	tx, err := createLSCCTx(ccname, ccver, lscc.UPGRADE, simresres)
	if err != nil {
		t.Fatalf("createTx returned err %s", err)
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("GetBytesEnvelope returned err %s", err)
	}

//好路径：由正确的MSP签名
	policy, err := getSignedByMSPMemberPolicy(mspid)
	if err != nil {
		t.Fatalf("failed getting policy, err %s", err)
	}

	bl := &common.Block{Data: &common.BlockData{Data: [][]byte{envBytes}}, Header: &common.BlockHeader{}}
	err = v.Validate(bl, "lscc", 0, 0, policy)
	assert.Error(t, err)
}

var id msp.SigningIdentity
var sid []byte
var mspid string
var chainId string = util.GetTestChainID()

type mockPolicyCheckerFactory struct {
}

func (c *mockPolicyCheckerFactory) NewPolicyChecker() policy.PolicyChecker {
	return &mockPolicyChecker{}
}

type mockPolicyChecker struct {
}

func (c *mockPolicyChecker) CheckPolicy(channelID, policyName string, signedProp *peer.SignedProposal) error {
	return nil
}

func (c *mockPolicyChecker) CheckPolicyBySignedData(channelID, policyName string, sd []*common.SignedData) error {
	return nil
}

func (c *mockPolicyChecker) CheckPolicyNoChannel(policyName string, signedProp *peer.SignedProposal) error {
	return nil
}

func createCollectionConfig(collectionName string, signaturePolicyEnvelope *common.SignaturePolicyEnvelope,
	requiredPeerCount int32, maximumPeerCount int32, blockToLive uint64,
) *common.CollectionConfig {
	signaturePolicy := &common.CollectionPolicyConfig_SignaturePolicy{
		SignaturePolicy: signaturePolicyEnvelope,
	}
	accessPolicy := &common.CollectionPolicyConfig{
		Payload: signaturePolicy,
	}

	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              collectionName,
				MemberOrgsPolicy:  accessPolicy,
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maximumPeerCount,
				BlockToLive:       blockToLive,
			},
		},
	}
}

func testValidateCollection(t *testing.T, v *Validator, collectionConfigs []*common.CollectionConfig, cdRWSet *ccprovider.ChaincodeData,
	lsccFunc string, ac channelconfig.ApplicationCapabilities, chid string,
) error {
	ccp := &common.CollectionConfigPackage{Config: collectionConfigs}
	ccpBytes, err := proto.Marshal(ccp)
	assert.NoError(t, err)
	assert.NotNil(t, ccpBytes)

	lsccargs := [][]byte{nil, nil, nil, nil, nil, ccpBytes}
	rwset := &kvrwset.KVRWSet{Writes: []*kvrwset.KVWrite{{Key: cdRWSet.Name}, {Key: privdata.BuildCollectionKVSKey(cdRWSet.Name), Value: ccpBytes}}}

	err = v.validateRWSetAndCollection(rwset, cdRWSet, lsccargs, lsccFunc, ac, chid)
	return err

}

func TestValidateRWSetAndCollectionForDeploy(t *testing.T) {
	var err error
	chid := "ch"
	ccid := "mycc"
	ccver := "1.0"
	cdRWSet := &ccprovider.ChaincodeData{Name: ccid, Version: ccver}

	state := make(map[string]map[string][]byte)
	state["lscc"] = make(map[string][]byte)

	v := newValidationInstance(state)

	ac := capabilities.NewApplicationProvider(map[string]*common.Capability{
		capabilities.ApplicationV1_1: {},
	})

	lsccFunc := lscc.DEPLOY
//测试1:rwset->error中的两个以上条目
	rwset := &kvrwset.KVRWSet{Writes: []*kvrwset.KVWrite{{Key: ccid}, {Key: "b"}, {Key: "c"}}}
	err = v.validateRWSetAndCollection(rwset, cdRWSet, nil, lsccFunc, ac, chid)
	assert.EqualError(t, err, "LSCC can only issue one or two putState upon deploy")

//测试2:集合配置包的密钥无效->错误
	rwset = &kvrwset.KVRWSet{Writes: []*kvrwset.KVWrite{{Key: ccid}, {Key: "b"}}}
	err = v.validateRWSetAndCollection(rwset, cdRWSet, nil, lsccFunc, ac, chid)
	assert.EqualError(t, err, "invalid key for the collection of chaincode mycc:1.0; expected 'mycc~collection', received 'b'")

//测试3：无集合配置包->成功
	rwset = &kvrwset.KVRWSet{Writes: []*kvrwset.KVWrite{{Key: ccid}}}
	err = v.validateRWSetAndCollection(rwset, cdRWSet, nil, lsccFunc, ac, chid)
	assert.NoError(t, err)

	lsccargs := [][]byte{nil, nil, nil, nil, nil, nil}
	err = v.validateRWSetAndCollection(rwset, cdRWSet, lsccargs, lsccFunc, ac, chid)
	assert.NoError(t, err)

//测试4:集合配置包的有效密钥->成功
	rwset = &kvrwset.KVRWSet{Writes: []*kvrwset.KVWrite{{Key: ccid}, {Key: privdata.BuildCollectionKVSKey(ccid)}}}
	err = v.validateRWSetAndCollection(rwset, cdRWSet, lsccargs, lsccFunc, ac, chid)
	assert.NoError(t, err)

//测试5:lscc参数的集合配置与rwset不匹配
	lsccargs = [][]byte{nil, nil, nil, nil, nil, []byte("barf")}
	err = v.validateRWSetAndCollection(rwset, cdRWSet, lsccargs, lsccFunc, ac, chid)
	assert.EqualError(t, err, "collection configuration arguments supplied for chaincode mycc:1.0 do not match the configuration in the lscc writeset")

//测试6:无效的集合配置包->错误
	rwset = &kvrwset.KVRWSet{Writes: []*kvrwset.KVWrite{{Key: ccid}, {Key: privdata.BuildCollectionKVSKey("mycc"), Value: []byte("barf")}}}
	err = v.validateRWSetAndCollection(rwset, cdRWSet, lsccargs, lsccFunc, ac, chid)
	assert.EqualError(t, err, "invalid collection configuration supplied for chaincode mycc:1.0")

//测试7:有效的收集配置包->成功
	collName1 := "mycollection1"
	collName2 := "mycollection2"
	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}
	policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
	var requiredPeerCount, maximumPeerCount int32
	var blockToLive uint64
	requiredPeerCount = 1
	maximumPeerCount = 2
	blockToLive = 10000
	coll1 := createCollectionConfig(collName1, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)
	coll2 := createCollectionConfig(collName2, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)

	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1, coll2}, cdRWSet, lsccFunc, ac, chid)
	assert.NoError(t, err)

//测试8：集合配置包中的重复集合->成功，因为对等方处于v1.1验证模式
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1, coll2, coll1}, cdRWSet, lsccFunc, ac, chid)
	assert.NoError(t, err)

//测试9：RequiredPeerCount>MaximumpeerCount->Success，因为对等机处于v1.1验证模式
	collName3 := "mycollection3"
	requiredPeerCount = 2
	maximumPeerCount = 1
	blockToLive = 10000
	coll3 := createCollectionConfig(collName3, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1, coll2, coll3}, cdRWSet, lsccFunc, ac, chid)
	assert.NoError(t, err)

//启用v1.2验证模式
	ac = capabilities.NewApplicationProvider(map[string]*common.Capability{
		capabilities.ApplicationV1_2: {},
	})

//测试10:集合配置包中的重复集合->错误
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1, coll2, coll1}, cdRWSet, lsccFunc, ac, chid)
	assert.EqualError(t, err, "collection-name: mycollection1 -- found duplicate collection configuration")

//测试11:RequiredPeerCount<0->错误
	requiredPeerCount = -2
	maximumPeerCount = 1
	blockToLive = 10000
	coll3 = createCollectionConfig(collName3, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1, coll2, coll3}, cdRWSet, lsccFunc, ac, chid)
	assert.EqualError(t, err, "collection-name: mycollection3 -- requiredPeerCount (1) cannot be less than zero (-2)",
		collName3, maximumPeerCount, requiredPeerCount)

//测试11:RequiredPeerCount>MaximumpeerCount->Error
	requiredPeerCount = 2
	maximumPeerCount = 1
	blockToLive = 10000
	coll3 = createCollectionConfig(collName3, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1, coll2, coll3}, cdRWSet, lsccFunc, ac, chid)
	assert.EqualError(t, err, "collection-name: mycollection3 -- maximum peer count (1) cannot be greater than the required peer count (2)")

//测试12：访问策略中组织的连接->错误
	requiredPeerCount = 1
	maximumPeerCount = 2
	policyEnvelope = cauthdsl.Envelope(cauthdsl.And(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
	coll3 = createCollectionConfig(collName3, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll3}, cdRWSet, lsccFunc, ac, chid)
	assert.EqualError(t, err, "collection-name: mycollection3 -- error in member org policy: signature policy is not an OR concatenation, NOutOf 2")

//测试13：使用分类帐上的现有集合配置部署->错误
	ccp := &common.CollectionConfigPackage{Config: []*common.CollectionConfig{coll1}}
	ccpBytes, err := proto.Marshal(ccp)
	assert.NoError(t, err)
	state["lscc"][privdata.BuildCollectionKVSKey(ccid)] = ccpBytes
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1}, cdRWSet, lsccFunc, ac, chid)
	assert.EqualError(t, err, "collection data should not exist for chaincode mycc:1.0")
}

func TestValidateRWSetAndCollectionForUpgrade(t *testing.T) {
	chid := "ch"
	ccid := "mycc"
	ccver := "1.0"
	cdRWSet := &ccprovider.ChaincodeData{Name: ccid, Version: ccver}

	state := make(map[string]map[string][]byte)
	state["lscc"] = make(map[string][]byte)

	v := newValidationInstance(state)

	ac := capabilities.NewApplicationProvider(map[string]*common.Capability{
		capabilities.ApplicationV1_2: {},
	})

	lsccFunc := lscc.UPGRADE

	collName1 := "mycollection1"
	collName2 := "mycollection2"
	collName3 := "mycollection3"
	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}
	policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
	var requiredPeerCount, maximumPeerCount int32
	var blockToLive uint64
	requiredPeerCount = 1
	maximumPeerCount = 2
	blockToLive = 3
	coll1 := createCollectionConfig(collName1, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)
	coll2 := createCollectionConfig(collName2, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)
	coll3 := createCollectionConfig(collName3, policyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)

	ccp := &common.CollectionConfigPackage{Config: []*common.CollectionConfig{coll1, coll2}}
	ccpBytes, err := proto.Marshal(ccp)
	assert.NoError(t, err)

//测试1:没有现有的集合配置包->成功
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1}, cdRWSet, lsccFunc, ac, chid)
	assert.NoError(t, err)

	state["lscc"][privdata.BuildCollectionKVSKey(ccid)] = ccpBytes

//测试2：与现有集合配置包完全相同->成功
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1, coll2}, cdRWSet, lsccFunc, ac, chid)
	assert.NoError(t, err)

//测试3:缺少一个现有集合（根据长度检查）->错误
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1}, cdRWSet, lsccFunc, ac, chid)
	assert.EqualError(t, err, "the following existing collections are missing in the new collection configuration package: [mycollection2]")

//测试4:缺少一个现有集合（根据集合名称检查）->错误
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1, coll3}, cdRWSet, lsccFunc, ac, chid)
	assert.EqualError(t, err, "the following existing collections are missing in the new collection configuration package: [mycollection2]")

//测试5：添加新集合以及现有集合->成功
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1, coll2, coll3}, cdRWSet, lsccFunc, ac, chid)
	assert.NoError(t, err)

	newBlockToLive := blockToLive + 1
	coll2 = createCollectionConfig(collName2, policyEnvelope, requiredPeerCount, maximumPeerCount, newBlockToLive)

//测试6：修改现有集合中的blocktoLive->error
	err = testValidateCollection(t, v, []*common.CollectionConfig{coll1, coll2, coll3}, cdRWSet, lsccFunc, ac, chid)
	assert.EqualError(t, err, "the BlockToLive in the following existing collections must not be modified: [mycollection2]")
}

func TestMain(m *testing.M) {
	testDir, err := ioutil.TempDir("", "v1.3-validation")
	if err != nil {
		fmt.Printf("Could not create temp dir: %s", err)
		os.Exit(-1)
	}
	defer os.RemoveAll(testDir)
	ccprovider.SetChaincodesPath(testDir)

	policy.RegisterPolicyCheckerFactory(&mockPolicyCheckerFactory{})

	mspGetter := func(cid string) []string {
		return []string{"SampleOrg"}
	}

	corepeer.MockSetMSPIDGetter(mspGetter)

//设置MSP管理器，以便我们可以签名/验证
	msptesttools.LoadMSPSetupForTesting()

	id, err = mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		fmt.Printf("GetSigningIdentity failed with err %s", err)
		os.Exit(-1)
	}

	sid, err = id.Serialize()
	if err != nil {
		fmt.Printf("Serialize failed with err %s", err)
		os.Exit(-1)
	}

//确定默认链中第一个MSP的MSP标识符
	var msp msp.MSP
	mspMgr := mspmgmt.GetManagerForChain(chainId)
	msps, err := mspMgr.GetMSPs()
	if err != nil {
		fmt.Printf("Could not retrieve the MSPs for the chain manager, err %s", err)
		os.Exit(-1)
	}
	if len(msps) == 0 {
		fmt.Printf("At least one MSP was expected")
		os.Exit(-1)
	}
	for _, m := range msps {
		msp = m
		break
	}
	mspid, err = msp.GetIdentifier()
	if err != nil {
		fmt.Printf("Failure getting the msp identifier, err %s", err)
		os.Exit(-1)
	}

//同时为“测试”链设置MSP
	mspmgmt.XXXSetMSPManager("mycc", mspmgmt.GetManagerForChain(util.GetTestChainID()))

	os.Exit(m.Run())
}

func TestInValidCollectionName(t *testing.T) {
	validNames := []string{"collection1", "collection_2"}
	inValidNames := []string{"collection.1", "collection%2", ""}

	for _, name := range validNames {
		assert.NoError(t, validateCollectionName(name), "Testing for name = "+name)
	}
	for _, name := range inValidNames {
		assert.Error(t, validateCollectionName(name), "Testing for name = "+name)
	}
}
