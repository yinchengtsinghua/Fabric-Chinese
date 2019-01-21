
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


package msp

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/stretchr/testify/assert"
)

func TestGetLocalMspConfig(t *testing.T) {
	mspDir, err := configtest.GetDevMspDir()
	assert.NoError(t, err)
	_, err = GetLocalMspConfig(mspDir, nil, "SampleOrg")
	assert.NoError(t, err)
}

func TestGetLocalMspConfigFails(t *testing.T) {
	_, err := GetLocalMspConfig("/tmp/", nil, "SampleOrg")
	assert.Error(t, err)
}

func TestGetPemMaterialFromDirWithFile(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "fabric-msp-test")
	assert.NoError(t, err)
	err = tempFile.Close()
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = getPemMaterialFromDir(tempFile.Name())
	assert.Error(t, err)
}

func TestGetPemMaterialFromDirWithSymlinks(t *testing.T) {
	mspDir, err := configtest.GetDevMspDir()
	assert.NoError(t, err)

	tempDir, err := ioutil.TempDir("", "fabric-msp-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dirSymlinkName := filepath.Join(tempDir, "..data")
	err = os.Symlink(filepath.Join(mspDir, "signcerts"), dirSymlinkName)
	assert.NoError(t, err)

	fileSymlinkTarget := filepath.Join("..data", "peer.pem")
	fileSymlinkName := filepath.Join(tempDir, "peer.pem")
	err = os.Symlink(fileSymlinkTarget, fileSymlinkName)
	assert.NoError(t, err)

	pemdataSymlink, err := getPemMaterialFromDir(tempDir)
	assert.NoError(t, err)
	expected, err := getPemMaterialFromDir(filepath.Join(mspDir, "signcerts"))
	assert.NoError(t, err)
	assert.Equal(t, pemdataSymlink, expected)
}

func TestReadFileUtils(t *testing.T) {
//测试读取具有空路径的文件是否不会崩溃
	_, err := readPemFile("")
	assert.Error(t, err)

//测试读取不是PEM文件的现有文件是否不会崩溃
	_, err = readPemFile("/dev/null")
	assert.Error(t, err)
}
