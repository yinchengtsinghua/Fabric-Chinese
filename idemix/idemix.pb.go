
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//由Protoc Gen Go生成的代码。不要编辑。
//来源：idemix/idemix.proto

package idemix //导入“github.com/hyperledger/fabric/idemix”

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

//ECP是由其坐标指定的椭圆曲线点。
//ECP对应于第一组（g1）的一个元素。
type ECP struct {
	X                    []byte   `protobuf:"bytes,1,opt,name=x,proto3" json:"x,omitempty"`
	Y                    []byte   `protobuf:"bytes,2,opt,name=y,proto3" json:"y,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ECP) Reset()         { *m = ECP{} }
func (m *ECP) String() string { return proto.CompactTextString(m) }
func (*ECP) ProtoMessage()    {}
func (*ECP) Descriptor() ([]byte, []int) {
	return fileDescriptor_idemix_ea623f6980eee47e, []int{0}
}
func (m *ECP) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ECP.Unmarshal(m, b)
}
func (m *ECP) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ECP.Marshal(b, m, deterministic)
}
func (dst *ECP) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ECP.Merge(dst, src)
}
func (m *ECP) XXX_Size() int {
	return xxx_messageInfo_ECP.Size(m)
}
func (m *ECP) XXX_DiscardUnknown() {
	xxx_messageInfo_ECP.DiscardUnknown(m)
}

var xxx_messageInfo_ECP proto.InternalMessageInfo

func (m *ECP) GetX() []byte {
	if m != nil {
		return m.X
	}
	return nil
}

func (m *ECP) GetY() []byte {
	if m != nil {
		return m.Y
	}
	return nil
}

//ECP2是由其坐标指定的椭圆曲线点。
//ECP2对应于第二组（G2）的一个元素。
type ECP2 struct {
	Xa                   []byte   `protobuf:"bytes,1,opt,name=xa,proto3" json:"xa,omitempty"`
	Xb                   []byte   `protobuf:"bytes,2,opt,name=xb,proto3" json:"xb,omitempty"`
	Ya                   []byte   `protobuf:"bytes,3,opt,name=ya,proto3" json:"ya,omitempty"`
	Yb                   []byte   `protobuf:"bytes,4,opt,name=yb,proto3" json:"yb,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ECP2) Reset()         { *m = ECP2{} }
func (m *ECP2) String() string { return proto.CompactTextString(m) }
func (*ECP2) ProtoMessage()    {}
func (*ECP2) Descriptor() ([]byte, []int) {
	return fileDescriptor_idemix_ea623f6980eee47e, []int{1}
}
func (m *ECP2) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ECP2.Unmarshal(m, b)
}
func (m *ECP2) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ECP2.Marshal(b, m, deterministic)
}
func (dst *ECP2) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ECP2.Merge(dst, src)
}
func (m *ECP2) XXX_Size() int {
	return xxx_messageInfo_ECP2.Size(m)
}
func (m *ECP2) XXX_DiscardUnknown() {
	xxx_messageInfo_ECP2.DiscardUnknown(m)
}

var xxx_messageInfo_ECP2 proto.InternalMessageInfo

func (m *ECP2) GetXa() []byte {
	if m != nil {
		return m.Xa
	}
	return nil
}

func (m *ECP2) GetXb() []byte {
	if m != nil {
		return m.Xb
	}
	return nil
}

func (m *ECP2) GetYa() []byte {
	if m != nil {
		return m.Ya
	}
	return nil
}

func (m *ECP2) GetYb() []byte {
	if m != nil {
		return m.Yb
	}
	return nil
}

//IssuerPublickey指定由以下内容组成的颁发者公钥
//属性名称-由颁发者颁发的凭证的属性名称列表
//h_sk、h_rand、h_attrs、w、bar_g1、bar_g2-与签名键、随机性和属性相对应的组元素
//proof_c，proof_s组成一个零知识的密钥知识证明
//哈希是附加到它的公钥的哈希
type IssuerPublicKey struct {
	AttributeNames       []string `protobuf:"bytes,1,rep,name=attribute_names,json=attributeNames,proto3" json:"attribute_names,omitempty"`
	HSk                  *ECP     `protobuf:"bytes,2,opt,name=h_sk,json=hSk,proto3" json:"h_sk,omitempty"`
	HRand                *ECP     `protobuf:"bytes,3,opt,name=h_rand,json=hRand,proto3" json:"h_rand,omitempty"`
	HAttrs               []*ECP   `protobuf:"bytes,4,rep,name=h_attrs,json=hAttrs,proto3" json:"h_attrs,omitempty"`
	W                    *ECP2    `protobuf:"bytes,5,opt,name=w,proto3" json:"w,omitempty"`
	BarG1                *ECP     `protobuf:"bytes,6,opt,name=bar_g1,json=barG1,proto3" json:"bar_g1,omitempty"`
	BarG2                *ECP     `protobuf:"bytes,7,opt,name=bar_g2,json=barG2,proto3" json:"bar_g2,omitempty"`
	ProofC               []byte   `protobuf:"bytes,8,opt,name=proof_c,json=proofC,proto3" json:"proof_c,omitempty"`
	ProofS               []byte   `protobuf:"bytes,9,opt,name=proof_s,json=proofS,proto3" json:"proof_s,omitempty"`
	Hash                 []byte   `protobuf:"bytes,10,opt,name=hash,proto3" json:"hash,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *IssuerPublicKey) Reset()         { *m = IssuerPublicKey{} }
func (m *IssuerPublicKey) String() string { return proto.CompactTextString(m) }
func (*IssuerPublicKey) ProtoMessage()    {}
func (*IssuerPublicKey) Descriptor() ([]byte, []int) {
	return fileDescriptor_idemix_ea623f6980eee47e, []int{2}
}
func (m *IssuerPublicKey) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IssuerPublicKey.Unmarshal(m, b)
}
func (m *IssuerPublicKey) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IssuerPublicKey.Marshal(b, m, deterministic)
}
func (dst *IssuerPublicKey) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IssuerPublicKey.Merge(dst, src)
}
func (m *IssuerPublicKey) XXX_Size() int {
	return xxx_messageInfo_IssuerPublicKey.Size(m)
}
func (m *IssuerPublicKey) XXX_DiscardUnknown() {
	xxx_messageInfo_IssuerPublicKey.DiscardUnknown(m)
}

var xxx_messageInfo_IssuerPublicKey proto.InternalMessageInfo

func (m *IssuerPublicKey) GetAttributeNames() []string {
	if m != nil {
		return m.AttributeNames
	}
	return nil
}

func (m *IssuerPublicKey) GetHSk() *ECP {
	if m != nil {
		return m.HSk
	}
	return nil
}

func (m *IssuerPublicKey) GetHRand() *ECP {
	if m != nil {
		return m.HRand
	}
	return nil
}

func (m *IssuerPublicKey) GetHAttrs() []*ECP {
	if m != nil {
		return m.HAttrs
	}
	return nil
}

func (m *IssuerPublicKey) GetW() *ECP2 {
	if m != nil {
		return m.W
	}
	return nil
}

func (m *IssuerPublicKey) GetBarG1() *ECP {
	if m != nil {
		return m.BarG1
	}
	return nil
}

func (m *IssuerPublicKey) GetBarG2() *ECP {
	if m != nil {
		return m.BarG2
	}
	return nil
}

func (m *IssuerPublicKey) GetProofC() []byte {
	if m != nil {
		return m.ProofC
	}
	return nil
}

func (m *IssuerPublicKey) GetProofS() []byte {
	if m != nil {
		return m.ProofS
	}
	return nil
}

func (m *IssuerPublicKey) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

//issuer key指定一个由
//isk-颁发者密钥和
//IssuerPublickey-颁发者公钥
type IssuerKey struct {
	Isk                  []byte           `protobuf:"bytes,1,opt,name=isk,proto3" json:"isk,omitempty"`
	Ipk                  *IssuerPublicKey `protobuf:"bytes,2,opt,name=ipk,proto3" json:"ipk,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *IssuerKey) Reset()         { *m = IssuerKey{} }
func (m *IssuerKey) String() string { return proto.CompactTextString(m) }
func (*IssuerKey) ProtoMessage()    {}
func (*IssuerKey) Descriptor() ([]byte, []int) {
	return fileDescriptor_idemix_ea623f6980eee47e, []int{3}
}
func (m *IssuerKey) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IssuerKey.Unmarshal(m, b)
}
func (m *IssuerKey) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IssuerKey.Marshal(b, m, deterministic)
}
func (dst *IssuerKey) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IssuerKey.Merge(dst, src)
}
func (m *IssuerKey) XXX_Size() int {
	return xxx_messageInfo_IssuerKey.Size(m)
}
func (m *IssuerKey) XXX_DiscardUnknown() {
	xxx_messageInfo_IssuerKey.DiscardUnknown(m)
}

var xxx_messageInfo_IssuerKey proto.InternalMessageInfo

func (m *IssuerKey) GetIsk() []byte {
	if m != nil {
		return m.Isk
	}
	return nil
}

func (m *IssuerKey) GetIpk() *IssuerPublicKey {
	if m != nil {
		return m.Ipk
	}
	return nil
}

//Credential指定一个由
//A、B、E、S—签名值
//attrs-属性值
type Credential struct {
	A                    *ECP     `protobuf:"bytes,1,opt,name=a,proto3" json:"a,omitempty"`
	B                    *ECP     `protobuf:"bytes,2,opt,name=b,proto3" json:"b,omitempty"`
	E                    []byte   `protobuf:"bytes,3,opt,name=e,proto3" json:"e,omitempty"`
	S                    []byte   `protobuf:"bytes,4,opt,name=s,proto3" json:"s,omitempty"`
	Attrs                [][]byte `protobuf:"bytes,5,rep,name=attrs,proto3" json:"attrs,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Credential) Reset()         { *m = Credential{} }
func (m *Credential) String() string { return proto.CompactTextString(m) }
func (*Credential) ProtoMessage()    {}
func (*Credential) Descriptor() ([]byte, []int) {
	return fileDescriptor_idemix_ea623f6980eee47e, []int{4}
}
func (m *Credential) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Credential.Unmarshal(m, b)
}
func (m *Credential) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Credential.Marshal(b, m, deterministic)
}
func (dst *Credential) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Credential.Merge(dst, src)
}
func (m *Credential) XXX_Size() int {
	return xxx_messageInfo_Credential.Size(m)
}
func (m *Credential) XXX_DiscardUnknown() {
	xxx_messageInfo_Credential.DiscardUnknown(m)
}

var xxx_messageInfo_Credential proto.InternalMessageInfo

func (m *Credential) GetA() *ECP {
	if m != nil {
		return m.A
	}
	return nil
}

func (m *Credential) GetB() *ECP {
	if m != nil {
		return m.B
	}
	return nil
}

func (m *Credential) GetE() []byte {
	if m != nil {
		return m.E
	}
	return nil
}

func (m *Credential) GetS() []byte {
	if m != nil {
		return m.S
	}
	return nil
}

func (m *Credential) GetAttrs() [][]byte {
	if m != nil {
		return m.Attrs
	}
	return nil
}

//CredRequest指定一个凭据请求对象，该对象由
//Nym-一个假名，它是对用户秘密的承诺。
//颁发者提供的随机非颁发者提供的非颁发者
//证明，证明
//Nym内的用户秘密
type CredRequest struct {
	Nym                  *ECP     `protobuf:"bytes,1,opt,name=nym,proto3" json:"nym,omitempty"`
	IssuerNonce          []byte   `protobuf:"bytes,2,opt,name=issuer_nonce,json=issuerNonce,proto3" json:"issuer_nonce,omitempty"`
	ProofC               []byte   `protobuf:"bytes,3,opt,name=proof_c,json=proofC,proto3" json:"proof_c,omitempty"`
	ProofS               []byte   `protobuf:"bytes,4,opt,name=proof_s,json=proofS,proto3" json:"proof_s,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CredRequest) Reset()         { *m = CredRequest{} }
func (m *CredRequest) String() string { return proto.CompactTextString(m) }
func (*CredRequest) ProtoMessage()    {}
func (*CredRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_idemix_ea623f6980eee47e, []int{5}
}
func (m *CredRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CredRequest.Unmarshal(m, b)
}
func (m *CredRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CredRequest.Marshal(b, m, deterministic)
}
func (dst *CredRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CredRequest.Merge(dst, src)
}
func (m *CredRequest) XXX_Size() int {
	return xxx_messageInfo_CredRequest.Size(m)
}
func (m *CredRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CredRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CredRequest proto.InternalMessageInfo

func (m *CredRequest) GetNym() *ECP {
	if m != nil {
		return m.Nym
	}
	return nil
}

func (m *CredRequest) GetIssuerNonce() []byte {
	if m != nil {
		return m.IssuerNonce
	}
	return nil
}

func (m *CredRequest) GetProofC() []byte {
	if m != nil {
		return m.ProofC
	}
	return nil
}

func (m *CredRequest) GetProofS() []byte {
	if m != nil {
		return m.ProofS
	}
	return nil
}

//签名指定一个签名对象，该对象由
//一个主、一个条、一个主、证明随机凭证签名值
//一个零知识的凭证
//以及相应的用户秘密和属性值
//nonce-用于签名的新nonce
//新的笔名（对用户秘密的承诺）
type Signature struct {
	APrime               *ECP                `protobuf:"bytes,1,opt,name=a_prime,json=aPrime,proto3" json:"a_prime,omitempty"`
	ABar                 *ECP                `protobuf:"bytes,2,opt,name=a_bar,json=aBar,proto3" json:"a_bar,omitempty"`
	BPrime               *ECP                `protobuf:"bytes,3,opt,name=b_prime,json=bPrime,proto3" json:"b_prime,omitempty"`
	ProofC               []byte              `protobuf:"bytes,4,opt,name=proof_c,json=proofC,proto3" json:"proof_c,omitempty"`
	ProofSSk             []byte              `protobuf:"bytes,5,opt,name=proof_s_sk,json=proofSSk,proto3" json:"proof_s_sk,omitempty"`
	ProofSE              []byte              `protobuf:"bytes,6,opt,name=proof_s_e,json=proofSE,proto3" json:"proof_s_e,omitempty"`
	ProofSR2             []byte              `protobuf:"bytes,7,opt,name=proof_s_r2,json=proofSR2,proto3" json:"proof_s_r2,omitempty"`
	ProofSR3             []byte              `protobuf:"bytes,8,opt,name=proof_s_r3,json=proofSR3,proto3" json:"proof_s_r3,omitempty"`
	ProofSSPrime         []byte              `protobuf:"bytes,9,opt,name=proof_s_s_prime,json=proofSSPrime,proto3" json:"proof_s_s_prime,omitempty"`
	ProofSAttrs          [][]byte            `protobuf:"bytes,10,rep,name=proof_s_attrs,json=proofSAttrs,proto3" json:"proof_s_attrs,omitempty"`
	Nonce                []byte              `protobuf:"bytes,11,opt,name=nonce,proto3" json:"nonce,omitempty"`
	Nym                  *ECP                `protobuf:"bytes,12,opt,name=nym,proto3" json:"nym,omitempty"`
	ProofSRNym           []byte              `protobuf:"bytes,13,opt,name=proof_s_r_nym,json=proofSRNym,proto3" json:"proof_s_r_nym,omitempty"`
	RevocationEpochPk    *ECP2               `protobuf:"bytes,14,opt,name=revocation_epoch_pk,json=revocationEpochPk,proto3" json:"revocation_epoch_pk,omitempty"`
	RevocationPkSig      []byte              `protobuf:"bytes,15,opt,name=revocation_pk_sig,json=revocationPkSig,proto3" json:"revocation_pk_sig,omitempty"`
	Epoch                int64               `protobuf:"varint,16,opt,name=epoch,proto3" json:"epoch,omitempty"`
	NonRevocationProof   *NonRevocationProof `protobuf:"bytes,17,opt,name=non_revocation_proof,json=nonRevocationProof,proto3" json:"non_revocation_proof,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *Signature) Reset()         { *m = Signature{} }
func (m *Signature) String() string { return proto.CompactTextString(m) }
func (*Signature) ProtoMessage()    {}
func (*Signature) Descriptor() ([]byte, []int) {
	return fileDescriptor_idemix_ea623f6980eee47e, []int{6}
}
func (m *Signature) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Signature.Unmarshal(m, b)
}
func (m *Signature) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Signature.Marshal(b, m, deterministic)
}
func (dst *Signature) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Signature.Merge(dst, src)
}
func (m *Signature) XXX_Size() int {
	return xxx_messageInfo_Signature.Size(m)
}
func (m *Signature) XXX_DiscardUnknown() {
	xxx_messageInfo_Signature.DiscardUnknown(m)
}

var xxx_messageInfo_Signature proto.InternalMessageInfo

func (m *Signature) GetAPrime() *ECP {
	if m != nil {
		return m.APrime
	}
	return nil
}

func (m *Signature) GetABar() *ECP {
	if m != nil {
		return m.ABar
	}
	return nil
}

func (m *Signature) GetBPrime() *ECP {
	if m != nil {
		return m.BPrime
	}
	return nil
}

func (m *Signature) GetProofC() []byte {
	if m != nil {
		return m.ProofC
	}
	return nil
}

func (m *Signature) GetProofSSk() []byte {
	if m != nil {
		return m.ProofSSk
	}
	return nil
}

func (m *Signature) GetProofSE() []byte {
	if m != nil {
		return m.ProofSE
	}
	return nil
}

func (m *Signature) GetProofSR2() []byte {
	if m != nil {
		return m.ProofSR2
	}
	return nil
}

func (m *Signature) GetProofSR3() []byte {
	if m != nil {
		return m.ProofSR3
	}
	return nil
}

func (m *Signature) GetProofSSPrime() []byte {
	if m != nil {
		return m.ProofSSPrime
	}
	return nil
}

func (m *Signature) GetProofSAttrs() [][]byte {
	if m != nil {
		return m.ProofSAttrs
	}
	return nil
}

func (m *Signature) GetNonce() []byte {
	if m != nil {
		return m.Nonce
	}
	return nil
}

func (m *Signature) GetNym() *ECP {
	if m != nil {
		return m.Nym
	}
	return nil
}

func (m *Signature) GetProofSRNym() []byte {
	if m != nil {
		return m.ProofSRNym
	}
	return nil
}

func (m *Signature) GetRevocationEpochPk() *ECP2 {
	if m != nil {
		return m.RevocationEpochPk
	}
	return nil
}

func (m *Signature) GetRevocationPkSig() []byte {
	if m != nil {
		return m.RevocationPkSig
	}
	return nil
}

func (m *Signature) GetEpoch() int64 {
	if m != nil {
		return m.Epoch
	}
	return 0
}

func (m *Signature) GetNonRevocationProof() *NonRevocationProof {
	if m != nil {
		return m.NonRevocationProof
	}
	return nil
}

//非吊销证明包含凭证未被吊销的证明
type NonRevocationProof struct {
	RevocationAlg        int32    `protobuf:"varint,1,opt,name=revocation_alg,json=revocationAlg,proto3" json:"revocation_alg,omitempty"`
	NonRevocationProof   []byte   `protobuf:"bytes,2,opt,name=non_revocation_proof,json=nonRevocationProof,proto3" json:"non_revocation_proof,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NonRevocationProof) Reset()         { *m = NonRevocationProof{} }
func (m *NonRevocationProof) String() string { return proto.CompactTextString(m) }
func (*NonRevocationProof) ProtoMessage()    {}
func (*NonRevocationProof) Descriptor() ([]byte, []int) {
	return fileDescriptor_idemix_ea623f6980eee47e, []int{7}
}
func (m *NonRevocationProof) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NonRevocationProof.Unmarshal(m, b)
}
func (m *NonRevocationProof) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NonRevocationProof.Marshal(b, m, deterministic)
}
func (dst *NonRevocationProof) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NonRevocationProof.Merge(dst, src)
}
func (m *NonRevocationProof) XXX_Size() int {
	return xxx_messageInfo_NonRevocationProof.Size(m)
}
func (m *NonRevocationProof) XXX_DiscardUnknown() {
	xxx_messageInfo_NonRevocationProof.DiscardUnknown(m)
}

var xxx_messageInfo_NonRevocationProof proto.InternalMessageInfo

func (m *NonRevocationProof) GetRevocationAlg() int32 {
	if m != nil {
		return m.RevocationAlg
	}
	return 0
}

func (m *NonRevocationProof) GetNonRevocationProof() []byte {
	if m != nil {
		return m.NonRevocationProof
	}
	return nil
}

//NymSignature指定对消息签名的签名对象
//关于笔名。它不同于标准的idemix.signature，事实上
//标准签名对象还证明了该笔名是基于
//CA（发行人），而NymSignature仅证明该假名的所有者
//在邮件上签名
type NymSignature struct {
//证明是菲亚特·沙米尔对ZKP的挑战
	ProofC []byte `protobuf:"bytes,1,opt,name=proof_c,json=proofC,proto3" json:"proof_c,omitempty"`
//proof_s_sk是用户密钥的s值证明知识
	ProofSSk []byte `protobuf:"bytes,2,opt,name=proof_s_sk,json=proofSSk,proto3" json:"proof_s_sk,omitempty"`
//证明化名是证明化名秘密的S值。
	ProofSRNym []byte `protobuf:"bytes,3,opt,name=proof_s_r_nym,json=proofSRNym,proto3" json:"proof_s_r_nym,omitempty"`
//nonce是用于签名的新nonce
	Nonce                []byte   `protobuf:"bytes,4,opt,name=nonce,proto3" json:"nonce,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NymSignature) Reset()         { *m = NymSignature{} }
func (m *NymSignature) String() string { return proto.CompactTextString(m) }
func (*NymSignature) ProtoMessage()    {}
func (*NymSignature) Descriptor() ([]byte, []int) {
	return fileDescriptor_idemix_ea623f6980eee47e, []int{8}
}
func (m *NymSignature) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NymSignature.Unmarshal(m, b)
}
func (m *NymSignature) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NymSignature.Marshal(b, m, deterministic)
}
func (dst *NymSignature) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NymSignature.Merge(dst, src)
}
func (m *NymSignature) XXX_Size() int {
	return xxx_messageInfo_NymSignature.Size(m)
}
func (m *NymSignature) XXX_DiscardUnknown() {
	xxx_messageInfo_NymSignature.DiscardUnknown(m)
}

var xxx_messageInfo_NymSignature proto.InternalMessageInfo

func (m *NymSignature) GetProofC() []byte {
	if m != nil {
		return m.ProofC
	}
	return nil
}

func (m *NymSignature) GetProofSSk() []byte {
	if m != nil {
		return m.ProofSSk
	}
	return nil
}

func (m *NymSignature) GetProofSRNym() []byte {
	if m != nil {
		return m.ProofSRNym
	}
	return nil
}

func (m *NymSignature) GetNonce() []byte {
	if m != nil {
		return m.Nonce
	}
	return nil
}

type CredentialRevocationInformation struct {
//epoch包含此CRI有效的epoch（时间窗口）
	Epoch int64 `protobuf:"varint,1,opt,name=epoch,proto3" json:"epoch,omitempty"`
//epoch_pk是吊销机构在这个时期使用的公钥。
	EpochPk *ECP2 `protobuf:"bytes,2,opt,name=epoch_pk,json=epochPk,proto3" json:"epoch_pk,omitempty"`
//epoch_pk_sig是epoch pk上的签名，在吊销机构的长期密钥下有效。
	EpochPkSig []byte `protobuf:"bytes,3,opt,name=epoch_pk_sig,json=epochPkSig,proto3" json:"epoch_pk_sig,omitempty"`
//revocation表示使用哪种撤销算法
	RevocationAlg int32 `protobuf:"varint,4,opt,name=revocation_alg,json=revocationAlg,proto3" json:"revocation_alg,omitempty"`
//吊销数据包含特定于所用吊销算法的数据
	RevocationData       []byte   `protobuf:"bytes,5,opt,name=revocation_data,json=revocationData,proto3" json:"revocation_data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CredentialRevocationInformation) Reset()         { *m = CredentialRevocationInformation{} }
func (m *CredentialRevocationInformation) String() string { return proto.CompactTextString(m) }
func (*CredentialRevocationInformation) ProtoMessage()    {}
func (*CredentialRevocationInformation) Descriptor() ([]byte, []int) {
	return fileDescriptor_idemix_ea623f6980eee47e, []int{9}
}
func (m *CredentialRevocationInformation) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CredentialRevocationInformation.Unmarshal(m, b)
}
func (m *CredentialRevocationInformation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CredentialRevocationInformation.Marshal(b, m, deterministic)
}
func (dst *CredentialRevocationInformation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CredentialRevocationInformation.Merge(dst, src)
}
func (m *CredentialRevocationInformation) XXX_Size() int {
	return xxx_messageInfo_CredentialRevocationInformation.Size(m)
}
func (m *CredentialRevocationInformation) XXX_DiscardUnknown() {
	xxx_messageInfo_CredentialRevocationInformation.DiscardUnknown(m)
}

var xxx_messageInfo_CredentialRevocationInformation proto.InternalMessageInfo

func (m *CredentialRevocationInformation) GetEpoch() int64 {
	if m != nil {
		return m.Epoch
	}
	return 0
}

func (m *CredentialRevocationInformation) GetEpochPk() *ECP2 {
	if m != nil {
		return m.EpochPk
	}
	return nil
}

func (m *CredentialRevocationInformation) GetEpochPkSig() []byte {
	if m != nil {
		return m.EpochPkSig
	}
	return nil
}

func (m *CredentialRevocationInformation) GetRevocationAlg() int32 {
	if m != nil {
		return m.RevocationAlg
	}
	return 0
}

func (m *CredentialRevocationInformation) GetRevocationData() []byte {
	if m != nil {
		return m.RevocationData
	}
	return nil
}

func init() {
	proto.RegisterType((*ECP)(nil), "ECP")
	proto.RegisterType((*ECP2)(nil), "ECP2")
	proto.RegisterType((*IssuerPublicKey)(nil), "IssuerPublicKey")
	proto.RegisterType((*IssuerKey)(nil), "IssuerKey")
	proto.RegisterType((*Credential)(nil), "Credential")
	proto.RegisterType((*CredRequest)(nil), "CredRequest")
	proto.RegisterType((*Signature)(nil), "Signature")
	proto.RegisterType((*NonRevocationProof)(nil), "NonRevocationProof")
	proto.RegisterType((*NymSignature)(nil), "NymSignature")
	proto.RegisterType((*CredentialRevocationInformation)(nil), "CredentialRevocationInformation")
}

func init() { proto.RegisterFile("idemix/idemix.proto", fileDescriptor_idemix_ea623f6980eee47e) }

var fileDescriptor_idemix_ea623f6980eee47e = []byte{
//gzip文件描述符或协议的816字节
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x55, 0xdd, 0x6e, 0xe2, 0x46,
	0x14, 0xd6, 0x60, 0x9b, 0x84, 0x83, 0x13, 0xb2, 0x93, 0xa8, 0x3b, 0xfd, 0x53, 0x59, 0xab, 0xdb,
	0x8d, 0x7a, 0x41, 0xba, 0x44, 0x7d, 0x80, 0x2c, 0xa5, 0xd5, 0xaa, 0x12, 0x42, 0xe6, 0xae, 0x37,
	0xa3, 0x19, 0x18, 0xb0, 0x05, 0xb6, 0xe9, 0xd8, 0x74, 0x71, 0x2f, 0xfa, 0x34, 0x7d, 0x9b, 0x5e,
	0xf4, 0x95, 0xaa, 0xf9, 0x01, 0x0f, 0x61, 0xb7, 0x57, 0xf1, 0x39, 0xdf, 0xf9, 0xe3, 0xfb, 0x3e,
	0xc7, 0x70, 0x9b, 0x2e, 0x44, 0x96, 0xee, 0x1f, 0xcc, 0x9f, 0xc1, 0x56, 0x16, 0x55, 0x11, 0xbd,
	0x02, 0x6f, 0x3c, 0x9a, 0xe2, 0x10, 0xd0, 0x9e, 0xa0, 0x3e, 0xba, 0x0f, 0x63, 0xb4, 0x57, 0x51,
	0x4d, 0x5a, 0x26, 0xaa, 0xa3, 0x9f, 0xc1, 0x1f, 0x8f, 0xa6, 0x43, 0x7c, 0x0d, 0xad, 0x3d, 0xb3,
	0x45, 0xad, 0x3d, 0xd3, 0x31, 0xb7, 0x65, 0xad, 0x3d, 0x57, 0x71, 0xcd, 0x88, 0x67, 0xe2, 0x5a,
	0xe3, 0x35, 0x27, 0xbe, 0x8d, 0x79, 0xf4, 0x77, 0x0b, 0x7a, 0xef, 0xcb, 0x72, 0x27, 0xe4, 0x74,
	0xc7, 0x37, 0xe9, 0xfc, 0x57, 0x51, 0xe3, 0x37, 0xd0, 0x63, 0x55, 0x25, 0x53, 0xbe, 0xab, 0x04,
	0xcd, 0x59, 0x26, 0x4a, 0x82, 0xfa, 0xde, 0x7d, 0x27, 0xbe, 0x3e, 0xa6, 0x27, 0x2a, 0x8b, 0x5f,
	0x82, 0x9f, 0xd0, 0x72, 0xad, 0xd7, 0x75, 0x87, 0xfe, 0x60, 0x3c, 0x9a, 0xc6, 0x5e, 0x32, 0x5b,
	0xe3, 0x2f, 0xa1, 0x9d, 0x50, 0xc9, 0xf2, 0x85, 0xde, 0x7c, 0x80, 0x82, 0x24, 0x66, 0xf9, 0x02,
	0x7f, 0x0d, 0x17, 0x09, 0x55, 0x93, 0x4a, 0xe2, 0xf7, 0xbd, 0x23, 0xda, 0x4e, 0x9e, 0x54, 0x0e,
	0xdf, 0x02, 0xfa, 0x40, 0x02, 0xdd, 0x16, 0x28, 0x60, 0x18, 0xa3, 0x0f, 0x6a, 0x20, 0x67, 0x92,
	0xae, 0xde, 0x92, 0xb6, 0x3b, 0x90, 0x33, 0xf9, 0xcb, 0xdb, 0x23, 0x38, 0x24, 0x17, 0xcf, 0xc1,
	0x21, 0x7e, 0x09, 0x17, 0x5b, 0x59, 0x14, 0x4b, 0x3a, 0x27, 0x97, 0xfa, 0x57, 0xb7, 0x75, 0x38,
	0x6a, 0x80, 0x92, 0x74, 0x1c, 0x60, 0x86, 0x31, 0xf8, 0x09, 0x2b, 0x13, 0x02, 0x3a, 0xab, 0x9f,
	0xa3, 0x27, 0xe8, 0x18, 0x96, 0x14, 0x3f, 0x37, 0xe0, 0xa5, 0xe5, 0xda, 0x92, 0xae, 0x1e, 0x71,
	0x04, 0x5e, 0xba, 0x3d, 0xf0, 0x70, 0x33, 0x78, 0x46, 0x68, 0xac, 0xc0, 0x68, 0x09, 0x30, 0x92,
	0x62, 0x21, 0xf2, 0x2a, 0x65, 0x1b, 0x8c, 0x01, 0x19, 0xd9, 0x0e, 0xe7, 0x22, 0xa6, 0x72, 0xfc,
	0x84, 0x4b, 0xc4, 0x95, 0xea, 0xc2, 0xca, 0x87, 0x84, 0x8a, 0x4a, 0x2b, 0x1e, 0x2a, 0xf1, 0x1d,
	0x04, 0x86, 0xc6, 0xa0, 0xef, 0xdd, 0x87, 0xb1, 0x09, 0xa2, 0x3f, 0xa1, 0xab, 0xf6, 0xc4, 0xe2,
	0xf7, 0x9d, 0x28, 0x2b, 0xfc, 0x19, 0x78, 0x79, 0x9d, 0x9d, 0xac, 0x52, 0x09, 0xfc, 0x0a, 0xc2,
	0x54, 0x9f, 0x49, 0xf3, 0x22, 0x9f, 0x0b, 0x6b, 0x99, 0xae, 0xc9, 0x4d, 0x54, 0xca, 0xa5, 0xce,
	0xfb, 0x14, 0x75, 0xbe, 0x4b, 0x5d, 0xf4, 0xaf, 0x0f, 0x9d, 0x59, 0xba, 0xca, 0x59, 0xb5, 0x93,
	0x42, 0x09, 0xcd, 0xe8, 0x56, 0xa6, 0x99, 0x38, 0x59, 0xdf, 0x66, 0x53, 0x95, 0xc3, 0x9f, 0x43,
	0xc0, 0x28, 0x67, 0xf2, 0xe4, 0x27, 0xfb, 0xec, 0x1d, 0x93, 0xaa, 0x93, 0xdb, 0x4e, 0xd7, 0x40,
	0x6d, 0x6e, 0x3a, 0x9d, 0xc3, 0xfc, 0x93, 0xc3, 0xbe, 0x02, 0xb0, 0x87, 0x29, 0x5b, 0x06, 0x1a,
	0xbb, 0x34, 0xb7, 0xcd, 0xd6, 0xf8, 0x0b, 0xe8, 0x1c, 0x50, 0xa1, 0x7d, 0x14, 0xc6, 0x66, 0xce,
	0x6c, 0xec, 0x76, 0x4a, 0xe3, 0xa3, 0x63, 0x67, 0x3c, 0x3c, 0x41, 0x1f, 0xad, 0x8f, 0x0e, 0xe8,
	0x23, 0x7e, 0x0d, 0xbd, 0xe3, 0x56, 0x7b, 0xb5, 0x71, 0x54, 0x68, 0x57, 0x9b, 0xab, 0x23, 0xb8,
	0x3a, 0x94, 0x19, 0xd9, 0x40, 0xcb, 0xd6, 0x35, 0x45, 0xc6, 0xfc, 0x77, 0x10, 0x18, 0x39, 0xba,
	0x7a, 0x80, 0x09, 0x0e, 0x1a, 0x86, 0xe7, 0x1a, 0x1e, 0x27, 0x4a, 0xaa, 0x2a, 0xae, 0x74, 0x17,
	0xd8, 0xcb, 0x26, 0x75, 0x86, 0x7f, 0x84, 0x5b, 0x29, 0xfe, 0x28, 0xe6, 0xac, 0x4a, 0x8b, 0x9c,
	0x8a, 0x6d, 0x31, 0x4f, 0xe8, 0x76, 0x4d, 0xae, 0xdd, 0xf7, 0xeb, 0x45, 0x53, 0x31, 0x56, 0x05,
	0xd3, 0x35, 0xfe, 0x1e, 0x9c, 0x24, 0xdd, 0xae, 0x69, 0x99, 0xae, 0x48, 0x4f, 0x4f, 0xef, 0x35,
	0xc0, 0x74, 0x3d, 0x4b, 0x57, 0xea, 0x66, 0x3d, 0x97, 0xdc, 0xf4, 0xd1, 0xbd, 0x17, 0x9b, 0x00,
	0x8f, 0xe1, 0x2e, 0x2f, 0x72, 0xea, 0x4e, 0x51, 0x57, 0x91, 0x17, 0x7a, 0xf3, 0xed, 0x60, 0x52,
	0xe4, 0x71, 0x33, 0x48, 0x41, 0x31, 0xce, 0xcf, 0x72, 0x51, 0x06, 0xf8, 0xbc, 0x12, 0xbf, 0x86,
	0x6b, 0x67, 0x30, 0xdb, 0xac, 0xb4, 0xc1, 0x82, 0xf8, 0xaa, 0xc9, 0x3e, 0x6d, 0x56, 0xf8, 0x87,
	0x4f, 0xdc, 0x60, 0xbc, 0xfe, 0xb1, 0x75, 0x7f, 0x41, 0x38, 0xa9, 0xb3, 0xc6, 0xc2, 0x8e, 0xd3,
	0xd0, 0xff, 0x38, 0xad, 0xf5, 0xcc, 0x69, 0x67, 0xc2, 0x78, 0x67, 0xc2, 0x1c, 0x95, 0xf6, 0x1d,
	0xa5, 0xa3, 0x7f, 0x10, 0x7c, 0xd3, 0xfc, 0x97, 0x68, 0xae, 0x7b, 0x9f, 0x2f, 0x0b, 0x99, 0xe9,
	0xc7, 0x86, 0x6f, 0xe4, 0xf2, 0xdd, 0x87, 0xcb, 0xa3, 0xba, 0x2d, 0x57, 0xdd, 0x0b, 0x61, 0x35,
	0xed, 0x43, 0x78, 0xa8, 0xd0, 0x72, 0xda, 0x9b, 0x2c, 0xac, 0x94, 0x3c, 0xa7, 0xd5, 0xff, 0x18,
	0xad, 0x6f, 0xc0, 0xf1, 0x00, 0x5d, 0xb0, 0x8a, 0xd9, 0x57, 0xcd, 0xe9, 0xfe, 0x89, 0x55, 0xec,
	0xdd, 0x77, 0xbf, 0x7d, 0xbb, 0x4a, 0xab, 0x64, 0xc7, 0x07, 0xf3, 0x22, 0x7b, 0x48, 0xea, 0xad,
	0x90, 0x1b, 0xb1, 0x58, 0x09, 0xf9, 0xb0, 0x64, 0x5c, 0xa6, 0x73, 0xfb, 0xd5, 0xe3, 0x6d, 0xfd,
	0xd9, 0x7b, 0xfc, 0x2f, 0x00, 0x00, 0xff, 0xff, 0xaa, 0xcc, 0x9c, 0x10, 0x0d, 0x07, 0x00, 0x00,
}
