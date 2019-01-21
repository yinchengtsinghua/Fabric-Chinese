
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
日立美国有限公司版权所有。

SPDX许可证标识符：Apache-2.0
**/


package node

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/viperutil"
	"github.com/hyperledger/fabric/core/handlers/library"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestStartCmd(t *testing.T) {
	defer viper.Reset()
	g := NewGomegaWithT(t)

	tempDir, err := ioutil.TempDir("", "startcmd")
	g.Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tempDir)

	viper.Set("peer.address", "localhost:6051")
	viper.Set("peer.listenAddress", "0.0.0.0:6051")
	viper.Set("peer.chaincodeListenAddress", "0.0.0.0:6052")
	viper.Set("peer.fileSystemPath", tempDir)
	viper.Set("chaincode.executetimeout", "30s")
	viper.Set("chaincode.mode", "dev")

	msptesttools.LoadMSPSetupForTesting()

	go func() {
		cmd := startCmd()
		assert.NoError(t, cmd.Execute(), "expected to successfully start command")
	}()

	grpcProbe := func(addr string) bool {
		c, err := grpc.Dial(addr, grpc.WithBlock(), grpc.WithInsecure())
		if err == nil {
			c.Close()
			return true
		}
		return false
	}
	g.Eventually(grpcProbe("localhost:6051")).Should(BeTrue())
}

func TestAdminHasSeparateListener(t *testing.T) {
	assert.False(t, adminHasSeparateListener("0.0.0.0:7051", ""))

	assert.Panics(t, func() {
		adminHasSeparateListener("foo", "blabla")
	})

	assert.Panics(t, func() {
		adminHasSeparateListener("0.0.0.0:7051", "blabla")
	})

	assert.False(t, adminHasSeparateListener("0.0.0.0:7051", "0.0.0.0:7051"))
	assert.False(t, adminHasSeparateListener("0.0.0.0:7051", "127.0.0.1:7051"))
	assert.True(t, adminHasSeparateListener("0.0.0.0:7051", "0.0.0.0:7055"))
}

func TestHandlerMap(t *testing.T) {
	config1 := `
  peer:
    handlers:
      authFilters:
        -
          name: filter1
          library: /opt/lib/filter1.so
        -
          name: filter2
  `
	viper.SetConfigType("yaml")
	err := viper.ReadConfig(bytes.NewBuffer([]byte(config1)))
	assert.NoError(t, err)

	libConf := library.Config{}
	err = viperutil.EnhancedExactUnmarshalKey("peer.handlers", &libConf)
	assert.NoError(t, err)
	assert.Len(t, libConf.AuthFilters, 2, "expected two filters")
	assert.Equal(t, "/opt/lib/filter1.so", libConf.AuthFilters[0].Library)
	assert.Equal(t, "filter2", libConf.AuthFilters[1].Name)
}

func TestComputeChaincodeEndpoint(t *testing.T) {
 /**场景1：未设置chaincodeaddress和chaincodelistenaddress**/
	viper.Set(chaincodeAddrKey, nil)
	viper.Set(chaincodeListenAddrKey, nil)
//场景1.1：对等地址为0.0.0.0
//ComputechaincodeEndpoint将返回错误
	peerAddress0 := "0.0.0.0"
	ccEndpoint, err := computeChaincodeEndpoint(peerAddress0)
	assert.Error(t, err)
	assert.Equal(t, "", ccEndpoint)
//场景1.2：对等地址不是0.0.0.0
//ChaincodeEndpoint将是对等地址：7052
	peerAddress := "127.0.0.1"
	ccEndpoint, err = computeChaincodeEndpoint(peerAddress)
	assert.NoError(t, err)
	assert.Equal(t, peerAddress+":7052", ccEndpoint)

 /**场景2：仅设置chaincodelistenaddress**/
//场景2.1:chaincodeListenAddress为0.0.0.0
	chaincodeListenPort := "8052"
	settingChaincodeListenAddress0 := "0.0.0.0:" + chaincodeListenPort
	viper.Set(chaincodeListenAddrKey, settingChaincodeListenAddress0)
	viper.Set(chaincodeAddrKey, nil)
//场景2.1.1：对等地址为0.0.0.0
//ComputechaincodeEndpoint将返回错误
	ccEndpoint, err = computeChaincodeEndpoint(peerAddress0)
	assert.Error(t, err)
	assert.Equal(t, "", ccEndpoint)
//场景2.1.2：对等地址不是0.0.0.0
//chaincodeEndpoint将是peerAddress:chaincodeListenPort
	ccEndpoint, err = computeChaincodeEndpoint(peerAddress)
	assert.NoError(t, err)
	assert.Equal(t, peerAddress+":"+chaincodeListenPort, ccEndpoint)
//场景2.2:chaincodeListenAddress不是0.0.0.0
//chaincodeEndpoint将是chaincodeListenAddress
	settingChaincodeListenAddress := "127.0.0.1:" + chaincodeListenPort
	viper.Set(chaincodeListenAddrKey, settingChaincodeListenAddress)
	viper.Set(chaincodeAddrKey, nil)
	ccEndpoint, err = computeChaincodeEndpoint(peerAddress)
	assert.NoError(t, err)
	assert.Equal(t, settingChaincodeListenAddress, ccEndpoint)
//场景2.3:chaincodeListenAddress无效
//ComputechaincodeEndpoint将返回错误
	settingChaincodeListenAddressInvalid := "abc"
	viper.Set(chaincodeListenAddrKey, settingChaincodeListenAddressInvalid)
	viper.Set(chaincodeAddrKey, nil)
	ccEndpoint, err = computeChaincodeEndpoint(peerAddress)
	assert.Error(t, err)
	assert.Equal(t, "", ccEndpoint)

 /**场景3：仅设置链码地址**/
//场景3.1：链码地址为0.0.0.0
//ComputechaincodeEndpoint将返回错误
	chaincodeAddressPort := "9052"
	settingChaincodeAddress0 := "0.0.0.0:" + chaincodeAddressPort
	viper.Set(chaincodeListenAddrKey, nil)
	viper.Set(chaincodeAddrKey, settingChaincodeAddress0)
	ccEndpoint, err = computeChaincodeEndpoint(peerAddress)
	assert.Error(t, err)
	assert.Equal(t, "", ccEndpoint)
//场景3.2:chaincodeaddress不是0.0.0.0
//chaincodeEndpoint将是chaincodeAddress
	settingChaincodeAddress := "127.0.0.2:" + chaincodeAddressPort
	viper.Set(chaincodeListenAddrKey, nil)
	viper.Set(chaincodeAddrKey, settingChaincodeAddress)
	ccEndpoint, err = computeChaincodeEndpoint(peerAddress)
	assert.NoError(t, err)
	assert.Equal(t, settingChaincodeAddress, ccEndpoint)
//场景3.3:chaincodeaddress无效
//ComputechaincodeEndpoint将返回错误
	settingChaincodeAddressInvalid := "bcd"
	viper.Set(chaincodeListenAddrKey, nil)
	viper.Set(chaincodeAddrKey, settingChaincodeAddressInvalid)
	ccEndpoint, err = computeChaincodeEndpoint(peerAddress)
	assert.Error(t, err)
	assert.Equal(t, "", ccEndpoint)

 /**场景4：同时设置chaincodeaddress和chaincodelistenaddress**/
//此方案与方案3相同：仅设置chaincodeaddress。
}
