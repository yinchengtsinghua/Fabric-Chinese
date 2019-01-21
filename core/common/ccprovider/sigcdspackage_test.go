
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package ccprovider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hyperledger/fabric/core/common/ccpackage"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func processSignedCDS(cds *pb.ChaincodeDeploymentSpec, policy *common.SignaturePolicyEnvelope, tofs bool) (*SignedCDSPackage, []byte, *ChaincodeData, error) {
	env, err := ccpackage.OwnerCreateSignedCCDepSpec(cds, policy, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not create package %s", err)
	}

	b := utils.MarshalOrPanic(env)

	ccpack := &SignedCDSPackage{}
	cd, err := ccpack.InitFromBuffer(b)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error owner creating package %s", err)
	}

	if tofs {
		if err = ccpack.PutChaincodeToFS(); err != nil {
			return nil, nil, nil, fmt.Errorf("error putting package on the FS %s", err)
		}
	}

	return ccpack, b, cd, nil
}

func TestPutSigCDSCC(t *testing.T) {
	ccdir := setupccdir()
	defer os.RemoveAll(ccdir)

	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"}, Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}

	ccpack, _, cd, err := processSignedCDS(cds, &common.SignaturePolicyEnvelope{Version: 1}, true)
	if err != nil {
		t.Fatalf("cannot create package %s", err)
		return
	}

	if err = ccpack.ValidateCC(cd); err != nil {
		t.Fatalf("error validating package %s", err)
		return
	}
}

func TestPutSignedCDSErrorPaths(t *testing.T) {
	ccdir := setupccdir()
	defer os.RemoveAll(ccdir)

	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"},
		Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}

	ccpack, b, _, err := processSignedCDS(cds, &common.SignaturePolicyEnvelope{Version: 1}, true)
	if err != nil {
		t.Fatalf("cannot create package %s", err)
		return
	}

//取出缓冲器
	ccpack.buf = nil
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uninitialized package", "Unexpected error putting package on the FS")

//把缓冲器放回去
	ccpack.buf = b
	id := ccpack.id
ccpack.id = nil //删除ID
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id cannot be nil if buf is not nil", "Unexpected error putting package on the FS")

	assert.Panics(t, func() {
		ccpack.GetId()
	}, "GetId should have paniced if chaincode package ID is nil")

//把身份证放回原处
	ccpack.id = id
	id1 := ccpack.GetId()
	assert.Equal(t, id, id1)

	savDepSpec := ccpack.sDepSpec
ccpack.sDepSpec = nil //删除已签名的链代码部署规范
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "depspec cannot be nil if buf is not nil", "Unexpected error putting package on the FS")
	assert.Panics(t, func() {
		ccpack.GetInstantiationPolicy()
	}, "GetChaincodeData should have paniced if signed chaincode deployment spec is nil")
	assert.Panics(t, func() {
		ccpack.GetDepSpecBytes()
	}, "GetDepSpecBytes should have paniced if signed chaincode deployment spec is nil")
ccpack.sDepSpec = savDepSpec //退回DEP规范
	sdepspec1 := ccpack.GetInstantiationPolicy()
	assert.NotNil(t, sdepspec1)
	depspecBytes := ccpack.GetDepSpecBytes()
	assert.NotNil(t, depspecBytes)

//放回签名的链代码部署规范
	depSpec := ccpack.depSpec
ccpack.depSpec = nil //删除链码部署规范
	assert.Panics(t, func() {
		ccpack.GetDepSpec()
	}, "GetDepSec should have paniced if chaincode deployment spec is nil")
	assert.Panics(t, func() {
		ccpack.GetChaincodeData()
	}, "GetChaincodeData should have paniced if chaincode deployment spec is nil")
ccpack.depSpec = depSpec //放回链码部署规范
	depSpec1 := ccpack.GetDepSpec()
	assert.NotNil(t, depSpec1)

	env := ccpack.env
ccpack.env = nil //取出信封
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "env cannot be nil if buf and depspec are not nil", "Unexpected error putting package on the FS")
ccpack.env = env //把信封放回去
	env1 := ccpack.GetPackageObject()
	assert.Equal(t, env, env1)

	data := ccpack.data
ccpack.data = nil //删除数据
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil data", "Unexpected error putting package on the FS")
ccpack.data = data //把数据放回去

	datab := ccpack.datab
ccpack.datab = nil //删除数据字节
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil data bytes", "Unexpected error putting package on the FS")
ccpack.datab = datab //放回数据字节

//删除chaincode目录
	os.RemoveAll(ccdir)
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err, "Expected error putting package on the FS")
}

func TestGetCDSDataErrorPaths(t *testing.T) {
	ccdir := setupccdir()
	defer os.RemoveAll(ccdir)

	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"},
		Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}

	ccpack, _, _, err := processSignedCDS(cds, &common.SignaturePolicyEnvelope{Version: 1}, true)
	if err != nil {
		t.Fatalf("cannot create package %s", err)
		return
	}

//错误案例1:传递给getcdsdata的签名链代码部署规范为零
	assert.Panics(t, func() {
		_, _, _, err = ccpack.getCDSData(nil)
	}, "getCDSData should have paniced when called with nil signed chaincode deployment spec")

//错误案例2:错误的链代码部署规范
	scdp := &pb.SignedChaincodeDeploymentSpec{ChaincodeDeploymentSpec: []byte("bad spec")}
	_, _, _, err = ccpack.getCDSData(scdp)
	assert.Error(t, err)

//错误案例3:实例化策略为零
	instPolicy := ccpack.sDepSpec.InstantiationPolicy
	ccpack.sDepSpec.InstantiationPolicy = nil
	_, _, _, err = ccpack.getCDSData(ccpack.sDepSpec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "instantiation policy cannot be nil for chaincode", "Unexpected error returned by getCDSData")
	ccpack.sDepSpec.InstantiationPolicy = instPolicy

	ccpack.sDepSpec.OwnerEndorsements = make([]*pb.Endorsement, 1)
	ccpack.sDepSpec.OwnerEndorsements[0] = &pb.Endorsement{}
	_, _, _, err = ccpack.getCDSData(ccpack.sDepSpec)
	assert.NoError(t, err)
}

func TestInitFromBufferErrorPaths(t *testing.T) {
	ccdir := setupccdir()
	defer os.RemoveAll(ccdir)

	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"},
		Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}

	ccpack, _, _, err := processSignedCDS(cds, &common.SignaturePolicyEnvelope{Version: 1}, true)
	if err != nil {
		t.Fatalf("cannot create package %s", err)
		return
	}

	_, err = ccpack.InitFromBuffer([]byte("bad buffer"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal envelope from bytes", "Unexpected error returned by InitFromBuffer")
}

func TestValidateSignedCCErrorPaths(t *testing.T) {
	ccdir := setupccdir()
	defer os.RemoveAll(ccdir)

	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"},
		Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}

	ccpack, _, _, err := processSignedCDS(cds, &common.SignaturePolicyEnvelope{Version: 1}, true)
	if err != nil {
		t.Fatalf("cannot create package %s", err)
		return
	}

//使用无效名称验证
	cd := &ChaincodeData{Name: "invalname", Version: "0"}
	err = ccpack.ValidateCC(cd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid chaincode data", "Unexpected error validating package")

	savDepSpec := ccpack.sDepSpec
	ccpack.sDepSpec = nil
	err = ccpack.ValidateCC(cd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uninitialized package", "Unexpected error validating package")
	ccpack.sDepSpec = savDepSpec

	cdspec := ccpack.sDepSpec.ChaincodeDeploymentSpec
	ccpack.sDepSpec.ChaincodeDeploymentSpec = nil
	err = ccpack.ValidateCC(cd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signed chaincode deployment spec cannot be nil in a package", "Unexpected error validating package")
	ccpack.sDepSpec.ChaincodeDeploymentSpec = cdspec

	depspec := ccpack.depSpec
	ccpack.depSpec = nil
	err = ccpack.ValidateCC(cd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chaincode deployment spec cannot be nil in a package", "Unexpected error validating package")
	ccpack.depSpec = depspec

	cd = &ChaincodeData{Name: "\027", Version: "0"}
	err = ccpack.ValidateCC(cd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `invalid chaincode name: "\x17"`)
}

func TestSigCDSGetCCPackage(t *testing.T) {
	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"}, Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}

	env, err := ccpackage.OwnerCreateSignedCCDepSpec(cds, &common.SignaturePolicyEnvelope{Version: 1}, nil)
	if err != nil {
		t.Fatalf("cannot create package")
		return
	}

	b := utils.MarshalOrPanic(env)

	ccpack, err := GetCCPackage(b)
	if err != nil {
		t.Fatalf("failed to get CCPackage %s", err)
		return
	}

	cccdspack, ok := ccpack.(*CDSPackage)
	if ok || cccdspack != nil {
		t.Fatalf("expected CDSPackage type cast to fail but succeeded")
		return
	}

	ccsignedcdspack, ok := ccpack.(*SignedCDSPackage)
	if !ok || ccsignedcdspack == nil {
		t.Fatalf("failed to get Signed CDS CCPackage")
		return
	}

	cds2 := ccsignedcdspack.GetDepSpec()
	if cds2 == nil {
		t.Fatalf("nil dep spec in Signed CDS CCPackage")
		return
	}

	if cds2.ChaincodeSpec.ChaincodeId.Name != cds.ChaincodeSpec.ChaincodeId.Name || cds2.ChaincodeSpec.ChaincodeId.Version != cds.ChaincodeSpec.ChaincodeId.Version {
		t.Fatalf("dep spec in Signed CDS CCPackage does not match %v != %v", cds, cds2)
		return
	}
}

func TestInvalidSigCDSGetCCPackage(t *testing.T) {
	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"}, Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}

	b := utils.MarshalOrPanic(cds)
	ccpack, err := GetCCPackage(b)
	if err != nil {
		t.Fatalf("failed to get CCPackage %s", err)
	}

	ccsignedcdspack, ok := ccpack.(*SignedCDSPackage)
	if ok || ccsignedcdspack != nil {
		t.Fatalf("expected failure to get Signed CDS CCPackage but succeeded")
	}
}

//切换fs上的链码并验证
func TestSignedCDSSwitchChaincodes(t *testing.T) {
	ccdir := setupccdir()
	defer os.RemoveAll(ccdir)

//有人用“badcode”修改了fs上的代码
	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"}, Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("badcode")}

//将错误代码写入fs
	badccpack, _, _, err := processSignedCDS(cds, &common.SignaturePolicyEnvelope{Version: 1}, true)
	if err != nil {
		t.Fatalf("error putting CDS to FS %s", err)
		return
	}

//从实例中模拟良好的代码链代码数据…
	cds.CodePackage = []byte("goodcode")

//…并为其生成CD（不要覆盖坏代码）
	_, _, goodcd, err := processSignedCDS(cds, &common.SignaturePolicyEnvelope{Version: 1}, false)
	if err != nil {
		t.Fatalf("error putting CDS to FS %s", err)
		return
	}

	if err = badccpack.ValidateCC(goodcd); err == nil {
		t.Fatalf("expected goodcd to fail against bad package but succeeded!")
		return
	}
}
