
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//由Protoc Gen Go生成的代码。不要编辑。
//来源：Ledger/query result/kv_query_result.proto

package queryresult //导入“github.com/hyperledger/fabric/protos/ledger/queryresult”

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import timestamp "github.com/golang/protobuf/ptypes/timestamp"

//引用导入以禁止错误（如果未使用）。
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

//这是一个编译时断言，以确保生成的文件
//与正在编译的proto包兼容。
//此行的编译错误可能意味着您的
//需要更新proto包。
const _ = proto.ProtoPackageIsVersion2 //请升级proto包

//kv——范围/执行查询的查询结果。保存一个键和相应的值。
type KV struct {
	Namespace            string   `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Key                  string   `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	Value                []byte   `protobuf:"bytes,3,opt,name=value,proto3" json:"value,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *KV) Reset()         { *m = KV{} }
func (m *KV) String() string { return proto.CompactTextString(m) }
func (*KV) ProtoMessage()    {}
func (*KV) Descriptor() ([]byte, []int) {
	return fileDescriptor_kv_query_result_8f59f813f4fe5e5a, []int{0}
}
func (m *KV) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_KV.Unmarshal(m, b)
}
func (m *KV) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_KV.Marshal(b, m, deterministic)
}
func (dst *KV) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KV.Merge(dst, src)
}
func (m *KV) XXX_Size() int {
	return xxx_messageInfo_KV.Size(m)
}
func (m *KV) XXX_DiscardUnknown() {
	xxx_messageInfo_KV.DiscardUnknown(m)
}

var xxx_messageInfo_KV proto.InternalMessageInfo

func (m *KV) GetNamespace() string {
	if m != nil {
		return m.Namespace
	}
	return ""
}

func (m *KV) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *KV) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

//keymodification——历史查询的查询结果。保存事务ID、值，
//时间戳，并删除由历史查询产生的标记。
type KeyModification struct {
	TxId                 string               `protobuf:"bytes,1,opt,name=tx_id,json=txId,proto3" json:"tx_id,omitempty"`
	Value                []byte               `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Timestamp            *timestamp.Timestamp `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	IsDelete             bool                 `protobuf:"varint,4,opt,name=is_delete,json=isDelete,proto3" json:"is_delete,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *KeyModification) Reset()         { *m = KeyModification{} }
func (m *KeyModification) String() string { return proto.CompactTextString(m) }
func (*KeyModification) ProtoMessage()    {}
func (*KeyModification) Descriptor() ([]byte, []int) {
	return fileDescriptor_kv_query_result_8f59f813f4fe5e5a, []int{1}
}
func (m *KeyModification) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_KeyModification.Unmarshal(m, b)
}
func (m *KeyModification) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_KeyModification.Marshal(b, m, deterministic)
}
func (dst *KeyModification) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KeyModification.Merge(dst, src)
}
func (m *KeyModification) XXX_Size() int {
	return xxx_messageInfo_KeyModification.Size(m)
}
func (m *KeyModification) XXX_DiscardUnknown() {
	xxx_messageInfo_KeyModification.DiscardUnknown(m)
}

var xxx_messageInfo_KeyModification proto.InternalMessageInfo

func (m *KeyModification) GetTxId() string {
	if m != nil {
		return m.TxId
	}
	return ""
}

func (m *KeyModification) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *KeyModification) GetTimestamp() *timestamp.Timestamp {
	if m != nil {
		return m.Timestamp
	}
	return nil
}

func (m *KeyModification) GetIsDelete() bool {
	if m != nil {
		return m.IsDelete
	}
	return false
}

func init() {
	proto.RegisterType((*KV)(nil), "queryresult.KV")
	proto.RegisterType((*KeyModification)(nil), "queryresult.KeyModification")
}

func init() {
	proto.RegisterFile("ledger/queryresult/kv_query_result.proto", fileDescriptor_kv_query_result_8f59f813f4fe5e5a)
}

var fileDescriptor_kv_query_result_8f59f813f4fe5e5a = []byte{
//gzip文件描述符或协议的286字节
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x51, 0x4f, 0x4b, 0xc3, 0x30,
	0x1c, 0xa5, 0xdd, 0x26, 0x6b, 0x26, 0x28, 0xd1, 0x43, 0x99, 0x82, 0x65, 0xa7, 0x9e, 0x12, 0xd1,
	0x83, 0x9e, 0xc5, 0x8b, 0x0e, 0x2f, 0x45, 0x3c, 0x78, 0x29, 0x69, 0xfb, 0x6b, 0x17, 0xda, 0x2e,
	0x35, 0x7f, 0xc6, 0xfa, 0x39, 0xfc, 0xc2, 0x62, 0xb2, 0xd9, 0x82, 0xb7, 0xbc, 0xf7, 0x7b, 0xef,
	0xf1, 0x78, 0x41, 0x71, 0x03, 0x45, 0x05, 0x92, 0x7e, 0x19, 0x90, 0xbd, 0x04, 0x65, 0x1a, 0x4d,
	0xeb, 0x5d, 0x6a, 0x61, 0xea, 0x30, 0xe9, 0xa4, 0xd0, 0x02, 0x2f, 0x46, 0x92, 0xe5, 0x4d, 0x25,
	0x44, 0xd5, 0x00, 0xb5, 0xa7, 0xcc, 0x94, 0x54, 0xf3, 0x16, 0x94, 0x66, 0x6d, 0xe7, 0xd4, 0xab,
	0x57, 0xe4, 0xaf, 0x3f, 0xf0, 0x35, 0x0a, 0xb6, 0xac, 0x05, 0xd5, 0xb1, 0x1c, 0x42, 0x2f, 0xf2,
	0xe2, 0x20, 0x19, 0x08, 0x7c, 0x8e, 0x26, 0x35, 0xf4, 0xa1, 0x6f, 0xf9, 0xdf, 0x27, 0xbe, 0x44,
	0xb3, 0x1d, 0x6b, 0x0c, 0x84, 0x93, 0xc8, 0x8b, 0x4f, 0x13, 0x07, 0x56, 0xdf, 0x1e, 0x3a, 0x5b,
	0x43, 0xff, 0x26, 0x0a, 0x5e, 0xf2, 0x9c, 0x69, 0x2e, 0xb6, 0xf8, 0x02, 0xcd, 0xf4, 0x3e, 0xe5,
	0xc5, 0x21, 0x75, 0xaa, 0xf7, 0x2f, 0xc5, 0x60, 0xf7, 0x47, 0x76, 0xfc, 0x88, 0x82, 0xbf, 0x76,
	0x36, 0x78, 0x71, 0xb7, 0x24, 0xae, 0x3f, 0x39, 0xf6, 0x27, 0xef, 0x47, 0x45, 0x32, 0x88, 0xf1,
	0x15, 0x0a, 0xb8, 0x4a, 0x0b, 0x68, 0x40, 0x43, 0x38, 0x8d, 0xbc, 0x78, 0x9e, 0xcc, 0xb9, 0x7a,
	0xb6, 0xf8, 0xa9, 0x46, 0xb7, 0x42, 0x56, 0x64, 0xd3, 0x77, 0x20, 0xdd, 0x88, 0xa4, 0x64, 0x99,
	0xe4, 0xb9, 0x0b, 0x55, 0xe4, 0x40, 0x8e, 0x66, 0xfb, 0x7c, 0xa8, 0xb8, 0xde, 0x98, 0x8c, 0xe4,
	0xa2, 0xa5, 0x23, 0x23, 0x75, 0x46, 0xb7, 0xa6, 0xa2, 0xff, 0xbf, 0x24, 0x3b, 0xb1, 0xa7, 0xfb,
	0x9f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xa2, 0xb7, 0x3e, 0x86, 0xaf, 0x01, 0x00, 0x00,
}
