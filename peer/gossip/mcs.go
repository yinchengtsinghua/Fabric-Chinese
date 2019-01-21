
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


package gossip

import (
	"bytes"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mgmt"
	pcommon "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var mcsLogger = flogging.MustGetLogger("peer.gossip.mcs")

//MSPMessageRyptoService实现MessageCryptoService接口
//使用对等MSP（本地和通道相关）
//
//为了使系统安全，必须
//MSP是最新的。频道的MSP通过以下方式更新：
//由订购服务分发的配置事务。
//
//还需要建立类似的机制来更新本地MSP。
//此实现假定这些机制都已就位并起作用。
type MSPMessageCryptoService struct {
	channelPolicyManagerGetter policies.ChannelPolicyManagerGetter
	localSigner                crypto.LocalSigner
	deserializer               mgmt.DeserializersManager
}

//newmcs创建MSPMessageRyptoService的新实例
//实现了MessageCryptoService。
//该方法接受输入：
//1。通过manager方法授予给定通道的策略管理器访问权限的policies.channelpolicymanagergetter。
//2。crypto.localsigner的实例
//三。标识反序列化程序管理器
func NewMCS(channelPolicyManagerGetter policies.ChannelPolicyManagerGetter, localSigner crypto.LocalSigner, deserializer mgmt.DeserializersManager) *MSPMessageCryptoService {
	return &MSPMessageCryptoService{channelPolicyManagerGetter: channelPolicyManagerGetter, localSigner: localSigner, deserializer: deserializer}
}

//validateIDentity验证远程对等机的标识。
//如果标识无效、已吊销、已过期，则返回错误。
//否则，返回零
func (s *MSPMessageCryptoService) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
//按照方法合同的规定，
//下面我们只检查PeerIdentity不是
//无效、吊销或过期。

	_, _, err := s.getValidatedIdentity(peerIdentity)
	return err
}

//getpkiidofcert返回对等身份的pki-id
//如果出现任何错误，方法返回nil
//对等机的pkid计算为对等机标识的sha2-256，其中
//应该是MSP标识的序列化版本。
//此方法不验证对等标识。
//这个验证应该在执行流程中适当地完成。
func (s *MSPMessageCryptoService) GetPKIidOfCert(peerIdentity api.PeerIdentityType) common.PKIidType {
//验证参数
	if len(peerIdentity) == 0 {
		mcsLogger.Error("Invalid Peer Identity. It must be different from nil.")

		return nil
	}

	sid, err := s.deserializer.Deserialize(peerIdentity)
	if err != nil {
		mcsLogger.Errorf("Failed getting validated identity from peer identity [% x]: [%s]", peerIdentity, err)

		return nil
	}

//连接msp id和idbytes
//IDbytes是标识的低级表示。
//它应该已经在它的最小表示中了。

	mspIdRaw := []byte(sid.Mspid)
	raw := append(mspIdRaw, sid.IdBytes...)

//搞砸
	digest, err := factory.GetDefault().Hash(raw, &bccsp.SHA256Opts{})
	if err != nil {
		mcsLogger.Errorf("Failed computing digest of serialized identity [% x]: [%s]", peerIdentity, err)

		return nil
	}

	return digest
}

//如果块已正确签名，并且声明的seqnum是
//块头包含的序列号。
//else返回错误
func (s *MSPMessageCryptoService) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
//-将signedBlock转换为common.block。
	block, err := utils.GetBlockFromBlockBytes(signedBlock)
	if err != nil {
		return fmt.Errorf("Failed unmarshalling block bytes on channel [%s]: [%s]", chainID, err)
	}

	if block.Header == nil {
		return fmt.Errorf("Invalid Block on channel [%s]. Header must be different from nil.", chainID)
	}

	blockSeqNum := block.Header.Number
	if seqNum != blockSeqNum {
		return fmt.Errorf("Claimed seqNum is [%d] but actual seqNum inside block is [%d]", seqNum, blockSeqNum)
	}

//-提取channelid并与chaineid进行比较
	channelID, err := utils.GetChainIDFromBlock(block)
	if err != nil {
		return fmt.Errorf("Failed getting channel id from block with id [%d] on channel [%s]: [%s]", block.Header.Number, chainID, err)
	}

	if channelID != string(chainID) {
		return fmt.Errorf("Invalid block's channel id. Expected [%s]. Given [%s]", chainID, channelID)
	}

//-Unmashal Medatada公司
	if block.Metadata == nil || len(block.Metadata.Metadata) == 0 {
		return fmt.Errorf("Block with id [%d] on channel [%s] does not have metadata. Block not valid.", block.Header.Number, chainID)
	}

	metadata, err := utils.GetMetadataFromBlock(block, pcommon.BlockMetadataIndex_SIGNATURES)
	if err != nil {
		return fmt.Errorf("Failed unmarshalling medatata for signatures [%s]", err)
	}

//-验证header.data hash是否等于block.data的hash
//这是为了确保头与此块携带的数据一致。
	if !bytes.Equal(block.Data.Hash(), block.Header.DataHash) {
		return fmt.Errorf("Header.DataHash is different from Hash(block.Data) for block with id [%d] on channel [%s]", block.Header.Number, chainID)
	}

//-获取块验证策略

//获取channelid的策略管理器
	cpm, ok := s.channelPolicyManagerGetter.Manager(channelID)
	if cpm == nil {
		return fmt.Errorf("Could not acquire policy manager for channel %s", channelID)
	}
//如果是请求的管理器，则OK为true；如果是默认管理器，则OK为false。
	mcsLogger.Debugf("Got policy manager for channel [%s] with flag [%t]", channelID, ok)

//获取块验证策略
	policy, ok := cpm.GetPolicy(policies.BlockValidation)
//如果是请求的策略，则OK为true；如果是默认策略，则OK为false。
	mcsLogger.Debugf("Got block validation policy for channel [%s] with flag [%t]", channelID, ok)

//-准备签名数据
	signatureSet := []*pcommon.SignedData{}
	for _, metadataSignature := range metadata.Signatures {
		shdr, err := utils.GetSignatureHeader(metadataSignature.SignatureHeader)
		if err != nil {
			return fmt.Errorf("Failed unmarshalling signature header for block with id [%d] on channel [%s]: [%s]", block.Header.Number, chainID, err)
		}
		signatureSet = append(
			signatureSet,
			&pcommon.SignedData{
				Identity:  shdr.Creator,
				Data:      util.ConcatenateBytes(metadata.Value, metadataSignature.SignatureHeader, block.Header.Bytes()),
				Signature: metadataSignature.Signature,
			},
		)
	}

//-评估政策
	return policy.Evaluate(signatureSet)
}

//用该对等方的签名密钥和输出对消息进行签名
//如果没有出现错误，则返回签名。
func (s *MSPMessageCryptoService) Sign(msg []byte) ([]byte, error) {
	return s.localSigner.Sign(msg)
}

//验证检查签名是否是对等验证密钥下消息的有效签名。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peerIdentity为零，则验证失败。
func (s *MSPMessageCryptoService) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	identity, chainID, err := s.getValidatedIdentity(peerIdentity)
	if err != nil {
		mcsLogger.Errorf("Failed getting validated identity from peer identity [%s]", err)

		return err
	}

	if len(chainID) == 0 {
//在这个阶段，这意味着对等身份
//属于此对等机的localmsp。
//直接验证签名
		return identity.Verify(message, signature)
	}

//在此阶段，必须验证签名
//违反频道的读者政策
//由chainID标识

	return s.VerifyByChannel(chainID, peerIdentity, signature, message)
}

//VerifyByChannel检查签名是否为消息的有效签名
//在对等机的验证密钥下，也在特定通道的上下文中。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peerIdentity为零，则验证失败。
func (s *MSPMessageCryptoService) VerifyByChannel(chainID common.ChainID, peerIdentity api.PeerIdentityType, signature, message []byte) error {
//验证参数
	if len(peerIdentity) == 0 {
		return errors.New("Invalid Peer Identity. It must be different from nil.")
	}

//获取通道chainID的策略管理器
	cpm, flag := s.channelPolicyManagerGetter.Manager(string(chainID))
	if cpm == nil {
		return fmt.Errorf("Could not acquire policy manager for channel %s", string(chainID))
	}
	mcsLogger.Debugf("Got policy manager for channel [%s] with flag [%t]", string(chainID), flag)

//获取频道读取器策略
	policy, flag := cpm.GetPolicy(policies.ChannelApplicationReaders)
	mcsLogger.Debugf("Got reader policy for channel [%s] with flag [%t]", string(chainID), flag)

	return policy.Evaluate(
		[]*pcommon.SignedData{{
			Data:      message,
			Identity:  []byte(peerIdentity),
			Signature: signature,
		}},
	)
}

func (s *MSPMessageCryptoService) Expiration(peerIdentity api.PeerIdentityType) (time.Time, error) {
	id, _, err := s.getValidatedIdentity(peerIdentity)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "Unable to extract msp.Identity from peer Identity")
	}
	return id.ExpiresAt(), nil

}

func (s *MSPMessageCryptoService) getValidatedIdentity(peerIdentity api.PeerIdentityType) (msp.Identity, common.ChainID, error) {
//验证参数
	if len(peerIdentity) == 0 {
		return nil, nil, errors.New("Invalid Peer Identity. It must be different from nil.")
	}

	sId, err := s.deserializer.Deserialize(peerIdentity)
	if err != nil {
		mcsLogger.Error("failed deserializing identity", err)
		return nil, nil, err
	}

//请注意，假定peerIdentity是标识的序列化。
//所以，第一步是身份反序列化，然后验证它。

//首先检查本地MSP。
//如果对等身份在该节点的同一组织中，则
//本地MSP需要对有效性做出最终决定。
//签名。
	lDes := s.deserializer.GetLocalDeserializer()
	identity, err := lDes.DeserializeIdentity([]byte(peerIdentity))
	if err == nil {
//没有错误意味着本地MSP成功地反序列化了标识。
//我们现在检查其他属性。
		if err := lDes.IsWellFormed(sId); err != nil {
			return nil, nil, errors.Wrap(err, "identity is not well formed")
		}
//TODO:以下检查将替换为对组织单位的检查
//当我们允许八卦网络拥有组织单位（MSP子部门）时
//作用域消息。
//以下检查与SecurityAdvisor orgByPeerIdentity一致
//实施。
//TODO:请注意，下面的检查将我们从事实中解救出来
//反序列化的IDentity尚未强制MSPIDS一致性。
//一旦反序列化实体被修复，就可以删除此检查。
		if identity.GetMSPIdentifier() == s.deserializer.GetLocalMSPIdentifier() {
//检查身份有效性

//注意，在这个阶段，我们不需要检查身份
//反对任何渠道的政策。
//如果需要，这将由调用函数完成。
			return identity, nil, identity.Validate()
		}
	}

//与经理核对
	for chainID, mspManager := range s.deserializer.GetChannelDeserializers() {
//反序列化标识
		identity, err := mspManager.DeserializeIdentity([]byte(peerIdentity))
		if err != nil {
			mcsLogger.Debugf("Failed deserialization identity [% x] on [%s]: [%s]", peerIdentity, chainID, err)
			continue
		}

//我们用这个MSP管理器成功地反序列化了标识。现在我们检查一下它是否成形。
		if err := mspManager.IsWellFormed(sId); err != nil {
			return nil, nil, errors.Wrap(err, "identity is not well formed")
		}

//检查身份有效性
//注意，在这个阶段，我们不需要检查身份
//反对任何渠道的政策。
//如果需要，这将由调用函数完成。

		if err := identity.Validate(); err != nil {
			mcsLogger.Debugf("Failed validating identity [% x] on [%s]: [%s]", peerIdentity, chainID, err)
			continue
		}

		mcsLogger.Debugf("Validation succeeded [% x] on [%s]", peerIdentity, chainID)

		return identity, common.ChainID(chainID), nil
	}

	return nil, nil, fmt.Errorf("Peer Identity [% x] cannot be validated. No MSP found able to do that.", peerIdentity)
}
