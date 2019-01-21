
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


package testutil

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"
)

//
func ConstructRandomBytes(t testing.TB, size int) []byte {
	value := make([]byte, size)
	_, err := rand.Read(value)
	if err != nil {
		t.Fatalf("Error while generating random bytes: %s", err)
	}
	return value
}

//
type TarFileEntry struct {
	Name, Body string
}

//
func CreateTarBytesForTest(testFiles []*TarFileEntry) []byte {
//
	buffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(buffer)

	for _, file := range testFiles {
		tarHeader := &tar.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Body)),
		}
		err := tarWriter.WriteHeader(tarHeader)
		if err != nil {
			return nil
		}
		_, err = tarWriter.Write([]byte(file.Body))
		if err != nil {
			return nil
		}
	}
//
	tarWriter.Close()
	return buffer.Bytes()
}

//
func CopyDir(srcroot, destroot string) error {
	_, lastSegment := filepath.Split(srcroot)
	destroot = filepath.Join(destroot, lastSegment)

	walkFunc := func(srcpath string, info os.FileInfo, err error) error {
		srcsubpath, err := filepath.Rel(srcroot, srcpath)
		if err != nil {
			return err
		}
		destpath := filepath.Join(destroot, srcsubpath)

if info.IsDir() { //它是一个dir，在dest中生成相应的dir
			if err = os.MkdirAll(destpath, info.Mode()); err != nil {
				return err
			}
			return nil
		}

//
		if err = copyFile(srcpath, destpath); err != nil {
			return err
		}
		return nil
	}

	return filepath.Walk(srcroot, walkFunc)
}

func copyFile(srcpath, destpath string) error {
	var srcFile, destFile *os.File
	var err error
	if srcFile, err = os.Open(srcpath); err != nil {
		return err
	}
	if destFile, err = os.Create(destpath); err != nil {
		return err
	}
	if _, err = io.Copy(destFile, srcFile); err != nil {
		return err
	}
	if err = srcFile.Close(); err != nil {
		return err
	}
	if err = destFile.Close(); err != nil {
		return err
	}
	return nil
}
