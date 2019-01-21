
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//由Protoc Gen Go生成的代码。不要编辑。
//来源：order/ab.proto

package orderer //导入“github.com/hyperledger/fabric/protos/order”

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import common "github.com/hyperledger/fabric/protos/common"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

//引用导入以禁止错误（如果未使用）。
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

//这是一个编译时断言，以确保生成的文件
//与正在编译的proto包兼容。
//此行的编译错误可能意味着您的
//需要更新proto包。
const _ = proto.ProtoPackageIsVersion2 //请升级proto包

type SeekInfo_SeekBehavior int32

const (
	SeekInfo_BLOCK_UNTIL_READY SeekInfo_SeekBehavior = 0
	SeekInfo_FAIL_IF_NOT_READY SeekInfo_SeekBehavior = 1
)

var SeekInfo_SeekBehavior_name = map[int32]string{
	0: "BLOCK_UNTIL_READY",
	1: "FAIL_IF_NOT_READY",
}
var SeekInfo_SeekBehavior_value = map[string]int32{
	"BLOCK_UNTIL_READY": 0,
	"FAIL_IF_NOT_READY": 1,
}

func (x SeekInfo_SeekBehavior) String() string {
	return proto.EnumName(SeekInfo_SeekBehavior_name, int32(x))
}
func (SeekInfo_SeekBehavior) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_ab_86effae0ebc2388c, []int{5, 0}
}

type BroadcastResponse struct {
//状态代码，可用于对成功/失败进行编程响应
	Status common.Status `protobuf:"varint,1,opt,name=status,proto3,enum=common.Status" json:"status,omitempty"`
//可能包含有关返回状态的其他信息的信息字符串
	Info                 string   `protobuf:"bytes,2,opt,name=info,proto3" json:"info,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BroadcastResponse) Reset()         { *m = BroadcastResponse{} }
func (m *BroadcastResponse) String() string { return proto.CompactTextString(m) }
func (*BroadcastResponse) ProtoMessage()    {}
func (*BroadcastResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_ab_86effae0ebc2388c, []int{0}
}
func (m *BroadcastResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BroadcastResponse.Unmarshal(m, b)
}
func (m *BroadcastResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BroadcastResponse.Marshal(b, m, deterministic)
}
func (dst *BroadcastResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BroadcastResponse.Merge(dst, src)
}
func (m *BroadcastResponse) XXX_Size() int {
	return xxx_messageInfo_BroadcastResponse.Size(m)
}
func (m *BroadcastResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_BroadcastResponse.DiscardUnknown(m)
}

var xxx_messageInfo_BroadcastResponse proto.InternalMessageInfo

func (m *BroadcastResponse) GetStatus() common.Status {
	if m != nil {
		return m.Status
	}
	return common.Status_UNKNOWN
}

func (m *BroadcastResponse) GetInfo() string {
	if m != nil {
		return m.Info
	}
	return ""
}

type SeekNewest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SeekNewest) Reset()         { *m = SeekNewest{} }
func (m *SeekNewest) String() string { return proto.CompactTextString(m) }
func (*SeekNewest) ProtoMessage()    {}
func (*SeekNewest) Descriptor() ([]byte, []int) {
	return fileDescriptor_ab_86effae0ebc2388c, []int{1}
}
func (m *SeekNewest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SeekNewest.Unmarshal(m, b)
}
func (m *SeekNewest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SeekNewest.Marshal(b, m, deterministic)
}
func (dst *SeekNewest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SeekNewest.Merge(dst, src)
}
func (m *SeekNewest) XXX_Size() int {
	return xxx_messageInfo_SeekNewest.Size(m)
}
func (m *SeekNewest) XXX_DiscardUnknown() {
	xxx_messageInfo_SeekNewest.DiscardUnknown(m)
}

var xxx_messageInfo_SeekNewest proto.InternalMessageInfo

type SeekOldest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SeekOldest) Reset()         { *m = SeekOldest{} }
func (m *SeekOldest) String() string { return proto.CompactTextString(m) }
func (*SeekOldest) ProtoMessage()    {}
func (*SeekOldest) Descriptor() ([]byte, []int) {
	return fileDescriptor_ab_86effae0ebc2388c, []int{2}
}
func (m *SeekOldest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SeekOldest.Unmarshal(m, b)
}
func (m *SeekOldest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SeekOldest.Marshal(b, m, deterministic)
}
func (dst *SeekOldest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SeekOldest.Merge(dst, src)
}
func (m *SeekOldest) XXX_Size() int {
	return xxx_messageInfo_SeekOldest.Size(m)
}
func (m *SeekOldest) XXX_DiscardUnknown() {
	xxx_messageInfo_SeekOldest.DiscardUnknown(m)
}

var xxx_messageInfo_SeekOldest proto.InternalMessageInfo

type SeekSpecified struct {
	Number               uint64   `protobuf:"varint,1,opt,name=number,proto3" json:"number,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SeekSpecified) Reset()         { *m = SeekSpecified{} }
func (m *SeekSpecified) String() string { return proto.CompactTextString(m) }
func (*SeekSpecified) ProtoMessage()    {}
func (*SeekSpecified) Descriptor() ([]byte, []int) {
	return fileDescriptor_ab_86effae0ebc2388c, []int{3}
}
func (m *SeekSpecified) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SeekSpecified.Unmarshal(m, b)
}
func (m *SeekSpecified) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SeekSpecified.Marshal(b, m, deterministic)
}
func (dst *SeekSpecified) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SeekSpecified.Merge(dst, src)
}
func (m *SeekSpecified) XXX_Size() int {
	return xxx_messageInfo_SeekSpecified.Size(m)
}
func (m *SeekSpecified) XXX_DiscardUnknown() {
	xxx_messageInfo_SeekSpecified.DiscardUnknown(m)
}

var xxx_messageInfo_SeekSpecified proto.InternalMessageInfo

func (m *SeekSpecified) GetNumber() uint64 {
	if m != nil {
		return m.Number
	}
	return 0
}

type SeekPosition struct {
//有效分配给类型的类型：
//*请参见位置\最新
//*请参见位置“最旧”
//*请参见指定的位置
	Type                 isSeekPosition_Type `protobuf_oneof:"Type"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *SeekPosition) Reset()         { *m = SeekPosition{} }
func (m *SeekPosition) String() string { return proto.CompactTextString(m) }
func (*SeekPosition) ProtoMessage()    {}
func (*SeekPosition) Descriptor() ([]byte, []int) {
	return fileDescriptor_ab_86effae0ebc2388c, []int{4}
}
func (m *SeekPosition) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SeekPosition.Unmarshal(m, b)
}
func (m *SeekPosition) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SeekPosition.Marshal(b, m, deterministic)
}
func (dst *SeekPosition) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SeekPosition.Merge(dst, src)
}
func (m *SeekPosition) XXX_Size() int {
	return xxx_messageInfo_SeekPosition.Size(m)
}
func (m *SeekPosition) XXX_DiscardUnknown() {
	xxx_messageInfo_SeekPosition.DiscardUnknown(m)
}

var xxx_messageInfo_SeekPosition proto.InternalMessageInfo

type isSeekPosition_Type interface {
	isSeekPosition_Type()
}

type SeekPosition_Newest struct {
	Newest *SeekNewest `protobuf:"bytes,1,opt,name=newest,proto3,oneof"`
}

type SeekPosition_Oldest struct {
	Oldest *SeekOldest `protobuf:"bytes,2,opt,name=oldest,proto3,oneof"`
}

type SeekPosition_Specified struct {
	Specified *SeekSpecified `protobuf:"bytes,3,opt,name=specified,proto3,oneof"`
}

func (*SeekPosition_Newest) isSeekPosition_Type() {}

func (*SeekPosition_Oldest) isSeekPosition_Type() {}

func (*SeekPosition_Specified) isSeekPosition_Type() {}

func (m *SeekPosition) GetType() isSeekPosition_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *SeekPosition) GetNewest() *SeekNewest {
	if x, ok := m.GetType().(*SeekPosition_Newest); ok {
		return x.Newest
	}
	return nil
}

func (m *SeekPosition) GetOldest() *SeekOldest {
	if x, ok := m.GetType().(*SeekPosition_Oldest); ok {
		return x.Oldest
	}
	return nil
}

func (m *SeekPosition) GetSpecified() *SeekSpecified {
	if x, ok := m.GetType().(*SeekPosition_Specified); ok {
		return x.Specified
	}
	return nil
}

//xxxoneoffuncs用于Proto包的内部使用。
func (*SeekPosition) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _SeekPosition_OneofMarshaler, _SeekPosition_OneofUnmarshaler, _SeekPosition_OneofSizer, []interface{}{
		(*SeekPosition_Newest)(nil),
		(*SeekPosition_Oldest)(nil),
		(*SeekPosition_Specified)(nil),
	}
}

func _SeekPosition_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*SeekPosition)
//类型
	switch x := m.Type.(type) {
	case *SeekPosition_Newest:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Newest); err != nil {
			return err
		}
	case *SeekPosition_Oldest:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Oldest); err != nil {
			return err
		}
	case *SeekPosition_Specified:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Specified); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("SeekPosition.Type has unexpected type %T", x)
	}
	return nil
}

func _SeekPosition_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*SeekPosition)
	switch tag {
case 1: //最新类型
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(SeekNewest)
		err := b.DecodeMessage(msg)
		m.Type = &SeekPosition_Newest{msg}
		return true, err
case 2: //最古老类型
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(SeekOldest)
		err := b.DecodeMessage(msg)
		m.Type = &SeekPosition_Oldest{msg}
		return true, err
case 3: //指定类型
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(SeekSpecified)
		err := b.DecodeMessage(msg)
		m.Type = &SeekPosition_Specified{msg}
		return true, err
	default:
		return false, nil
	}
}

func _SeekPosition_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*SeekPosition)
//类型
	switch x := m.Type.(type) {
	case *SeekPosition_Newest:
		s := proto.Size(x.Newest)
n += 1 //标签和电线
		n += proto.SizeVarint(uint64(s))
		n += s
	case *SeekPosition_Oldest:
		s := proto.Size(x.Oldest)
n += 1 //标签和电线
		n += proto.SizeVarint(uint64(s))
		n += s
	case *SeekPosition_Specified:
		s := proto.Size(x.Specified)
n += 1 //标签和电线
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

//seekinfo指定要返回的请求块的范围
//如果未找到起始位置，则立即返回错误
//否则，将返回块，直到遇到丢失的块，然后指定行为
//按指定的SeekBehavior。如果在指定“准备就绪”之前阻止，则答复将阻止到
//请求的块是可用的，如果指定了“未就绪”，则回复将返回一个
//指示找不到块的错误。请求无限期地返回所有块
//当它们被创建时，行为应该被设置为阻止，直到“准备就绪”，停止应该被设置为
//指定的最大值为14
type SeekInfo struct {
	Start                *SeekPosition         `protobuf:"bytes,1,opt,name=start,proto3" json:"start,omitempty"`
	Stop                 *SeekPosition         `protobuf:"bytes,2,opt,name=stop,proto3" json:"stop,omitempty"`
	Behavior             SeekInfo_SeekBehavior `protobuf:"varint,3,opt,name=behavior,proto3,enum=orderer.SeekInfo_SeekBehavior" json:"behavior,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *SeekInfo) Reset()         { *m = SeekInfo{} }
func (m *SeekInfo) String() string { return proto.CompactTextString(m) }
func (*SeekInfo) ProtoMessage()    {}
func (*SeekInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_ab_86effae0ebc2388c, []int{5}
}
func (m *SeekInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SeekInfo.Unmarshal(m, b)
}
func (m *SeekInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SeekInfo.Marshal(b, m, deterministic)
}
func (dst *SeekInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SeekInfo.Merge(dst, src)
}
func (m *SeekInfo) XXX_Size() int {
	return xxx_messageInfo_SeekInfo.Size(m)
}
func (m *SeekInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_SeekInfo.DiscardUnknown(m)
}

var xxx_messageInfo_SeekInfo proto.InternalMessageInfo

func (m *SeekInfo) GetStart() *SeekPosition {
	if m != nil {
		return m.Start
	}
	return nil
}

func (m *SeekInfo) GetStop() *SeekPosition {
	if m != nil {
		return m.Stop
	}
	return nil
}

func (m *SeekInfo) GetBehavior() SeekInfo_SeekBehavior {
	if m != nil {
		return m.Behavior
	}
	return SeekInfo_BLOCK_UNTIL_READY
}

type DeliverResponse struct {
//有效分配给类型的类型：
//*交付响应\状态
//*DeliverResponse_块
	Type                 isDeliverResponse_Type `protobuf_oneof:"Type"`
	XXX_NoUnkeyedLiteral struct{}               `json:"-"`
	XXX_unrecognized     []byte                 `json:"-"`
	XXX_sizecache        int32                  `json:"-"`
}

func (m *DeliverResponse) Reset()         { *m = DeliverResponse{} }
func (m *DeliverResponse) String() string { return proto.CompactTextString(m) }
func (*DeliverResponse) ProtoMessage()    {}
func (*DeliverResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_ab_86effae0ebc2388c, []int{6}
}
func (m *DeliverResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeliverResponse.Unmarshal(m, b)
}
func (m *DeliverResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeliverResponse.Marshal(b, m, deterministic)
}
func (dst *DeliverResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeliverResponse.Merge(dst, src)
}
func (m *DeliverResponse) XXX_Size() int {
	return xxx_messageInfo_DeliverResponse.Size(m)
}
func (m *DeliverResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DeliverResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DeliverResponse proto.InternalMessageInfo

type isDeliverResponse_Type interface {
	isDeliverResponse_Type()
}

type DeliverResponse_Status struct {
	Status common.Status `protobuf:"varint,1,opt,name=status,proto3,enum=common.Status,oneof"`
}

type DeliverResponse_Block struct {
	Block *common.Block `protobuf:"bytes,2,opt,name=block,proto3,oneof"`
}

func (*DeliverResponse_Status) isDeliverResponse_Type() {}

func (*DeliverResponse_Block) isDeliverResponse_Type() {}

func (m *DeliverResponse) GetType() isDeliverResponse_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *DeliverResponse) GetStatus() common.Status {
	if x, ok := m.GetType().(*DeliverResponse_Status); ok {
		return x.Status
	}
	return common.Status_UNKNOWN
}

func (m *DeliverResponse) GetBlock() *common.Block {
	if x, ok := m.GetType().(*DeliverResponse_Block); ok {
		return x.Block
	}
	return nil
}

//xxxoneoffuncs用于Proto包的内部使用。
func (*DeliverResponse) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _DeliverResponse_OneofMarshaler, _DeliverResponse_OneofUnmarshaler, _DeliverResponse_OneofSizer, []interface{}{
		(*DeliverResponse_Status)(nil),
		(*DeliverResponse_Block)(nil),
	}
}

func _DeliverResponse_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*DeliverResponse)
//类型
	switch x := m.Type.(type) {
	case *DeliverResponse_Status:
		b.EncodeVarint(1<<3 | proto.WireVarint)
		b.EncodeVarint(uint64(x.Status))
	case *DeliverResponse_Block:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Block); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("DeliverResponse.Type has unexpected type %T", x)
	}
	return nil
}

func _DeliverResponse_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*DeliverResponse)
	switch tag {
case 1: //类型状态
		if wire != proto.WireVarint {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeVarint()
		m.Type = &DeliverResponse_Status{common.Status(x)}
		return true, err
case 2: //类型块
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(common.Block)
		err := b.DecodeMessage(msg)
		m.Type = &DeliverResponse_Block{msg}
		return true, err
	default:
		return false, nil
	}
}

func _DeliverResponse_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*DeliverResponse)
//类型
	switch x := m.Type.(type) {
	case *DeliverResponse_Status:
n += 1 //标签和电线
		n += proto.SizeVarint(uint64(x.Status))
	case *DeliverResponse_Block:
		s := proto.Size(x.Block)
n += 1 //标签和电线
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

func init() {
	proto.RegisterType((*BroadcastResponse)(nil), "orderer.BroadcastResponse")
	proto.RegisterType((*SeekNewest)(nil), "orderer.SeekNewest")
	proto.RegisterType((*SeekOldest)(nil), "orderer.SeekOldest")
	proto.RegisterType((*SeekSpecified)(nil), "orderer.SeekSpecified")
	proto.RegisterType((*SeekPosition)(nil), "orderer.SeekPosition")
	proto.RegisterType((*SeekInfo)(nil), "orderer.SeekInfo")
	proto.RegisterType((*DeliverResponse)(nil), "orderer.DeliverResponse")
	proto.RegisterEnum("orderer.SeekInfo_SeekBehavior", SeekInfo_SeekBehavior_name, SeekInfo_SeekBehavior_value)
}

//引用导入以禁止错误（如果未使用）。
var _ context.Context
var _ grpc.ClientConn

//这是一个编译时断言，以确保生成的文件
//与正在编译的GRPC包兼容。
const _ = grpc.SupportPackageIsVersion4

//atomicbroadcast client是atomicbroadcast服务的客户端API。
//
//有关CTX使用和关闭/结束流式RPC的语义，请参阅https://godoc.org/google.golang.org/grpc clientconn.newstream。
type AtomicBroadcastClient interface {
//广播接收对每个公用信封的确认回复。信封按顺序排列，指示成功或失败类型。
	Broadcast(ctx context.Context, opts ...grpc.CallOption) (AtomicBroadcast_BroadcastClient, error)
//Deliver First需要一个deliver_seek_info类型的信封，其有效载荷数据为mashaled seek info消息，然后接收一个块回复流。
	Deliver(ctx context.Context, opts ...grpc.CallOption) (AtomicBroadcast_DeliverClient, error)
}

type atomicBroadcastClient struct {
	cc *grpc.ClientConn
}

func NewAtomicBroadcastClient(cc *grpc.ClientConn) AtomicBroadcastClient {
	return &atomicBroadcastClient{cc}
}

func (c *atomicBroadcastClient) Broadcast(ctx context.Context, opts ...grpc.CallOption) (AtomicBroadcast_BroadcastClient, error) {
	stream, err := c.cc.NewStream(ctx, &_AtomicBroadcast_serviceDesc.Streams[0], "/orderer.AtomicBroadcast/Broadcast", opts...)
	if err != nil {
		return nil, err
	}
	x := &atomicBroadcastBroadcastClient{stream}
	return x, nil
}

type AtomicBroadcast_BroadcastClient interface {
	Send(*common.Envelope) error
	Recv() (*BroadcastResponse, error)
	grpc.ClientStream
}

type atomicBroadcastBroadcastClient struct {
	grpc.ClientStream
}

func (x *atomicBroadcastBroadcastClient) Send(m *common.Envelope) error {
	return x.ClientStream.SendMsg(m)
}

func (x *atomicBroadcastBroadcastClient) Recv() (*BroadcastResponse, error) {
	m := new(BroadcastResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *atomicBroadcastClient) Deliver(ctx context.Context, opts ...grpc.CallOption) (AtomicBroadcast_DeliverClient, error) {
	stream, err := c.cc.NewStream(ctx, &_AtomicBroadcast_serviceDesc.Streams[1], "/orderer.AtomicBroadcast/Deliver", opts...)
	if err != nil {
		return nil, err
	}
	x := &atomicBroadcastDeliverClient{stream}
	return x, nil
}

type AtomicBroadcast_DeliverClient interface {
	Send(*common.Envelope) error
	Recv() (*DeliverResponse, error)
	grpc.ClientStream
}

type atomicBroadcastDeliverClient struct {
	grpc.ClientStream
}

func (x *atomicBroadcastDeliverClient) Send(m *common.Envelope) error {
	return x.ClientStream.SendMsg(m)
}

func (x *atomicBroadcastDeliverClient) Recv() (*DeliverResponse, error) {
	m := new(DeliverResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

//atomicbroadcast server是atomicbroadcast服务的服务器API。
type AtomicBroadcastServer interface {
//广播接收对每个公用信封的确认回复。信封按顺序排列，指示成功或失败类型。
	Broadcast(AtomicBroadcast_BroadcastServer) error
//Deliver First需要一个deliver_seek_info类型的信封，其有效载荷数据为mashaled seek info消息，然后接收一个块回复流。
	Deliver(AtomicBroadcast_DeliverServer) error
}

func RegisterAtomicBroadcastServer(s *grpc.Server, srv AtomicBroadcastServer) {
	s.RegisterService(&_AtomicBroadcast_serviceDesc, srv)
}

func _AtomicBroadcast_Broadcast_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(AtomicBroadcastServer).Broadcast(&atomicBroadcastBroadcastServer{stream})
}

type AtomicBroadcast_BroadcastServer interface {
	Send(*BroadcastResponse) error
	Recv() (*common.Envelope, error)
	grpc.ServerStream
}

type atomicBroadcastBroadcastServer struct {
	grpc.ServerStream
}

func (x *atomicBroadcastBroadcastServer) Send(m *BroadcastResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *atomicBroadcastBroadcastServer) Recv() (*common.Envelope, error) {
	m := new(common.Envelope)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _AtomicBroadcast_Deliver_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(AtomicBroadcastServer).Deliver(&atomicBroadcastDeliverServer{stream})
}

type AtomicBroadcast_DeliverServer interface {
	Send(*DeliverResponse) error
	Recv() (*common.Envelope, error)
	grpc.ServerStream
}

type atomicBroadcastDeliverServer struct {
	grpc.ServerStream
}

func (x *atomicBroadcastDeliverServer) Send(m *DeliverResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *atomicBroadcastDeliverServer) Recv() (*common.Envelope, error) {
	m := new(common.Envelope)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _AtomicBroadcast_serviceDesc = grpc.ServiceDesc{
	ServiceName: "orderer.AtomicBroadcast",
	HandlerType: (*AtomicBroadcastServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Broadcast",
			Handler:       _AtomicBroadcast_Broadcast_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "Deliver",
			Handler:       _AtomicBroadcast_Deliver_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "orderer/ab.proto",
}

func init() { proto.RegisterFile("orderer/ab.proto", fileDescriptor_ab_86effae0ebc2388c) }

var fileDescriptor_ab_86effae0ebc2388c = []byte{
//505字节的gzip文件描述符或协议
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x93, 0xdf, 0x6e, 0x12, 0x51,
	0x10, 0xc6, 0x59, 0xa5, 0xb4, 0x8c, 0x2d, 0xa5, 0xa7, 0x69, 0x43, 0xb8, 0x30, 0xcd, 0x26, 0x55,
	0x8c, 0xba, 0x6b, 0x30, 0xf1, 0x42, 0x4d, 0x0c, 0x6b, 0xdb, 0x40, 0x24, 0x60, 0x16, 0x7a, 0xa1,
	0x37, 0x64, 0x77, 0x19, 0x60, 0xed, 0xb2, 0x67, 0x73, 0xce, 0x01, 0xd3, 0xa7, 0xf0, 0x45, 0x7c,
	0x24, 0x1f, 0xc6, 0x9c, 0x3f, 0xbb, 0x48, 0xd5, 0x5e, 0xb1, 0x33, 0xf3, 0xfb, 0xce, 0x7c, 0x33,
	0x19, 0xa0, 0x4e, 0xd9, 0x14, 0x19, 0x32, 0x37, 0x08, 0x9d, 0x8c, 0x51, 0x41, 0xc9, 0xae, 0xc9,
	0x34, 0x8f, 0x23, 0xba, 0x5c, 0xd2, 0xd4, 0xd5, 0x3f, 0xba, 0x6a, 0x0f, 0xe1, 0xc8, 0x63, 0x34,
	0x98, 0x46, 0x01, 0x17, 0x3e, 0xf2, 0x8c, 0xa6, 0x1c, 0xc9, 0x13, 0xa8, 0x70, 0x11, 0x88, 0x15,
	0x6f, 0x58, 0x67, 0x56, 0xab, 0xd6, 0xae, 0x39, 0x46, 0x33, 0x52, 0x59, 0xdf, 0x54, 0x09, 0x81,
	0x72, 0x9c, 0xce, 0x68, 0xe3, 0xc1, 0x99, 0xd5, 0xaa, 0xfa, 0xea, 0xdb, 0xde, 0x07, 0x18, 0x21,
	0xde, 0x0c, 0xf0, 0x3b, 0x72, 0x91, 0x47, 0xc3, 0x64, 0x2a, 0xa3, 0xa7, 0x70, 0x20, 0xa3, 0x51,
	0x86, 0x51, 0x3c, 0x8b, 0x71, 0x4a, 0x4e, 0xa1, 0x92, 0xae, 0x96, 0x21, 0x32, 0xd5, 0xa8, 0xec,
	0x9b, 0xc8, 0xfe, 0x69, 0xc1, 0xbe, 0x24, 0x3f, 0x53, 0x1e, 0x8b, 0x98, 0xa6, 0xe4, 0x25, 0x54,
	0x52, 0xf5, 0xa2, 0x02, 0x1f, 0xb5, 0x8f, 0x1d, 0x33, 0x95, 0xb3, 0x69, 0xd6, 0x2d, 0xf9, 0x06,
	0x92, 0x38, 0x55, 0x2d, 0x95, 0xb5, 0xbb, 0xb8, 0x76, 0x23, 0x71, 0x0d, 0x91, 0x37, 0x50, 0xe5,
	0xb9, 0xa7, 0xc6, 0x43, 0xa5, 0x38, 0xdd, 0x52, 0x14, 0x8e, 0xbb, 0x25, 0x7f, 0x83, 0x7a, 0x15,
	0x28, 0x8f, 0x6f, 0x33, 0xb4, 0x7f, 0x59, 0xb0, 0x27, 0xb1, 0x5e, 0x3a, 0xa3, 0xe4, 0x39, 0xec,
	0x70, 0x11, 0xb0, 0xdc, 0xe9, 0xc9, 0xd6, 0x43, 0xf9, 0x40, 0xbe, 0x66, 0xc8, 0x33, 0x28, 0x73,
	0x41, 0x33, 0x63, 0xf3, 0x3f, 0xac, 0x42, 0xc8, 0x5b, 0xd8, 0x0b, 0x71, 0x11, 0xac, 0x63, 0xca,
	0x94, 0xc7, 0x5a, 0xfb, 0xf1, 0x16, 0x2e, 0x9b, 0xab, 0x0f, 0xcf, 0x50, 0x7e, 0xc1, 0xdb, 0xef,
	0xf5, 0x3a, 0xf3, 0x0a, 0x39, 0x81, 0x23, 0xaf, 0x3f, 0xfc, 0xf8, 0x69, 0x72, 0x3d, 0x18, 0xf7,
	0xfa, 0x13, 0xff, 0xb2, 0x73, 0xf1, 0xa5, 0x5e, 0x92, 0xe9, 0xab, 0x4e, 0xaf, 0x3f, 0xe9, 0x5d,
	0x4d, 0x06, 0xc3, 0xb1, 0x49, 0x5b, 0xf6, 0x37, 0x38, 0xbc, 0xc0, 0x24, 0x5e, 0x23, 0x2b, 0x2e,
	0xa4, 0x75, 0xff, 0x85, 0xc8, 0xdd, 0x9a, 0x1b, 0x39, 0x87, 0x9d, 0x30, 0xa1, 0xd1, 0x8d, 0x19,
	0xf1, 0x20, 0x07, 0x3d, 0x99, 0xec, 0x96, 0x7c, 0x5d, 0xcd, 0x57, 0xd9, 0xfe, 0x61, 0xc1, 0x61,
	0x47, 0xd0, 0x65, 0x1c, 0x15, 0x67, 0x49, 0x3e, 0x40, 0x75, 0x13, 0xd4, 0xf3, 0x07, 0x2e, 0xd3,
	0x35, 0x26, 0x34, 0xc3, 0x66, 0xb3, 0x58, 0xc3, 0x5f, 0x97, 0x6c, 0x97, 0x5a, 0xd6, 0x2b, 0x8b,
	0xbc, 0x83, 0x5d, 0x33, 0xc0, 0x3f, 0xe4, 0x8d, 0x42, 0x7e, 0x67, 0x48, 0x2d, 0xf6, 0xae, 0xe1,
	0x9c, 0xb2, 0xb9, 0xb3, 0xb8, 0xcd, 0x90, 0x25, 0x38, 0x9d, 0x23, 0x73, 0x66, 0x41, 0xc8, 0xe2,
	0x48, 0xff, 0x83, 0x78, 0x2e, 0xff, 0xfa, 0x62, 0x1e, 0x8b, 0xc5, 0x2a, 0x94, 0x0d, 0xdc, 0x3f,
	0x68, 0x57, 0xd3, 0xae, 0xa6, 0x5d, 0x43, 0x87, 0x15, 0x15, 0xbf, 0xfe, 0x1d, 0x00, 0x00, 0xff,
	0xff, 0x4b, 0x88, 0xa4, 0x39, 0xb1, 0x03, 0x00, 0x00,
}
