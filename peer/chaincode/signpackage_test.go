
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 版权所有Digital Asset Holdings，LLC 2016保留所有权利。

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
	"github.com/hyperledger/fabric/peer/common"
	pcommon "github.com/hyperledger/fabric/protos/common"
)

//签署现有包的帮助程序
func signExistingPackage(env *pcommon.Envelope, infile, outfile string) error {
	signer, err := common.GetDefaultSigner()
	if err != nil {
		return fmt.Errorf("Get default signer error: %v", err)
	}

	mockCF := &ChaincodeCmdFactory{Signer: signer}

	cmd := signpackageCmd(mockCF)
	addFlags(cmd)

	cmd.SetArgs([]string{infile, outfile})

	if err := cmd.Execute(); err != nil {
		return err
	}

	return nil
}

//testsignexistingpackage对现有包进行签名
func TestSignExistingPackage(t *testing.T) {
	pdir := newTempDir()
	defer os.RemoveAll(pdir)

	ccpackfile := pdir + "/ccpack.file"
	err := createSignedCDSPackage([]string{"-n", "somecc", "-p", "some/go/package", "-v", "0", "-s", "-S", ccpackfile}, true)
	if err != nil {
		t.Fatalf("error creating signed :%v", err)
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

	signedfile := pdir + "/signed.file"
	err = signExistingPackage(e, ccpackfile, signedfile)
	if err != nil {
		t.Fatalf("could not sign envelope")
	}

	b, err = ioutil.ReadFile(signedfile)
	if err != nil {
		t.Fatalf("signed package file %s not created", signedfile)
	}

	e = &pcommon.Envelope{}
	err = proto.Unmarshal(b, e)
	if err != nil {
		t.Fatalf("could not unmarshall signed envelope")
	}

	_, p, err := extractSignedCCDepSpec(e)
	if err != nil {
		t.Fatalf("could not extract signed dep spec")
	}

	if p.OwnerEndorsements == nil {
		t.Fatalf("expected endorsements")
	}

	if len(p.OwnerEndorsements) != 2 {
		t.Fatalf("expected 2 endorserments but found %d", len(p.OwnerEndorsements))
	}
}

//testfailsignunsignedpackage尝试对最初未签名的包进行签名
func TestFailSignUnsignedPackage(t *testing.T) {
	pdir := newTempDir()
	defer os.RemoveAll(pdir)

	ccpackfile := pdir + "/ccpack.file"
//不要签…没有“-S”
	err := createSignedCDSPackage([]string{"-n", "somecc", "-p", "some/go/package", "-v", "0", "-s", ccpackfile}, true)
	if err != nil {
		t.Fatalf("error creating signed :%v", err)
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

	signedfile := pdir + "/signed.file"
	err = signExistingPackage(e, ccpackfile, signedfile)
	if err == nil {
		t.Fatalf("expected signing a package that's not originally signed to fail")
	}
}
