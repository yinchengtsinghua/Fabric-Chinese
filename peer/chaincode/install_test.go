
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


package chaincode

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hyperledger/fabric/peer/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func initInstallTest(fsPath string, t *testing.T) (*cobra.Command, *ChaincodeCmdFactory) {
	viper.Set("peer.fileSystemPath", fsPath)
	cleanupInstallTest(fsPath)

//如果mkdir失败，一切都将失败…但它不应该
	if err := os.Mkdir(fsPath, 0755); err != nil {
		t.Fatalf("could not create install env")
	}

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	mockCF := &ChaincodeCmdFactory{
		Signer: signer,
	}

	cmd := installCmd(mockCF)
	addFlags(cmd)

	return cmd, mockCF
}

func cleanupInstallTest(fsPath string) {
	os.RemoveAll(fsPath)
}

//testbadversion测试生成安装命令
func TestBadVersion(t *testing.T) {
	fsPath := "/tmp/installtest"

	cmd, _ := initInstallTest(fsPath, t)
	defer cleanupInstallTest(fsPath)

	args := []string{"-n", "example02", "-p", "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"}
	cmd.SetArgs(args)

	if err := cmd.Execute(); err == nil {
		t.Fatal("Expected error executing install command for version not specified")
	}
}

//testnonexistentcc不存在的chaincode应按预期失败
func TestNonExistentCC(t *testing.T) {
	fsPath := "/tmp/installtest"

	cmd, _ := initInstallTest(fsPath, t)
	defer cleanupInstallTest(fsPath)

	args := []string{"-n", "badexample02", "-p", "github.com/hyperledger/fabric/examples/chaincode/go/bad_example02", "-v", "testversion"}
	cmd.SetArgs(args)

	if err := cmd.Execute(); err == nil {
		t.Fatal("Expected error executing install command for bad chaincode")
	}

	if _, err := os.Stat(fsPath + "/chaincodes/badexample02.testversion"); err == nil {
		t.Fatal("chaincode example02.testversion should not exist")
	}
}

//TestInstallFromPackage使用包安装
func TestInstallFromPackage(t *testing.T) {
	pdir := newTempDir()
	defer os.RemoveAll(pdir)

	ccpackfile := pdir + "/ccpack.file"
	err := createSignedCDSPackage([]string{"-n", "somecc", "-p", "some/go/package", "-v", "0", ccpackfile}, false)
	if err != nil {
		t.Fatalf("could not create package :%v", err)
	}

	fsPath := "/tmp/installtest"

	cmd, mockCF := initInstallTest(fsPath, t)
	defer cleanupInstallTest(fsPath)

	mockResponse := &pb.ProposalResponse{
		Response:    &pb.Response{Status: 200},
		Endorsement: &pb.Endorsement{},
	}
	mockEndorserClient := common.GetMockEndorserClient(mockResponse, nil)
	mockCF.EndorserClients = []pb.EndorserClient{mockEndorserClient}

	args := []string{ccpackfile}
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatal("error executing install command from package")
	}
}

//testinstallfrombadpackage测试bad包失败
func TestInstallFromBadPackage(t *testing.T) {
	pdir := newTempDir()
	defer os.RemoveAll(pdir)

	ccpackfile := pdir + "/ccpack.file"
	err := ioutil.WriteFile(ccpackfile, []byte("really bad CC package"), 0700)
	if err != nil {
		t.Fatalf("could not create package :%v", err)
	}

	fsPath := "/tmp/installtest"

	cmd, mockCF := initInstallTest(fsPath, t)
	defer cleanupInstallTest(fsPath)

//这不应该传到背书人那里，而背书人会成功地作出回应。
	mockResponse := &pb.ProposalResponse{
		Response:    &pb.Response{Status: 200},
		Endorsement: &pb.Endorsement{},
	}
	mockEndorserClient := common.GetMockEndorserClient(mockResponse, nil)
	mockCF.EndorserClients = []pb.EndorserClient{mockEndorserClient}

	args := []string{ccpackfile}
	cmd.SetArgs(args)

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error installing bad package")
	}
}
func installEx02(t *testing.T) error {
	defer viper.Reset()
	viper.Set("chaincode.mode", "dev")

	fsPath := "/tmp/installtest"
	cmd, mockCF := initInstallTest(fsPath, t)
	defer cleanupInstallTest(fsPath)

	mockResponse := &pb.ProposalResponse{
		Response:    &pb.Response{Status: 200},
		Endorsement: &pb.Endorsement{},
	}
	mockEndorserClient := common.GetMockEndorserClient(mockResponse, nil)
	mockCF.EndorserClients = []pb.EndorserClient{mockEndorserClient}

	args := []string{"-n", "example02", "-p", "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd", "-v", "anotherversion"}
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		return fmt.Errorf("Run chaincode upgrade cmd error:%v", err)
	}

	return nil
}

func TestInstall(t *testing.T) {
	if err := installEx02(t); err != nil {
		t.Fatalf("Install failed with error: %v", err)
	}
}
