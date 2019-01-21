
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


package golang

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	ccutil "github.com/hyperledger/fabric/core/chaincode/platforms/util"
)

var includeFileTypes = map[string]bool{
	".c":    true,
	".h":    true,
	".s":    true,
	".go":   true,
	".yaml": true,
	".json": true,
}

var logger = flogging.MustGetLogger("chaincode.platform.golang")

func getCodeFromFS(path string) (codegopath string, err error) {
	logger.Debugf("getCodeFromFS %s", path)
	gopath, err := getGopath()
	if err != nil {
		return "", err
	}

	tmppath := filepath.Join(gopath, "src", path)
	if err := ccutil.IsCodeExist(tmppath); err != nil {
		return "", fmt.Errorf("code does not exist %s", err)
	}

	return gopath, nil
}

type CodeDescriptor struct {
	Gopath, Pkg string
	Cleanup     func()
}

//collectchaincodefiles收集链码文件。如果路径是HTTP（S）URL，则
//先下载代码。
//
//注意：对于dev模式，用户手动构建和运行链码。提供的名称
//由用户等效为路径。
func getCode(path string) (*CodeDescriptor, error) {
	if path == "" {
		return nil, errors.New("Cannot collect files from empty chaincode path")
	}

//代码根将指向代码存在的目录
	var gopath string
	gopath, err := getCodeFromFS(path)
	if err != nil {
		return nil, fmt.Errorf("Error getting code %s", err)
	}

	return &CodeDescriptor{Gopath: gopath, Pkg: path, Cleanup: nil}, nil
}

type SourceDescriptor struct {
	Name, Path string
	IsMetadata bool
	Info       os.FileInfo
}
type SourceMap map[string]SourceDescriptor

type Sources []SourceDescriptor

func (s Sources) Len() int {
	return len(s)
}

func (s Sources) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Sources) Less(i, j int) bool {
	return strings.Compare(s[i].Name, s[j].Name) < 0
}

func findSource(gopath, pkg string) (SourceMap, error) {
	sources := make(SourceMap)
	tld := filepath.Join(gopath, "src", pkg)
	walkFn := func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info.IsDir() {

//允许将顶级chaincode目录导入chaincode包
			if path == tld {
				return nil
			}

//允许将META-INF元数据目录导入链码包tar。
//META-INF目录包含链码元数据工件，如statedb索引定义
			if isMetadataDir(path, tld) {
				logger.Debug("Files in META-INF directory will be included in code package tar:", path)
				return nil
			}

//不要将任何其他目录导入到chaincode包中
			logger.Debugf("skipping dir: %s", path)
			return filepath.SkipDir
		}

		ext := filepath.Ext(path)
//我们现在只需要“filetypes”源文件
		if _, ok := includeFileTypes[ext]; ok != true {
			return nil
		}

		name, err := filepath.Rel(gopath, path)
		if err != nil {
			return fmt.Errorf("error obtaining relative path for %s: %s", path, err)
		}

		sources[name] = SourceDescriptor{Name: name, Path: path, IsMetadata: isMetadataDir(path, tld), Info: info}

		return nil
	}

	if err := filepath.Walk(tld, walkFn); err != nil {
		return nil, fmt.Errorf("Error walking directory: %s", err)
	}

	return sources, nil
}

//ismetadatadir检查当前路径是否在chaincode目录根目录下的META-INF目录中。
func isMetadataDir(path, tld string) bool {
	return strings.HasPrefix(path, filepath.Join(tld, "META-INF"))
}
