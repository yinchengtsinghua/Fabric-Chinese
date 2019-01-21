
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
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	msgCryptoService = &naiveCryptoService{revokedIdentities: map[string]struct{}{}}
	dummyID          = api.PeerIdentityType("dummyID")
)

type naiveCryptoService struct {
	mock.Mock
	revokedIdentities map[string]struct{}
}

var noopPurgeTrigger = func(_ common.PKIidType, _ api.PeerIdentityType) {}

func init() {
	util.SetupTestLogging()
	msgCryptoService.On("Expiration", api.PeerIdentityType(dummyID)).Return(time.Now().Add(time.Hour), nil)
	msgCryptoService.On("Expiration", api.PeerIdentityType("yacovm")).Return(time.Now().Add(time.Hour), nil)
	msgCryptoService.On("Expiration", api.PeerIdentityType("not-yacovm")).Return(time.Now().Add(time.Hour), nil)
	msgCryptoService.On("Expiration", api.PeerIdentityType("invalidIdentity")).Return(time.Now().Add(time.Hour), nil)
}

func (cs *naiveCryptoService) OrgByPeerIdentity(id api.PeerIdentityType) api.OrgIdentityType {
	found := false
	for _, call := range cs.Mock.ExpectedCalls {
		if call.Method == "OrgByPeerIdentity" {
			found = true
		}
	}
	if !found {
		return nil
	}
	return cs.Called(id).Get(0).(api.OrgIdentityType)
}

func (cs *naiveCryptoService) Expiration(peerIdentity api.PeerIdentityType) (time.Time, error) {
	args := cs.Called(peerIdentity)
	t, err := args.Get(0), args.Get(1)
	if err == nil {
		return t.(time.Time), nil
	}
	return time.Time{}, err.(error)
}

func (cs *naiveCryptoService) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
	if _, isRevoked := cs.revokedIdentities[string(cs.GetPKIidOfCert(peerIdentity))]; isRevoked {
		return errors.New("revoked")
	}
	return nil
}

//getpkiidofcert返回对等身份的pki-id
func (*naiveCryptoService) GetPKIidOfCert(peerIdentity api.PeerIdentityType) common.PKIidType {
	return common.PKIidType(peerIdentity)
}

//verifyblock如果块被正确签名，则返回nil，
//else返回错误
func (*naiveCryptoService) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
	return nil
}

//verifybychannel验证上下文中消息上对等方的签名
//特定通道的
func (*naiveCryptoService) VerifyByChannel(_ common.ChainID, _ api.PeerIdentityType, _, _ []byte) error {
	return nil
}

//用该对等方的签名密钥和输出对消息进行签名
//如果没有出现错误，则返回签名。
func (*naiveCryptoService) Sign(msg []byte) ([]byte, error) {
	return msg, nil
}

//验证检查签名是否是对等验证密钥下消息的有效签名。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peercert为nil，则根据该对等方的验证密钥验证签名。
func (*naiveCryptoService) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	equal := bytes.Equal(signature, message)
	if !equal {
		return fmt.Errorf("Wrong certificate")
	}
	return nil
}

func TestPut(t *testing.T) {
	idStore := NewIdentityMapper(msgCryptoService, dummyID, noopPurgeTrigger, msgCryptoService)
	identity := []byte("yacovm")
	identity2 := []byte("not-yacovm")
	identity3 := []byte("invalidIdentity")
	msgCryptoService.revokedIdentities[string(identity3)] = struct{}{}
	pkiID := msgCryptoService.GetPKIidOfCert(api.PeerIdentityType(identity))
	pkiID2 := msgCryptoService.GetPKIidOfCert(api.PeerIdentityType(identity2))
	pkiID3 := msgCryptoService.GetPKIidOfCert(api.PeerIdentityType(identity3))
	assert.NoError(t, idStore.Put(pkiID, identity))
	assert.NoError(t, idStore.Put(pkiID, identity))
	assert.Error(t, idStore.Put(nil, identity))
	assert.Error(t, idStore.Put(pkiID2, nil))
	assert.Error(t, idStore.Put(pkiID2, identity))
	assert.Error(t, idStore.Put(pkiID, identity2))
	assert.Error(t, idStore.Put(pkiID3, identity3))
}

func TestGet(t *testing.T) {
	idStore := NewIdentityMapper(msgCryptoService, dummyID, noopPurgeTrigger, msgCryptoService)
	identity := []byte("yacovm")
	identity2 := []byte("not-yacovm")
	pkiID := msgCryptoService.GetPKIidOfCert(api.PeerIdentityType(identity))
	pkiID2 := msgCryptoService.GetPKIidOfCert(api.PeerIdentityType(identity2))
	assert.NoError(t, idStore.Put(pkiID, identity))
	cert, err := idStore.Get(pkiID)
	assert.NoError(t, err)
	assert.Equal(t, api.PeerIdentityType(identity), cert)
	cert, err = idStore.Get(pkiID2)
	assert.Nil(t, cert)
	assert.Error(t, err)
}

func TestVerify(t *testing.T) {
	idStore := NewIdentityMapper(msgCryptoService, dummyID, noopPurgeTrigger, msgCryptoService)
	identity := []byte("yacovm")
	identity2 := []byte("not-yacovm")
	pkiID := msgCryptoService.GetPKIidOfCert(api.PeerIdentityType(identity))
	pkiID2 := msgCryptoService.GetPKIidOfCert(api.PeerIdentityType(identity2))
	idStore.Put(pkiID, api.PeerIdentityType(identity))
	signed, err := idStore.Sign([]byte("bla bla"))
	assert.NoError(t, err)
	assert.NoError(t, idStore.Verify(pkiID, signed, []byte("bla bla")))
	assert.Error(t, idStore.Verify(pkiID2, signed, []byte("bla bla")))
}

func TestListInvalidIdentities(t *testing.T) {
	deletedIdentities := make(chan string, 1)
	assertDeletedIdentity := func(expected string) {
		select {
		case <-time.After(time.Second * 10):
			t.Fatalf("Didn't detect a deleted identity, expected %s to be deleted", expected)
		case actual := <-deletedIdentities:
			assert.Equal(t, expected, actual)
		}
	}
//将基于时间的过期时间限制设置为较小的值
	SetIdentityUsageThreshold(time.Millisecond * 500)
	assert.Equal(t, time.Millisecond*500, GetIdentityUsageThreshold())
	selfPKIID := msgCryptoService.GetPKIidOfCert(dummyID)
	idStore := NewIdentityMapper(msgCryptoService, dummyID, func(_ common.PKIidType, identity api.PeerIdentityType) {
		deletedIdentities <- string(identity)
	}, msgCryptoService)
	identity := []byte("yacovm")
//测试吊销的身份
	pkiID := msgCryptoService.GetPKIidOfCert(api.PeerIdentityType(identity))
	assert.NoError(t, idStore.Put(pkiID, api.PeerIdentityType(identity)))
	cert, err := idStore.Get(pkiID)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
//吊销证书
	msgCryptoService.revokedIdentities[string(pkiID)] = struct{}{}
	idStore.SuspectPeers(func(_ api.PeerIdentityType) bool {
		return true
	})
//确保它不再被发现
	cert, err = idStore.Get(pkiID)
	assert.Error(t, err)
	assert.Nil(t, cert)
	assertDeletedIdentity("yacovm")

//清除MCS吊销模拟
	msgCryptoService.revokedIdentities = map[string]struct{}{}
//现在，测试尚未使用的证书
//很长一段时间
//添加回标识
	pkiID = msgCryptoService.GetPKIidOfCert(api.PeerIdentityType(identity))
	assert.NoError(t, idStore.Put(pkiID, api.PeerIdentityType(identity)))
//同时检查它是否存在
	cert, err = idStore.Get(pkiID)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
	time.Sleep(time.Second * 3)
//确保它已经过期
	cert, err = idStore.Get(pkiID)
	assert.Error(t, err)
	assert.Nil(t, cert)
	assertDeletedIdentity("yacovm")
//确保我们自己的身份没有过期
	_, err = idStore.Get(selfPKIID)
	assert.NoError(t, err)

//现在测试一个经常使用的标识是否不会过期
//添加回标识
	pkiID = msgCryptoService.GetPKIidOfCert(api.PeerIdentityType(identity))
	assert.NoError(t, idStore.Put(pkiID, api.PeerIdentityType(identity)))
	stopChan := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case <-time.After(time.Millisecond * 10):
				idStore.Get(pkiID)
			}
		}
	}()
	time.Sleep(time.Second * 3)
//确保它没有过期，即使时间已经过去
	cert, err = idStore.Get(pkiID)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
	stopChan <- struct{}{}
//停止身份存储-这将使定期取消使用
//呼气停止
	idStore.Stop()
	time.Sleep(time.Second * 3)
//确保它没有过期，即使时间已经过去
	cert, err = idStore.Get(pkiID)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}

func TestExpiration(t *testing.T) {
	deletedIdentities := make(chan string, 1)
	SetIdentityUsageThreshold(time.Second * 500)
	idStore := NewIdentityMapper(msgCryptoService, dummyID, func(_ common.PKIidType, identity api.PeerIdentityType) {
		deletedIdentities <- string(identity)
	}, msgCryptoService)
	assertDeletedIdentity := func(expected string) {
		select {
		case <-time.After(time.Second * 10):
			t.Fatalf("Didn't detect a deleted identity, expected %s to be deleted", expected)
		case actual := <-deletedIdentities:
			assert.Equal(t, expected, actual)
		}
	}
	x509Identity := api.PeerIdentityType("x509Identity")
	expiredX509Identity := api.PeerIdentityType("expiredX509Identity")
	nonX509Identity := api.PeerIdentityType("nonX509Identity")
	notSupportedIdentity := api.PeerIdentityType("notSupportedIdentity")
	x509PkiID := idStore.GetPKIidOfCert(x509Identity)
	expiredX509PkiID := idStore.GetPKIidOfCert(expiredX509Identity)
	nonX509PkiID := idStore.GetPKIidOfCert(nonX509Identity)
	notSupportedPkiID := idStore.GetPKIidOfCert(notSupportedIdentity)
	msgCryptoService.On("Expiration", x509Identity).Return(time.Now().Add(time.Second), nil)
	msgCryptoService.On("Expiration", expiredX509Identity).Return(time.Now().Add(-time.Second), nil)
	msgCryptoService.On("Expiration", nonX509Identity).Return(time.Time{}, nil)
	msgCryptoService.On("Expiration", notSupportedIdentity).Return(time.Time{}, errors.New("no MSP supports given identity"))
//添加所有标识
	err := idStore.Put(x509PkiID, x509Identity)
	assert.NoError(t, err)
	err = idStore.Put(expiredX509PkiID, expiredX509Identity)
	assert.Equal(t, "identity expired", err.Error())
	err = idStore.Put(nonX509PkiID, nonX509Identity)
	assert.NoError(t, err)
	err = idStore.Put(notSupportedPkiID, notSupportedIdentity)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no MSP supports given identity")

//确保存储中存在X509证书和非X509证书
	returnedIdentity, err := idStore.Get(x509PkiID)
	assert.NoError(t, err)
	assert.NotEmpty(t, returnedIdentity)

	returnedIdentity, err = idStore.Get(nonX509PkiID)
	assert.NoError(t, err)
	assert.NotEmpty(t, returnedIdentity)

//等待X509标识过期
	time.Sleep(time.Second * 3)

//确保现在只存在非X509标识
	returnedIdentity, err = idStore.Get(x509PkiID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PKIID wasn't found")
	assert.Empty(t, returnedIdentity)
	assertDeletedIdentity("x509Identity")

	returnedIdentity, err = idStore.Get(nonX509PkiID)
	assert.NoError(t, err)
	assert.NotEmpty(t, returnedIdentity)

//确保当它被吊销时，不会为它取消过期计时器。
	msgCryptoService.revokedIdentities[string(nonX509PkiID)] = struct{}{}
	idStore.SuspectPeers(func(_ api.PeerIdentityType) bool {
		return true
	})
	assertDeletedIdentity("nonX509Identity")
	msgCryptoService.revokedIdentities = map[string]struct{}{}
}

func TestExpirationPanic(t *testing.T) {
	identity3 := []byte("invalidIdentity")
	msgCryptoService.revokedIdentities[string(identity3)] = struct{}{}
	assert.Panics(t, func() {
		NewIdentityMapper(msgCryptoService, identity3, noopPurgeTrigger, msgCryptoService)
	})
}

func TestIdentityInfo(t *testing.T) {
	cs := &naiveCryptoService{}
	alice := api.PeerIdentityType("alicePeer")
	bob := api.PeerIdentityType("bobPeer")
	aliceID := cs.GetPKIidOfCert(alice)
	bobId := cs.GetPKIidOfCert(bob)
	cs.On("OrgByPeerIdentity", dummyID).Return(api.OrgIdentityType("D"))
	cs.On("OrgByPeerIdentity", alice).Return(api.OrgIdentityType("A"))
	cs.On("OrgByPeerIdentity", bob).Return(api.OrgIdentityType("B"))
	cs.On("Expiration", mock.Anything).Return(time.Now().Add(time.Minute), nil)
	idStore := NewIdentityMapper(cs, dummyID, noopPurgeTrigger, cs)
	idStore.Put(aliceID, alice)
	idStore.Put(bobId, bob)
	for org, id := range idStore.IdentityInfo().ByOrg() {
		identity := string(id[0].Identity)
		pkiID := string(id[0].PKIId)
		orgId := string(id[0].Organization)
		assert.Equal(t, org, orgId)
		assert.Equal(t, strings.ToLower(org), string(identity[0]))
		assert.Equal(t, strings.ToLower(org), string(pkiID[0]))
	}
}
