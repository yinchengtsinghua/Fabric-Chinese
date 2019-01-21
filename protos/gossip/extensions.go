
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
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
)

//newgossipMessageComparator创建一个messageReplacingPolicy，给定要保留的最大块数。
func NewGossipMessageComparator(dataBlockStorageSize int) common.MessageReplacingPolicy {
	return (&msgComparator{dataBlockStorageSize: dataBlockStorageSize}).getMsgReplacingPolicy()
}

type msgComparator struct {
	dataBlockStorageSize int
}

func (mc *msgComparator) getMsgReplacingPolicy() common.MessageReplacingPolicy {
	return func(this interface{}, that interface{}) common.InvalidationResult {
		return mc.invalidationPolicy(this, that)
	}
}

func (mc *msgComparator) invalidationPolicy(this interface{}, that interface{}) common.InvalidationResult {
	thisMsg := this.(*SignedGossipMessage)
	thatMsg := that.(*SignedGossipMessage)

	if thisMsg.IsAliveMsg() && thatMsg.IsAliveMsg() {
		return aliveInvalidationPolicy(thisMsg.GetAliveMsg(), thatMsg.GetAliveMsg())
	}

	if thisMsg.IsDataMsg() && thatMsg.IsDataMsg() {
		return mc.dataInvalidationPolicy(thisMsg.GetDataMsg(), thatMsg.GetDataMsg())
	}

	if thisMsg.IsStateInfoMsg() && thatMsg.IsStateInfoMsg() {
		return mc.stateInvalidationPolicy(thisMsg.GetStateInfo(), thatMsg.GetStateInfo())
	}

	if thisMsg.IsIdentityMsg() && thatMsg.IsIdentityMsg() {
		return mc.identityInvalidationPolicy(thisMsg.GetPeerIdentity(), thatMsg.GetPeerIdentity())
	}

	if thisMsg.IsLeadershipMsg() && thatMsg.IsLeadershipMsg() {
		return leaderInvalidationPolicy(thisMsg.GetLeadershipMsg(), thatMsg.GetLeadershipMsg())
	}

	return common.MessageNoAction
}

func (mc *msgComparator) stateInvalidationPolicy(thisStateMsg *StateInfo, thatStateMsg *StateInfo) common.InvalidationResult {
	if !bytes.Equal(thisStateMsg.PkiId, thatStateMsg.PkiId) {
		return common.MessageNoAction
	}
	return compareTimestamps(thisStateMsg.Timestamp, thatStateMsg.Timestamp)
}

func (mc *msgComparator) identityInvalidationPolicy(thisIdentityMsg *PeerIdentity, thatIdentityMsg *PeerIdentity) common.InvalidationResult {
	if bytes.Equal(thisIdentityMsg.PkiId, thatIdentityMsg.PkiId) {
		return common.MessageInvalidated
	}

	return common.MessageNoAction
}

func (mc *msgComparator) dataInvalidationPolicy(thisDataMsg *DataMessage, thatDataMsg *DataMessage) common.InvalidationResult {
	if thisDataMsg.Payload.SeqNum == thatDataMsg.Payload.SeqNum {
		return common.MessageInvalidated
	}

	diff := abs(thisDataMsg.Payload.SeqNum, thatDataMsg.Payload.SeqNum)
	if diff <= uint64(mc.dataBlockStorageSize) {
		return common.MessageNoAction
	}

	if thisDataMsg.Payload.SeqNum > thatDataMsg.Payload.SeqNum {
		return common.MessageInvalidates
	}
	return common.MessageInvalidated
}

func aliveInvalidationPolicy(thisMsg *AliveMessage, thatMsg *AliveMessage) common.InvalidationResult {
	if !bytes.Equal(thisMsg.Membership.PkiId, thatMsg.Membership.PkiId) {
		return common.MessageNoAction
	}

	return compareTimestamps(thisMsg.Timestamp, thatMsg.Timestamp)
}

func leaderInvalidationPolicy(thisMsg *LeadershipMessage, thatMsg *LeadershipMessage) common.InvalidationResult {
	if !bytes.Equal(thisMsg.PkiId, thatMsg.PkiId) {
		return common.MessageNoAction
	}

	return compareTimestamps(thisMsg.Timestamp, thatMsg.Timestamp)
}

func compareTimestamps(thisTS *PeerTime, thatTS *PeerTime) common.InvalidationResult {
	if thisTS.IncNum == thatTS.IncNum {
		if thisTS.SeqNum > thatTS.SeqNum {
			return common.MessageInvalidates
		}

		return common.MessageInvalidated
	}
	if thisTS.IncNum < thatTS.IncNum {
		return common.MessageInvalidated
	}
	return common.MessageInvalidates
}

//isalivemsg返回此gossipmessage是否为alivemessage
func (m *GossipMessage) IsAliveMsg() bool {
	return m.GetAliveMsg() != nil
}

//isdatamsg返回此gossipmessage是否为数据消息
func (m *GossipMessage) IsDataMsg() bool {
	return m.GetDataMsg() != nil
}

//ISstateinfoPullRequestMsg返回此gossipMessage是否为stateinfoPullRequest
func (m *GossipMessage) IsStateInfoPullRequestMsg() bool {
	return m.GetStateInfoPullReq() != nil
}

//ISstateinfosSnapshot返回此gossipMessage是否为StateInfo快照
func (m *GossipMessage) IsStateInfoSnapshot() bool {
	return m.GetStateSnapshot() != nil
}

//ISstateinfomsg返回此gossipmessage是否为stateinfo消息
func (m *GossipMessage) IsStateInfoMsg() bool {
	return m.GetStateInfo() != nil
}

//ispullmsg返回此gossipMessage是否属于
//至牵引机构
func (m *GossipMessage) IsPullMsg() bool {
	return m.GetDataReq() != nil || m.GetDataUpdate() != nil ||
		m.GetHello() != nil || m.GetDataDig() != nil
}

//IsRemotestateMessage返回此gossipMessage是否与状态同步相关
func (m *GossipMessage) IsRemoteStateMessage() bool {
	return m.GetStateRequest() != nil || m.GetStateResponse() != nil
}

//getpullmsgtype返回此八卦消息所属的拉机制的阶段
//例如：你好、文摘等。
//如果这不是拉消息，则返回pullmsgtype_undefined。
func (m *GossipMessage) GetPullMsgType() PullMsgType {
	if helloMsg := m.GetHello(); helloMsg != nil {
		return helloMsg.MsgType
	}

	if digMsg := m.GetDataDig(); digMsg != nil {
		return digMsg.MsgType
	}

	if reqMsg := m.GetDataReq(); reqMsg != nil {
		return reqMsg.MsgType
	}

	if resMsg := m.GetDataUpdate(); resMsg != nil {
		return resMsg.MsgType
	}

	return PullMsgType_UNDEFINED
}

//IsChannelRestricted返回是否应路由此gossipMessage
//只在它的频道里
func (m *GossipMessage) IsChannelRestricted() bool {
	return m.Tag == GossipMessage_CHAN_AND_ORG || m.Tag == GossipMessage_CHAN_ONLY || m.Tag == GossipMessage_CHAN_OR_ORG
}

//IsorgRestricted返回是否只应路由此gossipMessage
//组织内部
func (m *GossipMessage) IsOrgRestricted() bool {
	return m.Tag == GossipMessage_CHAN_AND_ORG || m.Tag == GossipMessage_ORG_ONLY
}

//IsIdentityMsg返回此gossipMessage是否为标识消息
func (m *GossipMessage) IsIdentityMsg() bool {
	return m.GetPeerIdentity() != nil
}

//isdatareq返回此gossipMessage是否为数据请求消息
func (m *GossipMessage) IsDataReq() bool {
	return m.GetDataReq() != nil
}

//isprivatedatamsg返回此消息是否与私有数据相关
func (m *GossipMessage) IsPrivateDataMsg() bool {
	return m.GetPrivateReq() != nil || m.GetPrivateRes() != nil || m.GetPrivateData() != nil
}

//isack返回此八卦消息是否为确认消息
func (m *GossipMessage) IsAck() bool {
	return m.GetAck() != nil
}

//ISdataUpdate返回此gossipMessage是否为数据更新消息
func (m *GossipMessage) IsDataUpdate() bool {
	return m.GetDataUpdate() != nil
}

//ishellomsg返回此八卦消息是否为hello消息
func (m *GossipMessage) IsHelloMsg() bool {
	return m.GetHello() != nil
}

//isdigestmsg返回此gossipMessage是否为摘要消息
func (m *GossipMessage) IsDigestMsg() bool {
	return m.GetDataDig() != nil
}

//isleadershipmsg返回此八卦消息是否为领导层（领导人选举）消息
func (m *GossipMessage) IsLeadershipMsg() bool {
	return m.GetLeadershipMsg() != nil
}

//msgConsumer调用给定SignedGossipMessage的代码
type MsgConsumer func(message *SignedGossipMessage)

//IdentifierExtractor从SignedGossipMessage提取标识符
type IdentifierExtractor func(*SignedGossipMessage) string

//istagegal检查gossipmessage标记和内部类型
//如果标记与类型不匹配，则返回错误。
func (m *GossipMessage) IsTagLegal() error {
	if m.Tag == GossipMessage_UNDEFINED {
		return fmt.Errorf("Undefined tag")
	}
	if m.IsDataMsg() {
		if m.Tag != GossipMessage_CHAN_AND_ORG {
			return fmt.Errorf("Tag should be %s", GossipMessage_Tag_name[int32(GossipMessage_CHAN_AND_ORG)])
		}
		return nil
	}

	if m.IsAliveMsg() || m.GetMemReq() != nil || m.GetMemRes() != nil {
		if m.Tag != GossipMessage_EMPTY {
			return fmt.Errorf("Tag should be %s", GossipMessage_Tag_name[int32(GossipMessage_EMPTY)])
		}
		return nil
	}

	if m.IsIdentityMsg() {
		if m.Tag != GossipMessage_ORG_ONLY {
			return fmt.Errorf("Tag should be %s", GossipMessage_Tag_name[int32(GossipMessage_ORG_ONLY)])
		}
		return nil
	}

	if m.IsPullMsg() {
		switch m.GetPullMsgType() {
		case PullMsgType_BLOCK_MSG:
			if m.Tag != GossipMessage_CHAN_AND_ORG {
				return fmt.Errorf("Tag should be %s", GossipMessage_Tag_name[int32(GossipMessage_CHAN_AND_ORG)])
			}
			return nil
		case PullMsgType_IDENTITY_MSG:
			if m.Tag != GossipMessage_EMPTY {
				return fmt.Errorf("Tag should be %s", GossipMessage_Tag_name[int32(GossipMessage_EMPTY)])
			}
			return nil
		default:
			return fmt.Errorf("Invalid PullMsgType: %s", PullMsgType_name[int32(m.GetPullMsgType())])
		}
	}

	if m.IsStateInfoMsg() || m.IsStateInfoPullRequestMsg() || m.IsStateInfoSnapshot() || m.IsRemoteStateMessage() {
		if m.Tag != GossipMessage_CHAN_OR_ORG {
			return fmt.Errorf("Tag should be %s", GossipMessage_Tag_name[int32(GossipMessage_CHAN_OR_ORG)])
		}
		return nil
	}

	if m.IsLeadershipMsg() {
		if m.Tag != GossipMessage_CHAN_AND_ORG {
			return fmt.Errorf("Tag should be %s", GossipMessage_Tag_name[int32(GossipMessage_CHAN_AND_ORG)])
		}
		return nil
	}

	return fmt.Errorf("Unknown message type: %v", m)
}

//验证程序接收对等身份、签名和消息
//如果消息上的签名可以验证，则返回nil
//使用给定的标识。
type Verifier func(peerIdentity []byte, signature, message []byte) error

//签名者在消息上签名，然后返回（签名，无）
//成功时，零，失败时出错。
type Signer func(msg []byte) ([]byte, error)

//ReceivedMessage是一个八卦消息包装器，
//允许用户将消息发送到来源
//接收到的消息是从发送的。
//它还允许知道发送者的身份，
//要获取未封送gossipMessage的原始字节，
//以及这些原始字节上的签名。
type ReceivedMessage interface {

//response将一条八卦消息发送到发送此接收消息的来源。
	Respond(msg *GossipMessage)

//GetGossipMessage返回基础的GossipMessage
	GetGossipMessage() *SignedGossipMessage

//GetSourceMessage返回接收到的消息所在的信封
//建筑用
	GetSourceEnvelope() *Envelope

//getConnectionInfo返回有关远程对等机的信息
//发出信息的
	GetConnectionInfo() *ConnectionInfo

//ACK向发送方返回消息确认
//ACK可以接收一个错误，该错误指示与操作相关
//到消息失败
	Ack(err error)
}

//ConnectionInfo表示有关
//发送特定接收消息的远程对等机
type ConnectionInfo struct {
	ID       common.PKIidType
	Auth     *AuthInfo
	Identity api.PeerIdentityType
	Endpoint string
}

//string返回此connectioninfo的字符串表示形式
func (c *ConnectionInfo) String() string {
	return fmt.Sprintf("%s %v", c.Endpoint, c.ID)
}

//AuthInfo表示身份验证
//远程对等端提供的数据
//连接时
type AuthInfo struct {
	SignedData []byte
	Signature  []byte
}

//签名与给定的签名者签署流言蜚语。
//成功时返回信封，
//失败时的恐慌。
func (m *SignedGossipMessage) Sign(signer Signer) (*Envelope, error) {
//如果我们有秘密藏匿，不要覆盖它。
//备份，稍后恢复
	var secretEnvelope *SecretEnvelope
	if m.Envelope != nil {
		secretEnvelope = m.Envelope.SecretEnvelope
	}
	m.Envelope = nil
	payload, err := proto.Marshal(m.GossipMessage)
	if err != nil {
		return nil, err
	}
	sig, err := signer(payload)
	if err != nil {
		return nil, err
	}

	e := &Envelope{
		Payload:        payload,
		Signature:      sig,
		SecretEnvelope: secretEnvelope,
	}
	m.Envelope = e
	return e, nil
}

//NoopSign创建一个签名为nil的SignedGossipMessage
func (m *GossipMessage) NoopSign() (*SignedGossipMessage, error) {
	signer := func(msg []byte) ([]byte, error) {
		return nil, nil
	}
	sMsg := &SignedGossipMessage{
		GossipMessage: m,
	}
	_, err := sMsg.Sign(signer)
	return sMsg, err
}

//verify使用给定的验证器验证签名的八卦消息。
//成功时返回零，失败时返回错误。
func (m *SignedGossipMessage) Verify(peerIdentity []byte, verify Verifier) error {
	if m.Envelope == nil {
		return errors.New("Missing envelope")
	}
	if len(m.Envelope.Payload) == 0 {
		return errors.New("Empty payload")
	}
	if len(m.Envelope.Signature) == 0 {
		return errors.New("Empty signature")
	}
	payloadSigVerificationErr := verify(peerIdentity, m.Envelope.Signature, m.Envelope.Payload)
	if payloadSigVerificationErr != nil {
		return payloadSigVerificationErr
	}
	if m.Envelope.SecretEnvelope != nil {
		payload := m.Envelope.SecretEnvelope.Payload
		sig := m.Envelope.SecretEnvelope.Signature
		if len(payload) == 0 {
			return errors.New("Empty payload")
		}
		if len(sig) == 0 {
			return errors.New("Empty signature")
		}
		return verify(peerIdentity, sig, payload)
	}
	return nil
}

//ISSIGNED返回消息
//信封上有签名。
func (m *SignedGossipMessage) IsSigned() bool {
	return m.Envelope != nil && m.Envelope.Payload != nil && m.Envelope.Signature != nil
}

//TogossipMessage取消封送给定信封并创建
//从中签名的ossipMessage。
//如果取消封送失败，则返回错误。
func (e *Envelope) ToGossipMessage() (*SignedGossipMessage, error) {
	if e == nil {
		return nil, errors.New("nil envelope")
	}
	msg := &GossipMessage{}
	err := proto.Unmarshal(e.Payload, msg)
	if err != nil {
		return nil, fmt.Errorf("Failed unmarshaling GossipMessage from envelope: %v", err)
	}
	return &SignedGossipMessage{
		GossipMessage: msg,
		Envelope:      e,
	}, nil
}

//SignSecret对秘密有效载荷进行签名并创建
//一个秘密的信封。
func (e *Envelope) SignSecret(signer Signer, secret *Secret) error {
	payload, err := proto.Marshal(secret)
	if err != nil {
		return err
	}
	sig, err := signer(payload)
	if err != nil {
		return err
	}
	e.SecretEnvelope = &SecretEnvelope{
		Payload:   payload,
		Signature: sig,
	}
	return nil
}

//InternalEndpoint返回内部终结点
//在秘密信封或空字符串中
//如果发生故障。
func (s *SecretEnvelope) InternalEndpoint() string {
	secret := &Secret{}
	if err := proto.Unmarshal(s.Payload, secret); err != nil {
		return ""
	}
	return secret.GetInternalEndpoint()
}

//SignedGossipMessage包含一个八卦消息
//以及它来自的信封
type SignedGossipMessage struct {
	*Envelope
	*GossipMessage
}

func (p *Payload) toString() string {
	return fmt.Sprintf("Block message: {Data: %d bytes, seq: %d}", len(p.Data), p.SeqNum)
}

func (du *DataUpdate) toString() string {
	mType := PullMsgType_name[int32(du.MsgType)]
	return fmt.Sprintf("Type: %s, items: %d, nonce: %d", mType, len(du.Data), du.Nonce)
}

func (mr *MembershipResponse) toString() string {
	return fmt.Sprintf("MembershipResponse with Alive: %d, Dead: %d", len(mr.Alive), len(mr.Dead))
}

func (sis *StateInfoSnapshot) toString() string {
	return fmt.Sprintf("StateInfoSnapshot with %d items", len(sis.Elements))
}

//字符串返回字符串表示形式
//已签名的邮件
func (m *SignedGossipMessage) String() string {
	env := "No envelope"
	if m.Envelope != nil {
		var secretEnv string
		if m.SecretEnvelope != nil {
			pl := len(m.SecretEnvelope.Payload)
			sl := len(m.SecretEnvelope.Signature)
			secretEnv = fmt.Sprintf(" Secret payload: %d bytes, Secret Signature: %d bytes", pl, sl)
		}
		env = fmt.Sprintf("%d bytes, Signature: %d bytes%s", len(m.Envelope.Payload), len(m.Envelope.Signature), secretEnv)
	}
	gMsg := "No gossipMessage"
	if m.GossipMessage != nil {
		var isSimpleMsg bool
		if m.GetStateResponse() != nil {
			gMsg = fmt.Sprintf("StateResponse with %d items", len(m.GetStateResponse().Payloads))
		} else if m.IsDataMsg() && m.GetDataMsg().Payload != nil {
			gMsg = m.GetDataMsg().Payload.toString()
		} else if m.IsDataUpdate() {
			update := m.GetDataUpdate()
			gMsg = fmt.Sprintf("DataUpdate: %s", update.toString())
		} else if m.GetMemRes() != nil {
			gMsg = m.GetMemRes().toString()
		} else if m.IsStateInfoSnapshot() {
			gMsg = m.GetStateSnapshot().toString()
		} else if m.GetPrivateRes() != nil {
			gMsg = m.GetPrivateRes().ToString()
		} else {
			gMsg = m.GossipMessage.String()
			isSimpleMsg = true
		}
		if !isSimpleMsg {
			desc := fmt.Sprintf("Channel: %s, nonce: %d, tag: %s", string(m.Channel), m.Nonce, GossipMessage_Tag_name[int32(m.Tag)])
			gMsg = fmt.Sprintf("%s %s", desc, gMsg)
		}
	}
	return fmt.Sprintf("GossipMessage: %v, Envelope: %s", gMsg, env)
}

func (dd *DataRequest) FormattedDigests() []string {
	if dd.MsgType == PullMsgType_IDENTITY_MSG {
		return digestsToHex(dd.Digests)
	}

	return digestsAsStrings(dd.Digests)
}

func (dd *DataDigest) FormattedDigests() []string {
	if dd.MsgType == PullMsgType_IDENTITY_MSG {
		return digestsToHex(dd.Digests)
	}
	return digestsAsStrings(dd.Digests)
}

//hash返回pvtDataDigest字节的sha256表示形式
func (dig *PvtDataDigest) Hash() (string, error) {
	b, err := proto.Marshal(dig)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(util.ComputeSHA256(b)), nil
}

//ToString返回此RemotePvtDataResponse的字符串表示形式
func (res *RemotePvtDataResponse) ToString() string {
	a := make([]string, len(res.Elements))
	for i, el := range res.Elements {
		a[i] = fmt.Sprintf("%s with %d elements", el.Digest.String(), len(el.Payload))
	}
	return fmt.Sprintf("%v", a)
}

func digestsAsStrings(digests [][]byte) []string {
	a := make([]string, len(digests))
	for i, dig := range digests {
		a[i] = string(dig)
	}
	return a
}

func digestsToHex(digests [][]byte) []string {
	a := make([]string, len(digests))
	for i, dig := range digests {
		a[i] = hex.EncodeToString(dig)
	}
	return a
}

//LedgerHeight返回指定的分类帐高度
//在StateInfo消息中
func (msg *StateInfo) LedgerHeight() (uint64, error) {
	if msg.Properties != nil {
		return msg.Properties.LedgerHeight, nil
	}
	return 0, errors.New("properties undefined")
}

//ABS返回ABS（A-B）
func abs(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}
