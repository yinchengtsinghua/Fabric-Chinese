
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//由Protoc Gen Go生成的代码。不要编辑。
//来源：peer/resources.proto

package peer //导入“github.com/hyperledger/fabric/protos/peer”

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import common "github.com/hyperledger/fabric/protos/common"

//引用导入以禁止错误（如果未使用）。
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

//这是一个编译时断言，以确保生成的文件
//与正在编译的proto包兼容。
//此行的编译错误可能意味着您的
//需要更新proto包。
const _ = proto.ProtoPackageIsVersion2 //请升级proto包

//chaincodeidentifier标识一段chaincode。允许对等机接受
//此链码，安装的代码的哈希必须匹配，版本字符串也必须匹配
//包含在安装命令中。
type ChaincodeIdentifier struct {
	Hash                 []byte   `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	Version              string   `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ChaincodeIdentifier) Reset()         { *m = ChaincodeIdentifier{} }
func (m *ChaincodeIdentifier) String() string { return proto.CompactTextString(m) }
func (*ChaincodeIdentifier) ProtoMessage()    {}
func (*ChaincodeIdentifier) Descriptor() ([]byte, []int) {
	return fileDescriptor_resources_8467365451aa3557, []int{0}
}
func (m *ChaincodeIdentifier) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChaincodeIdentifier.Unmarshal(m, b)
}
func (m *ChaincodeIdentifier) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChaincodeIdentifier.Marshal(b, m, deterministic)
}
func (dst *ChaincodeIdentifier) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChaincodeIdentifier.Merge(dst, src)
}
func (m *ChaincodeIdentifier) XXX_Size() int {
	return xxx_messageInfo_ChaincodeIdentifier.Size(m)
}
func (m *ChaincodeIdentifier) XXX_DiscardUnknown() {
	xxx_messageInfo_ChaincodeIdentifier.DiscardUnknown(m)
}

var xxx_messageInfo_ChaincodeIdentifier proto.InternalMessageInfo

func (m *ChaincodeIdentifier) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *ChaincodeIdentifier) GetVersion() string {
	if m != nil {
		return m.Version
	}
	return ""
}

//chaincodevalidation指示对等端如何处理此chaincode的事务
//验证。目前唯一与织物一起提供的验证机制是标准的
//“vscc”验证机制。此内置验证方法使用认可策略
//它检查是否包含了足够数量的签名。“争论”
//字段对验证实现所需的任何参数进行编码。
type ChaincodeValidation struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Argument             []byte   `protobuf:"bytes,2,opt,name=argument,proto3" json:"argument,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ChaincodeValidation) Reset()         { *m = ChaincodeValidation{} }
func (m *ChaincodeValidation) String() string { return proto.CompactTextString(m) }
func (*ChaincodeValidation) ProtoMessage()    {}
func (*ChaincodeValidation) Descriptor() ([]byte, []int) {
	return fileDescriptor_resources_8467365451aa3557, []int{1}
}
func (m *ChaincodeValidation) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChaincodeValidation.Unmarshal(m, b)
}
func (m *ChaincodeValidation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChaincodeValidation.Marshal(b, m, deterministic)
}
func (dst *ChaincodeValidation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChaincodeValidation.Merge(dst, src)
}
func (m *ChaincodeValidation) XXX_Size() int {
	return xxx_messageInfo_ChaincodeValidation.Size(m)
}
func (m *ChaincodeValidation) XXX_DiscardUnknown() {
	xxx_messageInfo_ChaincodeValidation.DiscardUnknown(m)
}

var xxx_messageInfo_ChaincodeValidation proto.InternalMessageInfo

func (m *ChaincodeValidation) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *ChaincodeValidation) GetArgument() []byte {
	if m != nil {
		return m.Argument
	}
	return nil
}

//vsccargs作为参数通过
//chaincodevalidation消息的参数字段。
type VSCCArgs struct {
	EndorsementPolicyRef string   `protobuf:"bytes,1,opt,name=endorsement_policy_ref,json=endorsementPolicyRef,proto3" json:"endorsement_policy_ref,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *VSCCArgs) Reset()         { *m = VSCCArgs{} }
func (m *VSCCArgs) String() string { return proto.CompactTextString(m) }
func (*VSCCArgs) ProtoMessage()    {}
func (*VSCCArgs) Descriptor() ([]byte, []int) {
	return fileDescriptor_resources_8467365451aa3557, []int{2}
}
func (m *VSCCArgs) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VSCCArgs.Unmarshal(m, b)
}
func (m *VSCCArgs) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VSCCArgs.Marshal(b, m, deterministic)
}
func (dst *VSCCArgs) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VSCCArgs.Merge(dst, src)
}
func (m *VSCCArgs) XXX_Size() int {
	return xxx_messageInfo_VSCCArgs.Size(m)
}
func (m *VSCCArgs) XXX_DiscardUnknown() {
	xxx_messageInfo_VSCCArgs.DiscardUnknown(m)
}

var xxx_messageInfo_VSCCArgs proto.InternalMessageInfo

func (m *VSCCArgs) GetEndorsementPolicyRef() string {
	if m != nil {
		return m.EndorsementPolicyRef
	}
	return ""
}

//chaincode认可指示对等方如何认可事务。唯一
//如今与织物一起提供的认可机制是标准的“ESCC”机制。
//此代码简单地模拟生成一个rw集的建议，然后对结果进行签名。
//使用对等机的本地签名标识。
type ChaincodeEndorsement struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ChaincodeEndorsement) Reset()         { *m = ChaincodeEndorsement{} }
func (m *ChaincodeEndorsement) String() string { return proto.CompactTextString(m) }
func (*ChaincodeEndorsement) ProtoMessage()    {}
func (*ChaincodeEndorsement) Descriptor() ([]byte, []int) {
	return fileDescriptor_resources_8467365451aa3557, []int{3}
}
func (m *ChaincodeEndorsement) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChaincodeEndorsement.Unmarshal(m, b)
}
func (m *ChaincodeEndorsement) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChaincodeEndorsement.Marshal(b, m, deterministic)
}
func (dst *ChaincodeEndorsement) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChaincodeEndorsement.Merge(dst, src)
}
func (m *ChaincodeEndorsement) XXX_Size() int {
	return xxx_messageInfo_ChaincodeEndorsement.Size(m)
}
func (m *ChaincodeEndorsement) XXX_DiscardUnknown() {
	xxx_messageInfo_ChaincodeEndorsement.DiscardUnknown(m)
}

var xxx_messageInfo_ChaincodeEndorsement proto.InternalMessageInfo

func (m *ChaincodeEndorsement) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

//configtree封装通道和通道的资源配置。
//两种配置都表示为common.config
type ConfigTree struct {
	ChannelConfig        *common.Config `protobuf:"bytes,1,opt,name=channel_config,json=channelConfig,proto3" json:"channel_config,omitempty"`
	ResourcesConfig      *common.Config `protobuf:"bytes,2,opt,name=resources_config,json=resourcesConfig,proto3" json:"resources_config,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *ConfigTree) Reset()         { *m = ConfigTree{} }
func (m *ConfigTree) String() string { return proto.CompactTextString(m) }
func (*ConfigTree) ProtoMessage()    {}
func (*ConfigTree) Descriptor() ([]byte, []int) {
	return fileDescriptor_resources_8467365451aa3557, []int{4}
}
func (m *ConfigTree) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ConfigTree.Unmarshal(m, b)
}
func (m *ConfigTree) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ConfigTree.Marshal(b, m, deterministic)
}
func (dst *ConfigTree) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ConfigTree.Merge(dst, src)
}
func (m *ConfigTree) XXX_Size() int {
	return xxx_messageInfo_ConfigTree.Size(m)
}
func (m *ConfigTree) XXX_DiscardUnknown() {
	xxx_messageInfo_ConfigTree.DiscardUnknown(m)
}

var xxx_messageInfo_ConfigTree proto.InternalMessageInfo

func (m *ConfigTree) GetChannelConfig() *common.Config {
	if m != nil {
		return m.ChannelConfig
	}
	return nil
}

func (m *ConfigTree) GetResourcesConfig() *common.Config {
	if m != nil {
		return m.ResourcesConfig
	}
	return nil
}

func init() {
	proto.RegisterType((*ChaincodeIdentifier)(nil), "protos.ChaincodeIdentifier")
	proto.RegisterType((*ChaincodeValidation)(nil), "protos.ChaincodeValidation")
	proto.RegisterType((*VSCCArgs)(nil), "protos.VSCCArgs")
	proto.RegisterType((*ChaincodeEndorsement)(nil), "protos.ChaincodeEndorsement")
	proto.RegisterType((*ConfigTree)(nil), "protos.ConfigTree")
}

func init() { proto.RegisterFile("peer/resources.proto", fileDescriptor_resources_8467365451aa3557) }

var fileDescriptor_resources_8467365451aa3557 = []byte{
//gzip文件描述符或协议的326字节
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x91, 0xbf, 0x4f, 0xfb, 0x30,
	0x10, 0xc5, 0xd5, 0xea, 0xab, 0x2f, 0xed, 0x51, 0x0a, 0x32, 0x05, 0x55, 0x9d, 0xaa, 0x4c, 0x85,
	0x21, 0x91, 0xf8, 0x31, 0xb0, 0x01, 0x51, 0x07, 0x26, 0x50, 0x40, 0x1d, 0x58, 0x2a, 0xd7, 0xb9,
	0x24, 0x96, 0x12, 0x3b, 0x3a, 0xbb, 0x88, 0x2e, 0xfc, 0xed, 0x28, 0x76, 0x1b, 0x3a, 0x74, 0xf2,
	0x9d, 0xdf, 0xe7, 0x9e, 0x4e, 0xef, 0x60, 0x54, 0x23, 0x52, 0x44, 0x68, 0xf4, 0x9a, 0x04, 0x9a,
	0xb0, 0x26, 0x6d, 0x35, 0xfb, 0xef, 0x1e, 0x33, 0xb9, 0x10, 0xba, 0xaa, 0xb4, 0x8a, 0x84, 0x56,
	0x99, 0xcc, 0xed, 0xb7, 0x97, 0x83, 0x18, 0xce, 0xe3, 0x82, 0x4b, 0x25, 0x74, 0x8a, 0x2f, 0x29,
	0x2a, 0x2b, 0x33, 0x89, 0xc4, 0x18, 0xfc, 0x2b, 0xb8, 0x29, 0xc6, 0x9d, 0x69, 0x67, 0x36, 0x48,
	0x5c, 0xcd, 0xc6, 0x70, 0xf4, 0x85, 0x64, 0xa4, 0x56, 0xe3, 0xee, 0xb4, 0x33, 0xeb, 0x27, 0xbb,
	0x36, 0x98, 0xef, 0x99, 0x2c, 0x78, 0x29, 0x53, 0x6e, 0xa5, 0x56, 0x8d, 0x89, 0xe2, 0x15, 0x3a,
	0x93, 0x7e, 0xe2, 0x6a, 0x36, 0x81, 0x1e, 0xa7, 0x7c, 0x5d, 0xa1, 0xb2, 0xce, 0x65, 0x90, 0xb4,
	0x7d, 0xf0, 0x08, 0xbd, 0xc5, 0x7b, 0x1c, 0x3f, 0x51, 0x6e, 0xd8, 0x1d, 0x5c, 0xa2, 0x4a, 0x35,
	0x19, 0x6c, 0xa4, 0x65, 0xad, 0x4b, 0x29, 0x36, 0x4b, 0xc2, 0x6c, 0xeb, 0x36, 0xda, 0x53, 0xdf,
	0x9c, 0x98, 0x60, 0x16, 0x5c, 0xc3, 0xa8, 0x5d, 0x64, 0xfe, 0x07, 0x1c, 0xda, 0x24, 0xf8, 0x01,
	0x88, 0x5d, 0x16, 0x1f, 0x84, 0xc8, 0xee, 0x61, 0x28, 0x0a, 0xae, 0x14, 0x96, 0x4b, 0x9f, 0x90,
	0x63, 0x8f, 0x6f, 0x86, 0xa1, 0xcf, 0x2d, 0xf4, 0x6c, 0x72, 0xb2, 0xa5, 0x7c, 0xcb, 0x1e, 0xe0,
	0xac, 0x0d, 0x7c, 0x37, 0xd8, 0x3d, 0x38, 0x78, 0xda, 0x72, 0xfe, 0xe3, 0xf9, 0x15, 0x02, 0x4d,
	0x79, 0x58, 0x6c, 0x6a, 0xa4, 0x12, 0xd3, 0x1c, 0x29, 0xcc, 0xf8, 0x8a, 0xa4, 0xf0, 0x97, 0x31,
	0x61, 0x73, 0xce, 0xcf, 0xab, 0x5c, 0xda, 0x62, 0xbd, 0x6a, 0xcc, 0xa2, 0x3d, 0x34, 0xf2, 0x68,
	0xe4, 0xd1, 0xa8, 0x41, 0x57, 0xfe, 0xd2, 0xb7, 0xbf, 0x01, 0x00, 0x00, 0xff, 0xff, 0x5d, 0x4a,
	0x54, 0x47, 0x08, 0x02, 0x00, 0x00,
}
