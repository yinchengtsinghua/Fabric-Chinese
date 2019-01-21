
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
	"io/ioutil"
	"os"
	"testing"

	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func setupccdir() string {
	tempDir, err := ioutil.TempDir("/tmp", "ccprovidertest")
	if err != nil {
		panic(err)
	}
	SetChaincodesPath(tempDir)
	return tempDir
}

func processCDS(cds *pb.ChaincodeDeploymentSpec, tofs bool) (*CDSPackage, []byte, *ChaincodeData, error) {
	b := utils.MarshalOrPanic(cds)

	ccpack := &CDSPackage{}
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

func TestPutCDSCC(t *testing.T) {
	ccdir := setupccdir()
	defer os.RemoveAll(ccdir)

	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"}, Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}

	ccpack, _, cd, err := processCDS(cds, true)
	if err != nil {
		t.Fatalf("error putting CDS to FS %s", err)
		return
	}

	if err = ccpack.ValidateCC(cd); err != nil {
		t.Fatalf("error validating package %s", err)
		return
	}
}

func TestPutCDSErrorPaths(t *testing.T) {
	ccdir := setupccdir()
	defer os.RemoveAll(ccdir)

	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"},
		Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}

	ccpack, b, _, err := processCDS(cds, true)
	if err != nil {
		t.Fatalf("error putting CDS to FS %s", err)
		return
	}

//使用无效名称验证
	if err = ccpack.ValidateCC(&ChaincodeData{Name: "invalname", Version: "0"}); err == nil {
		t.Fatalf("expected error validating package")
		return
	}
//取出缓冲器
	ccpack.buf = nil
	if err = ccpack.PutChaincodeToFS(); err == nil {
		t.Fatalf("expected error putting package on the FS")
		return
	}

//把缓冲器放回原处，但要拆下DEPSPEC。
	ccpack.buf = b
	savDepSpec := ccpack.depSpec
	ccpack.depSpec = nil
	if err = ccpack.PutChaincodeToFS(); err == nil {
		t.Fatalf("expected error putting package on the FS")
		return
	}

//退回DEP规范
	ccpack.depSpec = savDepSpec

//…但删除chaincode目录
	os.RemoveAll(ccdir)
	if err = ccpack.PutChaincodeToFS(); err == nil {
		t.Fatalf("expected error putting package on the FS")
		return
	}
}

func TestCDSGetCCPackage(t *testing.T) {
	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"}, Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}

	b := utils.MarshalOrPanic(cds)

	ccpack, err := GetCCPackage(b)
	if err != nil {
		t.Fatalf("failed to get CDS CCPackage %s", err)
		return
	}

	cccdspack, ok := ccpack.(*CDSPackage)
	if !ok || cccdspack == nil {
		t.Fatalf("failed to get CDS CCPackage")
		return
	}

	cds2 := cccdspack.GetDepSpec()
	if cds2 == nil {
		t.Fatalf("nil dep spec in CDS CCPackage")
		return
	}

	if cds2.ChaincodeSpec.ChaincodeId.Name != cds.ChaincodeSpec.ChaincodeId.Name || cds2.ChaincodeSpec.ChaincodeId.Version != cds.ChaincodeSpec.ChaincodeId.Version {
		t.Fatalf("dep spec in CDS CCPackage does not match %v != %v", cds, cds2)
		return
	}
}

//切换fs上的链码并验证
func TestCDSSwitchChaincodes(t *testing.T) {
	ccdir := setupccdir()
	defer os.RemoveAll(ccdir)

//有人用“badcode”修改了fs上的代码
	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"}, Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("badcode")}

//将错误代码写入fs
	badccpack, _, _, err := processCDS(cds, true)
	if err != nil {
		t.Fatalf("error putting CDS to FS %s", err)
		return
	}

//从实例中模拟良好的代码链代码数据…
	cds.CodePackage = []byte("goodcode")

//…并为其生成CD（不要覆盖坏代码）
	_, _, goodcd, err := processCDS(cds, false)
	if err != nil {
		t.Fatalf("error putting CDS to FS %s", err)
		return
	}

	if err = badccpack.ValidateCC(goodcd); err == nil {
		t.Fatalf("expected goodcd to fail against bad package but succeeded!")
		return
	}
}

func TestPutChaincodeToFSErrorPaths(t *testing.T) {
	ccpack := &CDSPackage{}
	err := ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uninitialized package", "Unexpected error returned")

	ccpack.buf = []byte("hello")
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id cannot be nil if buf is not nil", "Unexpected error returned")

	ccpack.id = []byte("cc123")
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "depspec cannot be nil if buf is not nil", "Unexpected error returned")

	ccpack.depSpec = &pb.ChaincodeDeploymentSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"},
		Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("")}}}, CodePackage: []byte("code")}
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil data", "Unexpected error returned")

	ccpack.data = &CDSData{}
	err = ccpack.PutChaincodeToFS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil data bytes", "Unexpected error returned")
}

func TestValidateCCErrorPaths(t *testing.T) {
	cpack := &CDSPackage{}
	ccdata := &ChaincodeData{}
	err := cpack.ValidateCC(ccdata)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uninitialized package", "Unexpected error returned")

	cpack.depSpec = &pb.ChaincodeDeploymentSpec{
		CodePackage: []byte("code"),
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type:        1,
			ChaincodeId: &pb.ChaincodeID{Name: "testcc", Version: "0"},
			Input:       &pb.ChaincodeInput{Args: [][]byte{[]byte("")}},
		},
	}
	err = cpack.ValidateCC(ccdata)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil data", "Unexpected error returned")

//编码名称无效
	cpack = &CDSPackage{}
	ccdata = &ChaincodeData{Name: "\027"}
	cpack.depSpec = &pb.ChaincodeDeploymentSpec{
		CodePackage: []byte("code"),
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{Name: ccdata.Name, Version: "0"},
		},
	}
	cpack.data = &CDSData{}
	err = cpack.ValidateCC(ccdata)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `invalid chaincode name: "\x17"`)

//不匹配的名称
	cpack = &CDSPackage{}
	ccdata = &ChaincodeData{Name: "Tom"}
	cpack.depSpec = &pb.ChaincodeDeploymentSpec{
		CodePackage: []byte("code"),
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{Name: "Jerry", Version: "0"},
		},
	}
	cpack.data = &CDSData{}
	err = cpack.ValidateCC(ccdata)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `invalid chaincode data name:"Tom"  (name:"Jerry" version:"0" )`)

//不匹配的版本
	cpack = &CDSPackage{}
	ccdata = &ChaincodeData{Name: "Tom", Version: "1"}
	cpack.depSpec = &pb.ChaincodeDeploymentSpec{
		CodePackage: []byte("code"),
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{Name: ccdata.Name, Version: "0"},
		},
	}
	cpack.data = &CDSData{}
	err = cpack.ValidateCC(ccdata)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `invalid chaincode data name:"Tom" version:"1"  (name:"Tom" version:"0" )`)
}
