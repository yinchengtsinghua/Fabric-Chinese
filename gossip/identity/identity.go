
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


package identity

import (
	"bytes"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	errors "github.com/pkg/errors"
)

var (
//IdentityUsageThreshold设置标识
//无法在删除某些签名之前对其进行验证
	usageThreshold = time.Hour
)

//映射器保存pkiid之间的映射
//到对等方的证书（身份）
type Mapper interface {
//将标识与其给定的pkiid关联，并返回错误
//如果给定的pkiid与身份不匹配
	Put(pkiID common.PKIidType, identity api.PeerIdentityType) error

//get返回给定pkiid的标识，或者返回错误（如果是这样的标识）
//找不到
	Get(pkiID common.PKIidType) (api.PeerIdentityType, error)

//签名签署消息，成功时返回签名消息
//或者失败时出错
	Sign(msg []byte) ([]byte, error)

//验证验证验证签名消息
	Verify(vkID, signature, message []byte) error

//getpkiidofcert返回证书的pki-id
	GetPKIidOfCert(api.PeerIdentityType) common.PKIidType

//SuspectPeers重新验证与给定谓词匹配的所有对等方
	SuspectPeers(isSuspected api.PeerSuspector)

//IdentityInfo返回已知对等标识的信息
	IdentityInfo() api.PeerIdentitySet

//停止停止映射器的所有后台计算
	Stop()
}

type purgeTrigger func(pkiID common.PKIidType, identity api.PeerIdentityType)

//IdentityMapperImpl是实现映射器的结构
type identityMapperImpl struct {
	onPurge    purgeTrigger
	mcs        api.MessageCryptoService
	sa         api.SecurityAdvisor
	pkiID2Cert map[string]*storedIdentity
	sync.RWMutex
	stopChan chan struct{}
	sync.Once
	selfPKIID string
}

//newIdentityMapper方法，我们只需要一个对messageCryptoService的引用
func NewIdentityMapper(mcs api.MessageCryptoService, selfIdentity api.PeerIdentityType, onPurge purgeTrigger, sa api.SecurityAdvisor) Mapper {
	selfPKIID := mcs.GetPKIidOfCert(selfIdentity)
	idMapper := &identityMapperImpl{
		onPurge:    onPurge,
		mcs:        mcs,
		pkiID2Cert: make(map[string]*storedIdentity),
		stopChan:   make(chan struct{}),
		selfPKIID:  string(selfPKIID),
		sa:         sa,
	}
	if err := idMapper.Put(selfPKIID, selfIdentity); err != nil {
		panic(errors.Wrap(err, "Failed putting our own identity into the identity mapper"))
	}
	go idMapper.periodicalPurgeUnusedIdentities()
	return idMapper
}

func (is *identityMapperImpl) periodicalPurgeUnusedIdentities() {
	usageTh := GetIdentityUsageThreshold()
	for {
		select {
		case <-is.stopChan:
			return
		case <-time.After(usageTh / 10):
			is.SuspectPeers(func(_ api.PeerIdentityType) bool {
				return false
			})
		}
	}
}

//将标识与其给定的pkiid关联，并返回错误
//如果给定的pkiid与身份不匹配
func (is *identityMapperImpl) Put(pkiID common.PKIidType, identity api.PeerIdentityType) error {
	if pkiID == nil {
		return errors.New("PKIID is nil")
	}
	if identity == nil {
		return errors.New("identity is nil")
	}

	expirationDate, err := is.mcs.Expiration(identity)
	if err != nil {
		return errors.Wrap(err, "failed classifying identity")
	}

	if err := is.mcs.ValidateIdentity(identity); err != nil {
		return err
	}

	id := is.mcs.GetPKIidOfCert(identity)
	if !bytes.Equal(pkiID, id) {
		return errors.New("identity doesn't match the computed pkiID")
	}

	is.Lock()
	defer is.Unlock()
//检查标识是否已存在。
//如果是这样，就不需要覆盖它。
	if _, exists := is.pkiID2Cert[string(pkiID)]; exists {
		return nil
	}

	var expirationTimer *time.Timer
	if !expirationDate.IsZero() {
		if time.Now().After(expirationDate) {
			return errors.New("identity expired")
		}
//身份将在到期后一毫秒内被删除。
		timeToLive := expirationDate.Add(time.Millisecond).Sub(time.Now())
		expirationTimer = time.AfterFunc(timeToLive, func() {
			is.delete(pkiID, identity)
		})
	}

	is.pkiID2Cert[string(id)] = newStoredIdentity(pkiID, identity, expirationTimer, is.sa.OrgByPeerIdentity(identity))
	return nil
}

//get返回给定pkiid的标识，或者返回错误（如果是这样的标识）
//找不到
func (is *identityMapperImpl) Get(pkiID common.PKIidType) (api.PeerIdentityType, error) {
	is.RLock()
	defer is.RUnlock()
	storedIdentity, exists := is.pkiID2Cert[string(pkiID)]
	if !exists {
		return nil, errors.New("PKIID wasn't found")
	}
	return storedIdentity.fetchIdentity(), nil
}

//签名签署消息，成功时返回签名消息
//或者失败时出错
func (is *identityMapperImpl) Sign(msg []byte) ([]byte, error) {
	return is.mcs.Sign(msg)
}

func (is *identityMapperImpl) Stop() {
	is.Once.Do(func() {
		is.stopChan <- struct{}{}
	})
}

//验证验证验证签名消息
func (is *identityMapperImpl) Verify(vkID, signature, message []byte) error {
	cert, err := is.Get(vkID)
	if err != nil {
		return err
	}
	return is.mcs.Verify(cert, signature, message)
}

//getpkiidofcert返回证书的pki-id
func (is *identityMapperImpl) GetPKIidOfCert(identity api.PeerIdentityType) common.PKIidType {
	return is.mcs.GetPKIidOfCert(identity)
}

//SuspectPeers重新验证与给定谓词匹配的所有对等方
func (is *identityMapperImpl) SuspectPeers(isSuspected api.PeerSuspector) {
	for _, identity := range is.validateIdentities(isSuspected) {
		identity.cancelExpirationTimer()
		is.delete(identity.pkiID, identity.peerIdentity)
	}
}

//validateIdentities返回已吊销、过期或尚未过期的标识列表。
//长期使用
func (is *identityMapperImpl) validateIdentities(isSuspected api.PeerSuspector) []*storedIdentity {
	now := time.Now()
	usageTh := GetIdentityUsageThreshold()
	is.RLock()
	defer is.RUnlock()
	var revokedIdentities []*storedIdentity
	for pkiID, storedIdentity := range is.pkiID2Cert {
		if pkiID != is.selfPKIID && storedIdentity.fetchLastAccessTime().Add(usageTh).Before(now) {
			revokedIdentities = append(revokedIdentities, storedIdentity)
			continue
		}
		if !isSuspected(storedIdentity.peerIdentity) {
			continue
		}
		if err := is.mcs.ValidateIdentity(storedIdentity.fetchIdentity()); err != nil {
			revokedIdentities = append(revokedIdentities, storedIdentity)
		}
	}
	return revokedIdentities
}

//IdentityInfo返回已知对等标识的信息
func (is *identityMapperImpl) IdentityInfo() api.PeerIdentitySet {
	var res api.PeerIdentitySet
	is.RLock()
	defer is.RUnlock()
	for _, storedIdentity := range is.pkiID2Cert {
		res = append(res, api.PeerIdentityInfo{
			Identity:     storedIdentity.peerIdentity,
			PKIId:        storedIdentity.pkiID,
			Organization: storedIdentity.orgId,
		})
	}
	return res
}

func (is *identityMapperImpl) delete(pkiID common.PKIidType, identity api.PeerIdentityType) {
	is.Lock()
	defer is.Unlock()
	is.onPurge(pkiID, identity)
	delete(is.pkiID2Cert, string(pkiID))
}

type storedIdentity struct {
	pkiID           common.PKIidType
	lastAccessTime  int64
	peerIdentity    api.PeerIdentityType
	orgId           api.OrgIdentityType
	expirationTimer *time.Timer
}

func newStoredIdentity(pkiID common.PKIidType, identity api.PeerIdentityType, expirationTimer *time.Timer, org api.OrgIdentityType) *storedIdentity {
	return &storedIdentity{
		pkiID:           pkiID,
		lastAccessTime:  time.Now().UnixNano(),
		peerIdentity:    identity,
		expirationTimer: expirationTimer,
		orgId:           org,
	}
}

func (si *storedIdentity) fetchIdentity() api.PeerIdentityType {
	atomic.StoreInt64(&si.lastAccessTime, time.Now().UnixNano())
	return si.peerIdentity
}

func (si *storedIdentity) fetchLastAccessTime() time.Time {
	return time.Unix(0, atomic.LoadInt64(&si.lastAccessTime))
}

func (si *storedIdentity) cancelExpirationTimer() {
	if si.expirationTimer == nil {
		return
	}
	si.expirationTimer.Stop()
}

//setIdentityUsageThreshold设置标识的使用阈值。
//在给定时间内至少未使用一次的标识
//被清除
func SetIdentityUsageThreshold(duration time.Duration) {
	atomic.StoreInt64((*int64)(&usageThreshold), int64(duration))
}

//GetIdentityUsageThreshold返回标识的使用阈值。
//在使用阈值期间至少未使用一次的标识
//清除持续时间。
func GetIdentityUsageThreshold() time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(&usageThreshold)))
}
