
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司。保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package msp

import (
	"time"

	"github.com/hyperledger/fabric/protos/msp"
)

//IdentityDeserializer由MSPManger和MSP实现
type IdentityDeserializer interface {
//反序列化IDentity反序列化标识。
//如果标识与关联，则反序列化将失败
//与正在执行的MSP不同的MSP
//反序列化。
	DeserializeIdentity(serializedIdentity []byte) (Identity, error)

//iswell格式检查给定的标识是否可以反序列化为其提供程序特定的形式
	IsWellFormed(identity *msp.SerializedIdentity) error
}

//用于Hyperledger结构的成员资格服务提供程序API:
//
//“会员服务提供商”指的是
//向客户端和对等端提供（匿名）凭据的系统
//他们将参与Hyperledger/Fabric网络。客户使用这些
//用于验证其事务的凭据，对等方使用这些凭据
//验证交易处理结果（背书）。同时
//与系统的事务处理组件紧密相连，
//此接口旨在定义成员服务组件，其中
//一种可以顺利插入此的替代实现的方法
//不需要修改系统的核心事务处理组件。
//
//此文件包括成员资格服务提供程序接口，该接口包含
//对等成员身份服务提供程序接口的需求。

//MSPManager是定义一个或多个MSP的管理器的接口。这个
//基本上充当MSP调用的中介，并路由与MSP相关的调用
//到适当的MSP。
//此对象是不可变的，它被初始化一次，并且从未更改过。
type MSPManager interface {

//IdentityDeserializer接口需要由MSPManager实现
	IdentityDeserializer

//根据配置信息设置MSP管理器实例
	Setup(msps []MSP) error

//GetMSPS提供成员资格服务提供程序列表
	GetMSPs() (map[string]MSP, error)
}

//MSP是要实现的最小成员资格服务提供程序接口
//以适应对等功能
type MSP interface {

//IdentityDeserializer接口需要由MSP实现
	IdentityDeserializer

//根据配置信息设置MSP实例
	Setup(config *msp.MSPConfig) error

//GetVersion返回此MSP的版本
	GetVersion() MSPVersion

//GetType返回提供程序类型
	GetType() ProviderType

//GetIdentifier返回提供程序标识符
	GetIdentifier() (string, error)

//GetSigningIdentity返回与提供的标识符对应的签名标识
	GetSigningIdentity(identifier *IdentityIdentifier) (SigningIdentity, error)

//GetDefaultSigningIdentity返回默认签名标识
	GetDefaultSigningIdentity() (SigningIdentity, error)

//GettlsRootCerts返回此MSP的TLS根证书
	GetTLSRootCerts() [][]byte

//GettlIntermediateCenters返回此MSP的TLS中间根证书
	GetTLSIntermediateCerts() [][]byte

//验证检查提供的标识是否有效
	Validate(id Identity) error

//satisfiesprincipal检查标识是否匹配
//mspprincipal中提供的说明。支票可以
//涉及逐字节比较（如果主体是
//或可能需要MSP验证
	SatisfiesPrincipal(id Identity, principal *msp.MSPPrincipal) error
}

//ouidentifier表示一个组织单位和
//它的相关信任链标识符。
type OUIdentifier struct {
//certifiersidentifier是信任证书链的哈希
//与此组织单位相关
	CertifiersIdentifier []byte
//OrganizationUnitIdentifier定义
//用MSPIdentifier标识的MSP
	OrganizationalUnitIdentifier string
}

//从现在开始，在对等端和客户机API中有共享的接口
//会员服务提供商的。

//定义与“证书”关联的操作的标识接口。
//也就是说，身份的公共部分可以被认为是证书，
//并且只提供签名验证功能。这是要用的
//在对等端验证已签名事务的证书时
//并验证与这些证书相对应的签名。
type Identity interface {

//expires at返回标识过期的时间。
//如果返回的时间为零值，则表示
//该标识没有过期，或者其过期
//时间未知
	ExpiresAt() time.Time

//GetIdentifier返回该标识的标识符
	GetIdentifier() *IdentityIdentifier

//GetMSPIdentifier返回此实例的MSP ID
	GetMSPIdentifier() string

//验证使用控制此标识的规则来验证它。
//例如，如果是作为标识实现的结构tcert，则验证
//将根据假定的根证书检查tcert签名
//权威。
	Validate() error

//GetOrganizationalUnits返回零个或多个组织单位或
//只要这是公开的，这个身份就与之相关
//信息。某些MSP实现可能使用属性
//与此身份或的标识符公开关联的
//在此上提供签名的根证书颁发机构
//证书。
//实例：
//-如果标识是X.509证书，则此函数返回一个
//或以主题的可分辨名称编码的多个字符串
//型瓯
//TODO:对于基于X.509的标识，请检查是否需要专用类型
//对于ou，其中证书ou由
//签名人身份
	GetOrganizationalUnits() []*OUIdentifier

//如果这是匿名标识，则Anonymous返回true，否则返回false
	Anonymous() bool

//使用此标识作为引用验证某些消息上的签名
	Verify(msg []byte, sig []byte) error

//序列化将标识转换为字节
	Serialize() ([]byte, error)

//satisfiesprincipal检查此实例是否匹配
//mspprincipal中提供的说明。支票可以
//涉及逐字节比较（如果主体是
//或可能需要MSP验证
	SatisfiesPrincipal(principal *msp.MSPPrincipal) error
}

//SigningIdentity是标识的扩展，用于覆盖签名功能。
//例如，如果客户希望
//签署交易，或希望签署提案的织物背书人
//处理结果。
type SigningIdentity interface {

//扩展标识
	Identity

//在留言上签名
	Sign(msg []byte) ([]byte, error)

//GetPublicVersion返回此标识的公共部分
	GetPublicVersion() Identity
}

//IdentityIdentifier是特定
//通过提供程序标识符自然命名的标识。
type IdentityIdentifier struct {

//关联的成员身份服务提供程序的标识符
	Mspid string

//提供程序内标识的标识符
	Id string
}

//ProviderType指示标识提供程序的类型
type ProviderType int

//成员相对于成员API的ProviderType
const (
FABRIC ProviderType = iota //MSP为织物类型
IDEMIX                     //MSP为Idemix类型
OTHER                      //MSP是其他类型的

//注意：随着新类型添加到此集合中，
//必须扩展下面的msptypes映射
)

var mspTypeStrings = map[ProviderType]string{
	FABRIC: "bccsp",
	IDEMIX: "idemix",
}

//ProviderTypeToString返回表示ProviderType整数的字符串
func ProviderTypeToString(id ProviderType) string {
	if res, found := mspTypeStrings[id]; found {
		return res
	}

	return ""
}
