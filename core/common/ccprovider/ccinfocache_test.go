
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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/container/util"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

func getDepSpec(name string, path string, version string, initArgs [][]byte) (*peer.ChaincodeDeploymentSpec, error) {
	spec := &peer.ChaincodeSpec{Type: 1, ChaincodeId: &peer.ChaincodeID{Name: name, Path: path, Version: version}, Input: &peer.ChaincodeInput{Args: initArgs}}

	codePackageBytes := bytes.NewBuffer(nil)
	gz := gzip.NewWriter(codePackageBytes)
	tw := tar.NewWriter(gz)

	err := util.WriteBytesToPackage("src/garbage.go", []byte(name+path+version), tw)
	if err != nil {
		return nil, err
	}

	tw.Close()
	gz.Close()

	return &peer.ChaincodeDeploymentSpec{ChaincodeSpec: spec, CodePackage: codePackageBytes.Bytes()}, nil
}

func buildPackage(name string, path string, version string, initArgs [][]byte) (CCPackage, error) {
	depSpec, err := getDepSpec(name, path, version, initArgs)
	if err != nil {
		return nil, err
	}

	buf, err := proto.Marshal(depSpec)
	if err != nil {
		return nil, err
	}
	cccdspack := &CDSPackage{}
	if _, err := cccdspack.InitFromBuffer(buf); err != nil {
		return nil, err
	}

	return cccdspack, nil
}

type mockCCInfoFSStorageMgrImpl struct {
	CCMap map[string]CCPackage
}

func (m *mockCCInfoFSStorageMgrImpl) GetChaincode(ccname string, ccversion string) (CCPackage, error) {
	return m.CCMap[ccname+ccversion], nil
}

//这里我们测试缓存实现本身
func TestCCInfoCache(t *testing.T) {
	ccname := "foo"
	ccver := "1.0"
	ccpath := "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"

	ccinfoFs := &mockCCInfoFSStorageMgrImpl{CCMap: map[string]CCPackage{}}
	cccache := NewCCInfoCache(ccinfoFs)

//测试GeT端

//CC数据尚不在缓存中
	_, err := cccache.GetChaincodeData(ccname, ccver)
	assert.Error(t, err)

//放入文件系统
	pack, err := buildPackage(ccname, ccpath, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")})
	assert.NoError(t, err)
	ccinfoFs.CCMap[ccname+ccver] = pack

//希望它现在在缓存中
	cd1, err := cccache.GetChaincodeData(ccname, ccver)
	assert.NoError(t, err)

//它应该还在缓存中
	cd2, err := cccache.GetChaincodeData(ccname, ccver)
	assert.NoError(t, err)

//它们不是空的
	assert.NotNil(t, cd1)
	assert.NotNil(t, cd2)

//现在测试Put侧。
	ccver = "2.0"
//放入文件系统
	pack, err = buildPackage(ccname, ccpath, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")})
	assert.NoError(t, err)
	ccinfoFs.CCMap[ccname+ccver] = pack

//创建要放置的DEP规范
	_, err = getDepSpec(ccname, ccpath, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")})
	assert.NoError(t, err)

//希望它被缓存
	cd1, err = cccache.GetChaincodeData(ccname, ccver)
	assert.NoError(t, err)

//它应该还在缓存中
	cd2, err = cccache.GetChaincodeData(ccname, ccver)
	assert.NoError(t, err)

//它们不是空的
	assert.NotNil(t, cd1)
	assert.NotNil(t, cd2)
}

func TestPutChaincode(t *testing.T) {
	ccname := ""
	ccver := "1.0"
	ccpath := "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"

	ccinfoFs := &mockCCInfoFSStorageMgrImpl{CCMap: map[string]CCPackage{}}
	NewCCInfoCache(ccinfoFs)

//错误案例1:ccname为空
//创建要放置的DEP规范
	_, err := getDepSpec(ccname, ccpath, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")})
	assert.NoError(t, err)

//错误案例2:ccver为空
	ccname = "foo"
	ccver = ""
	_, err = getDepSpec(ccname, ccpath, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")})
	assert.NoError(t, err)

//错误案例3:ccfs.putchaincode返回错误
	ccinfoFs = &mockCCInfoFSStorageMgrImpl{CCMap: map[string]CCPackage{}}
	NewCCInfoCache(ccinfoFs)

	ccname = "foo"
	ccver = "1.0"
	_, err = getDepSpec(ccname, ccpath, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")})
	assert.NoError(t, err)
}

//在这里，我们在启用对等缓存之后测试它的内置缓存
func TestCCInfoFSPeerInstance(t *testing.T) {
	ccname := "bar"
	ccver := "1.0"
	ccpath := "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"

//CC数据尚不在缓存中
	_, err := GetChaincodeFromFS(ccname, ccver)
	assert.Error(t, err)

//创建要放置的DEP规范
	ds, err := getDepSpec(ccname, ccpath, ccver, [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")})
	assert.NoError(t, err)

//放它
	err = PutChaincodeIntoFS(ds)
	assert.NoError(t, err)

//获取所有已安装的链码，不应返回0个链码
	resp, err := GetInstalledChaincodes()
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotZero(t, len(resp.Chaincodes), "GetInstalledChaincodes should not have returned 0 chaincodes")

//获取链码数据
	_, err = GetChaincodeData(ccname, ccver)
	assert.NoError(t, err)
}

func TestGetInstalledChaincodesErrorPaths(t *testing.T) {
//获取现有的chaincode安装路径值并设置它
//做完测试后再回来
	cip := chaincodeInstallPath
	defer SetChaincodesPath(cip)

//创建一个临时目录并在末尾将其删除
	dir, err := ioutil.TempDir(os.TempDir(), "chaincodes")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

//将上面创建的目录设置为chaincode安装路径
	SetChaincodesPath(dir)
	err = ioutil.WriteFile(filepath.Join(dir, "idontexist.1.0"), []byte("test"), 0777)
	assert.NoError(t, err)
	resp, err := GetInstalledChaincodes()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(resp.Chaincodes),
		"Expected 0 chaincodes but GetInstalledChaincodes returned %s chaincodes", len(resp.Chaincodes))
}

func TestChaincodePackageExists(t *testing.T) {
	_, err := ChaincodePackageExists("foo1", "1.0")
	assert.Error(t, err)
}

func TestSetChaincodesPath(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "setchaincodes")
	if err != nil {
		assert.Fail(t, err.Error(), "Unable to create temp dir")
	}
	defer os.RemoveAll(dir)
	t.Logf("created temp dir %s", dir)

//获取现有的chaincode安装路径值并设置它
//做完测试后再回来
	cip := chaincodeInstallPath
	defer SetChaincodesPath(cip)

	f, err := ioutil.TempFile(dir, "chaincodes")
	assert.NoError(t, err)
	assert.Panics(t, func() {
		SetChaincodesPath(f.Name())
	}, "SetChaincodesPath should have paniced if a file is passed to it")

//以下代码适用于Mac，但不适用于CI
////使目录为只读
//err=os.chmod（目录，0444）
//断言.noError（t，err）
//cdir：=filepath.join（dir，“chaincodesdir”）。
//断言.panics（t，func（）
//设置链码速度（cdir）
//，“如果setchaincodespath无法统计dir”，它应该会恐慌。

////读取并执行目录
//err=os.chmod（目录，0555）
//断言.noError（t，err）
//断言.panics（t，func（）
//设置链码速度（cdir）
//，“如果setchaincodespath无法创建dir”，则它应该会恐慌。
}

var ccinfocachetestpath = "/tmp/ccinfocachetest"

func TestMain(m *testing.M) {
	os.RemoveAll(ccinfocachetestpath)

	SetChaincodesPath(ccinfocachetestpath)
	rc := m.Run()
	os.RemoveAll(ccinfocachetestpath)
	os.Exit(rc)
}
