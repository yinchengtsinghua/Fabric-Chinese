
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


package chaincode

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/msp"
	msptesttools "github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/peer/common"
	pcommon "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
)

func TestMain(m *testing.M) {
	err := msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		panic(fmt.Sprintf("Fatal error when reading MSP config: %s", err))
	}

	os.Exit(m.Run())
}

func newTempDir() string {
	tempDir, err := ioutil.TempDir("/tmp", "packagetest-")
	if err != nil {
		panic(err)
	}
	return tempDir
}

func mockCDSFactory(spec *pb.ChaincodeSpec) (*pb.ChaincodeDeploymentSpec, error) {
	return &pb.ChaincodeDeploymentSpec{ChaincodeSpec: spec, CodePackage: []byte("somecode")}, nil
}

func extractSignedCCDepSpec(env *pcommon.Envelope) (*pcommon.ChannelHeader, *pb.SignedChaincodeDeploymentSpec, error) {
	p := &pcommon.Payload{}
	err := proto.Unmarshal(env.Payload, p)
	if err != nil {
		return nil, nil, err
	}
	ch := &pcommon.ChannelHeader{}
	err = proto.Unmarshal(p.Header.ChannelHeader, ch)
	if err != nil {
		return nil, nil, err
	}

	sp := &pb.SignedChaincodeDeploymentSpec{}
	err = proto.Unmarshal(p.Data, sp)
	if err != nil {
		return nil, nil, err
	}

	return ch, sp, nil
}

//testcdspackage测试生成旧的chaincodedeploymentspec安装
//我们至少会继续支持一段时间
func TestCDSPackage(t *testing.T) {
	pdir := newTempDir()
	defer os.RemoveAll(pdir)

	ccpackfile := pdir + "/ccpack.file"
	err := createSignedCDSPackage([]string{"-n", "somecc", "-p", "some/go/package", "-v", "0", ccpackfile}, false)
	if err != nil {
		t.Fatalf("Run chaincode package cmd error:%v", err)
	}

	b, err := ioutil.ReadFile(ccpackfile)
	if err != nil {
		t.Fatalf("package file %s not created", ccpackfile)
	}
	cds := &pb.ChaincodeDeploymentSpec{}
	err = proto.Unmarshal(b, cds)
	if err != nil {
		t.Fatalf("could not unmarshall package into CDS")
	}
}

//创建SignedChaincodeDeploymentSpec的帮助程序
func createSignedCDSPackage(args []string, sign bool) error {
	var signer msp.SigningIdentity
	var err error
	if sign {

		signer, err = common.GetDefaultSigner()
		if err != nil {
			return fmt.Errorf("Get default signer error: %v", err)
		}
	}

	mockCF := &ChaincodeCmdFactory{Signer: signer}

	cmd := packageCmd(mockCF, mockCDSFactory)
	addFlags(cmd)

	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		return err
	}

	return nil
}

//testsignedcddspackage生成新的信封封装
//CDS政策
func TestSignedCDSPackage(t *testing.T) {
	pdir := newTempDir()
	defer os.RemoveAll(pdir)

	ccpackfile := pdir + "/ccpack.file"
	err := createSignedCDSPackage([]string{"-n", "somecc", "-p", "some/go/package", "-v", "0", "-s", ccpackfile}, false)
	if err != nil {
		t.Fatalf("could not create signed cds package %s", err)
	}

	b, err := ioutil.ReadFile(ccpackfile)
	if err != nil {
		t.Fatalf("package file %s not created", ccpackfile)
	}

	e := &pcommon.Envelope{}
	err = proto.Unmarshal(b, e)
	if err != nil {
		t.Fatalf("could not unmarshall envelope")
	}

	_, p, err := extractSignedCCDepSpec(e)
	if err != nil {
		t.Fatalf("could not extract signed dep spec")
	}

	if p.OwnerEndorsements != nil {
		t.Fatalf("expected no signtures but found endorsements")
	}
}

//testsignedcdsdpackagewithsignature生成新的封装信封
//CD、政策和与当地MSP签署包
func TestSignedCDSPackageWithSignature(t *testing.T) {
	pdir := newTempDir()
	defer os.RemoveAll(pdir)

	ccpackfile := pdir + "/ccpack.file"
	err := createSignedCDSPackage([]string{"-n", "somecc", "-p", "some/go/package", "-v", "0", "-s", "-S", ccpackfile}, true)
	if err != nil {
		t.Fatalf("could not create signed cds package %s", err)
	}

	b, err := ioutil.ReadFile(ccpackfile)
	if err != nil {
		t.Fatalf("package file %s not created", ccpackfile)
	}
	e := &pcommon.Envelope{}
	err = proto.Unmarshal(b, e)
	if err != nil {
		t.Fatalf("could not unmarshall envelope")
	}

	_, p, err := extractSignedCCDepSpec(e)
	if err != nil {
		t.Fatalf("could not extract signed dep spec")
	}

	if p.OwnerEndorsements == nil {
		t.Fatalf("expected signtures and found nil")
	}
}

func TestNoOwnerToSign(t *testing.T) {
	pdir := newTempDir()
	defer os.RemoveAll(pdir)

	ccpackfile := pdir + "/ccpack.file"
//注“-s”需要签名人，但我们正在传递fase
	err := createSignedCDSPackage([]string{"-n", "somecc", "-p", "some/go/package", "-v", "0", "-s", "-S", ccpackfile}, false)

	if err == nil {
		t.Fatalf("Expected error with nil signer but succeeded")
	}
}

func TestInvalidPolicy(t *testing.T) {
	pdir := newTempDir()
	defer os.RemoveAll(pdir)

	ccpackfile := pdir + "/ccpack.file"
	err := createSignedCDSPackage([]string{"-n", "somecc", "-p", "some/go/package", "-v", "0", "-s", "-i", "AND('a bad policy')", ccpackfile}, false)

	if err == nil {
		t.Fatalf("Expected error with nil signer but succeeded")
	}
}
