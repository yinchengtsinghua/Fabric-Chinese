
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//由Protoc Gen Go生成的代码。不要编辑。
//来源：peer/transaction.proto

package peer //导入“github.com/hyperledger/fabric/protos/peer”

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/golang/protobuf/ptypes/timestamp"
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

type TxValidationCode int32

const (
	TxValidationCode_VALID                        TxValidationCode = 0
	TxValidationCode_NIL_ENVELOPE                 TxValidationCode = 1
	TxValidationCode_BAD_PAYLOAD                  TxValidationCode = 2
	TxValidationCode_BAD_COMMON_HEADER            TxValidationCode = 3
	TxValidationCode_BAD_CREATOR_SIGNATURE        TxValidationCode = 4
	TxValidationCode_INVALID_ENDORSER_TRANSACTION TxValidationCode = 5
	TxValidationCode_INVALID_CONFIG_TRANSACTION   TxValidationCode = 6
	TxValidationCode_UNSUPPORTED_TX_PAYLOAD       TxValidationCode = 7
	TxValidationCode_BAD_PROPOSAL_TXID            TxValidationCode = 8
	TxValidationCode_DUPLICATE_TXID               TxValidationCode = 9
	TxValidationCode_ENDORSEMENT_POLICY_FAILURE   TxValidationCode = 10
	TxValidationCode_MVCC_READ_CONFLICT           TxValidationCode = 11
	TxValidationCode_PHANTOM_READ_CONFLICT        TxValidationCode = 12
	TxValidationCode_UNKNOWN_TX_TYPE              TxValidationCode = 13
	TxValidationCode_TARGET_CHAIN_NOT_FOUND       TxValidationCode = 14
	TxValidationCode_MARSHAL_TX_ERROR             TxValidationCode = 15
	TxValidationCode_NIL_TXACTION                 TxValidationCode = 16
	TxValidationCode_EXPIRED_CHAINCODE            TxValidationCode = 17
	TxValidationCode_CHAINCODE_VERSION_CONFLICT   TxValidationCode = 18
	TxValidationCode_BAD_HEADER_EXTENSION         TxValidationCode = 19
	TxValidationCode_BAD_CHANNEL_HEADER           TxValidationCode = 20
	TxValidationCode_BAD_RESPONSE_PAYLOAD         TxValidationCode = 21
	TxValidationCode_BAD_RWSET                    TxValidationCode = 22
	TxValidationCode_ILLEGAL_WRITESET             TxValidationCode = 23
	TxValidationCode_INVALID_WRITESET             TxValidationCode = 24
	TxValidationCode_NOT_VALIDATED                TxValidationCode = 254
	TxValidationCode_INVALID_OTHER_REASON         TxValidationCode = 255
)

var TxValidationCode_name = map[int32]string{
	0:   "VALID",
	1:   "NIL_ENVELOPE",
	2:   "BAD_PAYLOAD",
	3:   "BAD_COMMON_HEADER",
	4:   "BAD_CREATOR_SIGNATURE",
	5:   "INVALID_ENDORSER_TRANSACTION",
	6:   "INVALID_CONFIG_TRANSACTION",
	7:   "UNSUPPORTED_TX_PAYLOAD",
	8:   "BAD_PROPOSAL_TXID",
	9:   "DUPLICATE_TXID",
	10:  "ENDORSEMENT_POLICY_FAILURE",
	11:  "MVCC_READ_CONFLICT",
	12:  "PHANTOM_READ_CONFLICT",
	13:  "UNKNOWN_TX_TYPE",
	14:  "TARGET_CHAIN_NOT_FOUND",
	15:  "MARSHAL_TX_ERROR",
	16:  "NIL_TXACTION",
	17:  "EXPIRED_CHAINCODE",
	18:  "CHAINCODE_VERSION_CONFLICT",
	19:  "BAD_HEADER_EXTENSION",
	20:  "BAD_CHANNEL_HEADER",
	21:  "BAD_RESPONSE_PAYLOAD",
	22:  "BAD_RWSET",
	23:  "ILLEGAL_WRITESET",
	24:  "INVALID_WRITESET",
	254: "NOT_VALIDATED",
	255: "INVALID_OTHER_REASON",
}
var TxValidationCode_value = map[string]int32{
	"VALID":                        0,
	"NIL_ENVELOPE":                 1,
	"BAD_PAYLOAD":                  2,
	"BAD_COMMON_HEADER":            3,
	"BAD_CREATOR_SIGNATURE":        4,
	"INVALID_ENDORSER_TRANSACTION": 5,
	"INVALID_CONFIG_TRANSACTION":   6,
	"UNSUPPORTED_TX_PAYLOAD":       7,
	"BAD_PROPOSAL_TXID":            8,
	"DUPLICATE_TXID":               9,
	"ENDORSEMENT_POLICY_FAILURE":   10,
	"MVCC_READ_CONFLICT":           11,
	"PHANTOM_READ_CONFLICT":        12,
	"UNKNOWN_TX_TYPE":              13,
	"TARGET_CHAIN_NOT_FOUND":       14,
	"MARSHAL_TX_ERROR":             15,
	"NIL_TXACTION":                 16,
	"EXPIRED_CHAINCODE":            17,
	"CHAINCODE_VERSION_CONFLICT":   18,
	"BAD_HEADER_EXTENSION":         19,
	"BAD_CHANNEL_HEADER":           20,
	"BAD_RESPONSE_PAYLOAD":         21,
	"BAD_RWSET":                    22,
	"ILLEGAL_WRITESET":             23,
	"INVALID_WRITESET":             24,
	"NOT_VALIDATED":                254,
	"INVALID_OTHER_REASON":         255,
}

func (x TxValidationCode) String() string {
	return proto.EnumName(TxValidationCode_name, int32(x))
}
func (TxValidationCode) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_transaction_4fbd1a0e1a50cfab, []int{0}
}

//键级元数据映射中的保留项
type MetaDataKeys int32

const (
	MetaDataKeys_VALIDATION_PARAMETER MetaDataKeys = 0
)

var MetaDataKeys_name = map[int32]string{
	0: "VALIDATION_PARAMETER",
}
var MetaDataKeys_value = map[string]int32{
	"VALIDATION_PARAMETER": 0,
}

func (x MetaDataKeys) String() string {
	return proto.EnumName(MetaDataKeys_name, int32(x))
}
func (MetaDataKeys) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_transaction_4fbd1a0e1a50cfab, []int{1}
}

//此消息对于验证签名是必要的
//（在签名字段中）超过事务的字节（在
//TransactionBytes字段）。
type SignedTransaction struct {
//事务的字节。无损检测
	TransactionBytes []byte `protobuf:"bytes,1,opt,name=transaction_bytes,json=transactionBytes,proto3" json:"transaction_bytes,omitempty"`
//签名的公钥所在的TransactionBytes的签名
//Transaction的头字段可能有多个
//事务处理，所以有多个头，但应该有相同的头
//所有标题中的事务处理程序标识（cert）
	Signature            []byte   `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SignedTransaction) Reset()         { *m = SignedTransaction{} }
func (m *SignedTransaction) String() string { return proto.CompactTextString(m) }
func (*SignedTransaction) ProtoMessage()    {}
func (*SignedTransaction) Descriptor() ([]byte, []int) {
	return fileDescriptor_transaction_4fbd1a0e1a50cfab, []int{0}
}
func (m *SignedTransaction) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SignedTransaction.Unmarshal(m, b)
}
func (m *SignedTransaction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SignedTransaction.Marshal(b, m, deterministic)
}
func (dst *SignedTransaction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SignedTransaction.Merge(dst, src)
}
func (m *SignedTransaction) XXX_Size() int {
	return xxx_messageInfo_SignedTransaction.Size(m)
}
func (m *SignedTransaction) XXX_DiscardUnknown() {
	xxx_messageInfo_SignedTransaction.DiscardUnknown(m)
}

var xxx_messageInfo_SignedTransaction proto.InternalMessageInfo

func (m *SignedTransaction) GetTransactionBytes() []byte {
	if m != nil {
		return m.TransactionBytes
	}
	return nil
}

func (m *SignedTransaction) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

//processedTransaction包装一个信封，其中包含一个事务和一个指示
//通过提交对等方来验证或使事务无效。
//用例是GetTransactionByID API需要检索事务信封
//从块存储，并将其返回到客户机，并指示事务
//通过提交对等方验证或使其无效。以便最初提交的
//未修改事务信封，将返回processedTransaction包装。
type ProcessedTransaction struct {
//包含已处理事务的信封。
	TransactionEnvelope *common.Envelope `protobuf:"bytes,1,opt,name=transactionEnvelope,proto3" json:"transactionEnvelope,omitempty"`
//通过提交对等方来指示事务是已验证还是已失效。
	ValidationCode       int32    `protobuf:"varint,2,opt,name=validationCode,proto3" json:"validationCode,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ProcessedTransaction) Reset()         { *m = ProcessedTransaction{} }
func (m *ProcessedTransaction) String() string { return proto.CompactTextString(m) }
func (*ProcessedTransaction) ProtoMessage()    {}
func (*ProcessedTransaction) Descriptor() ([]byte, []int) {
	return fileDescriptor_transaction_4fbd1a0e1a50cfab, []int{1}
}
func (m *ProcessedTransaction) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProcessedTransaction.Unmarshal(m, b)
}
func (m *ProcessedTransaction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProcessedTransaction.Marshal(b, m, deterministic)
}
func (dst *ProcessedTransaction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProcessedTransaction.Merge(dst, src)
}
func (m *ProcessedTransaction) XXX_Size() int {
	return xxx_messageInfo_ProcessedTransaction.Size(m)
}
func (m *ProcessedTransaction) XXX_DiscardUnknown() {
	xxx_messageInfo_ProcessedTransaction.DiscardUnknown(m)
}

var xxx_messageInfo_ProcessedTransaction proto.InternalMessageInfo

func (m *ProcessedTransaction) GetTransactionEnvelope() *common.Envelope {
	if m != nil {
		return m.TransactionEnvelope
	}
	return nil
}

func (m *ProcessedTransaction) GetValidationCode() int32 {
	if m != nil {
		return m.ValidationCode
	}
	return 0
}

//要发送到订购服务的事务。事务包含
//一个或多个交易。每一项交易都将一项提议约束到
//可能有多个动作。事务是原子的，意味着
//事务中的所有操作都将被提交或不提交。注意
//虽然事务可能包含多个头，但header.creator
//每个字段必须相同。
//一个客户可以自由发布多个独立的提案，每个提案
//它们的头（头）和请求有效载荷（chaincodeProposalPayload）。各
//提案得到独立批准，产生了一项行动。
//（提案负责人签名）每个背书人一个签名。任意数量
//独立提案（及其行动）可能包括在交易中。
//以确保它们被原子化处理。
type Transaction struct {
//有效负载是一个事务操作数组。数组是必需的
//每个事务包含多个操作
	Actions              []*TransactionAction `protobuf:"bytes,1,rep,name=actions,proto3" json:"actions,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *Transaction) Reset()         { *m = Transaction{} }
func (m *Transaction) String() string { return proto.CompactTextString(m) }
func (*Transaction) ProtoMessage()    {}
func (*Transaction) Descriptor() ([]byte, []int) {
	return fileDescriptor_transaction_4fbd1a0e1a50cfab, []int{2}
}
func (m *Transaction) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Transaction.Unmarshal(m, b)
}
func (m *Transaction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Transaction.Marshal(b, m, deterministic)
}
func (dst *Transaction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Transaction.Merge(dst, src)
}
func (m *Transaction) XXX_Size() int {
	return xxx_messageInfo_Transaction.Size(m)
}
func (m *Transaction) XXX_DiscardUnknown() {
	xxx_messageInfo_Transaction.DiscardUnknown(m)
}

var xxx_messageInfo_Transaction proto.InternalMessageInfo

func (m *Transaction) GetActions() []*TransactionAction {
	if m != nil {
		return m.Actions
	}
	return nil
}

//交易行为将提案与其行为绑定在一起。中的类型字段
//标题指定要应用于分类帐的操作类型。
type TransactionAction struct {
//建议操作的标题，即建议标题
	Header []byte `protobuf:"bytes,1,opt,name=header,proto3" json:"header,omitempty"`
//由标题中的类型定义的操作的有效负载
//chaincode，它是chaincodeactionPayload的字节数
	Payload              []byte   `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TransactionAction) Reset()         { *m = TransactionAction{} }
func (m *TransactionAction) String() string { return proto.CompactTextString(m) }
func (*TransactionAction) ProtoMessage()    {}
func (*TransactionAction) Descriptor() ([]byte, []int) {
	return fileDescriptor_transaction_4fbd1a0e1a50cfab, []int{3}
}
func (m *TransactionAction) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TransactionAction.Unmarshal(m, b)
}
func (m *TransactionAction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TransactionAction.Marshal(b, m, deterministic)
}
func (dst *TransactionAction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TransactionAction.Merge(dst, src)
}
func (m *TransactionAction) XXX_Size() int {
	return xxx_messageInfo_TransactionAction.Size(m)
}
func (m *TransactionAction) XXX_DiscardUnknown() {
	xxx_messageInfo_TransactionAction.DiscardUnknown(m)
}

var xxx_messageInfo_TransactionAction proto.InternalMessageInfo

func (m *TransactionAction) GetHeader() []byte {
	if m != nil {
		return m.Header
	}
	return nil
}

func (m *TransactionAction) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

//ChaincodeActionPayload是用于事务处理的消息
//当头的类型设置为chaincode时的有效负载。它承载着
//chaincodeProposalPayLoad和要应用于分类帐的已背书操作。
type ChaincodeActionPayload struct {
//此字段包含来自的chaincodeProposalPayLoad消息的字节
//应用程序之后的原始调用（本质上是参数）
//可见性函数的。主要的可见性模式是“满”（即
//此处包含整个chaincodeProposalPayLoad消息），“hash”（仅限
//包含chaincodeProposalPayLoad消息的哈希）或
//“没什么”。此字段将用于检查
//ProposalResponsePayLoad.ProposalHash。对于链码类型，
//ProposalResponsePayLoad.ProposalHash应为h（ProposalHeader
//f（chaincodeProposalPayLoad）），其中f是可见性函数。
	ChaincodeProposalPayload []byte `protobuf:"bytes,1,opt,name=chaincode_proposal_payload,json=chaincodeProposalPayload,proto3" json:"chaincode_proposal_payload,omitempty"`
//要应用于分类帐的操作列表
	Action               *ChaincodeEndorsedAction `protobuf:"bytes,2,opt,name=action,proto3" json:"action,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *ChaincodeActionPayload) Reset()         { *m = ChaincodeActionPayload{} }
func (m *ChaincodeActionPayload) String() string { return proto.CompactTextString(m) }
func (*ChaincodeActionPayload) ProtoMessage()    {}
func (*ChaincodeActionPayload) Descriptor() ([]byte, []int) {
	return fileDescriptor_transaction_4fbd1a0e1a50cfab, []int{4}
}
func (m *ChaincodeActionPayload) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChaincodeActionPayload.Unmarshal(m, b)
}
func (m *ChaincodeActionPayload) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChaincodeActionPayload.Marshal(b, m, deterministic)
}
func (dst *ChaincodeActionPayload) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChaincodeActionPayload.Merge(dst, src)
}
func (m *ChaincodeActionPayload) XXX_Size() int {
	return xxx_messageInfo_ChaincodeActionPayload.Size(m)
}
func (m *ChaincodeActionPayload) XXX_DiscardUnknown() {
	xxx_messageInfo_ChaincodeActionPayload.DiscardUnknown(m)
}

var xxx_messageInfo_ChaincodeActionPayload proto.InternalMessageInfo

func (m *ChaincodeActionPayload) GetChaincodeProposalPayload() []byte {
	if m != nil {
		return m.ChaincodeProposalPayload
	}
	return nil
}

func (m *ChaincodeActionPayload) GetAction() *ChaincodeEndorsedAction {
	if m != nil {
		return m.Action
	}
	return nil
}

//chaincodeRemarkedAction包含有关
//具体建议
type ChaincodeEndorsedAction struct {
//这是由
//背书人。回想一下，对于链式代码类型，
//ProposalResponsePayLoad的扩展字段带有一个链式代码操作
	ProposalResponsePayload []byte `protobuf:"bytes,1,opt,name=proposal_response_payload,json=proposalResponsePayload,proto3" json:"proposal_response_payload,omitempty"`
//提案的背书，基本上是背书人的签字
//投标责任书
	Endorsements         []*Endorsement `protobuf:"bytes,2,rep,name=endorsements,proto3" json:"endorsements,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *ChaincodeEndorsedAction) Reset()         { *m = ChaincodeEndorsedAction{} }
func (m *ChaincodeEndorsedAction) String() string { return proto.CompactTextString(m) }
func (*ChaincodeEndorsedAction) ProtoMessage()    {}
func (*ChaincodeEndorsedAction) Descriptor() ([]byte, []int) {
	return fileDescriptor_transaction_4fbd1a0e1a50cfab, []int{5}
}
func (m *ChaincodeEndorsedAction) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChaincodeEndorsedAction.Unmarshal(m, b)
}
func (m *ChaincodeEndorsedAction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChaincodeEndorsedAction.Marshal(b, m, deterministic)
}
func (dst *ChaincodeEndorsedAction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChaincodeEndorsedAction.Merge(dst, src)
}
func (m *ChaincodeEndorsedAction) XXX_Size() int {
	return xxx_messageInfo_ChaincodeEndorsedAction.Size(m)
}
func (m *ChaincodeEndorsedAction) XXX_DiscardUnknown() {
	xxx_messageInfo_ChaincodeEndorsedAction.DiscardUnknown(m)
}

var xxx_messageInfo_ChaincodeEndorsedAction proto.InternalMessageInfo

func (m *ChaincodeEndorsedAction) GetProposalResponsePayload() []byte {
	if m != nil {
		return m.ProposalResponsePayload
	}
	return nil
}

func (m *ChaincodeEndorsedAction) GetEndorsements() []*Endorsement {
	if m != nil {
		return m.Endorsements
	}
	return nil
}

func init() {
	proto.RegisterType((*SignedTransaction)(nil), "protos.SignedTransaction")
	proto.RegisterType((*ProcessedTransaction)(nil), "protos.ProcessedTransaction")
	proto.RegisterType((*Transaction)(nil), "protos.Transaction")
	proto.RegisterType((*TransactionAction)(nil), "protos.TransactionAction")
	proto.RegisterType((*ChaincodeActionPayload)(nil), "protos.ChaincodeActionPayload")
	proto.RegisterType((*ChaincodeEndorsedAction)(nil), "protos.ChaincodeEndorsedAction")
	proto.RegisterEnum("protos.TxValidationCode", TxValidationCode_name, TxValidationCode_value)
	proto.RegisterEnum("protos.MetaDataKeys", MetaDataKeys_name, MetaDataKeys_value)
}

func init() { proto.RegisterFile("peer/transaction.proto", fileDescriptor_transaction_4fbd1a0e1a50cfab) }

var fileDescriptor_transaction_4fbd1a0e1a50cfab = []byte{
//gzip文件描述符或协议的877字节
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x55, 0x41, 0x6f, 0xe2, 0x46,
	0x14, 0x5e, 0xb2, 0x4d, 0xd2, 0x0c, 0x24, 0x19, 0x06, 0x42, 0x08, 0x8a, 0xba, 0x2b, 0x0e, 0x55,
	0xba, 0x95, 0x40, 0xca, 0x1e, 0x2a, 0x55, 0xbd, 0x0c, 0xf6, 0x24, 0x58, 0x6b, 0x66, 0xac, 0xf1,
	0x40, 0x48, 0x0f, 0x1d, 0x19, 0x98, 0x25, 0xa8, 0x60, 0x23, 0xdb, 0x59, 0x35, 0xd7, 0xfe, 0x80,
	0xf6, 0xd2, 0xdf, 0xdb, 0x56, 0xe3, 0xb1, 0x81, 0x24, 0xed, 0x05, 0x33, 0xdf, 0xfb, 0xde, 0x7b,
	0xdf, 0xf7, 0x9e, 0x35, 0x06, 0x8d, 0xb5, 0x52, 0x71, 0x37, 0x8d, 0x83, 0x30, 0x09, 0xa6, 0xe9,
	0x22, 0x0a, 0x3b, 0xeb, 0x38, 0x4a, 0x23, 0x74, 0x90, 0x3d, 0x92, 0xd6, 0xbb, 0x79, 0x14, 0xcd,
	0x97, 0xaa, 0x9b, 0x1d, 0x27, 0x8f, 0x9f, 0xbb, 0xe9, 0x62, 0xa5, 0x92, 0x34, 0x58, 0xad, 0x0d,
	0xb1, 0x75, 0x99, 0x15, 0x58, 0xc7, 0xd1, 0x3a, 0x4a, 0x82, 0xa5, 0x8c, 0x55, 0xb2, 0x8e, 0xc2,
	0x44, 0xe5, 0xd1, 0xda, 0x34, 0x5a, 0xad, 0xa2, 0xb0, 0x6b, 0x1e, 0x06, 0x6c, 0xff, 0x02, 0xaa,
	0xfe, 0x62, 0x1e, 0xaa, 0x99, 0xd8, 0xb6, 0x45, 0xdf, 0x83, 0xea, 0x8e, 0x0a, 0x39, 0x79, 0x4a,
	0x55, 0xd2, 0x2c, 0xbd, 0x2f, 0x5d, 0x55, 0x38, 0xdc, 0x09, 0xf4, 0x34, 0x8e, 0x2e, 0xc1, 0x51,
	0xb2, 0x98, 0x87, 0x41, 0xfa, 0x18, 0xab, 0xe6, 0x5e, 0x46, 0xda, 0x02, 0xed, 0xdf, 0x4b, 0xa0,
	0xee, 0xc5, 0xd1, 0x54, 0x25, 0xc9, 0xf3, 0x1e, 0x3d, 0x50, 0xdb, 0x29, 0x45, 0xc2, 0x2f, 0x6a,
	0x19, 0xad, 0x55, 0xd6, 0xa5, 0x7c, 0x0d, 0x3b, 0xb9, 0xc8, 0x02, 0xe7, 0xff, 0x45, 0x46, 0xdf,
	0x82, 0x93, 0x2f, 0xc1, 0x72, 0x31, 0x0b, 0x34, 0x6a, 0x45, 0x33, 0xd3, 0x7f, 0x9f, 0xbf, 0x40,
	0xdb, 0x3d, 0x50, 0xde, 0x6d, 0xfd, 0x11, 0x1c, 0x9a, 0x7f, 0xda, 0xd4, 0xdb, 0xab, 0xf2, 0xf5,
	0x85, 0x19, 0x46, 0xd2, 0xd9, 0x61, 0xe1, 0xec, 0x97, 0x17, 0xcc, 0x36, 0x01, 0xd5, 0x57, 0x51,
	0xd4, 0x00, 0x07, 0x0f, 0x2a, 0x98, 0xa9, 0x38, 0x9f, 0x4e, 0x7e, 0x42, 0x4d, 0x70, 0xb8, 0x0e,
	0x9e, 0x96, 0x51, 0x30, 0xcb, 0x27, 0x52, 0x1c, 0xdb, 0x7f, 0x96, 0x40, 0xc3, 0x7a, 0x08, 0x16,
	0xe1, 0x34, 0x9a, 0x29, 0x53, 0xc5, 0x33, 0x21, 0xf4, 0x13, 0x68, 0x4d, 0x8b, 0x88, 0xdc, 0x2c,
	0xb1, 0xa8, 0x63, 0x1a, 0x34, 0x37, 0x0c, 0x2f, 0x27, 0x14, 0xd9, 0x3f, 0x80, 0x03, 0x23, 0x2d,
	0xeb, 0x58, 0xbe, 0x7e, 0x57, 0x78, 0xda, 0x74, 0x23, 0xe1, 0x2c, 0x8a, 0x13, 0x35, 0xcb, 0x9d,
	0xe5, 0xf4, 0xf6, 0x1f, 0x25, 0x70, 0xfe, 0x3f, 0x1c, 0xf4, 0x23, 0xb8, 0x78, 0xf5, 0x36, 0xbd,
	0x50, 0x74, 0x5e, 0x10, 0x78, 0x1e, 0xdf, 0x0a, 0xaa, 0x28, 0x53, 0x6d, 0xa5, 0xc2, 0x34, 0x69,
	0xee, 0x65, 0xa3, 0xae, 0x15, 0xb2, 0xc8, 0x36, 0xc6, 0x9f, 0x11, 0x3f, 0xfc, 0xb5, 0x0f, 0xa0,
	0xf8, 0x6d, 0xf4, 0x6c, 0x85, 0xe8, 0x08, 0xec, 0x8f, 0xb0, 0xeb, 0xd8, 0xf0, 0x0d, 0x82, 0xa0,
	0x42, 0x1d, 0x57, 0x12, 0x3a, 0x22, 0x2e, 0xf3, 0x08, 0x2c, 0xa1, 0x53, 0x50, 0xee, 0x61, 0x5b,
	0x7a, 0xf8, 0xde, 0x65, 0xd8, 0x86, 0x7b, 0xe8, 0x0c, 0x54, 0x35, 0x60, 0xb1, 0xc1, 0x80, 0x51,
	0xd9, 0x27, 0xd8, 0x26, 0x1c, 0xbe, 0x45, 0x17, 0xe0, 0x2c, 0x83, 0x39, 0xc1, 0x82, 0x71, 0xe9,
	0x3b, 0xb7, 0x14, 0x8b, 0x21, 0x27, 0xf0, 0x2b, 0xf4, 0x1e, 0x5c, 0x3a, 0x34, 0xeb, 0x20, 0x09,
	0xb5, 0x19, 0xf7, 0x09, 0x97, 0x82, 0x63, 0xea, 0x63, 0x4b, 0x38, 0x8c, 0xc2, 0x7d, 0xf4, 0x0d,
	0x68, 0x15, 0x0c, 0x8b, 0xd1, 0x1b, 0xe7, 0xf6, 0x59, 0xfc, 0x00, 0xb5, 0x40, 0x63, 0x48, 0xfd,
	0xa1, 0xe7, 0x31, 0x2e, 0x88, 0x2d, 0xc5, 0x78, 0xa3, 0xe7, 0xb0, 0xd0, 0xe3, 0x71, 0xe6, 0x31,
	0x1f, 0xbb, 0x52, 0x8c, 0x1d, 0x1b, 0x7e, 0x8d, 0x10, 0x38, 0xb1, 0x87, 0x9e, 0xeb, 0x58, 0x58,
	0x10, 0x83, 0x1d, 0xe9, 0x36, 0xb9, 0x80, 0x01, 0xa1, 0x42, 0x7a, 0xcc, 0x75, 0xac, 0x7b, 0x79,
	0x83, 0x1d, 0x57, 0x0b, 0x05, 0xa8, 0x01, 0xd0, 0x60, 0x64, 0x59, 0x92, 0x13, 0x6c, 0x84, 0xb8,
	0x8e, 0x25, 0x60, 0x59, 0x7b, 0xf3, 0xfa, 0x98, 0x0a, 0x36, 0x78, 0x11, 0xaa, 0xa0, 0x1a, 0x38,
	0x1d, 0xd2, 0x4f, 0x94, 0xdd, 0x51, 0xad, 0x4a, 0xdc, 0x7b, 0x04, 0x1e, 0x6b, 0xb9, 0x02, 0xf3,
	0x5b, 0x22, 0xa4, 0xd5, 0xc7, 0x0e, 0x95, 0x94, 0x09, 0x79, 0xc3, 0x86, 0xd4, 0x86, 0x27, 0xa8,
	0x0e, 0xe0, 0x00, 0x73, 0xbf, 0x9f, 0x29, 0x95, 0x84, 0x73, 0xc6, 0xe1, 0x69, 0x31, 0x77, 0x31,
	0xce, 0x2d, 0x43, 0x6d, 0x8b, 0x8c, 0x3d, 0x87, 0x13, 0xdb, 0x14, 0xb1, 0x98, 0x4d, 0x60, 0x55,
	0x5b, 0xd8, 0x1c, 0xe5, 0x88, 0x70, 0xdf, 0x61, 0x74, 0xab, 0x07, 0xa1, 0x26, 0xa8, 0xeb, 0x69,
	0x98, 0xb5, 0x48, 0x32, 0x16, 0x84, 0x6a, 0x0a, 0xac, 0x69, 0x73, 0xd9, 0x82, 0xfa, 0x98, 0x52,
	0xe2, 0x16, 0x8b, 0xab, 0x17, 0x19, 0x9c, 0xf8, 0x1e, 0xa3, 0x3e, 0xd9, 0x4c, 0xf6, 0x0c, 0x1d,
	0x83, 0xa3, 0x2c, 0x72, 0xe7, 0x13, 0x01, 0x1b, 0x5a, 0xb9, 0xe3, 0xba, 0xe4, 0x16, 0xbb, 0xf2,
	0x8e, 0x3b, 0x82, 0x68, 0xf4, 0x3c, 0x43, 0xf3, 0xd5, 0x6d, 0xd0, 0x26, 0x42, 0xe0, 0x58, 0x9b,
	0xce, 0x70, 0x2c, 0x88, 0x0d, 0xff, 0x2e, 0xa1, 0x0b, 0x50, 0x2f, 0x98, 0x4c, 0xf4, 0x09, 0xd7,
	0xb3, 0xf4, 0x19, 0x85, 0xff, 0x94, 0x3e, 0x5c, 0x81, 0xca, 0x40, 0xa5, 0x81, 0x1d, 0xa4, 0xc1,
	0x27, 0xf5, 0x94, 0x68, 0x4d, 0x79, 0xaa, 0xb6, 0xe7, 0x61, 0x8e, 0x07, 0x44, 0x10, 0x0e, 0xdf,
	0xf4, 0xa6, 0xa0, 0x1d, 0xc5, 0xf3, 0xce, 0xc3, 0xd3, 0x5a, 0xc5, 0x4b, 0x35, 0x9b, 0xab, 0xb8,
	0xf3, 0x39, 0x98, 0xc4, 0x8b, 0x69, 0xf1, 0xee, 0xeb, 0x6b, 0xba, 0x87, 0x76, 0xae, 0x13, 0x2f,
	0x98, 0xfe, 0x1a, 0xcc, 0xd5, 0xcf, 0xdf, 0xcd, 0x17, 0xe9, 0xc3, 0xe3, 0x44, 0xdf, 0x7e, 0xdd,
	0x9d, 0xf4, 0xae, 0x49, 0x37, 0x17, 0x7f, 0xd2, 0xd5, 0xe9, 0x13, 0xf3, 0x51, 0xf8, 0xf8, 0x6f,
	0x00, 0x00, 0x00, 0xff, 0xff, 0xba, 0xd3, 0xb2, 0x32, 0x35, 0x06, 0x00, 0x00,
}
