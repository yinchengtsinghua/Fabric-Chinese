
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
	pb "github.com/hyperledger/fabric/protos/peer"
)

//-----CDSData------

//cdsdata是在CC实例化时存储在LSCC中的数据。
//用于CD包装。这需要为chaincodedata序列化
//因此，Protobuf格式
type CDSData struct {
//来自chaincodedeploymentspec的代码包的codehash哈希
	CodeHash []byte `protobuf:"bytes,1,opt,name=codehash,proto3"`

//来自chaincodedeploymentspec的名称和版本的metadatahash哈希
	MetaDataHash []byte `protobuf:"bytes,2,opt,name=metadatahash,proto3"`
}

//----实现proto的mar/unmashal函数所需的proto.message函数

//重置重置
func (data *CDSData) Reset() { *data = CDSData{} }

//字符串转换为字符串
func (data *CDSData) String() string { return proto.CompactTextString(data) }

//原始信息的存在只是为了让原始人快乐
func (*CDSData) ProtoMessage() {}

//等于数据等于其他
func (data *CDSData) Equals(other *CDSData) bool {
	return other != nil && bytes.Equal(data.CodeHash, other.CodeHash) && bytes.Equal(data.MetaDataHash, other.MetaDataHash)
}

//-------CDS包装-----------

//cdspackage封装了chaincodedeploymentspec。
type CDSPackage struct {
	buf     []byte
	depSpec *pb.ChaincodeDeploymentSpec
	data    *CDSData
	datab   []byte
	id      []byte
}

//重置数据
func (ccpack *CDSPackage) reset() {
	*ccpack = CDSPackage{}
}

//GetID根据包计算获取链码的指纹
func (ccpack *CDSPackage) GetId() []byte {
//这必须在创建包并初始化它之后进行
//如果这些步骤失败，则不应调用getid（）。
	if ccpack.id == nil {
		panic("GetId called on uninitialized package")
	}
	return ccpack.id
}

//getdepspec从包中获取chaincodedeploymentspec
func (ccpack *CDSPackage) GetDepSpec() *pb.ChaincodeDeploymentSpec {
//这必须在创建包并初始化它之后进行
//如果这些步骤失败，则不应调用getDepspec（）。
	if ccpack.depSpec == nil {
		panic("GetDepSpec called on uninitialized package")
	}
	return ccpack.depSpec
}

//GetDepspecBytes从包中获取序列化的chaincodeDeploymentsPec
func (ccpack *CDSPackage) GetDepSpecBytes() []byte {
//这必须在创建包并初始化它之后进行
//如果这些步骤失败，则不应调用getDepspecBytes（）。
	if ccpack.buf == nil {
		panic("GetDepSpecBytes called on uninitialized package")
	}
	return ccpack.buf
}

//getPackageObject以proto.message形式获取chaincodeDeploymentsPec
func (ccpack *CDSPackage) GetPackageObject() proto.Message {
	return ccpack.depSpec
}

//getchaincodedata获取链码数据
func (ccpack *CDSPackage) GetChaincodeData() *ChaincodeData {
//这必须在创建包并初始化它之后进行
//如果这些步骤失败，则不应调用getchaincodedata（）。
	if ccpack.depSpec == nil || ccpack.datab == nil || ccpack.id == nil {
		panic("GetChaincodeData called on uninitialized package")
	}
	return &ChaincodeData{Name: ccpack.depSpec.ChaincodeSpec.ChaincodeId.Name, Version: ccpack.depSpec.ChaincodeSpec.ChaincodeId.Version, Data: ccpack.datab, Id: ccpack.id}
}

func (ccpack *CDSPackage) getCDSData(cds *pb.ChaincodeDeploymentSpec) ([]byte, []byte, *CDSData, error) {
//检查是否有零参数。这是一个断言，getcdsddata
//从未对未通过/成功的包调用
//包初始化。
	if cds == nil {
		panic("nil cds")
	}

	b, err := proto.Marshal(cds)
	if err != nil {
		return nil, nil, nil, err
	}

	if err = factory.InitFactories(nil); err != nil {
		return nil, nil, nil, fmt.Errorf("Internal error, BCCSP could not be initialized : %s", err)
	}

//立即计算哈希
	hash, err := factory.GetDefault().GetHash(&bccsp.SHAOpts{})
	if err != nil {
		return nil, nil, nil, err
	}

	cdsdata := &CDSData{}

//代码哈希
	hash.Write(cds.CodePackage)
	cdsdata.CodeHash = hash.Sum(nil)

	hash.Reset()

//元数据散列
	hash.Write([]byte(cds.ChaincodeSpec.ChaincodeId.Name))
	hash.Write([]byte(cds.ChaincodeSpec.ChaincodeId.Version))

	cdsdata.MetaDataHash = hash.Sum(nil)

	b, err = proto.Marshal(cdsdata)
	if err != nil {
		return nil, nil, nil, err
	}

	hash.Reset()

//计算身份证
	hash.Write(cdsdata.CodeHash)
	hash.Write(cdsdata.MetaDataHash)

	id := hash.Sum(nil)

	return b, id, cdsdata, nil
}

//如果未找到链码或链码不是
//链码部署规范
func (ccpack *CDSPackage) ValidateCC(ccdata *ChaincodeData) error {
	if ccpack.depSpec == nil {
		return fmt.Errorf("uninitialized package")
	}

	if ccpack.data == nil {
		return fmt.Errorf("nil data")
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

	otherdata := &CDSData{}
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
func (ccpack *CDSPackage) InitFromBuffer(buf []byte) (*ChaincodeData, error) {
//incase ccpack is reused
	ccpack.reset()

	depSpec := &pb.ChaincodeDeploymentSpec{}
	err := proto.Unmarshal(buf, depSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment spec from bytes")
	}

	databytes, id, data, err := ccpack.getCDSData(depSpec)
	if err != nil {
		return nil, err
	}

	ccpack.buf = buf
	ccpack.depSpec = depSpec
	ccpack.data = data
	ccpack.datab = databytes
	ccpack.id = id

	return ccpack.GetChaincodeData(), nil
}

//initfromfs从文件系统返回链码及其包
func (ccpack *CDSPackage) InitFromPath(ccname string, ccversion string, path string) ([]byte, *pb.ChaincodeDeploymentSpec, error) {
//如果重复使用ccpack
	ccpack.reset()

	buf, err := GetChaincodePackageFromPath(ccname, ccversion, path)
	if err != nil {
		return nil, nil, err
	}

	ccdata, err := ccpack.InitFromBuffer(buf)
	if err != nil {
		return nil, nil, err
	}

	if err := ccpack.ValidateCC(ccdata); err != nil {
		return nil, nil, err
	}

	return ccpack.buf, ccpack.depSpec, nil
}

//initfromfs从文件系统返回链码及其包
func (ccpack *CDSPackage) InitFromFS(ccname string, ccversion string) ([]byte, *pb.ChaincodeDeploymentSpec, error) {
	return ccpack.InitFromPath(ccname, ccversion, chaincodeInstallPath)
}

//putchaincodetofs-将链码序列化到文件系统上的包
func (ccpack *CDSPackage) PutChaincodeToFS() error {
	if ccpack.buf == nil {
		return fmt.Errorf("uninitialized package")
	}

	if ccpack.id == nil {
		return fmt.Errorf("id cannot be nil if buf is not nil")
	}

	if ccpack.depSpec == nil {
		return fmt.Errorf("depspec cannot be nil if buf is not nil")
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
