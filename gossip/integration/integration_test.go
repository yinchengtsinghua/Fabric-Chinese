
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


package integration

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func init() {
	util.SetupTestLogging()
}

var (
	cryptSvc = &cryptoService{}
	secAdv   = &secAdviser{}
)
var defaultSecureDialOpts = func() []grpc.DialOption {
	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, grpc.WithInsecure())
	return dialOpts
}

//这只是一个演示如何实例化八卦组件的测试
func TestNewGossipCryptoService(t *testing.T) {
	setupTestEnv()
	s1 := grpc.NewServer()
	s2 := grpc.NewServer()
	s3 := grpc.NewServer()
	ll1, _ := net.Listen("tcp", fmt.Sprintf("%s:%d", "", 5611))
	ll2, _ := net.Listen("tcp", fmt.Sprintf("%s:%d", "", 5612))
	ll3, _ := net.Listen("tcp", fmt.Sprintf("%s:%d", "", 5613))
	endpoint1 := "localhost:5611"
	endpoint2 := "localhost:5612"
	endpoint3 := "localhost:5613"
	msptesttools.LoadMSPSetupForTesting()
	peerIdentity, _ := mgmt.GetLocalSigningIdentityOrPanic().Serialize()
	g1, err := NewGossipComponent(peerIdentity, endpoint1, s1, secAdv, cryptSvc,
		defaultSecureDialOpts, nil)
	assert.NoError(t, err)
	g2, err := NewGossipComponent(peerIdentity, endpoint2, s2, secAdv, cryptSvc,
		defaultSecureDialOpts, nil, endpoint1)
	assert.NoError(t, err)
	g3, err := NewGossipComponent(peerIdentity, endpoint3, s3, secAdv, cryptSvc,
		defaultSecureDialOpts, nil, endpoint1)
	assert.NoError(t, err)
	defer g1.Stop()
	defer g2.Stop()
	defer g3.Stop()
	go s1.Serve(ll1)
	go s2.Serve(ll2)
	go s3.Serve(ll3)
}

func setupTestEnv() {
	viper.SetConfigName("core")
	viper.SetEnvPrefix("CORE")
	configtest.AddDevConfigPath(nil)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
if err != nil { //处理读取配置文件时的错误
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
}

type secAdviser struct {
}

func (sa *secAdviser) OrgByPeerIdentity(api.PeerIdentityType) api.OrgIdentityType {
	return api.OrgIdentityType("SampleOrg")
}

type cryptoService struct {
}

func (s *cryptoService) Expiration(peerIdentity api.PeerIdentityType) (time.Time, error) {
	return time.Now().Add(time.Hour), nil
}

func (s *cryptoService) GetPKIidOfCert(peerIdentity api.PeerIdentityType) common.PKIidType {
	return common.PKIidType(peerIdentity)
}

func (s *cryptoService) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
	return nil
}

func (s *cryptoService) Sign(msg []byte) ([]byte, error) {
	return msg, nil
}

func (s *cryptoService) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	return nil
}

func (s *cryptoService) VerifyByChannel(chainID common.ChainID, peerIdentity api.PeerIdentityType, signature, message []byte) error {
	return nil
}

func (s *cryptoService) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
	return nil
}
