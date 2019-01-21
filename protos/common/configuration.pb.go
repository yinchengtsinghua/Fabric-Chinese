
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//由Protoc Gen Go生成的代码。不要编辑。
//来源：common/configuration.proto

package common //导入“github.com/hyperledger/fabric/protos/common”

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

//引用导入以禁止错误（如果未使用）。
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

//这是一个编译时断言，以确保生成的文件
//与正在编译的proto包兼容。
//此行的编译错误可能意味着您的
//需要更新proto包。
const _ = proto.ProtoPackageIsVersion2 //请升级proto包

//hashingalgorithm作为类型链的配置项编码到配置事务中。
//关键字为“hashingalgorithm”，值为hashingalgorithm，作为封送的protobuf字节
type HashingAlgorithm struct {
//目前支持的算法有：shake256
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *HashingAlgorithm) Reset()         { *m = HashingAlgorithm{} }
func (m *HashingAlgorithm) String() string { return proto.CompactTextString(m) }
func (*HashingAlgorithm) ProtoMessage()    {}
func (*HashingAlgorithm) Descriptor() ([]byte, []int) {
	return fileDescriptor_configuration_c60fbe5ebb3de531, []int{0}
}
func (m *HashingAlgorithm) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_HashingAlgorithm.Unmarshal(m, b)
}
func (m *HashingAlgorithm) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_HashingAlgorithm.Marshal(b, m, deterministic)
}
func (dst *HashingAlgorithm) XXX_Merge(src proto.Message) {
	xxx_messageInfo_HashingAlgorithm.Merge(dst, src)
}
func (m *HashingAlgorithm) XXX_Size() int {
	return xxx_messageInfo_HashingAlgorithm.Size(m)
}
func (m *HashingAlgorithm) XXX_DiscardUnknown() {
	xxx_messageInfo_HashingAlgorithm.DiscardUnknown(m)
}

var xxx_messageInfo_HashingAlgorithm proto.InternalMessageInfo

func (m *HashingAlgorithm) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

//blockdatahashingstructure作为的配置项编码到配置事务中。
//类型链，键为“blockdatahashingstructure”，值为hashingalgorithm，作为封送的protobuf字节
type BlockDataHashingStructure struct {
//width指定计算blockdatahash时要使用的Merkle树的宽度
//为了复制平面散列，请将此宽度设置为max_int32
	Width                uint32   `protobuf:"varint,1,opt,name=width,proto3" json:"width,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BlockDataHashingStructure) Reset()         { *m = BlockDataHashingStructure{} }
func (m *BlockDataHashingStructure) String() string { return proto.CompactTextString(m) }
func (*BlockDataHashingStructure) ProtoMessage()    {}
func (*BlockDataHashingStructure) Descriptor() ([]byte, []int) {
	return fileDescriptor_configuration_c60fbe5ebb3de531, []int{1}
}
func (m *BlockDataHashingStructure) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BlockDataHashingStructure.Unmarshal(m, b)
}
func (m *BlockDataHashingStructure) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BlockDataHashingStructure.Marshal(b, m, deterministic)
}
func (dst *BlockDataHashingStructure) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BlockDataHashingStructure.Merge(dst, src)
}
func (m *BlockDataHashingStructure) XXX_Size() int {
	return xxx_messageInfo_BlockDataHashingStructure.Size(m)
}
func (m *BlockDataHashingStructure) XXX_DiscardUnknown() {
	xxx_messageInfo_BlockDataHashingStructure.DiscardUnknown(m)
}

var xxx_messageInfo_BlockDataHashingStructure proto.InternalMessageInfo

func (m *BlockDataHashingStructure) GetWidth() uint32 {
	if m != nil {
		return m.Width
	}
	return 0
}

//orderAddresses作为类型链的配置项编码到配置事务中。
//键为“orderAddresses”，值为orderAddresses，作为封送的protobuf字节
type OrdererAddresses struct {
	Addresses            []string `protobuf:"bytes,1,rep,name=addresses,proto3" json:"addresses,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OrdererAddresses) Reset()         { *m = OrdererAddresses{} }
func (m *OrdererAddresses) String() string { return proto.CompactTextString(m) }
func (*OrdererAddresses) ProtoMessage()    {}
func (*OrdererAddresses) Descriptor() ([]byte, []int) {
	return fileDescriptor_configuration_c60fbe5ebb3de531, []int{2}
}
func (m *OrdererAddresses) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrdererAddresses.Unmarshal(m, b)
}
func (m *OrdererAddresses) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrdererAddresses.Marshal(b, m, deterministic)
}
func (dst *OrdererAddresses) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrdererAddresses.Merge(dst, src)
}
func (m *OrdererAddresses) XXX_Size() int {
	return xxx_messageInfo_OrdererAddresses.Size(m)
}
func (m *OrdererAddresses) XXX_DiscardUnknown() {
	xxx_messageInfo_OrdererAddresses.DiscardUnknown(m)
}

var xxx_messageInfo_OrdererAddresses proto.InternalMessageInfo

func (m *OrdererAddresses) GetAddresses() []string {
	if m != nil {
		return m.Addresses
	}
	return nil
}

//联合体代表创建渠道的联合体上下文。
type Consortium struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Consortium) Reset()         { *m = Consortium{} }
func (m *Consortium) String() string { return proto.CompactTextString(m) }
func (*Consortium) ProtoMessage()    {}
func (*Consortium) Descriptor() ([]byte, []int) {
	return fileDescriptor_configuration_c60fbe5ebb3de531, []int{3}
}
func (m *Consortium) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Consortium.Unmarshal(m, b)
}
func (m *Consortium) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Consortium.Marshal(b, m, deterministic)
}
func (dst *Consortium) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Consortium.Merge(dst, src)
}
func (m *Consortium) XXX_Size() int {
	return xxx_messageInfo_Consortium.Size(m)
}
func (m *Consortium) XXX_DiscardUnknown() {
	xxx_messageInfo_Consortium.DiscardUnknown(m)
}

var xxx_messageInfo_Consortium proto.InternalMessageInfo

func (m *Consortium) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

//功能消息定义特定二进制文件必须实现的功能
//使二进制文件能够安全地参与到通道中。能力
//消息在/channel级别、/channel/application级别和
///channel/order级别。
//
///channel级别的功能定义了订购方和对等方的功能
//二进制文件必须满足。这些功能可能是一种新的MSP类型，
//或新的策略类型。
//
///channel/order级别的功能定义了必须支持的功能
//但这与对等方的行为无关。例如
//如果医嘱者更改了它如何构造新通道的逻辑，则只有所有医嘱者
//必须同意新的逻辑。同龄人不需要意识到这种变化，因为
//它们只在通道构建之后才与通道交互。
//
//最后，/channel/application级别的功能定义了对等端
//二进制文件必须满足，但与订购方无关。例如，如果
//Peer添加了一个新的utxo事务类型，或者更改了链码生命周期需求，
//所有的同行都必须同意新的逻辑。但是，订购者从不检查事务
//这是深刻的，因此不需要意识到变化。
//
//这些消息中定义的功能字符串通常对应于发布
//二进制版本（例如“v1.1”），最初用作
//升级网络，从一组逻辑切换到新的逻辑。
//
//尽管对于v1.1版，订购方必须在
//由于/channel、/channel/order之间的拆分，网络，继续前进
//和/渠道/应用程序功能。订购方和
//独立升级的应用程序网络（除了
//在/channel级别定义的新功能）。
type Capabilities struct {
	Capabilities         map[string]*Capability `protobuf:"bytes,1,rep,name=capabilities,proto3" json:"capabilities,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}               `json:"-"`
	XXX_unrecognized     []byte                 `json:"-"`
	XXX_sizecache        int32                  `json:"-"`
}

func (m *Capabilities) Reset()         { *m = Capabilities{} }
func (m *Capabilities) String() string { return proto.CompactTextString(m) }
func (*Capabilities) ProtoMessage()    {}
func (*Capabilities) Descriptor() ([]byte, []int) {
	return fileDescriptor_configuration_c60fbe5ebb3de531, []int{4}
}
func (m *Capabilities) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Capabilities.Unmarshal(m, b)
}
func (m *Capabilities) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Capabilities.Marshal(b, m, deterministic)
}
func (dst *Capabilities) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Capabilities.Merge(dst, src)
}
func (m *Capabilities) XXX_Size() int {
	return xxx_messageInfo_Capabilities.Size(m)
}
func (m *Capabilities) XXX_DiscardUnknown() {
	xxx_messageInfo_Capabilities.DiscardUnknown(m)
}

var xxx_messageInfo_Capabilities proto.InternalMessageInfo

func (m *Capabilities) GetCapabilities() map[string]*Capability {
	if m != nil {
		return m.Capabilities
	}
	return nil
}

//目前，功能是一条空消息。它被定义为一个原型
//消息而不是常量，以便我们可以用其他字段扩展功能
//如果将来需要的话。目前，一种能力正在
//功能映射要求支持该功能。
type Capability struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Capability) Reset()         { *m = Capability{} }
func (m *Capability) String() string { return proto.CompactTextString(m) }
func (*Capability) ProtoMessage()    {}
func (*Capability) Descriptor() ([]byte, []int) {
	return fileDescriptor_configuration_c60fbe5ebb3de531, []int{5}
}
func (m *Capability) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Capability.Unmarshal(m, b)
}
func (m *Capability) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Capability.Marshal(b, m, deterministic)
}
func (dst *Capability) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Capability.Merge(dst, src)
}
func (m *Capability) XXX_Size() int {
	return xxx_messageInfo_Capability.Size(m)
}
func (m *Capability) XXX_DiscardUnknown() {
	xxx_messageInfo_Capability.DiscardUnknown(m)
}

var xxx_messageInfo_Capability proto.InternalMessageInfo

func init() {
	proto.RegisterType((*HashingAlgorithm)(nil), "common.HashingAlgorithm")
	proto.RegisterType((*BlockDataHashingStructure)(nil), "common.BlockDataHashingStructure")
	proto.RegisterType((*OrdererAddresses)(nil), "common.OrdererAddresses")
	proto.RegisterType((*Consortium)(nil), "common.Consortium")
	proto.RegisterType((*Capabilities)(nil), "common.Capabilities")
	proto.RegisterMapType((map[string]*Capability)(nil), "common.Capabilities.CapabilitiesEntry")
	proto.RegisterType((*Capability)(nil), "common.Capability")
}

func init() {
	proto.RegisterFile("common/configuration.proto", fileDescriptor_configuration_c60fbe5ebb3de531)
}

var fileDescriptor_configuration_c60fbe5ebb3de531 = []byte{
//gzip文件描述符或协议的314字节
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x91, 0x41, 0x6b, 0xf2, 0x40,
	0x10, 0x86, 0x89, 0x7e, 0x0a, 0x8e, 0x7e, 0x60, 0x97, 0x1e, 0xac, 0xf4, 0x10, 0x42, 0x91, 0x40,
	0x21, 0x69, 0xed, 0xa5, 0xf4, 0xa6, 0xb6, 0x50, 0x7a, 0x29, 0xc4, 0x5b, 0x6f, 0x9b, 0x64, 0x4c,
	0x16, 0x93, 0x5d, 0x99, 0xdd, 0xb4, 0xe4, 0x57, 0xf5, 0x2f, 0x16, 0xb3, 0x16, 0x23, 0xf6, 0x36,
	0xcf, 0xce, 0xf3, 0xce, 0xce, 0xb2, 0x30, 0x4d, 0x54, 0x59, 0x2a, 0x19, 0x26, 0x4a, 0x6e, 0x44,
	0x56, 0x11, 0x37, 0x42, 0xc9, 0x60, 0x47, 0xca, 0x28, 0xd6, 0xb7, 0x3d, 0x6f, 0x06, 0xe3, 0x57,
	0xae, 0x73, 0x21, 0xb3, 0x45, 0x91, 0x29, 0x12, 0x26, 0x2f, 0x19, 0x83, 0x7f, 0x92, 0x97, 0x38,
	0x71, 0x5c, 0xc7, 0x1f, 0x44, 0x4d, 0xed, 0xdd, 0xc3, 0xd5, 0xb2, 0x50, 0xc9, 0xf6, 0x99, 0x1b,
	0x7e, 0x08, 0xac, 0x0d, 0x55, 0x89, 0xa9, 0x08, 0xd9, 0x25, 0xf4, 0xbe, 0x44, 0x6a, 0xf2, 0x26,
	0xf1, 0x3f, 0xb2, 0xe0, 0xdd, 0xc1, 0xf8, 0x9d, 0x52, 0x24, 0xa4, 0x45, 0x9a, 0x12, 0x6a, 0x8d,
	0x9a, 0x5d, 0xc3, 0x80, 0xff, 0xc2, 0xc4, 0x71, 0xbb, 0xfe, 0x20, 0x3a, 0x1e, 0x78, 0x2e, 0xc0,
	0x4a, 0x49, 0xad, 0xc8, 0x88, 0xea, 0xef, 0x35, 0xbe, 0x1d, 0x18, 0xad, 0xf8, 0x8e, 0xc7, 0xa2,
	0x10, 0x46, 0xa0, 0x66, 0x6f, 0x30, 0x4a, 0x5a, 0xdc, 0xcc, 0x1c, 0xce, 0x67, 0x81, 0x7d, 0x5e,
	0xd0, 0x76, 0x4f, 0xe0, 0x45, 0x1a, 0xaa, 0xa3, 0x93, 0xec, 0x74, 0x0d, 0x17, 0x67, 0x0a, 0x1b,
	0x43, 0x77, 0x8b, 0xf5, 0x61, 0x89, 0x7d, 0xc9, 0x7c, 0xe8, 0x7d, 0xf2, 0xa2, 0xc2, 0x49, 0xc7,
	0x75, 0xfc, 0xe1, 0x9c, 0x9d, 0xdd, 0x55, 0x47, 0x56, 0x78, 0xea, 0x3c, 0x3a, 0xde, 0x08, 0xe0,
	0xd8, 0x58, 0xae, 0xe1, 0x46, 0x51, 0x16, 0xe4, 0xf5, 0x0e, 0xa9, 0xc0, 0x34, 0x43, 0x0a, 0x36,
	0x3c, 0x26, 0x91, 0xd8, 0x6f, 0xd1, 0x87, 0x59, 0x1f, 0xb7, 0x99, 0x30, 0x79, 0x15, 0xef, 0x31,
	0x6c, 0xc9, 0xa1, 0x95, 0x43, 0x2b, 0x87, 0x56, 0x8e, 0xfb, 0x0d, 0x3e, 0xfc, 0x04, 0x00, 0x00,
	0xff, 0xff, 0xd6, 0x7e, 0xb4, 0x89, 0xf0, 0x01, 0x00, 0x00,
}
