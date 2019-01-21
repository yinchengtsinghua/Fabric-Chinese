
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//由Protoc Gen Go生成的代码。不要编辑。
//来源：common/collection.proto

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

//CollectionConfigPackage表示CollectionConfig的数组
//消息；需要额外的结构，因为重复的其中一个是
//被protobuf语法禁止
type CollectionConfigPackage struct {
	Config               []*CollectionConfig `protobuf:"bytes,1,rep,name=config,proto3" json:"config,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *CollectionConfigPackage) Reset()         { *m = CollectionConfigPackage{} }
func (m *CollectionConfigPackage) String() string { return proto.CompactTextString(m) }
func (*CollectionConfigPackage) ProtoMessage()    {}
func (*CollectionConfigPackage) Descriptor() ([]byte, []int) {
	return fileDescriptor_collection_12a2cf6632dc7d83, []int{0}
}
func (m *CollectionConfigPackage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CollectionConfigPackage.Unmarshal(m, b)
}
func (m *CollectionConfigPackage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CollectionConfigPackage.Marshal(b, m, deterministic)
}
func (dst *CollectionConfigPackage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CollectionConfigPackage.Merge(dst, src)
}
func (m *CollectionConfigPackage) XXX_Size() int {
	return xxx_messageInfo_CollectionConfigPackage.Size(m)
}
func (m *CollectionConfigPackage) XXX_DiscardUnknown() {
	xxx_messageInfo_CollectionConfigPackage.DiscardUnknown(m)
}

var xxx_messageInfo_CollectionConfigPackage proto.InternalMessageInfo

func (m *CollectionConfigPackage) GetConfig() []*CollectionConfig {
	if m != nil {
		return m.Config
	}
	return nil
}

//collectionconfig定义集合对象的配置；
//它当前包含一个静态类型。
//动态集合被延迟。
type CollectionConfig struct {
//有效分配给有效负载的类型：
//*集合配置\u静态集合配置
	Payload              isCollectionConfig_Payload `protobuf_oneof:"payload"`
	XXX_NoUnkeyedLiteral struct{}                   `json:"-"`
	XXX_unrecognized     []byte                     `json:"-"`
	XXX_sizecache        int32                      `json:"-"`
}

func (m *CollectionConfig) Reset()         { *m = CollectionConfig{} }
func (m *CollectionConfig) String() string { return proto.CompactTextString(m) }
func (*CollectionConfig) ProtoMessage()    {}
func (*CollectionConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_collection_12a2cf6632dc7d83, []int{1}
}
func (m *CollectionConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CollectionConfig.Unmarshal(m, b)
}
func (m *CollectionConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CollectionConfig.Marshal(b, m, deterministic)
}
func (dst *CollectionConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CollectionConfig.Merge(dst, src)
}
func (m *CollectionConfig) XXX_Size() int {
	return xxx_messageInfo_CollectionConfig.Size(m)
}
func (m *CollectionConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_CollectionConfig.DiscardUnknown(m)
}

var xxx_messageInfo_CollectionConfig proto.InternalMessageInfo

type isCollectionConfig_Payload interface {
	isCollectionConfig_Payload()
}

type CollectionConfig_StaticCollectionConfig struct {
	StaticCollectionConfig *StaticCollectionConfig `protobuf:"bytes,1,opt,name=static_collection_config,json=staticCollectionConfig,proto3,oneof"`
}

func (*CollectionConfig_StaticCollectionConfig) isCollectionConfig_Payload() {}

func (m *CollectionConfig) GetPayload() isCollectionConfig_Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *CollectionConfig) GetStaticCollectionConfig() *StaticCollectionConfig {
	if x, ok := m.GetPayload().(*CollectionConfig_StaticCollectionConfig); ok {
		return x.StaticCollectionConfig
	}
	return nil
}

//xxxoneoffuncs用于Proto包的内部使用。
func (*CollectionConfig) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _CollectionConfig_OneofMarshaler, _CollectionConfig_OneofUnmarshaler, _CollectionConfig_OneofSizer, []interface{}{
		(*CollectionConfig_StaticCollectionConfig)(nil),
	}
}

func _CollectionConfig_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*CollectionConfig)
//有效载荷
	switch x := m.Payload.(type) {
	case *CollectionConfig_StaticCollectionConfig:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.StaticCollectionConfig); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("CollectionConfig.Payload has unexpected type %T", x)
	}
	return nil
}

func _CollectionConfig_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*CollectionConfig)
	switch tag {
case 1: //payload.static_collection_配置
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(StaticCollectionConfig)
		err := b.DecodeMessage(msg)
		m.Payload = &CollectionConfig_StaticCollectionConfig{msg}
		return true, err
	default:
		return false, nil
	}
}

func _CollectionConfig_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*CollectionConfig)
//有效载荷
	switch x := m.Payload.(type) {
	case *CollectionConfig_StaticCollectionConfig:
		s := proto.Size(x.StaticCollectionConfig)
n += 1 //标签和电线
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

//StaticCollectionConfig构成
//静态集合对象。静态集合是
//在链码实例化时已知，不能更改。
//动态集合被延迟。
type StaticCollectionConfig struct {
//指示的链代码中集合的名称
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
//对配置块中驻留/管理的策略的引用
//定义哪些组织可以访问此集合的私有数据
	MemberOrgsPolicy *CollectionPolicyConfig `protobuf:"bytes,2,opt,name=member_orgs_policy,json=memberOrgsPolicy,proto3" json:"member_orgs_policy,omitempty"`
//将发送到的对等机私有数据的最小数目
//背书。如果至少传播到
//没有达到这一数量的同龄人。
	RequiredPeerCount int32 `protobuf:"varint,3,opt,name=required_peer_count,json=requiredPeerCount,proto3" json:"required_peer_count,omitempty"`
//将私人数据发送到的对等机的最大数目
//背书后。此数字必须大于所需的“对等”计数。
	MaximumPeerCount int32 `protobuf:"varint,4,opt,name=maximum_peer_count,json=maximumPeerCount,proto3" json:"maximum_peer_count,omitempty"`
//收集数据过期后的块数。
//例如，如果该值设置为10，则最后由块编号100修改的键
//将在111号块处清除。零值的处理方式与maxuint64相同。
	BlockToLive uint64 `protobuf:"varint,5,opt,name=block_to_live,json=blockToLive,proto3" json:"block_to_live,omitempty"`
//成员只读访问表示是否仅集合成员客户端
//可以读取私有数据（如果设置为true），甚至非成员也可以
//读取数据（如果设置为false，例如如果要实现更细粒度的
//链码中的访问逻辑）
	MemberOnlyRead       bool     `protobuf:"varint,6,opt,name=member_only_read,json=memberOnlyRead,proto3" json:"member_only_read,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StaticCollectionConfig) Reset()         { *m = StaticCollectionConfig{} }
func (m *StaticCollectionConfig) String() string { return proto.CompactTextString(m) }
func (*StaticCollectionConfig) ProtoMessage()    {}
func (*StaticCollectionConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_collection_12a2cf6632dc7d83, []int{2}
}
func (m *StaticCollectionConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StaticCollectionConfig.Unmarshal(m, b)
}
func (m *StaticCollectionConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StaticCollectionConfig.Marshal(b, m, deterministic)
}
func (dst *StaticCollectionConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StaticCollectionConfig.Merge(dst, src)
}
func (m *StaticCollectionConfig) XXX_Size() int {
	return xxx_messageInfo_StaticCollectionConfig.Size(m)
}
func (m *StaticCollectionConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_StaticCollectionConfig.DiscardUnknown(m)
}

var xxx_messageInfo_StaticCollectionConfig proto.InternalMessageInfo

func (m *StaticCollectionConfig) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *StaticCollectionConfig) GetMemberOrgsPolicy() *CollectionPolicyConfig {
	if m != nil {
		return m.MemberOrgsPolicy
	}
	return nil
}

func (m *StaticCollectionConfig) GetRequiredPeerCount() int32 {
	if m != nil {
		return m.RequiredPeerCount
	}
	return 0
}

func (m *StaticCollectionConfig) GetMaximumPeerCount() int32 {
	if m != nil {
		return m.MaximumPeerCount
	}
	return 0
}

func (m *StaticCollectionConfig) GetBlockToLive() uint64 {
	if m != nil {
		return m.BlockToLive
	}
	return 0
}

func (m *StaticCollectionConfig) GetMemberOnlyRead() bool {
	if m != nil {
		return m.MemberOnlyRead
	}
	return false
}

//收集策略配置。最初，配置只能
//包含签名策略。在未来，签名政策可能是
//更一般的政策。而不是包含实际的策略，
//配置将来可能包含对策略的字符串引用。
type CollectionPolicyConfig struct {
//有效分配给有效负载的类型：
//*集合策略配置\u签名策略
	Payload              isCollectionPolicyConfig_Payload `protobuf_oneof:"payload"`
	XXX_NoUnkeyedLiteral struct{}                         `json:"-"`
	XXX_unrecognized     []byte                           `json:"-"`
	XXX_sizecache        int32                            `json:"-"`
}

func (m *CollectionPolicyConfig) Reset()         { *m = CollectionPolicyConfig{} }
func (m *CollectionPolicyConfig) String() string { return proto.CompactTextString(m) }
func (*CollectionPolicyConfig) ProtoMessage()    {}
func (*CollectionPolicyConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_collection_12a2cf6632dc7d83, []int{3}
}
func (m *CollectionPolicyConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CollectionPolicyConfig.Unmarshal(m, b)
}
func (m *CollectionPolicyConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CollectionPolicyConfig.Marshal(b, m, deterministic)
}
func (dst *CollectionPolicyConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CollectionPolicyConfig.Merge(dst, src)
}
func (m *CollectionPolicyConfig) XXX_Size() int {
	return xxx_messageInfo_CollectionPolicyConfig.Size(m)
}
func (m *CollectionPolicyConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_CollectionPolicyConfig.DiscardUnknown(m)
}

var xxx_messageInfo_CollectionPolicyConfig proto.InternalMessageInfo

type isCollectionPolicyConfig_Payload interface {
	isCollectionPolicyConfig_Payload()
}

type CollectionPolicyConfig_SignaturePolicy struct {
	SignaturePolicy *SignaturePolicyEnvelope `protobuf:"bytes,1,opt,name=signature_policy,json=signaturePolicy,proto3,oneof"`
}

func (*CollectionPolicyConfig_SignaturePolicy) isCollectionPolicyConfig_Payload() {}

func (m *CollectionPolicyConfig) GetPayload() isCollectionPolicyConfig_Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *CollectionPolicyConfig) GetSignaturePolicy() *SignaturePolicyEnvelope {
	if x, ok := m.GetPayload().(*CollectionPolicyConfig_SignaturePolicy); ok {
		return x.SignaturePolicy
	}
	return nil
}

//xxxoneoffuncs用于Proto包的内部使用。
func (*CollectionPolicyConfig) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _CollectionPolicyConfig_OneofMarshaler, _CollectionPolicyConfig_OneofUnmarshaler, _CollectionPolicyConfig_OneofSizer, []interface{}{
		(*CollectionPolicyConfig_SignaturePolicy)(nil),
	}
}

func _CollectionPolicyConfig_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*CollectionPolicyConfig)
//有效载荷
	switch x := m.Payload.(type) {
	case *CollectionPolicyConfig_SignaturePolicy:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.SignaturePolicy); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("CollectionPolicyConfig.Payload has unexpected type %T", x)
	}
	return nil
}

func _CollectionPolicyConfig_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*CollectionPolicyConfig)
	switch tag {
case 1: //有效载荷.签名\政策
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(SignaturePolicyEnvelope)
		err := b.DecodeMessage(msg)
		m.Payload = &CollectionPolicyConfig_SignaturePolicy{msg}
		return true, err
	default:
		return false, nil
	}
}

func _CollectionPolicyConfig_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*CollectionPolicyConfig)
//有效载荷
	switch x := m.Payload.(type) {
	case *CollectionPolicyConfig_SignaturePolicy:
		s := proto.Size(x.SignaturePolicy)
n += 1 //标签和电线
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

//collectioncriteria定义与之对应的私有数据的元素
//对某一交易和收款
type CollectionCriteria struct {
	Channel              string   `protobuf:"bytes,1,opt,name=channel,proto3" json:"channel,omitempty"`
	TxId                 string   `protobuf:"bytes,2,opt,name=tx_id,json=txId,proto3" json:"tx_id,omitempty"`
	Collection           string   `protobuf:"bytes,3,opt,name=collection,proto3" json:"collection,omitempty"`
	Namespace            string   `protobuf:"bytes,4,opt,name=namespace,proto3" json:"namespace,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CollectionCriteria) Reset()         { *m = CollectionCriteria{} }
func (m *CollectionCriteria) String() string { return proto.CompactTextString(m) }
func (*CollectionCriteria) ProtoMessage()    {}
func (*CollectionCriteria) Descriptor() ([]byte, []int) {
	return fileDescriptor_collection_12a2cf6632dc7d83, []int{4}
}
func (m *CollectionCriteria) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CollectionCriteria.Unmarshal(m, b)
}
func (m *CollectionCriteria) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CollectionCriteria.Marshal(b, m, deterministic)
}
func (dst *CollectionCriteria) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CollectionCriteria.Merge(dst, src)
}
func (m *CollectionCriteria) XXX_Size() int {
	return xxx_messageInfo_CollectionCriteria.Size(m)
}
func (m *CollectionCriteria) XXX_DiscardUnknown() {
	xxx_messageInfo_CollectionCriteria.DiscardUnknown(m)
}

var xxx_messageInfo_CollectionCriteria proto.InternalMessageInfo

func (m *CollectionCriteria) GetChannel() string {
	if m != nil {
		return m.Channel
	}
	return ""
}

func (m *CollectionCriteria) GetTxId() string {
	if m != nil {
		return m.TxId
	}
	return ""
}

func (m *CollectionCriteria) GetCollection() string {
	if m != nil {
		return m.Collection
	}
	return ""
}

func (m *CollectionCriteria) GetNamespace() string {
	if m != nil {
		return m.Namespace
	}
	return ""
}

func init() {
	proto.RegisterType((*CollectionConfigPackage)(nil), "common.CollectionConfigPackage")
	proto.RegisterType((*CollectionConfig)(nil), "common.CollectionConfig")
	proto.RegisterType((*StaticCollectionConfig)(nil), "common.StaticCollectionConfig")
	proto.RegisterType((*CollectionPolicyConfig)(nil), "common.CollectionPolicyConfig")
	proto.RegisterType((*CollectionCriteria)(nil), "common.CollectionCriteria")
}

func init() { proto.RegisterFile("common/collection.proto", fileDescriptor_collection_12a2cf6632dc7d83) }

var fileDescriptor_collection_12a2cf6632dc7d83 = []byte{
//gzip文件描述符或协议的480字节
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x93, 0x51, 0x6b, 0xdb, 0x30,
	0x10, 0xc7, 0xeb, 0x36, 0x4d, 0xe7, 0x0b, 0xdb, 0x32, 0x95, 0xa5, 0x66, 0x8c, 0x2e, 0x84, 0x3d,
	0x18, 0x36, 0x9c, 0xd1, 0x7d, 0x83, 0x86, 0x41, 0xc7, 0x02, 0x0b, 0xea, 0x9e, 0xfa, 0x62, 0x14,
	0xf9, 0xea, 0x88, 0xca, 0x92, 0x2b, 0x2b, 0x21, 0x7e, 0xdc, 0x97, 0xd9, 0xe7, 0x1c, 0x91, 0xec,
	0x24, 0x0d, 0x79, 0xf3, 0xdd, 0xff, 0x77, 0xe7, 0xbb, 0xfb, 0xdb, 0x70, 0xc5, 0x75, 0x51, 0x68,
	0x35, 0xe6, 0x5a, 0x4a, 0xe4, 0x56, 0x68, 0x95, 0x94, 0x46, 0x5b, 0x4d, 0xba, 0x5e, 0xf8, 0xf0,
	0xbe, 0x01, 0x4a, 0x2d, 0x05, 0x17, 0x58, 0x79, 0x79, 0xf4, 0x0b, 0xae, 0x26, 0xdb, 0x92, 0x89,
	0x56, 0x8f, 0x22, 0x9f, 0x31, 0xfe, 0xc4, 0x72, 0x24, 0xdf, 0xa0, 0xcb, 0x5d, 0x22, 0x0a, 0x86,
	0x67, 0x71, 0xef, 0x26, 0x4a, 0x7c, 0x8b, 0xe4, 0xb0, 0x80, 0x36, 0xdc, 0xa8, 0x86, 0xfe, 0xa1,
	0x46, 0x1e, 0x20, 0xaa, 0x2c, 0xb3, 0x82, 0xa7, 0xbb, 0xd1, 0xd2, 0x6d, 0xdf, 0x20, 0xee, 0xdd,
	0x5c, 0xb7, 0x7d, 0xef, 0x1d, 0x77, 0xd8, 0xe1, 0xee, 0x84, 0x0e, 0xaa, 0xa3, 0xca, 0x6d, 0x08,
	0x17, 0x25, 0xab, 0xa5, 0x66, 0xd9, 0xe8, 0xdf, 0x29, 0x0c, 0x8e, 0xd7, 0x13, 0x02, 0x1d, 0xc5,
	0x0a, 0x74, 0x6f, 0x0b, 0xa9, 0x7b, 0x26, 0x53, 0x20, 0x05, 0x16, 0x73, 0x34, 0xa9, 0x36, 0x79,
	0x95, 0xba, 0xa3, 0xd4, 0xd1, 0xe9, 0xcb, 0x79, 0x76, 0x9d, 0x66, 0x4e, 0x6f, 0xb6, 0xed, 0xfb,
	0xca, 0xdf, 0x26, 0xaf, 0x7c, 0x9e, 0x24, 0x70, 0x69, 0xf0, 0x79, 0x29, 0x0c, 0x66, 0x69, 0x89,
	0x68, 0x52, 0xae, 0x97, 0xca, 0x46, 0x67, 0xc3, 0x20, 0x3e, 0xa7, 0xef, 0x5a, 0x69, 0x86, 0x68,
	0x26, 0x1b, 0x81, 0x7c, 0x05, 0x52, 0xb0, 0xb5, 0x28, 0x96, 0xc5, 0x3e, 0xde, 0x71, 0x78, 0xbf,
	0x51, 0x76, 0xf4, 0x08, 0x5e, 0xcf, 0xa5, 0xe6, 0x4f, 0xa9, 0xd5, 0xa9, 0x14, 0x2b, 0x8c, 0xce,
	0x87, 0x41, 0xdc, 0xa1, 0x3d, 0x97, 0xfc, 0xa3, 0xa7, 0x62, 0x85, 0x24, 0x86, 0x7e, 0xbb, 0x8f,
	0x92, 0x75, 0x6a, 0x90, 0x65, 0x51, 0x77, 0x18, 0xc4, 0xaf, 0xe8, 0x9b, 0x66, 0x5a, 0x25, 0x6b,
	0x8a, 0x2c, 0x1b, 0x3d, 0xc3, 0xe0, 0xf8, 0x5e, 0x64, 0x0a, 0xfd, 0x4a, 0xe4, 0x8a, 0xd9, 0xa5,
	0xc1, 0xf6, 0x22, 0xde, 0xa1, 0x4f, 0x5b, 0x87, 0x5a, 0xdd, 0x17, 0xfe, 0x50, 0x2b, 0x94, 0xba,
	0xc4, 0xbb, 0x13, 0xfa, 0xb6, 0x7a, 0x29, 0xed, 0x7b, 0xf3, 0x37, 0x00, 0xb2, 0xe7, 0x8a, 0x11,
	0x16, 0x8d, 0x60, 0x24, 0x82, 0x0b, 0xbe, 0x60, 0x4a, 0xa1, 0x6c, 0xac, 0x69, 0x43, 0x72, 0x09,
	0xe7, 0x76, 0x9d, 0x8a, 0xcc, 0x19, 0x12, 0xd2, 0x8e, 0x5d, 0xff, 0xcc, 0xc8, 0x35, 0xc0, 0xee,
	0x0b, 0x72, 0xb7, 0x0d, 0xe9, 0x5e, 0x86, 0x7c, 0x84, 0x70, 0x63, 0x6d, 0x55, 0x32, 0x8e, 0xee,
	0x96, 0x21, 0xdd, 0x25, 0x6e, 0xef, 0xe1, 0xb3, 0x36, 0x79, 0xb2, 0xa8, 0x4b, 0x34, 0x12, 0xb3,
	0x1c, 0x4d, 0xf2, 0xc8, 0xe6, 0x46, 0x70, 0xff, 0x1f, 0x54, 0xcd, 0x86, 0x0f, 0x5f, 0x72, 0x61,
	0x17, 0xcb, 0xf9, 0x26, 0x1c, 0xef, 0xc1, 0x63, 0x0f, 0x8f, 0x3d, 0x3c, 0xf6, 0xf0, 0xbc, 0xeb,
	0xc2, 0xef, 0xff, 0x03, 0x00, 0x00, 0xff, 0xff, 0xf0, 0x3b, 0x7c, 0x15, 0x7d, 0x03, 0x00, 0x00,
}
