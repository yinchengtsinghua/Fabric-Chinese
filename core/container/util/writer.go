
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


package util

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/platforms/ccmetadata"
	"github.com/pkg/errors"
)

var vmLogger = flogging.MustGetLogger("container")

//创建发送到Docker的tar包时排除这些文件类型
//已生成。可以排除类和其他临时文件
var javaExcludeFileTypes = map[string]bool{
	".class": true,
}

//writefoldertotarpackage将源文件写入tarball。
//此实用程序用于节点JS链码打包，但不用于golang链码。
//正如golang/platform.go中实现的那样，golang chaincode具有更复杂的文件打包。
func WriteFolderToTarPackage(tw *tar.Writer, srcPath string, excludeDirs []string, includeFileTypeMap map[string]bool, excludeFileTypeMap map[string]bool) error {
	fileCount := 0
	rootDirectory := srcPath

//如果通过，则修剪尾随斜线
	if rootDirectory[len(rootDirectory)-1] == '/' {
		rootDirectory = rootDirectory[:len(rootDirectory)-1]
	}

	vmLogger.Debugf("rootDirectory = %s", rootDirectory)

//必要时附加“/”
	updatedExcludeDirs := make([]string, 0)
	for _, excludeDir := range excludeDirs {
		if excludeDir != "" && strings.LastIndex(excludeDir, "/") < len(excludeDir)-1 {
			excludeDir = excludeDir + "/"
			updatedExcludeDirs = append(updatedExcludeDirs, excludeDir)
		}
	}

	rootDirLen := len(rootDirectory)
	walkFn := func(localpath string, info os.FileInfo, err error) error {

//如果localpath包含.git，则忽略
		if strings.Contains(localpath, ".git") {
			return nil
		}

		if info.Mode().IsDir() {
			return nil
		}

//排除任何带有excludedir前缀的文件。它们应该已经在焦油中了
		for _, excludeDir := range updatedExcludeDirs {
			if strings.Index(localpath, excludeDir) == rootDirLen+1 {
				return nil
			}
		}
//由于作用域，我们可以引用外部rootdirectory变量
		if len(localpath[rootDirLen:]) == 0 {
			return nil
		}
		ext := filepath.Ext(localpath)

		if includeFileTypeMap != nil {
//我们现在只需要“filetypes”源文件
			if _, ok := includeFileTypeMap[ext]; ok != true {
				return nil
			}
		}

//排除给定的文件类型
		if excludeFileTypeMap != nil {
			if exclude, ok := excludeFileTypeMap[ext]; ok && exclude {
				return nil
			}
		}

		var packagepath string

//如果文件是元数据，请保留/META-INF目录，例如：META-INF/statedb/couchdb/indexes/indexowner.json
//否则，文件是源代码，请将其放在/src dir中，例如：src/marbles\u chaincode.js
		if strings.HasPrefix(localpath, filepath.Join(rootDirectory, "META-INF")) {
			packagepath = localpath[rootDirLen+1:]

//将tar包路径拆分为tar包目录和文件名
			_, filename := filepath.Split(packagepath)

//隐藏文件不支持作为元数据，因此忽略它们。
//用户通常不知道隐藏的文件在那里，可能无法删除它们，因此警告用户而不是出错。
			if strings.HasPrefix(filename, ".") {
				vmLogger.Warningf("Ignoring hidden file in metadata directory: %s", packagepath)
				return nil
			}

			fileBytes, errRead := ioutil.ReadFile(localpath)
			if errRead != nil {
				return errRead
			}

//验证元数据文件是否包含在tar中
//验证基于文件的完全限定路径
			err = ccmetadata.ValidateMetadataFile(packagepath, fileBytes)
			if err != nil {
				return err
			}

} else { //文件不是元数据，包含在SRC中
			packagepath = fmt.Sprintf("src%s", localpath[rootDirLen:])
		}

		err = WriteFileToPackage(localpath, packagepath, tw)
		if err != nil {
			return fmt.Errorf("Error writing file to package: %s", err)
		}
		fileCount++

		return nil
	}

	if err := filepath.Walk(rootDirectory, walkFn); err != nil {
		vmLogger.Infof("Error walking rootDirectory: %s", err)
		return err
	}
//如果找不到文件，则返回错误
	if fileCount == 0 {
		return errors.Errorf("no source files found in '%s'", srcPath)
	}
	return nil
}

//从源路径打包Java项目到TAR文件
func WriteJavaProjectToPackage(tw *tar.Writer, srcPath string) error {

	vmLogger.Debugf("Packaging Java project from path %s", srcPath)

	if err := WriteFolderToTarPackage(tw, srcPath, []string{"target", "build", "out"}, nil, javaExcludeFileTypes); err != nil {

		vmLogger.Errorf("Error writing folder to tar package %s", err)
		return err
	}
//写出tar文件
	if err := tw.Close(); err != nil {
		return err
	}
	return nil

}

//writefiletopackage将文件写入tarball
func WriteFileToPackage(localpath string, packagepath string, tw *tar.Writer) error {
	vmLogger.Debug("Writing file to tarball:", packagepath)
	fd, err := os.Open(localpath)
	if err != nil {
		return fmt.Errorf("%s: %s", localpath, err)
	}
	defer fd.Close()

	is := bufio.NewReader(fd)
	return WriteStreamToPackage(is, localpath, packagepath, tw)

}

//writestreamtopackage将字节（从文件读取器）写入tarball
func WriteStreamToPackage(is io.Reader, localpath string, packagepath string, tw *tar.Writer) error {
	info, err := os.Stat(localpath)
	if err != nil {
		return fmt.Errorf("%s: %s", localpath, err)
	}
	header, err := tar.FileInfoHeader(info, localpath)
	if err != nil {
		return fmt.Errorf("Error getting FileInfoHeader: %s", err)
	}

//让我们把tar中的方差去掉，用零时间使报头完全相同。
	oldname := header.Name
	var zeroTime time.Time
	header.AccessTime = zeroTime
	header.ModTime = zeroTime
	header.ChangeTime = zeroTime
	header.Name = packagepath
	header.Mode = 0100644
	header.Uid = 500
	header.Gid = 500

	if err = tw.WriteHeader(header); err != nil {
		return fmt.Errorf("Error write header for (path: %s, oldname:%s,newname:%s,sz:%d) : %s", localpath, oldname, packagepath, header.Size, err)
	}
	if _, err := io.Copy(tw, is); err != nil {
		return fmt.Errorf("Error copy (path: %s, oldname:%s,newname:%s,sz:%d) : %s", localpath, oldname, packagepath, header.Size, err)
	}

	return nil
}

func WriteBytesToPackage(name string, payload []byte, tw *tar.Writer) error {
//使用零时间使标题相同
	var zeroTime time.Time
	tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(payload)), ModTime: zeroTime, AccessTime: zeroTime, ChangeTime: zeroTime})
	tw.Write(payload)

	return nil
}
