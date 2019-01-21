
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


package peer_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/core/comm"
	testpb "github.com/hyperledger/fabric/core/comm/testdata/grpc"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/msp"
	cb "github.com/hyperledger/fabric/protos/common"
	mspproto "github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

//GRPC连接的默认超时
var timeout = time.Second * 1

//要在GRPCServer中注册的测试服务器
type testServiceServer struct{}

func (tss *testServiceServer) EmptyCall(context.Context, *testpb.Empty) (*testpb.Empty, error) {
	return new(testpb.Empty), nil
}

//CreateCertPool从PEM编码的证书数组创建一个x509.certpool
func createCertPool(rootCAs [][]byte) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	for _, rootCA := range rootCAs {
		if !certPool.AppendCertsFromPEM(rootCA) {
			return nil, errors.New("Failed to load root certificates")
		}
	}
	return certPool, nil
}

//用于重新调用测试服务的EmptyCall的Helper函数
func invokeEmptyCall(address string, dialOptions []grpc.DialOption) (*testpb.Empty, error) {
//添加拨号选项
	dialOptions = append(dialOptions, grpc.WithBlock())
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
//创建GRPC客户端连接
	clientConn, err := grpc.DialContext(ctx, address, dialOptions...)
	if err != nil {
		return nil, err
	}
	defer clientConn.Close()

//创建GRPC客户端
	client := testpb.NewTestServiceClient(clientConn)

//调用服务
	empty, err := client.EmptyCall(context.Background(), new(testpb.Empty))
	if err != nil {
		return nil, err
	}

	return empty, nil
}

//用于生成给定根证书的mspconfig的helper函数
func createMSPConfig(rootCerts, tlsRootCerts, tlsIntermediateCerts [][]byte,
	mspID string) (*mspproto.MSPConfig, error) {

	fmspconf := &mspproto.FabricMSPConfig{
		RootCerts:            rootCerts,
		TlsRootCerts:         tlsRootCerts,
		TlsIntermediateCerts: tlsIntermediateCerts,
		Name:                 mspID,
	}

	fmpsjs, err := proto.Marshal(fmspconf)
	if err != nil {
		return nil, err
	}
	mspconf := &mspproto.MSPConfig{Config: fmpsjs, Type: int32(msp.FABRIC)}
	return mspconf, nil
}

func createConfigBlock(chainID string, appMSPConf, ordererMSPConf *mspproto.MSPConfig,
	appOrgID, ordererOrgID string) (*cb.Block, error) {
	block, err := configtxtest.MakeGenesisBlockFromMSPs(chainID, appMSPConf, ordererMSPConf, appOrgID, ordererOrgID)
	if block == nil || err != nil {
		return block, err
	}

	txsFilter := util.NewTxValidationFlagsSetValue(len(block.Data.Data), pb.TxValidationCode_VALID)
	block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter

	return block, nil
}

func TestUpdateRootsFromConfigBlock(t *testing.T) {
//从测试数据加载测试证书
	org1CA, err := ioutil.ReadFile(filepath.Join("testdata", "Org1-cert.pem"))
	require.NoError(t, err)
	org1Server1Key, err := ioutil.ReadFile(filepath.Join("testdata", "Org1-server1-key.pem"))
	require.NoError(t, err)
	org1Server1Cert, err := ioutil.ReadFile(filepath.Join("testdata", "Org1-server1-cert.pem"))
	require.NoError(t, err)
	org2CA, err := ioutil.ReadFile(filepath.Join("testdata", "Org2-cert.pem"))
	require.NoError(t, err)
	org2Server1Key, err := ioutil.ReadFile(filepath.Join("testdata", "Org2-server1-key.pem"))
	require.NoError(t, err)
	org2Server1Cert, err := ioutil.ReadFile(filepath.Join("testdata", "Org2-server1-cert.pem"))
	require.NoError(t, err)
	org2IntermediateCA, err := ioutil.ReadFile(filepath.Join("testdata", "Org2-child1-cert.pem"))
	require.NoError(t, err)
	org2IntermediateServer1Key, err := ioutil.ReadFile(filepath.Join("testdata", "Org2-child1-server1-key.pem"))
	require.NoError(t, err)
	org2IntermediateServer1Cert, err := ioutil.ReadFile(filepath.Join("testdata", "Org2-child1-server1-cert.pem"))
	require.NoError(t, err)
	ordererOrgCA, err := ioutil.ReadFile(filepath.Join("testdata", "Org3-cert.pem"))
	require.NoError(t, err)
	ordererOrgServer1Key, err := ioutil.ReadFile(filepath.Join("testdata", "Org3-server1-key.pem"))
	require.NoError(t, err)
	ordererOrgServer1Cert, err := ioutil.ReadFile(filepath.Join("testdata", "Org3-server1-cert.pem"))
	require.NoError(t, err)

//创建测试mspconfigs
	org1MSPConf, err := createMSPConfig([][]byte{org2CA}, [][]byte{org1CA}, [][]byte{}, "Org1MSP")
	require.NoError(t, err)
	org2MSPConf, err := createMSPConfig([][]byte{org1CA}, [][]byte{org2CA}, [][]byte{}, "Org2MSP")
	require.NoError(t, err)
	org2IntermediateMSPConf, err := createMSPConfig([][]byte{org1CA}, [][]byte{org2CA}, [][]byte{org2IntermediateCA}, "Org2IntermediateMSP")
	require.NoError(t, err)
	ordererOrgMSPConf, err := createMSPConfig([][]byte{org1CA}, [][]byte{ordererOrgCA}, [][]byte{}, "OrdererOrgMSP")
	require.NoError(t, err)

//创建测试通道创建块
	channel1Block, err := createConfigBlock("channel1", org1MSPConf, ordererOrgMSPConf, "Org1MSP", "OrdererOrgMSP")
	require.NoError(t, err)
	channel2Block, err := createConfigBlock("channel2", org2MSPConf, ordererOrgMSPConf, "Org2MSP", "OrdererOrgMSP")
	require.NoError(t, err)
	channel3Block, err := createConfigBlock("channel3", org2IntermediateMSPConf, ordererOrgMSPConf, "Org2IntermediateMSP", "OrdererOrgMSP")
	require.NoError(t, err)

	createChannel := func(cid string, block *cb.Block) {
		testDir, err := ioutil.TempDir("", "peer-pkg")
		require.NoError(t, err)
		defer os.RemoveAll(testDir)

		viper.Set("peer.tls.enabled", true)
		viper.Set("peer.tls.cert.file", filepath.Join("testdata", "Org1-server1-cert.pem"))
		viper.Set("peer.tls.key.file", filepath.Join("testdata", "Org1-server1-key.pem"))
		viper.Set("peer.tls.rootcert.file", filepath.Join("testdata", "Org1-cert.pem"))
		viper.Set("peer.fileSystemPath", testDir)
		err = peer.Default.CreateChainFromBlock(block, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create config block (%s)", err)
		}
		t.Logf("Channel %s MSPIDs: (%s)", cid, peer.Default.GetMSPIDs(cid))
	}

	org1CertPool, err := createCertPool([][]byte{org1CA})
	require.NoError(t, err)

//使用服务器证书作为客户端证书
	org1ClientCert, err := tls.X509KeyPair(org1Server1Cert, org1Server1Key)
	require.NoError(t, err)

	org1Creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{org1ClientCert},
		RootCAs:      org1CertPool,
	})

	org2ClientCert, err := tls.X509KeyPair(org2Server1Cert, org2Server1Key)
	require.NoError(t, err)
	org2Creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{org2ClientCert},
		RootCAs:      org1CertPool,
	})

	org2IntermediateClientCert, err := tls.X509KeyPair(org2IntermediateServer1Cert, org2IntermediateServer1Key)
	require.NoError(t, err)
	org2IntermediateCreds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{org2IntermediateClientCert},
		RootCAs:      org1CertPool,
	})

	ordererOrgClientCert, err := tls.X509KeyPair(ordererOrgServer1Cert, ordererOrgServer1Key)
	require.NoError(t, err)

	ordererOrgCreds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{ordererOrgClientCert},
		RootCAs:      org1CertPool,
	})

//基本功能测试
	var tests = []struct {
		name          string
		serverConfig  comm.ServerConfig
		createChannel func()
		goodOptions   []grpc.DialOption
		badOptions    []grpc.DialOption
		numAppCAs     int
		numOrdererCAs int
	}{
		{
			name: "MutualTLSOrg1Org1",
			serverConfig: comm.ServerConfig{
				SecOpts: &comm.SecureOptions{
					UseTLS:            true,
					Certificate:       org1Server1Cert,
					Key:               org1Server1Key,
					ServerRootCAs:     [][]byte{org1CA},
					RequireClientCert: true,
				},
			},
			createChannel: func() { createChannel("channel1", channel1Block) },
			goodOptions:   []grpc.DialOption{grpc.WithTransportCredentials(org1Creds)},
			badOptions:    []grpc.DialOption{grpc.WithTransportCredentials(ordererOrgCreds)},
numAppCAs:     3, //每个通道也有一个默认的MSP
			numOrdererCAs: 1,
		},
		{
			name: "MutualTLSOrg1Org2",
			serverConfig: comm.ServerConfig{
				SecOpts: &comm.SecureOptions{
					UseTLS:            true,
					Certificate:       org1Server1Cert,
					Key:               org1Server1Key,
					ServerRootCAs:     [][]byte{org1CA},
					RequireClientCert: true,
				},
			},
			createChannel: func() { createChannel("channel2", channel2Block) },
			goodOptions: []grpc.DialOption{
				grpc.WithTransportCredentials(org2Creds),
			},
			badOptions: []grpc.DialOption{
				grpc.WithTransportCredentials(ordererOrgCreds),
			},
			numAppCAs:     6,
			numOrdererCAs: 2,
		},
		{
			name: "MutualTLSOrg1Org2Intermediate",
			serverConfig: comm.ServerConfig{
				SecOpts: &comm.SecureOptions{
					UseTLS:            true,
					Certificate:       org1Server1Cert,
					Key:               org1Server1Key,
					ServerRootCAs:     [][]byte{org1CA},
					RequireClientCert: true,
				},
			},
			createChannel: func() { createChannel("channel3", channel3Block) },
			goodOptions: []grpc.DialOption{
				grpc.WithTransportCredentials(org2IntermediateCreds),
			},
			badOptions: []grpc.DialOption{
				grpc.WithTransportCredentials(ordererOrgCreds),
			},
			numAppCAs:     10,
			numOrdererCAs: 3,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Running test %s ...", test.name)
			server, err := peer.NewPeerServer("localhost:0", test.serverConfig)
			if err != nil {
				t.Fatalf("NewPeerServer failed with error [%s]", err)
			} else {
				assert.NoError(t, err, "NewPeerServer should not have returned an error")
				assert.NotNil(t, server, "NewPeerServer should have created a server")
//注册GRPC测试服务
				testpb.RegisterTestServiceServer(server.Server(), &testServiceServer{})
				go server.Start()
				defer server.Stop()

//提取动态侦听端口
				_, port, err := net.SplitHostPort(server.Listener().Addr().String())
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("listenAddress: %s", server.Listener().Addr())
				testAddress := "localhost:" + port
				t.Logf("testAddress: %s", testAddress)

//使用良好的选项调用EmptyCall服务，但应失败
//直到创建通道并更新根CA
				_, err = invokeEmptyCall(testAddress, test.goodOptions)
				assert.Error(t, err, "Expected error invoking the EmptyCall service ")

//创建通道应更新受信任的客户端根目录
				test.createChannel()

//确保我们有预期数量的CA
				appCAs, ordererCAs := comm.GetCredentialSupport().GetClientRootCAs()
				assert.Equal(t, test.numAppCAs, len(appCAs), "Did not find expected number of app CAs for channel")
				assert.Equal(t, test.numOrdererCAs, len(ordererCAs), "Did not find expected number of orderer CAs for channel")

//使用良好的选项调用EmptyCall服务
				_, err = invokeEmptyCall(testAddress, test.goodOptions)
				assert.NoError(t, err, "Failed to invoke the EmptyCall service")

//使用错误的选项调用EmptyCall服务
				_, err = invokeEmptyCall(testAddress, test.badOptions)
				assert.Error(t, err, "Expected error using bad dial options")
			}
		})
	}
}
