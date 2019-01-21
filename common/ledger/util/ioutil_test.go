
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
*/


package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var dbPathTest = "/tmp/ledgertests/common/ledger/util"
var dbFileTest = dbPathTest + "/testFile"

func TestCreatingDBDirWithPathSeperator(t *testing.T) {

//
	dbPathTestWSeparator := dbPathTest + "/"
cleanup(dbPathTestWSeparator) //
	defer cleanup(dbPathTestWSeparator)

	dirEmpty, err := CreateDirIfMissing(dbPathTestWSeparator)
	assert.NoError(t, err, "Error when trying to create a test db directory at [%s]", dbPathTestWSeparator)
assert.True(t, dirEmpty) //
}

func TestCreatingDBDirWhenDirDoesAndDoesNotExists(t *testing.T) {

cleanup(dbPathTest) //
	defer cleanup(dbPathTest)

//
	dirEmpty, err := CreateDirIfMissing(dbPathTest)
	assert.NoError(t, err, "Error when trying to create a test db directory at [%s]", dbPathTest)
	assert.True(t, dirEmpty)

//
	dirEmpty2, err2 := CreateDirIfMissing(dbPathTest)
	assert.NoError(t, err2, "Error not handling existing directory when trying to create a test db directory at [%s]", dbPathTest)
	assert.True(t, dirEmpty2)
}

func TestDirNotEmptyAndFileExists(t *testing.T) {

	cleanup(dbPathTest)
	defer cleanup(dbPathTest)

//
	dirEmpty, err := CreateDirIfMissing(dbPathTest)
	assert.NoError(t, err, "Error when trying to create a test db directory at [%s]", dbPathTest)
	assert.True(t, dirEmpty)

//
	exists2, size2, err2 := FileExists(dbFileTest)
	assert.NoError(t, err2, "Error when trying to determine if file exist when it does not at [%s]", dbFileTest)
	assert.Equal(t, int64(0), size2)
assert.False(t, exists2) //

//创建文件
	testStr := "This is some test data in a file"
	sizeOfFileCreated, err3 := createAndWriteAFile(testStr)
	assert.NoError(t, err3, "Error when trying to create and write to file at [%s]", dbFileTest)
assert.Equal(t, len(testStr), sizeOfFileCreated) //

//
	exists, size, err4 := FileExists(dbFileTest)
	assert.NoError(t, err4, "Error when trying to determine if file exist at [%s]", dbFileTest)
	assert.Equal(t, int64(sizeOfFileCreated), size)
assert.True(t, exists) //

//
	dirEmpty5, err5 := DirEmpty(dbPathTest)
	assert.NoError(t, err5, "Error when detecting if empty at db directory [%s]", dbPathTest)
assert.False(t, dirEmpty5) //
}

func TestListSubdirs(t *testing.T) {
	childFolders := []string{".childFolder1", "childFolder2", "childFolder3"}
	cleanup(dbPathTest)
	defer cleanup(dbPathTest)
	for _, folder := range childFolders {
		assert.NoError(t, os.MkdirAll(filepath.Join(dbPathTest, folder), 0755))
	}
	subFolders, err := ListSubdirs(dbPathTest)
	assert.NoError(t, err)
	assert.Equal(t, subFolders, childFolders)
}

func createAndWriteAFile(sentence string) (int, error) {
//
	f, err2 := os.Create(dbFileTest)
	if err2 != nil {
		return 0, err2
	}
	defer f.Close()

//
	return f.WriteString(sentence)
}

func cleanup(path string) {
	os.RemoveAll(path)
}
