
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


package api

import (
	"time"

	"github.com/hyperledger/fabric/gossip/common"
	"google.golang.org/grpc"
)

//MessageCryptoService是八卦组件和
//对等机的加密层，并由八卦组件用于验证，
//并验证远程对等机和它们发送的数据，以及验证
//从订购服务接收到块。
type MessageCryptoService interface {

//getpkiidofcert返回对等身份的pki-id
//如果出现任何错误，方法返回nil
//此方法不验证对等标识。
//这个验证应该在执行流程中适当地完成。
	GetPKIidOfCert(peerIdentity PeerIdentityType) common.PKIidType

//如果块已正确签名，并且声明的seqnum是
//块头包含的序列号。
//else返回错误
	VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error

//用该对等方的签名密钥和输出对消息进行签名
//如果没有出现错误，则返回签名。
	Sign(msg []byte) ([]byte, error)

//验证检查签名是否是对等验证密钥下消息的有效签名。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peerIdentity为零，则验证失败。
	Verify(peerIdentity PeerIdentityType, signature, message []byte) error

//VerifyByChannel检查签名是否为消息的有效签名
//在对等机的验证密钥下，也在特定通道的上下文中。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peerIdentity为零，则验证失败。
	VerifyByChannel(chainID common.ChainID, peerIdentity PeerIdentityType, signature, message []byte) error

//validateIDentity验证远程对等机的标识。
//如果标识无效、已吊销、已过期，则返回错误。
//否则，返回零
	ValidateIdentity(peerIdentity PeerIdentityType) error

//过期退货：
//-身份过期时间，无
//以防过期
//零值时间，时间，零
//以防过期
//-零值，如果不能
//已确定身份是否可以过期
	Expiration(peerIdentity PeerIdentityType) (time.Time, error)
}

//peerIdentityInfo聚合对等机的标识，
//以及有关它的其他元数据
type PeerIdentityInfo struct {
	PKIId        common.PKIidType
	Identity     PeerIdentityType
	Organization OrgIdentityType
}

//PeerIdentitySet聚合PeerIdentityInfo切片
type PeerIdentitySet []PeerIdentityInfo

//byorg对其对等组织设置的对等标识进行排序
func (pis PeerIdentitySet) ByOrg() map[string]PeerIdentitySet {
	m := make(map[string]PeerIdentitySet)
	for _, id := range pis {
		m[string(id.Organization)] = append(m[string(id.Organization)], id)
	}
	return m
}

//ByOrg根据其对等方的pki ID对peerIdentity集进行排序
func (pis PeerIdentitySet) ByID() map[string]PeerIdentityInfo {
	m := make(map[string]PeerIdentityInfo)
	for _, id := range pis {
		m[string(id.PKIId)] = id
	}
	return m
}

//peerIdentityType是对等方的证书
type PeerIdentityType []byte

//PeerSuspencector返回是否怀疑具有给定标识的对等机
//被撤销或其CA被撤销
type PeerSuspector func(identity PeerIdentityType) bool

//PeerSecureDialOpts返回用于连接级别的GRPC拨号选项
//与远程对等端点通信时的安全性
type PeerSecureDialOpts func() []grpc.DialOption

//对等签名定义对等签名
//在给定的消息上
type PeerSignature struct {
	Signature    []byte
	Message      []byte
	PeerIdentity PeerIdentityType
}
