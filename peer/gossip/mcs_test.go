
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package gossip

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/localmsp"
	mockscrypto "github.com/hyperledger/fabric/common/mocks/crypto"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/peer/gossip/mocks"
	"github.com/hyperledger/fabric/protos/common"
	pmsp "github.com/hyperledger/fabric/protos/msp"
	protospeer "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPKIidOfCert(t *testing.T) {
	deserializersManager := &mocks.DeserializersManager{
		LocalDeserializer: &mocks.IdentityDeserializer{Identity: []byte("Alice"), Msg: []byte("msg1"), Mock: mock.Mock{}},
	}
	msgCryptoService := NewMCS(&mocks.ChannelPolicyManagerGetterWithManager{},
		&mockscrypto.LocalSigner{Identity: []byte("Alice")},
		deserializersManager,
	)

	peerIdentity := []byte("Alice")
	pkid := msgCryptoService.GetPKIidOfCert(peerIdentity)

//检查pkid是否不为零
	assert.NotNil(t, pkid, "PKID must be different from nil")
//检查pkid是否正确计算
	id, err := deserializersManager.Deserialize(peerIdentity)
	assert.NoError(t, err, "Failed getting validated identity from [% x]", []byte(peerIdentity))
	idRaw := append([]byte(id.Mspid), id.IdBytes...)
	assert.NoError(t, err, "Failed marshalling identity identifier [% x]: [%s]", peerIdentity, err)
	digest, err := factory.GetDefault().Hash(idRaw, &bccsp.SHA256Opts{})
	assert.NoError(t, err, "Failed computing digest of serialized identity [% x]", []byte(peerIdentity))
	assert.Equal(t, digest, []byte(pkid), "PKID must be the SHA2-256 of peerIdentity")

//通过将mspid与idbytes连接来计算pki-id。
//确保代码中没有引入其他字段
	v := reflect.Indirect(reflect.ValueOf(id)).Type()
	fieldsThatStartWithXXX := 0
	for i := 0; i < v.NumField(); i++ {
		if strings.Index(v.Field(i).Name, "XXX_") == 0 {
			fieldsThatStartWithXXX++
		}
	}
	assert.Equal(t, 2+fieldsThatStartWithXXX, v.NumField())
}

func TestPKIidOfNil(t *testing.T) {
	msgCryptoService := NewMCS(&mocks.ChannelPolicyManagerGetter{}, localmsp.NewSigner(), mgmt.NewDeserializersManager())

	pkid := msgCryptoService.GetPKIidOfCert(nil)
//检查pkid是否不为零
	assert.Nil(t, pkid, "PKID must be nil")
}

func TestValidateIdentity(t *testing.T) {
	deserializersManager := &mocks.DeserializersManager{
		LocalDeserializer: &mocks.IdentityDeserializer{Identity: []byte("Alice"), Msg: []byte("msg1"), Mock: mock.Mock{}},
		ChannelDeserializers: map[string]msp.IdentityDeserializer{
			"A": &mocks.IdentityDeserializer{Identity: []byte("Bob"), Msg: []byte("msg2"), Mock: mock.Mock{}},
		},
	}
	msgCryptoService := NewMCS(
		&mocks.ChannelPolicyManagerGetterWithManager{},
		&mockscrypto.LocalSigner{Identity: []byte("Charlie")},
		deserializersManager,
	)

	err := msgCryptoService.ValidateIdentity([]byte("Alice"))
	assert.NoError(t, err)

	err = msgCryptoService.ValidateIdentity([]byte("Bob"))
	assert.NoError(t, err)

	err = msgCryptoService.ValidateIdentity([]byte("Charlie"))
	assert.Error(t, err)

	err = msgCryptoService.ValidateIdentity(nil)
	assert.Error(t, err)

//现在，假装身份没有形成
	deserializersManager.ChannelDeserializers["A"].(*mocks.IdentityDeserializer).On("IsWellFormed", mock.Anything).Return(errors.New("invalid form"))
	err = msgCryptoService.ValidateIdentity([]byte("Bob"))
	assert.Error(t, err)
	assert.Equal(t, "identity is not well formed: invalid form", err.Error())

	deserializersManager.LocalDeserializer.(*mocks.IdentityDeserializer).On("IsWellFormed", mock.Anything).Return(errors.New("invalid form"))
	err = msgCryptoService.ValidateIdentity([]byte("Alice"))
	assert.Error(t, err)
	assert.Equal(t, "identity is not well formed: invalid form", err.Error())
}

func TestSign(t *testing.T) {
	msgCryptoService := NewMCS(
		&mocks.ChannelPolicyManagerGetter{},
		&mockscrypto.LocalSigner{Identity: []byte("Alice")},
		mgmt.NewDeserializersManager(),
	)

	msg := []byte("Hello World!!!")
	sigma, err := msgCryptoService.Sign(msg)
	assert.NoError(t, err, "Failed generating signature")
	assert.NotNil(t, sigma, "Signature must be different from nil")
}

func TestVerify(t *testing.T) {
	msgCryptoService := NewMCS(
		&mocks.ChannelPolicyManagerGetterWithManager{
			Managers: map[string]policies.Manager{
				"A": &mocks.ChannelPolicyManager{
					Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: []byte("Bob"), Msg: []byte("msg2"), Mock: mock.Mock{}}},
				},
				"B": &mocks.ChannelPolicyManager{
					Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: []byte("Charlie"), Msg: []byte("msg3"), Mock: mock.Mock{}}},
				},
				"C": nil,
			},
		},
		&mockscrypto.LocalSigner{Identity: []byte("Alice")},
		&mocks.DeserializersManager{
			LocalDeserializer: &mocks.IdentityDeserializer{Identity: []byte("Alice"), Msg: []byte("msg1"), Mock: mock.Mock{}},
			ChannelDeserializers: map[string]msp.IdentityDeserializer{
				"A": &mocks.IdentityDeserializer{Identity: []byte("Bob"), Msg: []byte("msg2"), Mock: mock.Mock{}},
				"B": &mocks.IdentityDeserializer{Identity: []byte("Charlie"), Msg: []byte("msg3"), Mock: mock.Mock{}},
				"C": &mocks.IdentityDeserializer{Identity: []byte("Dave"), Msg: []byte("msg4"), Mock: mock.Mock{}},
			},
		},
	)

	msg := []byte("msg1")
	sigma, err := msgCryptoService.Sign(msg)
	assert.NoError(t, err, "Failed generating signature")

	err = msgCryptoService.Verify(api.PeerIdentityType("Alice"), sigma, msg)
	assert.NoError(t, err, "Alice should verify the signature")

	err = msgCryptoService.Verify(api.PeerIdentityType("Bob"), sigma, msg)
	assert.Error(t, err, "Bob should not verify the signature")

	err = msgCryptoService.Verify(api.PeerIdentityType("Charlie"), sigma, msg)
	assert.Error(t, err, "Charlie should not verify the signature")

	sigma, err = msgCryptoService.Sign(msg)
	assert.NoError(t, err)
	err = msgCryptoService.Verify(api.PeerIdentityType("Dave"), sigma, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Could not acquire policy manager")

//检查无效参数
	assert.Error(t, msgCryptoService.Verify(nil, sigma, msg))
}

func TestVerifyBlock(t *testing.T) {
	aliceSigner := &mockscrypto.LocalSigner{Identity: []byte("Alice")}
	policyManagerGetter := &mocks.ChannelPolicyManagerGetterWithManager{
		Managers: map[string]policies.Manager{
			"A": &mocks.ChannelPolicyManager{
				Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: []byte("Bob"), Msg: []byte("msg2"), Mock: mock.Mock{}}},
			},
			"B": &mocks.ChannelPolicyManager{
				Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: []byte("Charlie"), Msg: []byte("msg3"), Mock: mock.Mock{}}},
			},
			"C": &mocks.ChannelPolicyManager{
				Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: []byte("Alice"), Msg: []byte("msg1"), Mock: mock.Mock{}}},
			},
			"D": &mocks.ChannelPolicyManager{
				Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: []byte("Alice"), Msg: []byte("msg1"), Mock: mock.Mock{}}},
			},
		},
	}

	msgCryptoService := NewMCS(
		policyManagerGetter,
		aliceSigner,
		&mocks.DeserializersManager{
			LocalDeserializer: &mocks.IdentityDeserializer{Identity: []byte("Alice"), Msg: []byte("msg1"), Mock: mock.Mock{}},
			ChannelDeserializers: map[string]msp.IdentityDeserializer{
				"A": &mocks.IdentityDeserializer{Identity: []byte("Bob"), Msg: []byte("msg2"), Mock: mock.Mock{}},
				"B": &mocks.IdentityDeserializer{Identity: []byte("Charlie"), Msg: []byte("msg3"), Mock: mock.Mock{}},
			},
		},
	)

//-准备测试有效块，Alice签字。
	blockRaw, msg := mockBlock(t, "C", 42, aliceSigner, nil)
	policyManagerGetter.Managers["C"].(*mocks.ChannelPolicyManager).Policy.(*mocks.Policy).Deserializer.(*mocks.IdentityDeserializer).Msg = msg
	blockRaw2, msg2 := mockBlock(t, "D", 42, aliceSigner, nil)
	policyManagerGetter.Managers["D"].(*mocks.ChannelPolicyManager).Policy.(*mocks.Policy).Deserializer.(*mocks.IdentityDeserializer).Msg = msg2

//-验证块
	assert.NoError(t, msgCryptoService.VerifyBlock([]byte("C"), 42, blockRaw))
//声明的序列号错误
	err := msgCryptoService.VerifyBlock([]byte("C"), 43, blockRaw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "but actual seqNum inside block is")
	delete(policyManagerGetter.Managers, "D")
	nilPolMgrErr := msgCryptoService.VerifyBlock([]byte("D"), 42, blockRaw2)
	assert.Contains(t, nilPolMgrErr.Error(), "Could not acquire policy manager")
	assert.Error(t, nilPolMgrErr)
	assert.Error(t, msgCryptoService.VerifyBlock([]byte("A"), 42, blockRaw))
	assert.Error(t, msgCryptoService.VerifyBlock([]byte("B"), 42, blockRaw))

//-准备测试无效块（错误数据有），Alice对其进行签名。
	blockRaw, msg = mockBlock(t, "C", 42, aliceSigner, []byte{0})
	policyManagerGetter.Managers["C"].(*mocks.ChannelPolicyManager).Policy.(*mocks.Policy).Deserializer.(*mocks.IdentityDeserializer).Msg = msg

//-验证块
	assert.Error(t, msgCryptoService.VerifyBlock([]byte("C"), 42, blockRaw))

//检查无效参数
	assert.Error(t, msgCryptoService.VerifyBlock([]byte("C"), 42, []byte{0, 1, 2, 3, 4}))
	assert.Error(t, msgCryptoService.VerifyBlock([]byte("C"), 42, nil))
}

func mockBlock(t *testing.T, channel string, seqNum uint64, localSigner crypto.LocalSigner, dataHash []byte) ([]byte, []byte) {
	block := common.NewBlock(seqNum, nil)

//向块引用通道“c”添加假事务
	sProp, _ := utils.MockSignedEndorserProposalOrPanic(channel, &protospeer.ChaincodeSpec{}, []byte("transactor"), []byte("transactor's signature"))
	sPropRaw, err := utils.Marshal(sProp)
	assert.NoError(t, err, "Failed marshalling signed proposal")
	block.Data.Data = [][]byte{sPropRaw}

//计算block.data的哈希值并放入头中
	if len(dataHash) != 0 {
		block.Header.DataHash = dataHash
	} else {
		block.Header.DataHash = block.Data.Hash()
	}

//将签名者的签名添加到块
	shdr, err := localSigner.NewSignatureHeader()
	assert.NoError(t, err, "Failed generating signature header")

	blockSignature := &common.MetadataSignature{
		SignatureHeader: utils.MarshalOrPanic(shdr),
	}

//注意，此值有意为零，因为此元数据仅与签名有关，因此没有其他元数据。
//元数据项已签名之外所需的信息。
	blockSignatureValue := []byte(nil)

	msg := util.ConcatenateBytes(blockSignatureValue, blockSignature.SignatureHeader, block.Header.Bytes())
	blockSignature.Signature, err = localSigner.Sign(msg)
	assert.NoError(t, err, "Failed signing block")

	block.Metadata.Metadata[common.BlockMetadataIndex_SIGNATURES] = utils.MarshalOrPanic(&common.Metadata{
		Value: blockSignatureValue,
		Signatures: []*common.MetadataSignature{
			blockSignature,
		},
	})

	blockRaw, err := proto.Marshal(block)
	assert.NoError(t, err, "Failed marshalling block")

	return blockRaw, msg
}

func TestExpiration(t *testing.T) {
	expirationDate := time.Now().Add(time.Minute)
	id1 := &pmsp.SerializedIdentity{
		Mspid:   "X509BasedMSP",
		IdBytes: []byte("X509BasedIdentity"),
	}

	x509IdentityBytes, _ := proto.Marshal(id1)

	id2 := &pmsp.SerializedIdentity{
		Mspid:   "nonX509BasedMSP",
		IdBytes: []byte("nonX509RawIdentity"),
	}

	nonX509IdentityBytes, _ := proto.Marshal(id2)

	deserializersManager := &mocks.DeserializersManager{
		LocalDeserializer: &mocks.IdentityDeserializer{
			Identity: []byte{1, 2, 3},
			Msg:      []byte{1, 2, 3},
		},
		ChannelDeserializers: map[string]msp.IdentityDeserializer{
			"X509BasedMSP": &mocks.IdentityDeserializerWithExpiration{
				Expiration: expirationDate,
				IdentityDeserializer: &mocks.IdentityDeserializer{
					Identity: x509IdentityBytes,
					Msg:      []byte("x509IdentityBytes"),
				},
			},
			"nonX509BasedMSP": &mocks.IdentityDeserializer{
				Identity: nonX509IdentityBytes,
				Msg:      []byte("nonX509IdentityBytes"),
			},
		},
	}
	msgCryptoService := NewMCS(
		&mocks.ChannelPolicyManagerGetterWithManager{},
		&mockscrypto.LocalSigner{Identity: []byte("Yacov")},
		deserializersManager,
	)

//绿色路径我检查到期日期是否如预期
	exp, err := msgCryptoService.Expiration(x509IdentityBytes)
	assert.NoError(t, err)
	assert.Equal(t, expirationDate.Second(), exp.Second())

//绿色路径II-非X509标识的过期时间为零
	exp, err = msgCryptoService.Expiration(nonX509IdentityBytes)
	assert.NoError(t, err)
	assert.Zero(t, exp)

//错误路径I-损坏X509标识并确保返回错误
	x509IdentityBytes = append(x509IdentityBytes, 0, 0, 0, 0, 0, 0)
	exp, err = msgCryptoService.Expiration(x509IdentityBytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No MSP found able to do that")
	assert.Zero(t, exp)
}
