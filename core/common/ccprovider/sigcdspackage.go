
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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/common/ccpackage"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//-----已签名的dcdsdata------

//SignedCDDSData是在CC实例化时存储在LSCC中的数据。
//对于已签名的dcdspackage。这需要为chaincodedata序列化
//因此，Protobuf格式
type SignedCDSData struct {
	CodeHash      []byte `protobuf:"bytes,1,opt,name=hash"`
	MetaDataHash  []byte `protobuf:"bytes,2,opt,name=metadatahash"`
	SignatureHash []byte `protobuf:"bytes,3,opt,name=signaturehash"`
}

//----实现proto的mar/unmashal函数所需的proto.message函数

//重置重置
func (data *SignedCDSData) Reset() { *data = SignedCDSData{} }

//字符串转换为字符串
func (data *SignedCDSData) String() string { return proto.CompactTextString(data) }

//原始信息的存在只是为了让原始人快乐
func (*SignedCDSData) ProtoMessage() {}

//等于数据等于其他
func (data *SignedCDSData) Equals(other *SignedCDSData) bool {
	return other != nil &&
		bytes.Equal(data.CodeHash, other.CodeHash) &&
		bytes.Equal(data.MetaDataHash, other.MetaDataHash) &&
		bytes.Equal(data.SignatureHash, other.SignatureHash)
}

//--------已签名的dcdspackage-------

//SignedCdsPackage封装了SignedChaincodeDeploymentSpec。
type SignedCDSPackage struct {
	buf      []byte
	depSpec  *pb.ChaincodeDeploymentSpec
	sDepSpec *pb.SignedChaincodeDeploymentSpec
	env      *common.Envelope
	data     *SignedCDSData
	datab    []byte
	id       []byte
}

//重置数据
func (ccpack *SignedCDSPackage) reset() {
	*ccpack = SignedCDSPackage{}
}

//GetID根据包计算获取链码的指纹
func (ccpack *SignedCDSPackage) GetId() []byte {
//这必须在创建包并初始化它之后进行
//如果这些步骤失败，则不应调用getid（）。
	if ccpack.id == nil {
		panic("GetId called on uninitialized package")
	}
	return ccpack.id
}

//getdepspec从包中获取chaincodedeploymentspec
func (ccpack *SignedCDSPackage) GetDepSpec() *pb.ChaincodeDeploymentSpec {
//这必须在创建包并初始化它之后进行
//如果这些步骤失败，则不应调用getDepspec（）。
	if ccpack.depSpec == nil {
		panic("GetDepSpec called on uninitialized package")
	}
	return ccpack.depSpec
}

//GetInstantiationPolicy从包中获取实例化策略
func (ccpack *SignedCDSPackage) GetInstantiationPolicy() []byte {
	if ccpack.sDepSpec == nil {
		panic("GetInstantiationPolicy called on uninitialized package")
	}
	return ccpack.sDepSpec.InstantiationPolicy
}

//GetDepspecBytes从包中获取序列化的chaincodeDeploymentsPec
func (ccpack *SignedCDSPackage) GetDepSpecBytes() []byte {
//这必须在创建包并初始化它之后进行
//如果这些步骤失败，则不应调用getDepspecBytes（）。
	if ccpack.sDepSpec == nil || ccpack.sDepSpec.ChaincodeDeploymentSpec == nil {
		panic("GetDepSpecBytes called on uninitialized package")
	}
	return ccpack.sDepSpec.ChaincodeDeploymentSpec
}

//getPackageObject以proto.message形式获取chaincodeDeploymentsPec
func (ccpack *SignedCDSPackage) GetPackageObject() proto.Message {
	return ccpack.env
}

//getchaincodedata获取链码数据
func (ccpack *SignedCDSPackage) GetChaincodeData() *ChaincodeData {
//这必须在创建包并初始化它之后进行
//如果这些步骤失败，则不应调用getchaincodedata（）。
	if ccpack.depSpec == nil || ccpack.datab == nil || ccpack.id == nil {
		panic("GetChaincodeData called on uninitialized package")
	}

	var instPolicy []byte
	if ccpack.sDepSpec != nil {
		instPolicy = ccpack.sDepSpec.InstantiationPolicy
	}

	return &ChaincodeData{
		Name:                ccpack.depSpec.ChaincodeSpec.ChaincodeId.Name,
		Version:             ccpack.depSpec.ChaincodeSpec.ChaincodeId.Version,
		Data:                ccpack.datab,
		Id:                  ccpack.id,
		InstantiationPolicy: instPolicy,
	}
}

func (ccpack *SignedCDSPackage) getCDSData(scds *pb.SignedChaincodeDeploymentSpec) ([]byte, []byte, *SignedCDSData, error) {
//检查是否有零参数。这是一个断言，getcdsddata
//从未对未通过/成功的包调用
//包初始化。
	if scds == nil {
		panic("nil cds")
	}

	cds := &pb.ChaincodeDeploymentSpec{}
	err := proto.Unmarshal(scds.ChaincodeDeploymentSpec, cds)
	if err != nil {
		return nil, nil, nil, err
	}

	if err = factory.InitFactories(nil); err != nil {
		return nil, nil, nil, fmt.Errorf("Internal error, BCCSP could not be initialized : %s", err)
	}

//获取哈希对象
	hash, err := factory.GetDefault().GetHash(&bccsp.SHAOpts{})
	if err != nil {
		return nil, nil, nil, err
	}

	scdsdata := &SignedCDSData{}

//获取代码哈希
	hash.Write(cds.CodePackage)
	scdsdata.CodeHash = hash.Sum(nil)

	hash.Reset()

//获取元数据哈希
	hash.Write([]byte(cds.ChaincodeSpec.ChaincodeId.Name))
	hash.Write([]byte(cds.ChaincodeSpec.ChaincodeId.Version))

	scdsdata.MetaDataHash = hash.Sum(nil)

	hash.Reset()

//获取签名哈希
	if scds.InstantiationPolicy == nil {
		return nil, nil, nil, fmt.Errorf("instantiation policy cannot be nil for chaincode (%s:%s)", cds.ChaincodeSpec.ChaincodeId.Name, cds.ChaincodeSpec.ChaincodeId.Version)
	}

	hash.Write(scds.InstantiationPolicy)
	for _, o := range scds.OwnerEndorsements {
		hash.Write(o.Endorser)
	}
	scdsdata.SignatureHash = hash.Sum(nil)

//马歇尔数据
	b, err := proto.Marshal(scdsdata)
	if err != nil {
		return nil, nil, nil, err
	}

	hash.Reset()

//计算身份证
	hash.Write(scdsdata.CodeHash)
	hash.Write(scdsdata.MetaDataHash)
	hash.Write(scdsdata.SignatureHash)

	id := hash.Sum(nil)

	return b, id, scdsdata, nil
}

//如果未找到链码或链码不是
//链码部署规范
func (ccpack *SignedCDSPackage) ValidateCC(ccdata *ChaincodeData) error {
	if ccpack.sDepSpec == nil {
		return fmt.Errorf("uninitialized package")
	}

	if ccpack.sDepSpec.ChaincodeDeploymentSpec == nil {
		return fmt.Errorf("signed chaincode deployment spec cannot be nil in a package")
	}

	if ccpack.depSpec == nil {
		return fmt.Errorf("chaincode deployment spec cannot be nil in a package")
	}

//这是一个黑客。当名称无效时，lscc需要特定的lscc错误，因此
//有自己的验证代码。由于导入周期的原因，我们不能使用该错误。
//不幸的是，我们还需要检查
//Protobuf很乐意反序列化垃圾，我们假设有一些路径
//成功的解组意味着一切都可以工作，但如果失败了，我们会尝试解组
//变成不同的东西。
	if !isPrintable(ccdata.Name) {
		return fmt.Errorf("invalid chaincode name: %q", ccdata.Name)
	}

	if ccdata.Name != ccpack.depSpec.ChaincodeSpec.ChaincodeId.Name || ccdata.Version != ccpack.depSpec.ChaincodeSpec.ChaincodeId.Version {
		return fmt.Errorf("invalid chaincode data %v (%v)", ccdata, ccpack.depSpec.ChaincodeSpec.ChaincodeId)
	}

	otherdata := &SignedCDSData{}
	err := proto.Unmarshal(ccdata.Data, otherdata)
	if err != nil {
		return err
	}

	if !ccpack.data.Equals(otherdata) {
		return fmt.Errorf("data mismatch")
	}

	return nil
}

//initfrombuffer设置缓冲区（如果有效）并返回chaincodedata
func (ccpack *SignedCDSPackage) InitFromBuffer(buf []byte) (*ChaincodeData, error) {
//如果重复使用ccpack
	ccpack.reset()

	env := &common.Envelope{}
	err := proto.Unmarshal(buf, env)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal envelope from bytes")
	}
	cHdr, sDepSpec, err := ccpackage.ExtractSignedCCDepSpec(env)
	if err != nil {
		return nil, err
	}

	if cHdr.Type != int32(common.HeaderType_CHAINCODE_PACKAGE) {
		return nil, fmt.Errorf("invalid type of envelope for chaincode package")
	}

	depSpec := &pb.ChaincodeDeploymentSpec{}
	err = proto.Unmarshal(sDepSpec.ChaincodeDeploymentSpec, depSpec)
	if err != nil {
		return nil, fmt.Errorf("error getting deployment spec")
	}

	databytes, id, data, err := ccpack.getCDSData(sDepSpec)
	if err != nil {
		return nil, err
	}

	ccpack.buf = buf
	ccpack.sDepSpec = sDepSpec
	ccpack.depSpec = depSpec
	ccpack.env = env
	ccpack.data = data
	ccpack.datab = databytes
	ccpack.id = id

	return ccpack.GetChaincodeData(), nil
}

//initfromfs从文件系统返回链码及其包
func (ccpack *SignedCDSPackage) InitFromFS(ccname string, ccversion string) ([]byte, *pb.ChaincodeDeploymentSpec, error) {
	return ccpack.InitFromPath(ccname, ccversion, chaincodeInstallPath)
}

//initfrompath从文件系统返回链码及其包
func (ccpack *SignedCDSPackage) InitFromPath(ccname string, ccversion string, path string) ([]byte, *pb.ChaincodeDeploymentSpec, error) {
//如果重复使用ccpack
	ccpack.reset()

	buf, err := GetChaincodePackageFromPath(ccname, ccversion, path)
	if err != nil {
		return nil, nil, err
	}

	if _, err = ccpack.InitFromBuffer(buf); err != nil {
		return nil, nil, err
	}

	return ccpack.buf, ccpack.depSpec, nil
}

//putchaincodetofs-将链码序列化到文件系统上的包
func (ccpack *SignedCDSPackage) PutChaincodeToFS() error {
	if ccpack.buf == nil {
		return fmt.Errorf("uninitialized package")
	}

	if ccpack.id == nil {
		return fmt.Errorf("id cannot be nil if buf is not nil")
	}

	if ccpack.sDepSpec == nil || ccpack.depSpec == nil {
		return fmt.Errorf("depspec cannot be nil if buf is not nil")
	}

	if ccpack.env == nil {
		return fmt.Errorf("env cannot be nil if buf and depspec are not nil")
	}

	if ccpack.data == nil {
		return fmt.Errorf("nil data")
	}

	if ccpack.datab == nil {
		return fmt.Errorf("nil data bytes")
	}

	ccname := ccpack.depSpec.ChaincodeSpec.ChaincodeId.Name
	ccversion := ccpack.depSpec.ChaincodeSpec.ChaincodeId.Version

//链码存在时返回错误
	path := fmt.Sprintf("%s/%s.%s", chaincodeInstallPath, ccname, ccversion)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("chaincode %s exists", path)
	}

	if err := ioutil.WriteFile(path, ccpack.buf, 0644); err != nil {
		return err
	}

	return nil
}
