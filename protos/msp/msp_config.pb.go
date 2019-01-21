
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//由Protoc Gen Go生成的代码。不要编辑。
//来源：msp/msp_config.proto

package msp //导入“github.com/hyperledger/fabric/protos/msp”

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

//mspconfig收集的所有配置信息
//一个MSP。配置字段应以某种方式取消编排
//那取决于类型
type MSPConfig struct {
//类型保存MSP的类型；默认类型将
//属于实现基于X.509的提供程序的结构类型
	Type int32 `protobuf:"varint,1,opt,name=type,proto3" json:"type,omitempty"`
//配置是与MSP相关的配置信息
	Config               []byte   `protobuf:"bytes,2,opt,name=config,proto3" json:"config,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MSPConfig) Reset()         { *m = MSPConfig{} }
func (m *MSPConfig) String() string { return proto.CompactTextString(m) }
func (*MSPConfig) ProtoMessage()    {}
func (*MSPConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_msp_config_e749e5bd1d6d997b, []int{0}
}
func (m *MSPConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MSPConfig.Unmarshal(m, b)
}
func (m *MSPConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MSPConfig.Marshal(b, m, deterministic)
}
func (dst *MSPConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MSPConfig.Merge(dst, src)
}
func (m *MSPConfig) XXX_Size() int {
	return xxx_messageInfo_MSPConfig.Size(m)
}
func (m *MSPConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_MSPConfig.DiscardUnknown(m)
}

var xxx_messageInfo_MSPConfig proto.InternalMessageInfo

func (m *MSPConfig) GetType() int32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *MSPConfig) GetConfig() []byte {
	if m != nil {
		return m.Config
	}
	return nil
}

//fabricmspconfig收集
//织物MSP。
//这里我们假设一个默认的证书验证策略，其中
//任何由列出的rootca证书签名的证书都将
//在本MSP下视为有效。
//此MSP可能附带签名标识，也可能不附带签名标识。如果确实如此，
//它还可以发布签名标识。如果没有，它只能
//用于验证证书。
type FabricMSPConfig struct {
//名称保留MSP的标识符；MSP标识符
//由管理此MSP的应用程序选择。
//例如，假设MSP的默认实现，
//基于X.509，考虑单个发行人，
//这可以引用主题ou字段或颁发者ou字段。
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
//此MSP信任的根证书列表
//它们在证书验证时使用（请参见
//以下中间证书注释）
	RootCerts [][]byte `protobuf:"bytes,2,rep,name=root_certs,json=rootCerts,proto3" json:"root_certs,omitempty"`
//此MSP信任的中间证书列表；
//它们在证书验证时使用，如下所示：
//验证尝试从证书生成路径
//待验证（位于路径一端）和
//rootcerts字段中的一个证书（位于
//路径的另一端）。如果路径长于
//2、中间的证书在
//中级证书池
	IntermediateCerts [][]byte `protobuf:"bytes,3,rep,name=intermediate_certs,json=intermediateCerts,proto3" json:"intermediate_certs,omitempty"`
//表示此MSP管理员的标识
	Admins [][]byte `protobuf:"bytes,4,rep,name=admins,proto3" json:"admins,omitempty"`
//身份吊销列表
	RevocationList [][]byte `protobuf:"bytes,5,rep,name=revocation_list,json=revocationList,proto3" json:"revocation_list,omitempty"`
//SigningIdentity保存有关签名标识的信息
//此对等机将被使用，并且将由
//之前定义的MSP
	SigningIdentity *SigningIdentityInfo `protobuf:"bytes,6,opt,name=signing_identity,json=signingIdentity,proto3" json:"signing_identity,omitempty"`
//OrganizationalUnitIdentifiers包含一个或多个
//属于的结构组织单位标识符
//此MSP配置
	OrganizationalUnitIdentifiers []*FabricOUIdentifier `protobuf:"bytes,7,rep,name=organizational_unit_identifiers,json=organizationalUnitIdentifiers,proto3" json:"organizational_unit_identifiers,omitempty"`
//FabricCryptoConfig包含配置参数
//对于此MSP使用的加密算法
	CryptoConfig *FabricCryptoConfig `protobuf:"bytes,8,opt,name=crypto_config,json=cryptoConfig,proto3" json:"crypto_config,omitempty"`
//此MSP信任的TLS根证书列表。
//它们由gettlsrootcerts返回。
	TlsRootCerts [][]byte `protobuf:"bytes,9,rep,name=tls_root_certs,json=tlsRootCerts,proto3" json:"tls_root_certs,omitempty"`
//此MSP信任的TLS中间证书列表；
//它们由gettlIntermediateCenter返回。
	TlsIntermediateCerts [][]byte `protobuf:"bytes,10,rep,name=tls_intermediate_certs,json=tlsIntermediateCerts,proto3" json:"tls_intermediate_certs,omitempty"`
//结构节点包含用于区分客户机和对等机与订购方的配置
//基于OU。
	FabricNodeOus        *FabricNodeOUs `protobuf:"bytes,11,opt,name=fabric_node_ous,json=fabricNodeOus,proto3" json:"fabric_node_ous,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *FabricMSPConfig) Reset()         { *m = FabricMSPConfig{} }
func (m *FabricMSPConfig) String() string { return proto.CompactTextString(m) }
func (*FabricMSPConfig) ProtoMessage()    {}
func (*FabricMSPConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_msp_config_e749e5bd1d6d997b, []int{1}
}
func (m *FabricMSPConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FabricMSPConfig.Unmarshal(m, b)
}
func (m *FabricMSPConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FabricMSPConfig.Marshal(b, m, deterministic)
}
func (dst *FabricMSPConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FabricMSPConfig.Merge(dst, src)
}
func (m *FabricMSPConfig) XXX_Size() int {
	return xxx_messageInfo_FabricMSPConfig.Size(m)
}
func (m *FabricMSPConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_FabricMSPConfig.DiscardUnknown(m)
}

var xxx_messageInfo_FabricMSPConfig proto.InternalMessageInfo

func (m *FabricMSPConfig) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *FabricMSPConfig) GetRootCerts() [][]byte {
	if m != nil {
		return m.RootCerts
	}
	return nil
}

func (m *FabricMSPConfig) GetIntermediateCerts() [][]byte {
	if m != nil {
		return m.IntermediateCerts
	}
	return nil
}

func (m *FabricMSPConfig) GetAdmins() [][]byte {
	if m != nil {
		return m.Admins
	}
	return nil
}

func (m *FabricMSPConfig) GetRevocationList() [][]byte {
	if m != nil {
		return m.RevocationList
	}
	return nil
}

func (m *FabricMSPConfig) GetSigningIdentity() *SigningIdentityInfo {
	if m != nil {
		return m.SigningIdentity
	}
	return nil
}

func (m *FabricMSPConfig) GetOrganizationalUnitIdentifiers() []*FabricOUIdentifier {
	if m != nil {
		return m.OrganizationalUnitIdentifiers
	}
	return nil
}

func (m *FabricMSPConfig) GetCryptoConfig() *FabricCryptoConfig {
	if m != nil {
		return m.CryptoConfig
	}
	return nil
}

func (m *FabricMSPConfig) GetTlsRootCerts() [][]byte {
	if m != nil {
		return m.TlsRootCerts
	}
	return nil
}

func (m *FabricMSPConfig) GetTlsIntermediateCerts() [][]byte {
	if m != nil {
		return m.TlsIntermediateCerts
	}
	return nil
}

func (m *FabricMSPConfig) GetFabricNodeOus() *FabricNodeOUs {
	if m != nil {
		return m.FabricNodeOus
	}
	return nil
}

//FabricCryptoConfig包含配置参数
//对于MSP使用的加密算法
//此配置引用
type FabricCryptoConfig struct {
//SignatureHashFamily是表示要使用的哈希系列的字符串
//在签名和验证操作期间。
//允许值为“sha2”和“sha3”。
	SignatureHashFamily string `protobuf:"bytes,1,opt,name=signature_hash_family,json=signatureHashFamily,proto3" json:"signature_hash_family,omitempty"`
//IdentityIdentifierHashFunction是表示哈希函数的字符串
//用于计算MSP标识的标识标识符。
//允许值为“sha256”、“sha384”和“sha3_256”、“sha3_384”。
	IdentityIdentifierHashFunction string   `protobuf:"bytes,2,opt,name=identity_identifier_hash_function,json=identityIdentifierHashFunction,proto3" json:"identity_identifier_hash_function,omitempty"`
	XXX_NoUnkeyedLiteral           struct{} `json:"-"`
	XXX_unrecognized               []byte   `json:"-"`
	XXX_sizecache                  int32    `json:"-"`
}

func (m *FabricCryptoConfig) Reset()         { *m = FabricCryptoConfig{} }
func (m *FabricCryptoConfig) String() string { return proto.CompactTextString(m) }
func (*FabricCryptoConfig) ProtoMessage()    {}
func (*FabricCryptoConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_msp_config_e749e5bd1d6d997b, []int{2}
}
func (m *FabricCryptoConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FabricCryptoConfig.Unmarshal(m, b)
}
func (m *FabricCryptoConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FabricCryptoConfig.Marshal(b, m, deterministic)
}
func (dst *FabricCryptoConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FabricCryptoConfig.Merge(dst, src)
}
func (m *FabricCryptoConfig) XXX_Size() int {
	return xxx_messageInfo_FabricCryptoConfig.Size(m)
}
func (m *FabricCryptoConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_FabricCryptoConfig.DiscardUnknown(m)
}

var xxx_messageInfo_FabricCryptoConfig proto.InternalMessageInfo

func (m *FabricCryptoConfig) GetSignatureHashFamily() string {
	if m != nil {
		return m.SignatureHashFamily
	}
	return ""
}

func (m *FabricCryptoConfig) GetIdentityIdentifierHashFunction() string {
	if m != nil {
		return m.IdentityIdentifierHashFunction
	}
	return ""
}

//idemixmspconfig收集
//一个IDEMIX MSP。
type IdemixMSPConfig struct {
//名称保存MSP的标识符
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
//IPK表示（序列化）颁发者公钥
	Ipk []byte `protobuf:"bytes,2,opt,name=ipk,proto3" json:"ipk,omitempty"`
//签名者可以包含加密材料来配置默认签名者
	Signer *IdemixMSPSignerConfig `protobuf:"bytes,3,opt,name=signer,proto3" json:"signer,omitempty"`
//吊销\u pk是用于吊销凭据的公钥
	RevocationPk []byte `protobuf:"bytes,4,opt,name=revocation_pk,json=revocationPk,proto3" json:"revocation_pk,omitempty"`
//epoch表示用于撤销的当前epoch（时间间隔）
	Epoch                int64    `protobuf:"varint,5,opt,name=epoch,proto3" json:"epoch,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *IdemixMSPConfig) Reset()         { *m = IdemixMSPConfig{} }
func (m *IdemixMSPConfig) String() string { return proto.CompactTextString(m) }
func (*IdemixMSPConfig) ProtoMessage()    {}
func (*IdemixMSPConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_msp_config_e749e5bd1d6d997b, []int{3}
}
func (m *IdemixMSPConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IdemixMSPConfig.Unmarshal(m, b)
}
func (m *IdemixMSPConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IdemixMSPConfig.Marshal(b, m, deterministic)
}
func (dst *IdemixMSPConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IdemixMSPConfig.Merge(dst, src)
}
func (m *IdemixMSPConfig) XXX_Size() int {
	return xxx_messageInfo_IdemixMSPConfig.Size(m)
}
func (m *IdemixMSPConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_IdemixMSPConfig.DiscardUnknown(m)
}

var xxx_messageInfo_IdemixMSPConfig proto.InternalMessageInfo

func (m *IdemixMSPConfig) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *IdemixMSPConfig) GetIpk() []byte {
	if m != nil {
		return m.Ipk
	}
	return nil
}

func (m *IdemixMSPConfig) GetSigner() *IdemixMSPSignerConfig {
	if m != nil {
		return m.Signer
	}
	return nil
}

func (m *IdemixMSPConfig) GetRevocationPk() []byte {
	if m != nil {
		return m.RevocationPk
	}
	return nil
}

func (m *IdemixMSPConfig) GetEpoch() int64 {
	if m != nil {
		return m.Epoch
	}
	return 0
}

//idemixmspsignerconfig包含用于设置idemix签名标识的加密材料
type IdemixMSPSignerConfig struct {
//cred表示默认签名者的序列化IDemix凭据
	Cred []byte `protobuf:"bytes,1,opt,name=cred,proto3" json:"cred,omitempty"`
//sk是默认签名者的密钥，对应于凭证凭证凭证
	Sk []byte `protobuf:"bytes,2,opt,name=sk,proto3" json:"sk,omitempty"`
//组织单元标识符定义默认签名者所在的组织单元
	OrganizationalUnitIdentifier string `protobuf:"bytes,3,opt,name=organizational_unit_identifier,json=organizationalUnitIdentifier,proto3" json:"organizational_unit_identifier,omitempty"`
//角色定义默认签名者是管理员、对等方、成员还是客户端
	Role int32 `protobuf:"varint,4,opt,name=role,proto3" json:"role,omitempty"`
//注册ID包含此签名者的注册ID
	EnrollmentId string `protobuf:"bytes,5,opt,name=enrollment_id,json=enrollmentId,proto3" json:"enrollment_id,omitempty"`
//凭证吊销信息包含一个系列化的凭证吊销信息
	CredentialRevocationInformation []byte   `protobuf:"bytes,6,opt,name=credential_revocation_information,json=credentialRevocationInformation,proto3" json:"credential_revocation_information,omitempty"`
	XXX_NoUnkeyedLiteral            struct{} `json:"-"`
	XXX_unrecognized                []byte   `json:"-"`
	XXX_sizecache                   int32    `json:"-"`
}

func (m *IdemixMSPSignerConfig) Reset()         { *m = IdemixMSPSignerConfig{} }
func (m *IdemixMSPSignerConfig) String() string { return proto.CompactTextString(m) }
func (*IdemixMSPSignerConfig) ProtoMessage()    {}
func (*IdemixMSPSignerConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_msp_config_e749e5bd1d6d997b, []int{4}
}
func (m *IdemixMSPSignerConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IdemixMSPSignerConfig.Unmarshal(m, b)
}
func (m *IdemixMSPSignerConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IdemixMSPSignerConfig.Marshal(b, m, deterministic)
}
func (dst *IdemixMSPSignerConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IdemixMSPSignerConfig.Merge(dst, src)
}
func (m *IdemixMSPSignerConfig) XXX_Size() int {
	return xxx_messageInfo_IdemixMSPSignerConfig.Size(m)
}
func (m *IdemixMSPSignerConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_IdemixMSPSignerConfig.DiscardUnknown(m)
}

var xxx_messageInfo_IdemixMSPSignerConfig proto.InternalMessageInfo

func (m *IdemixMSPSignerConfig) GetCred() []byte {
	if m != nil {
		return m.Cred
	}
	return nil
}

func (m *IdemixMSPSignerConfig) GetSk() []byte {
	if m != nil {
		return m.Sk
	}
	return nil
}

func (m *IdemixMSPSignerConfig) GetOrganizationalUnitIdentifier() string {
	if m != nil {
		return m.OrganizationalUnitIdentifier
	}
	return ""
}

func (m *IdemixMSPSignerConfig) GetRole() int32 {
	if m != nil {
		return m.Role
	}
	return 0
}

func (m *IdemixMSPSignerConfig) GetEnrollmentId() string {
	if m != nil {
		return m.EnrollmentId
	}
	return ""
}

func (m *IdemixMSPSignerConfig) GetCredentialRevocationInformation() []byte {
	if m != nil {
		return m.CredentialRevocationInformation
	}
	return nil
}

//signingIdentityInfo表示配置信息
//与对等方用于生成的签名标识相关
//赞同
type SigningIdentityInfo struct {
//公共签名者携带签名的公共信息
//身份。对于X.509提供商，这将由
//X.509证书
	PublicSigner []byte `protobuf:"bytes,1,opt,name=public_signer,json=publicSigner,proto3" json:"public_signer,omitempty"`
//privatesigner表示对
//对等签名身份
	PrivateSigner        *KeyInfo `protobuf:"bytes,2,opt,name=private_signer,json=privateSigner,proto3" json:"private_signer,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SigningIdentityInfo) Reset()         { *m = SigningIdentityInfo{} }
func (m *SigningIdentityInfo) String() string { return proto.CompactTextString(m) }
func (*SigningIdentityInfo) ProtoMessage()    {}
func (*SigningIdentityInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_msp_config_e749e5bd1d6d997b, []int{5}
}
func (m *SigningIdentityInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SigningIdentityInfo.Unmarshal(m, b)
}
func (m *SigningIdentityInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SigningIdentityInfo.Marshal(b, m, deterministic)
}
func (dst *SigningIdentityInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SigningIdentityInfo.Merge(dst, src)
}
func (m *SigningIdentityInfo) XXX_Size() int {
	return xxx_messageInfo_SigningIdentityInfo.Size(m)
}
func (m *SigningIdentityInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_SigningIdentityInfo.DiscardUnknown(m)
}

var xxx_messageInfo_SigningIdentityInfo proto.InternalMessageInfo

func (m *SigningIdentityInfo) GetPublicSigner() []byte {
	if m != nil {
		return m.PublicSigner
	}
	return nil
}

func (m *SigningIdentityInfo) GetPrivateSigner() *KeyInfo {
	if m != nil {
		return m.PrivateSigner
	}
	return nil
}

//keyinfo表示已经存储的（秘密）密钥
//在要导入到
//BCCSP密钥存储。在以后的版本中，它还可能包含
//密钥库标识符
type KeyInfo struct {
//默认密钥库中密钥的标识符；用于
//软件BCCSP和HSM BCCSP的情况是
//滑雪键
	KeyIdentifier string `protobuf:"bytes,1,opt,name=key_identifier,json=keyIdentifier,proto3" json:"key_identifier,omitempty"`
//要导入的密钥的密钥材料（可选）；这是
//正确编码的密钥字节，前缀为密钥类型
	KeyMaterial          []byte   `protobuf:"bytes,2,opt,name=key_material,json=keyMaterial,proto3" json:"key_material,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *KeyInfo) Reset()         { *m = KeyInfo{} }
func (m *KeyInfo) String() string { return proto.CompactTextString(m) }
func (*KeyInfo) ProtoMessage()    {}
func (*KeyInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_msp_config_e749e5bd1d6d997b, []int{6}
}
func (m *KeyInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_KeyInfo.Unmarshal(m, b)
}
func (m *KeyInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_KeyInfo.Marshal(b, m, deterministic)
}
func (dst *KeyInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KeyInfo.Merge(dst, src)
}
func (m *KeyInfo) XXX_Size() int {
	return xxx_messageInfo_KeyInfo.Size(m)
}
func (m *KeyInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_KeyInfo.DiscardUnknown(m)
}

var xxx_messageInfo_KeyInfo proto.InternalMessageInfo

func (m *KeyInfo) GetKeyIdentifier() string {
	if m != nil {
		return m.KeyIdentifier
	}
	return ""
}

func (m *KeyInfo) GetKeyMaterial() []byte {
	if m != nil {
		return m.KeyMaterial
	}
	return nil
}

//FabricouIdentifier表示组织单位和
//它的相关信任链标识符。
type FabricOUIdentifier struct {
//证书表示证书链中的第二个证书。
//（注意，证书链中的第一个证书应该是
//作为身份证明）。
//必须与根或中间CA的证书相对应
//由该邮件所属的MSP识别。
//从该证书开始，计算证书链
//并绑定到指定的OrganizationUnitIdentifier
	Certificate []byte `protobuf:"bytes,1,opt,name=certificate,proto3" json:"certificate,omitempty"`
//OrganizationUnitIdentifier定义
//用MSPIdentifier标识的MSP
	OrganizationalUnitIdentifier string   `protobuf:"bytes,2,opt,name=organizational_unit_identifier,json=organizationalUnitIdentifier,proto3" json:"organizational_unit_identifier,omitempty"`
	XXX_NoUnkeyedLiteral         struct{} `json:"-"`
	XXX_unrecognized             []byte   `json:"-"`
	XXX_sizecache                int32    `json:"-"`
}

func (m *FabricOUIdentifier) Reset()         { *m = FabricOUIdentifier{} }
func (m *FabricOUIdentifier) String() string { return proto.CompactTextString(m) }
func (*FabricOUIdentifier) ProtoMessage()    {}
func (*FabricOUIdentifier) Descriptor() ([]byte, []int) {
	return fileDescriptor_msp_config_e749e5bd1d6d997b, []int{7}
}
func (m *FabricOUIdentifier) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FabricOUIdentifier.Unmarshal(m, b)
}
func (m *FabricOUIdentifier) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FabricOUIdentifier.Marshal(b, m, deterministic)
}
func (dst *FabricOUIdentifier) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FabricOUIdentifier.Merge(dst, src)
}
func (m *FabricOUIdentifier) XXX_Size() int {
	return xxx_messageInfo_FabricOUIdentifier.Size(m)
}
func (m *FabricOUIdentifier) XXX_DiscardUnknown() {
	xxx_messageInfo_FabricOUIdentifier.DiscardUnknown(m)
}

var xxx_messageInfo_FabricOUIdentifier proto.InternalMessageInfo

func (m *FabricOUIdentifier) GetCertificate() []byte {
	if m != nil {
		return m.Certificate
	}
	return nil
}

func (m *FabricOUIdentifier) GetOrganizationalUnitIdentifier() string {
	if m != nil {
		return m.OrganizationalUnitIdentifier
	}
	return ""
}

//FabricNodeous包含用于区分客户机和对等机与订购方的配置
//基于U.如果启用了节点识别，则MSP标识
//不包含任何指定OU的将被视为无效。
type FabricNodeOUs struct {
//如果为true，则不包含任何指定OU的MSP标识将被视为无效。
	Enable bool `protobuf:"varint,1,opt,name=enable,proto3" json:"enable,omitempty"`
//客户机的OU标识符
	ClientOuIdentifier *FabricOUIdentifier `protobuf:"bytes,2,opt,name=client_ou_identifier,json=clientOuIdentifier,proto3" json:"client_ou_identifier,omitempty"`
//对等方的OU标识符
	PeerOuIdentifier     *FabricOUIdentifier `protobuf:"bytes,3,opt,name=peer_ou_identifier,json=peerOuIdentifier,proto3" json:"peer_ou_identifier,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *FabricNodeOUs) Reset()         { *m = FabricNodeOUs{} }
func (m *FabricNodeOUs) String() string { return proto.CompactTextString(m) }
func (*FabricNodeOUs) ProtoMessage()    {}
func (*FabricNodeOUs) Descriptor() ([]byte, []int) {
	return fileDescriptor_msp_config_e749e5bd1d6d997b, []int{8}
}
func (m *FabricNodeOUs) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FabricNodeOUs.Unmarshal(m, b)
}
func (m *FabricNodeOUs) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FabricNodeOUs.Marshal(b, m, deterministic)
}
func (dst *FabricNodeOUs) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FabricNodeOUs.Merge(dst, src)
}
func (m *FabricNodeOUs) XXX_Size() int {
	return xxx_messageInfo_FabricNodeOUs.Size(m)
}
func (m *FabricNodeOUs) XXX_DiscardUnknown() {
	xxx_messageInfo_FabricNodeOUs.DiscardUnknown(m)
}

var xxx_messageInfo_FabricNodeOUs proto.InternalMessageInfo

func (m *FabricNodeOUs) GetEnable() bool {
	if m != nil {
		return m.Enable
	}
	return false
}

func (m *FabricNodeOUs) GetClientOuIdentifier() *FabricOUIdentifier {
	if m != nil {
		return m.ClientOuIdentifier
	}
	return nil
}

func (m *FabricNodeOUs) GetPeerOuIdentifier() *FabricOUIdentifier {
	if m != nil {
		return m.PeerOuIdentifier
	}
	return nil
}

func init() {
	proto.RegisterType((*MSPConfig)(nil), "msp.MSPConfig")
	proto.RegisterType((*FabricMSPConfig)(nil), "msp.FabricMSPConfig")
	proto.RegisterType((*FabricCryptoConfig)(nil), "msp.FabricCryptoConfig")
	proto.RegisterType((*IdemixMSPConfig)(nil), "msp.IdemixMSPConfig")
	proto.RegisterType((*IdemixMSPSignerConfig)(nil), "msp.IdemixMSPSignerConfig")
	proto.RegisterType((*SigningIdentityInfo)(nil), "msp.SigningIdentityInfo")
	proto.RegisterType((*KeyInfo)(nil), "msp.KeyInfo")
	proto.RegisterType((*FabricOUIdentifier)(nil), "msp.FabricOUIdentifier")
	proto.RegisterType((*FabricNodeOUs)(nil), "msp.FabricNodeOUs")
}

func init() { proto.RegisterFile("msp/msp_config.proto", fileDescriptor_msp_config_e749e5bd1d6d997b) }

var fileDescriptor_msp_config_e749e5bd1d6d997b = []byte{
//gzip文件描述符或协议的847字节
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x55, 0x5f, 0x6f, 0xe3, 0x44,
	0x10, 0x57, 0x92, 0x26, 0x77, 0x99, 0x38, 0x49, 0xd9, 0xeb, 0x15, 0x0b, 0x71, 0x77, 0xa9, 0x01,
	0x91, 0x17, 0x52, 0xa9, 0x87, 0x84, 0x84, 0x78, 0xba, 0xc2, 0x09, 0x03, 0xa5, 0xd5, 0x56, 0x7d,
	0xe1, 0xc5, 0xda, 0xd8, 0x9b, 0x64, 0x65, 0x7b, 0xd7, 0xda, 0x5d, 0x9f, 0x08, 0xe2, 0x99, 0x2f,
	0xc0, 0x77, 0xe0, 0x99, 0x57, 0xbe, 0x1d, 0xda, 0x3f, 0x8d, 0x9d, 0x6b, 0x15, 0x78, 0x9b, 0x9d,
	0xf9, 0xcd, 0xcf, 0xb3, 0xbf, 0x99, 0x59, 0xc3, 0x49, 0xa9, 0xaa, 0xf3, 0x52, 0x55, 0x49, 0x2a,
	0xf8, 0x8a, 0xad, 0x17, 0x95, 0x14, 0x5a, 0xa0, 0x5e, 0xa9, 0xaa, 0xe8, 0x2b, 0x18, 0x5e, 0xdd,
	0xde, 0x5c, 0x5a, 0x3f, 0x42, 0x70, 0xa4, 0xb7, 0x15, 0x0d, 0x3b, 0xb3, 0xce, 0xbc, 0x8f, 0xad,
	0x8d, 0x4e, 0x61, 0xe0, 0xb2, 0xc2, 0xee, 0xac, 0x33, 0x0f, 0xb0, 0x3f, 0x45, 0x7f, 0x1f, 0xc1,
	0xf4, 0x2d, 0x59, 0x4a, 0x96, 0xee, 0xe5, 0x73, 0x52, 0xba, 0xfc, 0x21, 0xb6, 0x36, 0x7a, 0x01,
	0x20, 0x85, 0xd0, 0x49, 0x4a, 0xa5, 0x56, 0x61, 0x77, 0xd6, 0x9b, 0x07, 0x78, 0x68, 0x3c, 0x97,
	0xc6, 0x81, 0xbe, 0x00, 0xc4, 0xb8, 0xa6, 0xb2, 0xa4, 0x19, 0x23, 0x9a, 0x7a, 0x58, 0xcf, 0xc2,
	0x3e, 0x68, 0x47, 0x1c, 0xfc, 0x14, 0x06, 0x24, 0x2b, 0x19, 0x57, 0xe1, 0x91, 0x85, 0xf8, 0x13,
	0xfa, 0x1c, 0xa6, 0x92, 0xbe, 0x13, 0x29, 0xd1, 0x4c, 0xf0, 0xa4, 0x60, 0x4a, 0x87, 0x7d, 0x0b,
	0x98, 0x34, 0xee, 0x9f, 0x98, 0xd2, 0xe8, 0x12, 0x8e, 0x15, 0x5b, 0x73, 0xc6, 0xd7, 0x09, 0xcb,
	0x28, 0xd7, 0x4c, 0x6f, 0xc3, 0xc1, 0xac, 0x33, 0x1f, 0x5d, 0x84, 0x8b, 0x52, 0x55, 0x8b, 0x5b,
	0x17, 0x8c, 0x7d, 0x2c, 0xe6, 0x2b, 0x81, 0xa7, 0x6a, 0xdf, 0x89, 0x12, 0x78, 0x25, 0xe4, 0x9a,
	0x70, 0xf6, 0x9b, 0x25, 0x26, 0x45, 0x52, 0x73, 0xa6, 0x3d, 0xe1, 0x8a, 0x51, 0xa9, 0xc2, 0x27,
	0xb3, 0xde, 0x7c, 0x74, 0xf1, 0xa1, 0xe5, 0x74, 0x32, 0x5d, 0xdf, 0xc5, 0xbb, 0x38, 0x7e, 0xb1,
	0x9f, 0x7f, 0xc7, 0x99, 0x6e, 0xa2, 0x0a, 0x7d, 0x03, 0xe3, 0x54, 0x6e, 0x2b, 0x2d, 0x7c, 0xc7,
	0xc2, 0xa7, 0xb6, 0xc4, 0x36, 0xdd, 0xa5, 0x8d, 0x3b, 0xe1, 0x71, 0x90, 0xb6, 0x4e, 0xe8, 0x53,
	0x98, 0xe8, 0x42, 0x25, 0x2d, 0xd9, 0x87, 0x56, 0x8b, 0x40, 0x17, 0x0a, 0xef, 0x94, 0xff, 0x12,
	0x4e, 0x0d, 0xea, 0x11, 0xf5, 0xc1, 0xa2, 0x4f, 0x74, 0xa1, 0xe2, 0x07, 0x0d, 0xf8, 0x1a, 0xa6,
	0x2b, 0xfb, 0xfd, 0x84, 0x8b, 0x8c, 0x26, 0xa2, 0x56, 0xe1, 0xc8, 0xd6, 0x86, 0x5a, 0xb5, 0xfd,
	0x2c, 0x32, 0x7a, 0x7d, 0xa7, 0xf0, 0x78, 0xd5, 0x1c, 0x6b, 0x15, 0xfd, 0xd9, 0x01, 0xf4, 0xb0,
	0x78, 0x74, 0x01, 0xcf, 0x8d, 0xc0, 0x44, 0xd7, 0x92, 0x26, 0x1b, 0xa2, 0x36, 0xc9, 0x8a, 0x94,
	0xac, 0xd8, 0xfa, 0x31, 0x7a, 0xb6, 0x0b, 0x7e, 0x4f, 0xd4, 0xe6, 0xad, 0x0d, 0xa1, 0x18, 0xce,
	0xee, 0xdb, 0xd7, 0x92, 0xdd, 0x67, 0xd7, 0x3c, 0x35, 0xb2, 0xda, 0x81, 0x1d, 0xe2, 0x97, 0xf7,
	0xc0, 0x46, 0x60, 0x4b, 0xe4, 0x51, 0xd1, 0x5f, 0x1d, 0x98, 0xc6, 0x19, 0x2d, 0xd9, 0xaf, 0x87,
	0x07, 0xf9, 0x18, 0x7a, 0xac, 0xca, 0xfd, 0x16, 0x18, 0x13, 0x5d, 0xc0, 0xc0, 0xd4, 0x46, 0x65,
	0xd8, 0xb3, 0x12, 0x7c, 0x64, 0x25, 0xd8, 0x71, 0xdd, 0xda, 0x98, 0xef, 0x90, 0x47, 0xa2, 0x4f,
	0x60, 0xdc, 0x1a, 0xd4, 0x2a, 0x0f, 0x8f, 0x2c, 0x5f, 0xd0, 0x38, 0x6f, 0x72, 0x74, 0x02, 0x7d,
	0x5a, 0x89, 0x74, 0x13, 0xf6, 0x67, 0x9d, 0x79, 0x0f, 0xbb, 0x43, 0xf4, 0x47, 0x17, 0x9e, 0x3f,
	0x4a, 0x6e, 0xca, 0x4d, 0x25, 0xcd, 0x6c, 0xb9, 0x01, 0xb6, 0x36, 0x9a, 0x40, 0x57, 0xdd, 0x57,
	0xdb, 0x55, 0x39, 0xfa, 0x16, 0x5e, 0x1e, 0x9e, 0x59, 0x7b, 0x89, 0x21, 0xfe, 0xf8, 0xd0, 0x64,
	0x9a, 0x2f, 0x49, 0x51, 0x50, 0x5b, 0x75, 0x1f, 0x5b, 0xdb, 0x5c, 0x89, 0x72, 0x29, 0x8a, 0xa2,
	0xa4, 0xdc, 0x10, 0xda, 0xaa, 0x87, 0x38, 0x68, 0x9c, 0x71, 0x86, 0x7e, 0x80, 0x33, 0x53, 0x96,
	0x21, 0x22, 0x45, 0xd2, 0x92, 0x80, 0xf1, 0x95, 0x90, 0xa5, 0xb5, 0xed, 0x22, 0x06, 0xf8, 0x55,
	0x03, 0xc4, 0x3b, 0x5c, 0xdc, 0xc0, 0x22, 0x01, 0xcf, 0x1e, 0x59, 0x53, 0x53, 0x47, 0x55, 0x2f,
	0x0b, 0x96, 0x26, 0xbe, 0x2b, 0x4e, 0x8e, 0xc0, 0x39, 0x9d, 0x60, 0xe8, 0x35, 0x4c, 0x2a, 0xc9,
	0xde, 0x99, 0x61, 0xf7, 0xa8, 0xae, 0xed, 0x5d, 0x60, 0x7b, 0xf7, 0x23, 0x75, 0x1b, 0x3f, 0xf6,
	0x18, 0x97, 0x14, 0xdd, 0xc2, 0x13, 0x1f, 0x41, 0x9f, 0xc1, 0x24, 0xa7, 0xed, 0x99, 0xf3, 0x33,
	0x32, 0xce, 0x69, 0x6b, 0xc0, 0xd0, 0x19, 0x04, 0x06, 0x56, 0x12, 0x4d, 0x25, 0x23, 0x85, 0xef,
	0xc3, 0x28, 0xa7, 0xdb, 0x2b, 0xef, 0x8a, 0x7e, 0xbf, 0x5f, 0x86, 0xf6, 0xc3, 0x80, 0x66, 0x30,
	0x32, 0x4b, 0xc8, 0x56, 0x2c, 0x25, 0x9a, 0xfa, 0x2b, 0xb4, 0x5d, 0xff, 0xa3, 0x91, 0xdd, 0xff,
	0x6e, 0x64, 0xf4, 0x4f, 0x07, 0xc6, 0x7b, 0xcb, 0x6a, 0x9e, 0x56, 0xca, 0xc9, 0xb2, 0x70, 0x1f,
	0x7d, 0x8a, 0xfd, 0x09, 0xc5, 0x70, 0x92, 0x16, 0xcc, 0xb4, 0x56, 0xd4, 0xef, 0x7f, 0xe5, 0xc0,
	0x0b, 0x87, 0x5c, 0xd2, 0x75, 0xdd, 0xba, 0xdc, 0x77, 0x80, 0x2a, 0x4a, 0xe5, 0x7b, 0x44, 0xbd,
	0xc3, 0x44, 0xc7, 0x26, 0xa5, 0x4d, 0xf3, 0x26, 0x81, 0x33, 0x21, 0xd7, 0x8b, 0xcd, 0xb6, 0xa2,
	0xb2, 0xa0, 0xd9, 0x9a, 0xca, 0x85, 0x7b, 0x68, 0xdc, 0x8f, 0x4d, 0x19, 0xa6, 0x37, 0xc7, 0x57,
	0xaa, 0x72, 0xeb, 0x71, 0x43, 0xd2, 0x9c, 0xac, 0xe9, 0x2f, 0xf3, 0x35, 0xd3, 0x9b, 0x7a, 0xb9,
	0x48, 0x45, 0x79, 0xde, 0xca, 0x3d, 0x77, 0xb9, 0xe7, 0x2e, 0xd7, 0xfc, 0x26, 0x97, 0x03, 0x6b,
	0xbf, 0xfe, 0x37, 0x00, 0x00, 0xff, 0xff, 0x54, 0x67, 0x46, 0xdb, 0x38, 0x07, 0x00, 0x00,
}
