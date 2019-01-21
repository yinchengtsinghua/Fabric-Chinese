
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


package ccprovider

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

var ccproviderLogger = flogging.MustGetLogger("ccprovider")

var chaincodeInstallPath string

//ccpackage封装了一个chaincode包，可以
//原始链代码部署
//已签署的haincodedeploymentspec
//尝试将接口保持在最小
//用于可能泛化的接口。
type CCPackage interface {
//initfrombuffer从字节初始化包
	InitFromBuffer(buf []byte) (*ChaincodeData, error)

//initfromfs从文件系统获取链码（也包括原始字节）
	InitFromFS(ccname string, ccversion string) ([]byte, *pb.ChaincodeDeploymentSpec, error)

//putchaincodetofs将链码写入文件系统
	PutChaincodeToFS() error

//getdepspec从包中获取chaincodedeploymentspec
	GetDepSpec() *pb.ChaincodeDeploymentSpec

//GetDepspecBytes从包中获取序列化的chaincodeDeploymentsPec
	GetDepSpecBytes() []byte

//validatecc验证并返回对应于
//链表数据。验证基于来自chaincodedata的元数据
//此方法的一个用途是在启动前验证链码
	ValidateCC(ccdata *ChaincodeData) error

//getPackageObject以proto.message形式获取对象
	GetPackageObject() proto.Message

//getchaincodedata获取链码数据
	GetChaincodeData() *ChaincodeData

//GetID根据包计算获取链码的指纹
	GetId() []byte
}

//setchaincodespath设置此对等机的链码路径
func SetChaincodesPath(path string) {
	if s, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path, 0755); err != nil {
				panic(fmt.Sprintf("Could not create chaincodes install path: %s", err))
			}
		} else {
			panic(fmt.Sprintf("Could not stat chaincodes install path: %s", err))
		}
	} else if !s.IsDir() {
		panic(fmt.Errorf("chaincode path exists but not a dir: %s", path))
	}

	chaincodeInstallPath = path
}

func GetChaincodePackage(ccname string, ccversion string) ([]byte, error) {
	return GetChaincodePackageFromPath(ccname, ccversion, chaincodeInstallPath)
}

//isprintable由cdspackage和signedcddspackage验证用于
//在可打印的未格式化协议字段中检测垃圾字符串
//需要个字符。
func isPrintable(name string) bool {
	notASCII := func(r rune) bool {
		return !unicode.IsPrint(r)
	}
	return strings.IndexFunc(name, notASCII) == -1
}

//getchaincodepackage从文件系统返回chaincode包
func GetChaincodePackageFromPath(ccname string, ccversion string, ccInstallPath string) ([]byte, error) {
	path := fmt.Sprintf("%s/%s.%s", ccInstallPath, ccname, ccversion)
	var ccbytes []byte
	var err error
	if ccbytes, err = ioutil.ReadFile(path); err != nil {
		return nil, err
	}
	return ccbytes, nil
}

//chaincode package exists返回文件系统中是否存在chaincode包。
func ChaincodePackageExists(ccname string, ccversion string) (bool, error) {
	path := filepath.Join(chaincodeInstallPath, ccname+"."+ccversion)
	_, err := os.Stat(path)
	if err == nil {
//chaincodepackage已存在
		return true, nil
	}
	return false, err
}

type CCCacheSupport interface {
//get chaincode是缓存获取链码数据所必需的
	GetChaincode(ccname string, ccversion string) (CCPackage, error)
}

//ccinfofsimpl提供了在fs上CC的实现和对它的访问
//实施CCacheSupport
type CCInfoFSImpl struct{}

//getchaincodefromfs这是一个用于隐藏包实现的包装器。
//它使用chaincodeinstallpath调用getchaincodedefcompath
func (cifs *CCInfoFSImpl) GetChaincode(ccname string, ccversion string) (CCPackage, error) {
	return cifs.GetChaincodeFromPath(ccname, ccversion, chaincodeInstallPath)
}

func (cifs *CCInfoFSImpl) GetChaincodeCodePackage(ccname, ccversion string) ([]byte, error) {
	ccpack, err := cifs.GetChaincode(ccname, ccversion)
	if err != nil {
		return nil, err
	}
	return ccpack.GetDepSpec().Bytes(), nil
}

//getchaincodefrompath这是用于隐藏包实现的包装。
func (*CCInfoFSImpl) GetChaincodeFromPath(ccname string, ccversion string, path string) (CCPackage, error) {
//尝试原始光盘
	cccdspack := &CDSPackage{}
	_, _, err := cccdspack.InitFromPath(ccname, ccversion, path)
	if err != nil {
//尝试签署的光盘
		ccscdspack := &SignedCDSPackage{}
		_, _, err = ccscdspack.InitFromPath(ccname, ccversion, path)
		if err != nil {
			return nil, err
		}
		return ccscdspack, nil
	}
	return cccdspack, nil
}

//putchaincodeintofs是用于放置原始chaincodedeploymentspec的包装器
//使用cdspackage。这只在UTS中使用
func (*CCInfoFSImpl) PutChaincode(depSpec *pb.ChaincodeDeploymentSpec) (CCPackage, error) {
	buf, err := proto.Marshal(depSpec)
	if err != nil {
		return nil, err
	}
	cccdspack := &CDSPackage{}
	if _, err := cccdspack.InitFromBuffer(buf); err != nil {
		return nil, err
	}
	err = cccdspack.PutChaincodeToFS()
	if err != nil {
		return nil, err
	}

	return cccdspack, nil
}

//目录枚举器枚举目录
type DirEnumerator func(string) ([]os.FileInfo, error)

//chaincodeextractor从给定路径提取链码
type ChaincodeExtractor func(ccname string, ccversion string, path string) (CCPackage, error)

//ListInstalledChaincodes检索已安装的链代码
func (cifs *CCInfoFSImpl) ListInstalledChaincodes(dir string, ls DirEnumerator, ccFromPath ChaincodeExtractor) ([]chaincode.InstalledChaincode, error) {
	var chaincodes []chaincode.InstalledChaincode
	if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
		return nil, nil
	}
	files, err := ls(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading directory %s", dir)
	}

	for _, f := range files {
//跳过目录，我们只对普通文件感兴趣
		if f.IsDir() {
			continue
		}
//链码文件名的类型为“name.version”
//我们只对这个名字感兴趣。
//跳过不符合“a.b”文件命名约定的文件
		i := strings.Index(f.Name(), ".")
		if i == -1 {
			ccproviderLogger.Info("Skipping", f.Name(), "because of missing separator '.'")
			continue
		}
ccName := f.Name()[:i]      //分隔符前的所有内容
ccVersion := f.Name()[i+1:] //分隔符之后的所有内容

		ccPackage, err := ccFromPath(ccName, ccVersion, dir)
		if err != nil {
			ccproviderLogger.Warning("Failed obtaining chaincode information about", ccName, ccVersion, ":", err)
			return nil, errors.Wrapf(err, "failed obtaining information about %s, version %s", ccName, ccVersion)
		}

		chaincodes = append(chaincodes, chaincode.InstalledChaincode{
			Name:    ccName,
			Version: ccVersion,
			Id:      ccPackage.GetId(),
		})
	}
	ccproviderLogger.Debug("Returning", chaincodes)
	return chaincodes, nil
}

//ccinfo fstoragemgr是缓存使用的存储管理器，或者
//缓存被绕过
var ccInfoFSProvider = &CCInfoFSImpl{}

//ccinfo缓存是缓存实例本身
var ccInfoCache = NewCCInfoCache(ccInfoFSProvider)

//getchaincodefromfs从文件系统检索链码信息
func GetChaincodeFromFS(ccname string, ccversion string) (CCPackage, error) {
	return ccInfoFSProvider.GetChaincode(ccname, ccversion)
}

//putchaincodeintofs将链码信息放入文件系统（和
//如果启用了缓存，或者直接
//来自文件系统，否则
func PutChaincodeIntoFS(depSpec *pb.ChaincodeDeploymentSpec) error {
	_, err := ccInfoFSProvider.PutChaincode(depSpec)
	return err
}

//getchaincodedata从缓存中获取链码数据（如果有）
func GetChaincodeData(ccname string, ccversion string) (*ChaincodeData, error) {
	ccproviderLogger.Debugf("Getting chaincode data for <%s, %s> from cache", ccname, ccversion)
	return ccInfoCache.GetChaincodeData(ccname, ccversion)
}

func CheckInstantiationPolicy(name, version string, cdLedger *ChaincodeData) error {
	ccdata, err := GetChaincodeData(name, version)
	if err != nil {
		return err
	}

//我们有来自金融服务部的信息，检查一下政策
//与文件系统上的匹配（如果指定了一个）；
//此检查是必需的，因为此对等计算机的管理员
//可能为其指定了实例化策略
//例如，要确保链代码
//仅在某些通道上实例化；恶意
//另一方面，对等机可能创建了一个部署
//试图绕过实例化的事务
//政策。这张支票是为了确保
//发生，即对等方将拒绝调用
//这些条件下的链码。更多信息
//https://jira.hyperledger.org/browse/fab-3156
	if ccdata.InstantiationPolicy != nil {
		if !bytes.Equal(ccdata.InstantiationPolicy, cdLedger.InstantiationPolicy) {
			return fmt.Errorf("Instantiation policy mismatch for cc %s/%s", name, version)
		}
	}

	return nil
}

//getcpackage逐个尝试每个已知的包实现
//直到找到合适的包裹
func GetCCPackage(buf []byte) (CCPackage, error) {
//尝试原始光盘
	cds := &CDSPackage{}
	if ccdata, err := cds.InitFromBuffer(buf); err != nil {
		cds = nil
	} else {
		err = cds.ValidateCC(ccdata)
		if err != nil {
			cds = nil
		}
	}

//尝试签署的光盘
	scds := &SignedCDSPackage{}
	if ccdata, err := scds.InitFromBuffer(buf); err != nil {
		scds = nil
	} else {
		err = scds.ValidateCC(ccdata)
		if err != nil {
			scds = nil
		}
	}

	if cds != nil && scds != nil {
//两种方法都被成功地解开，这就是为什么
//希望Proto因输入错误而失败是致命的缺陷。
		ccproviderLogger.Errorf("Could not determine chaincode package type, guessing SignedCDS")
		return scds, nil
	}

	if cds != nil {
		return cds, nil
	}

	if scds != nil {
		return scds, nil
	}

	return nil, errors.New("could not unmarshal chaincode package to CDS or SignedCDS")
}

//GetInstalledChaincodes返回一个映射，其键是链码ID，并且
//值是具有
//已通过搜索在对等机上安装（但不一定实例化）
//链码安装路径
func GetInstalledChaincodes() (*pb.ChaincodeQueryResponse, error) {
	files, err := ioutil.ReadDir(chaincodeInstallPath)
	if err != nil {
		return nil, err
	}

//用于存储LSCC中所有链码条目信息的数组
	var ccInfoArray []*pb.ChaincodeInfo

	for _, file := range files {
//在第一个句点处拆分，因为链代码版本可以包含句点，而
//链码名称不能
		fileNameArray := strings.SplitN(file.Name(), ".", 2)

//检查长度是否如预期的那样为2，否则跳到下一个CC文件
		if len(fileNameArray) == 2 {
			ccname := fileNameArray[0]
			ccversion := fileNameArray[1]
			ccpack, err := GetChaincodeFromFS(ccname, ccversion)
			if err != nil {
//文件系统上的链码已被篡改或
//在chaincodes目录中找到了一个非chaincode文件
				ccproviderLogger.Errorf("Unreadable chaincode file found on filesystem: %s", file.Name())
				continue
			}

			cdsfs := ccpack.GetDepSpec()

			name := cdsfs.GetChaincodeSpec().GetChaincodeId().Name
			version := cdsfs.GetChaincodeSpec().GetChaincodeId().Version
			if name != ccname || version != ccversion {
//链码文件名中的链码名称/版本已修改
//由外部实体
				ccproviderLogger.Errorf("Chaincode file's name/version has been modified on the filesystem: %s", file.Name())
				continue
			}

			path := cdsfs.GetChaincodeSpec().ChaincodeId.Path
//因为这只是一个已安装的链代码，所以这些应该是空白的
			input, escc, vscc := "", "", ""

			ccInfo := &pb.ChaincodeInfo{Name: name, Version: version, Path: path, Input: input, Escc: escc, Vscc: vscc, Id: ccpack.GetId()}

//add this specific chaincode's metadata to the array of all chaincodes
			ccInfoArray = append(ccInfoArray, ccInfo)
		}
	}
//向查询中添加包含所有实例化链码信息的数组
//响应原
	cqr := &pb.ChaincodeQueryResponse{Chaincodes: ccInfoArray}

	return cqr, nil
}

//ccContext传递此参数，而不是字符串参数
type CCContext struct {
//名称链代码名称
	Name string

//用于构造链码图像和寄存器的版本
	Version string
}

//getcanonicalname返回与建议上下文关联的规范名称
func (cccid *CCContext) GetCanonicalName() string {
	return cccid.Name + ":" + cccid.Version
}

//--------chaincodedefinition-chaincodedata接口------
//chaincodedefinition描述了对等方决定是否认可的所有必要信息。
//针对特定链码的建议以及是否验证事务。
type ChaincodeDefinition interface {
//ccname返回这个chaincode的名称（它放在chaincoderegistry中的名称）。
	CCName() string

//哈希返回链码的哈希。
	Hash() []byte

//ccversion返回链代码的版本。
	CCVersion() string

//validation返回如何验证此链码的事务。
//返回的字符串是验证方法的名称（通常为“vscc”）。
//返回的字节是验证的参数（在
//“vscc”，这是封送的pb.vsccargs消息）。
	Validation() (string, []byte)

//认可返回如何认可此链码的建议。
//字符串返回的是认可方法（通常为“escc”）的名称。
	Endorsement() string
}

//--------chaincodedata存储在lscc上------

//chaincodedata定义要由proto序列化的链代码的数据结构
//类型通过在实例化后直接使用特定包来提供附加检查。
//数据类型为specifc（请参阅cdspackage和signedcddspackage）
type ChaincodeData struct {
//链码的名称
	Name string `protobuf:"bytes,1,opt,name=name"`

//链码的版本
	Version string `protobuf:"bytes,2,opt,name=version"`

//链代码实例的ESCC
	Escc string `protobuf:"bytes,3,opt,name=escc"`

//链代码实例的vscc
	Vscc string `protobuf:"bytes,4,opt,name=vscc"`

//链码实例的策略认可策略
	Policy []byte `protobuf:"bytes,5,opt,name=policy,proto3"`

//Data data specific to the package
	Data []byte `protobuf:"bytes,6,opt,name=data,proto3"`

//作为CC唯一指纹的链码的ID这不是
//目前在任何地方使用，但作为一个很好的捕眼器
	Id []byte `protobuf:"bytes,7,opt,name=id,proto3"`

//链代码的实例化策略
	InstantiationPolicy []byte `protobuf:"bytes,8,opt,name=instantiation_policy,proto3"`
}

//ccname返回这个chaincode的名称（它放在chaincoderegistry中的名称）。
func (cd *ChaincodeData) CCName() string {
	return cd.Name
}

//哈希返回链码的哈希。
func (cd *ChaincodeData) Hash() []byte {
	return cd.Id
}

//ccversion返回链代码的版本。
func (cd *ChaincodeData) CCVersion() string {
	return cd.Version
}

//validation返回如何验证此链码的事务。
//返回的字符串是验证方法的名称（通常为“vscc”）。
//返回的字节是验证的参数（在
//“vscc”，这是封送的pb.vsccargs消息）。
func (cd *ChaincodeData) Validation() (string, []byte) {
	return cd.Vscc, cd.Policy
}

//认可返回如何认可此链码的建议。
//字符串返回的是认可方法（通常为“escc”）的名称。
func (cd *ChaincodeData) Endorsement() string {
	return cd.Escc
}

//implement functions needed from proto.Message for proto's mar/unmarshal functions

//重置重置
func (cd *ChaincodeData) Reset() { *cd = ChaincodeData{} }

//字符串转换为字符串
func (cd *ChaincodeData) String() string { return proto.CompactTextString(cd) }

//原始信息的存在只是为了让原始人快乐
func (*ChaincodeData) ProtoMessage() {}

//chaincodecontainerinfo是启动/停止chaincode所需数据的另一个同义词。
type ChaincodeContainerInfo struct {
	Name        string
	Version     string
	Path        string
	Type        string
	CodePackage []byte

//containerType不是一个好名字，但“docker”和“system”是有效的类型
	ContainerType string
}

//事务参数是绑定到特定事务的参数。
//以及调用chaincode所需的。
type TransactionParams struct {
	TxID                 string
	ChannelID            string
	SignedProp           *pb.SignedProposal
	Proposal             *pb.Proposal
	TXSimulator          ledger.TxSimulator
	HistoryQueryExecutor ledger.HistoryQueryExecutor
	CollectionStore      privdata.CollectionStore
	IsInitTransaction    bool

//这是传递给链码的附加数据
	ProposalDecorations map[string][]byte
}

//ChaincodeProvider提供了一个抽象层，即
//用于不同的包与中的代码交互
//不导入chaincode包；更多方法
//如有必要，应在下面添加
type ChaincodeProvider interface {
//执行执行对链码和输入执行标准链码调用
	Execute(txParams *TransactionParams, cccid *CCContext, input *pb.ChaincodeInput) (*pb.Response, *pb.ChaincodeEvent, error)
//executeLegacyInit是执行链代码部署规范的特殊情况，
//它不在LSCC中，是旧生命周期所需的
	ExecuteLegacyInit(txParams *TransactionParams, cccid *CCContext, spec *pb.ChaincodeDeploymentSpec) (*pb.Response, *pb.ChaincodeEvent, error)
//停止停止链码给出
	Stop(ccci *ChaincodeContainerInfo) error
}

func DeploymentSpecToChaincodeContainerInfo(cds *pb.ChaincodeDeploymentSpec) *ChaincodeContainerInfo {
	return &ChaincodeContainerInfo{
		Name:          cds.Name(),
		Version:       cds.Version(),
		Path:          cds.Path(),
		Type:          cds.CCType(),
		ContainerType: cds.ExecEnv.String(),
	}
}
