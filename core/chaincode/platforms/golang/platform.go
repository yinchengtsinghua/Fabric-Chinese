
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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/ccmetadata"
	"github.com/hyperledger/fabric/core/chaincode/platforms/util"
	cutil "github.com/hyperledger/fabric/core/container/util"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

//Go中编写的链码平台
type Platform struct {
}

//
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func decodeUrl(path string) (string, error) {
	var urlLocation string
if strings.HasPrefix(path, "http://“”{
		urlLocation = path[7:]
} else if strings.HasPrefix(path, "https://“”{
		urlLocation = path[8:]
	} else {
		urlLocation = path
	}

	if len(urlLocation) < 2 {
		return "", errors.New("ChaincodeSpec's path/URL invalid")
	}

	if strings.LastIndex(urlLocation, "/") == len(urlLocation)-1 {
		urlLocation = urlLocation[:len(urlLocation)-1]
	}

	return urlLocation, nil
}

func getGopath() (string, error) {
	env, err := getGoEnv()
	if err != nil {
		return "", err
	}
//只取gopath的第一个元素
	splitGoPath := filepath.SplitList(env["GOPATH"])
	if len(splitGoPath) == 0 {
		return "", fmt.Errorf("invalid GOPATH environment variable value: %s", env["GOPATH"])
	}
	return splitGoPath[0], nil
}

func filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

//name返回此平台的名称
func (goPlatform *Platform) Name() string {
	return pb.ChaincodeSpec_GOLANG.String()
}

//validatespec验证go chaincodes
func (goPlatform *Platform) ValidatePath(rawPath string) error {
	path, err := url.Parse(rawPath)
	if err != nil || path == nil {
		return fmt.Errorf("invalid path: %s", err)
	}

//除了下载和测试，我们没有真正好的方法来检查远程URL的存在。
//不管怎样，我们稍后再做。但是我们可以并且应该测试本地路径的存在性。
//将空方案视为本地文件系统路径
	if path.Scheme == "" {
		gopath, err := getGopath()
		if err != nil {
			return err
		}
		pathToCheck := filepath.Join(gopath, "src", rawPath)
		exists, err := pathExists(pathToCheck)
		if err != nil {
			return fmt.Errorf("error validating chaincode path: %s", err)
		}
		if !exists {
			return fmt.Errorf("path to chaincode does not exist: %s", pathToCheck)
		}
	}
	return nil
}

func (goPlatform *Platform) ValidateCodePackage(code []byte) error {

	if len(code) == 0 {
//如果未包含代码包，则不验证任何内容
		return nil
	}

//FAB-2122：扫描提供的tarball，确保它只包含以下源代码
///SRC/$packagename。我们不想让像/pkg/shady.a这样的东西安装在
//$gopath在容器中。注意，我们现在看不到比这条路更深的地方
//知道现在只有go/cgo编译器才能执行。我们将删除源
//从系统编译后作为额外的保护层。
//
//应该注意的是，我们不能用这些技术抓住每一个威胁。因此，
//容器本身需要是最后一道防线，并配置为
//在执行约束时有弹性。但是，我们还是应该尽最大努力保持
//垃圾尽可能从系统中排出。
	re := regexp.MustCompile(`^(/)?(src|META-INF)/.*`)
	is := bytes.NewReader(code)
	gr, err := gzip.NewReader(is)
	if err != nil {
		return fmt.Errorf("failure opening codepackage gzip stream: %s", err)
	}
	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err != nil {
//只有当没有更多的条目需要扫描时，我们才能到达这里。
			break
		}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
		if !re.MatchString(header.Name) {
			return fmt.Errorf("illegal file detected in payload: \"%s\"", header.Name)
		}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//检查文件模式是否有意义
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//可接受标志：
//isreg==0100000
//-rw rw rw-=0666
//
//在此上下文中，任何其他内容都是可疑的，将被拒绝
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
		if header.Mode&^0100666 != 0 {
			return fmt.Errorf("illegal file mode detected for file %s: %o", header.Name, header.Mode)
		}
	}

	return nil
}

//供应商任何尚不在链码主包中的包
//或者被它出卖。我们取主包的名称和文件列表
//以前已经确定包含包的依赖项的。
//对于任何需要出售的内容，我们只需更新其路径规范。
//其他的一切，我们都没有动过。
func vendorDependencies(pkg string, files Sources) {

	exclusions := make([]string, 0)
	elements := strings.Split(pkg, "/")

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//首先，在我们的主包中的某个地方添加任何已经自动添加到
//“排除”。对于“foo/bar/baz”软件包，我们希望确保我们不是汽车供应商
//以下任一项：
//
//[“foo/vendor”，“foo/bar/vendor”，“foo/bar/baz/vendor”]
//
//因此，我们使用递归路径构建过程来形成这个列表
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	prev := filepath.Join("src")
	for _, element := range elements {
		curr := filepath.Join(prev, element)
		vendor := filepath.Join(curr, "vendor")
		exclusions = append(exclusions, vendor)
		prev = curr
	}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//接下来，将我们的主包添加到“排除项”列表中
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	exclusions = append(exclusions, filepath.Join("src", pkg))

	count := len(files)
	sem := make(chan bool, count)

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//现在启动一个并行进程，检查文件中的每个文件是否匹配
//任何排除的模式。任何匹配项都会被重命名，以便它们被自动贩卖。
//在src/$pkg/供应商下。
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	vendorPath := filepath.Join("src", pkg, "vendor")
	for i, file := range files {
		go func(i int, file SourceDescriptor) {
			excluded := false

			for _, exclusion := range exclusions {
				if strings.HasPrefix(file.Name, exclusion) == true {
					excluded = true
					break
				}
			}

			if excluded == false {
				origName := file.Name
				file.Name = strings.Replace(origName, "src", vendorPath, 1)
				logger.Debugf("vendoring %s -> %s", origName, file.Name)
			}

			files[i] = file
			sem <- true
		}(i, file)
	}

	for i := 0; i < count; i++ {
		<-sem
	}
}

//以.tar.gz格式为golang生成一系列src/$pkg条目的部署负载
func (goPlatform *Platform) GetDeploymentPayload(path string) ([]byte, error) {

	var err error

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//从HTTP或文件系统检索代码描述符
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	code, err := getCode(path)
	if err != nil {
		return nil, err
	}
	if code.Cleanup != nil {
		defer code.Cleanup()
	}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//为执行go list指令而更新我们的环境
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	env, err := getGoEnv()
	if err != nil {
		return nil, err
	}
	gopaths := splitEnvPaths(env["GOPATH"])
	goroots := splitEnvPaths(env["GOROOT"])
	gopaths[code.Gopath] = true
	env["GOPATH"] = flattenEnvPaths(gopaths)

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	imports, err := listImports(env, code.Pkg)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining imports: %s", err)
	}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//删除ccenv或系统提供的任何导入
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	var provided = map[string]bool{
		"github.com/hyperledger/fabric/core/chaincode/shim": true,
		"github.com/hyperledger/fabric/protos/peer":         true,
	}

//golang“伪包”-实际上不存在的包
	var pseudo = map[string]bool{
		"C": true,
	}

	imports = filter(imports, func(pkg string) bool {
//如果由ccenv提供，则删除
		if _, ok := provided[pkg]; ok == true {
			logger.Debugf("Discarding provided package %s", pkg)
			return false
		}

//删除伪包
		if _, ok := pseudo[pkg]; ok == true {
			logger.Debugf("Discarding pseudo-package %s", pkg)
			return false
		}

//如果由Goroot提供，则删除
		for goroot := range goroots {
			fqp := filepath.Join(goroot, "src", pkg)
			exists, err := pathExists(fqp)
			if err == nil && exists {
				logger.Debugf("Discarding GOROOT package %s", pkg)
				return false
			}
		}

//否则，我们保留它
		logger.Debugf("Accepting import: %s", pkg)
		return true
	})

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//
//筛选后保留的导入
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	deps := make(map[string]bool)

	for _, pkg := range imports {
//
//
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
		transitives, err := listDeps(env, pkg)
		if err != nil {
			return nil, fmt.Errorf("Error obtaining dependencies for %s: %s", pkg, err)
		}

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//将所有结果与顶级列表合并
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//合并直接依赖项…
		deps[pkg] = true

//…然后所有的过渡
		for _, dep := range transitives {
			deps[dep] = true
		}
	}

//剔除“”（如果存在）
	delete(deps, "")

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//从我们的一阶代码包中查找源代码…
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	fileMap, err := findSource(code.Gopath, code.Pkg)
	if err != nil {
		return nil, err
	}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//…然后是我们的代码包所具有的任何非系统依赖项的源代码
//从筛选列表
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	for dep := range deps {

		logger.Debugf("processing dep: %s", dep)

//每个依赖项都应该在gopath或goroot中。我们对包装不感兴趣
//任何系统包。然而，做出这一决定的官方方式（go list）
//每一个部门都太贵了，所以我们作弊。我们假设任何
//找不到必须是系统包，并以静默方式跳过它们
		for gopath := range gopaths {
			fqp := filepath.Join(gopath, "src", dep)
			exists, err := pathExists(fqp)

			logger.Debugf("checking: %s exists: %v", fqp, exists)

			if err == nil && exists {

//我们只有在找到它时才能到这里，所以继续加载它的代码
				files, err := findSource(gopath, dep)
				if err != nil {
					return nil, err
				}

//手动合并地图
				for _, file := range files {
					fileMap[file.Name] = file
				}
			}
		}
	}

	logger.Debugf("done")

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//重新处理到列表中以便于处理
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	files := make(Sources, 0)
	for _, file := range fileMap {
		files = append(files, file)
	}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//将非包依赖项重新映射到包/供应商
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	vendorDependencies(code.Pkg, files)

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	sort.Sort(files)

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//写下我们的焦油包裹
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	payload := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(payload)
	tw := tar.NewWriter(gw)

	for _, file := range files {

//file.path表示操作系统本地路径
//file.name表示tar包路径

//如果文件是元数据而不是golang代码，请删除前导的go代码路径，例如：
//原始文件名：src/github.com/hyperledger/fabric/examples/chaincode/go/marbles02/meta-inf/statedb/couchdb/indexes/indexowner.json
//更新的文件名：META-INF/statedb/couchdb/indexes/indexowner.json
		if file.IsMetadata {

			file.Name, err = filepath.Rel(filepath.Join("src", code.Pkg), file.Name)
			if err != nil {
				return nil, fmt.Errorf("This error was caused by bad packaging of the metadata.  The file [%s] is marked as MetaFile, however not located under META-INF   Error:[%s]", file.Name, err)
			}

//将tar位置（file.name）拆分为tar包目录和文件名
			_, filename := filepath.Split(file.Name)

//隐藏文件不支持作为元数据，因此忽略它们。
//用户通常不知道隐藏的文件在那里，可能无法删除它们，因此警告用户而不是出错。
			if strings.HasPrefix(filename, ".") {
				logger.Warningf("Ignoring hidden file in metadata directory: %s", file.Name)
				continue
			}

			fileBytes, err := ioutil.ReadFile(file.Path)
			if err != nil {
				return nil, err
			}

//验证元数据文件是否包含在tar中
//验证基于传递的带有路径的文件名
			err = ccmetadata.ValidateMetadataFile(file.Name, fileBytes)
			if err != nil {
				return nil, err
			}
		}

		err = cutil.WriteFileToPackage(file.Path, file.Name, tw)
		if err != nil {
			return nil, fmt.Errorf("Error writing %s to tar: %s", file.Name, err)
		}
	}

	err = tw.Close()
	if err == nil {
		err = gw.Close()
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create tar for chaincode")
	}

	return payload.Bytes(), nil
}

func (goPlatform *Platform) GenerateDockerfile() (string, error) {

	var buf []string

	buf = append(buf, "FROM "+cutil.GetDockerfileFromConfig("chaincode.golang.runtime"))
	buf = append(buf, "ADD binpackage.tar /usr/local/bin")

	dockerFileContents := strings.Join(buf, "\n")

	return dockerFileContents, nil
}

const staticLDFlagsOpts = "-ldflags \"-linkmode external -extldflags '-static'\""
const dynamicLDFlagsOpts = ""

func getLDFlagsOpts() string {
	if viper.GetBool("chaincode.golang.dynamicLink") {
		return dynamicLDFlagsOpts
	}
	return staticLDFlagsOpts
}

func (goPlatform *Platform) GenerateDockerBuild(path string, code []byte, tw *tar.Writer) error {
	pkgname, err := decodeUrl(path)
	if err != nil {
		return fmt.Errorf("could not decode url: %s", err)
	}

	ldflagsOpt := getLDFlagsOpts()
	logger.Infof("building chaincode with ldflagsOpt: '%s'", ldflagsOpt)

	codepackage := bytes.NewReader(code)
	binpackage := bytes.NewBuffer(nil)
	err = util.DockerBuild(util.DockerBuildOptions{
		Cmd:          fmt.Sprintf("GOPATH=/chaincode/input:$GOPATH go build  %s -o /chaincode/output/chaincode %s", ldflagsOpt, pkgname),
		InputStream:  codepackage,
		OutputStream: binpackage,
	})
	if err != nil {
		return err
	}

	return cutil.WriteBytesToPackage("binpackage.tar", binpackage.Bytes(), tw)
}

//GetMetadataProvider获取给定部署规范的元数据提供程序
func (goPlatform *Platform) GetMetadataProvider(code []byte) platforms.MetadataProvider {
	return &ccmetadata.TargzMetadataProvider{Code: code}
}
