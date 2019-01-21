
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有Digital Asset Holdings，LLC保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package channel

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/peer/common"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type timeoutOrderer struct {
	counter int
	net.Listener
	*grpc.Server
	nextExpectedSeek uint64
	t                *testing.T
	blockChannel     chan uint64
}

func newOrderer(port int, t *testing.T) *timeoutOrderer {
	srv := grpc.NewServer()
	lsnr, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		panic(err)
	}
	o := &timeoutOrderer{Server: srv,
		Listener:         lsnr,
		t:                t,
		nextExpectedSeek: uint64(1),
		blockChannel:     make(chan uint64, 1),
		counter:          int(1),
	}
	orderer.RegisterAtomicBroadcastServer(srv, o)
	go srv.Serve(lsnr)
	return o
}

func (o *timeoutOrderer) Shutdown() {
	o.Server.Stop()
	o.Listener.Close()
}

func (*timeoutOrderer) Broadcast(orderer.AtomicBroadcast_BroadcastServer) error {
	panic("Should not have been called")
}

func (o *timeoutOrderer) SendBlock(seq uint64) {
	o.blockChannel <- seq
}

func (o *timeoutOrderer) Deliver(stream orderer.AtomicBroadcast_DeliverServer) error {
	o.timeoutIncrement()
	if o.counter > 5 {
		o.sendBlock(stream, 0)
	}
	return nil
}

func (o *timeoutOrderer) sendBlock(stream orderer.AtomicBroadcast_DeliverServer, seq uint64) {
	block := &cb.Block{
		Header: &cb.BlockHeader{
			Number: seq,
		},
	}
	stream.Send(&orderer.DeliverResponse{
		Type: &orderer.DeliverResponse_Block{Block: block},
	})
}

func (o *timeoutOrderer) timeoutIncrement() {
	o.counter++
}

var once sync.Once

///mock-deliver客户端
type mockDeliverClient struct {
	err error
}

func (m *mockDeliverClient) readBlock() (*cb.Block, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &cb.Block{}, nil
}

func (m *mockDeliverClient) GetSpecifiedBlock(num uint64) (*cb.Block, error) {
	return m.readBlock()
}

func (m *mockDeliverClient) GetOldestBlock() (*cb.Block, error) {
	return m.readBlock()
}

func (m *mockDeliverClient) GetNewestBlock() (*cb.Block, error) {
	return m.readBlock()
}

func (m *mockDeliverClient) Close() error {
	return nil
}

//init msp初始化msp
func InitMSP() {
	once.Do(initMSP)
}

func initMSP() {
	err := msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		panic(fmt.Errorf("Fatal error when reading MSP config: err %s", err))
	}
}

func mockBroadcastClientFactory() (common.BroadcastClient, error) {
	return common.GetMockBroadcastClient(nil), nil
}

func TestCreateChain(t *testing.T) {
	defer resetFlags()

	InitMSP()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchain := "mockchain"

	defer os.Remove(mockchain + ".block")

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	mockCF := &ChannelCmdFactory{
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
		DeliverClient:    &mockDeliverClient{},
	}

	cmd := createCmd(mockCF)

	AddFlags(cmd)

	args := []string{"-c", mockchain, "-o", "localhost:7050"}
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fail()
		t.Errorf("expected create command to succeed")
	}

	filename := mockchain + ".block"
	if _, err := os.Stat(filename); err != nil {
		t.Fail()
		t.Errorf("expected %s to exist", filename)
	}
}

func TestCreateChainWithOutputBlock(t *testing.T) {
	defer resetFlags()

	InitMSP()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchain := "mockchain"

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	mockCF := &ChannelCmdFactory{
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
		DeliverClient:    &mockDeliverClient{},
	}

	cmd := createCmd(mockCF)
	AddFlags(cmd)

	tempDir, err := ioutil.TempDir("", "create-output")
	if err != nil {
		t.Fatalf("failed to create temporary directory")
	}
	defer os.RemoveAll(tempDir)

	outputBlockPath := filepath.Join(tempDir, "output.block")
	args := []string{"-c", mockchain, "-o", "localhost:7050", "--outputBlock", outputBlockPath}
	cmd.SetArgs(args)
	defer func() { outputBlock = "" }()

	err = cmd.Execute()
	assert.NoError(t, err, "execute should succeed")

	_, err = os.Stat(outputBlockPath)
	assert.NoErrorf(t, err, "expected %s to exist", outputBlockPath)
}

func TestCreateChainWithDefaultAnchorPeers(t *testing.T) {
	defer resetFlags()

	InitMSP()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchain := "mockchain"

	defer os.Remove(mockchain + ".block")

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	mockCF := &ChannelCmdFactory{
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
		DeliverClient:    &mockDeliverClient{},
	}

	cmd := createCmd(mockCF)
	AddFlags(cmd)
	args := []string{"-c", mockchain, "-o", "localhost:7050"}
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fail()
		t.Errorf("expected create command to succeed")
	}
}

func TestCreateChainWithWaitSuccess(t *testing.T) {
	defer resetFlags()

	InitMSP()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchain := "mockchain"

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	mockCF := &ChannelCmdFactory{
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
		DeliverClient:    &mockDeliverClient{err: nil},
	}
	fakeOrderer := newOrderer(8101, t)
	defer fakeOrderer.Shutdown()

	cmd := createCmd(mockCF)
	AddFlags(cmd)
	args := []string{"-c", mockchain, "-o", "localhost:8101", "-t", "10s"}
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fail()
		t.Errorf("expected create command to succeed")
	}
}

func TestCreateChainWithTimeoutErr(t *testing.T) {
	defer viper.Reset()
	defer resetFlags()

	InitMSP()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchain := "mockchain"

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	mockCF := &ChannelCmdFactory{
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
		DeliverClient:    &mockDeliverClient{err: errors.New("bobsled")},
	}
	fakeOrderer := newOrderer(8102, t)
	defer fakeOrderer.Shutdown()

//失败-连接到订购方，但等待通道超时
//被创造
	cmd := createCmd(mockCF)
	AddFlags(cmd)
	channelCmd.AddCommand(cmd)
	args := []string{"create", "-c", mockchain, "-o", "localhost:8102", "-t", "10ms"}
	channelCmd.SetArgs(args)

	if err := channelCmd.Execute(); err == nil {
		t.Error("expected create chain to fail with deliver error")
	} else {
		assert.Contains(t, err.Error(), "timeout waiting for channel creation")
	}

//故障-指向错误的端口并超时连接到订购方
	args = []string{"create", "-c", mockchain, "-o", "localhost:0", "--connTimeout", "10ms"}
	channelCmd.SetArgs(args)

	if err := channelCmd.Execute(); err == nil {
		t.Error("expected create chain to fail with deliver error")
	} else {
		assert.Contains(t, err.Error(), "failed connecting")
	}
}

func TestCreateChainBCFail(t *testing.T) {
	defer resetFlags()

	InitMSP()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchain := "mockchain"

	defer os.Remove(mockchain + ".block")

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	sendErr := errors.New("luge")
	mockCF := &ChannelCmdFactory{
		BroadcastFactory: func() (common.BroadcastClient, error) {
			return common.GetMockBroadcastClient(sendErr), nil
		},
		Signer:        signer,
		DeliverClient: &mockDeliverClient{},
	}

	cmd := createCmd(mockCF)
	AddFlags(cmd)
	args := []string{"-c", mockchain, "-o", "localhost:7050"}
	cmd.SetArgs(args)

	expectedErrMsg := sendErr.Error()
	if err := cmd.Execute(); err == nil {
		t.Error("expected create chain to fail with broadcast error")
	} else {
		if err.Error() != expectedErrMsg {
			t.Errorf("Run create chain get unexpected error: %s(expected %s)", err.Error(), expectedErrMsg)
		}
	}
}

func TestCreateChainDeliverFail(t *testing.T) {
	defer resetFlags()

	InitMSP()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchain := "mockchain"

	defer os.Remove(mockchain + ".block")

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	sendErr := fmt.Errorf("skeleton")
	mockCF := &ChannelCmdFactory{
		BroadcastFactory: func() (common.BroadcastClient, error) {
			return common.GetMockBroadcastClient(sendErr), nil
		},
		Signer:        signer,
		DeliverClient: &mockDeliverClient{err: sendErr},
	}
	cmd := createCmd(mockCF)
	AddFlags(cmd)
	args := []string{"-c", mockchain, "-o", "localhost:7050"}
	cmd.SetArgs(args)

	expectedErrMsg := sendErr.Error()
	if err := cmd.Execute(); err == nil {
		t.Errorf("expected create chain to fail with deliver error")
	} else {
		if err.Error() != expectedErrMsg {
			t.Errorf("Run create chain get unexpected error: %s(expected %s)", err.Error(), expectedErrMsg)
		}
	}
}

func createTxFile(filename string, typ cb.HeaderType, channelID string) (*cb.Envelope, error) {
	ch := &cb.ChannelHeader{Type: int32(typ), ChannelId: channelID}
	data, err := proto.Marshal(ch)
	if err != nil {
		return nil, err
	}

	p := &cb.Payload{Header: &cb.Header{ChannelHeader: data}}
	data, err = proto.Marshal(p)
	if err != nil {
		return nil, err
	}

	env := &cb.Envelope{Payload: data}
	data, err = proto.Marshal(env)
	if err != nil {
		return nil, err
	}

	if err = ioutil.WriteFile(filename, data, 0644); err != nil {
		return nil, err
	}

	return env, nil
}

func TestCreateChainFromTx(t *testing.T) {
	defer resetFlags()
	InitMSP()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchannel := "mockchannel"
	dir, err := ioutil.TempDir("", "createtestfromtx-")
	if err != nil {
		t.Fatalf("couldn't create temp dir")
	}
defer os.RemoveAll(dir) //清理

//这可以通过create命令创建
	defer os.Remove(mockchannel + ".block")

	file := filepath.Join(dir, mockchannel)

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}
	mockCF := &ChannelCmdFactory{
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
		DeliverClient:    &mockDeliverClient{},
	}

	cmd := createCmd(mockCF)
	AddFlags(cmd)

//错误案例0
	args := []string{"-c", "", "-f", file, "-o", "localhost:7050"}
	cmd.SetArgs(args)
	err = cmd.Execute()
	assert.Error(t, err, "Create command should have failed because channel ID is not specified")
	assert.Contains(t, err.Error(), "must supply channel ID")

//错误案例1
	args = []string{"-c", mockchannel, "-f", file, "-o", "localhost:7050"}
	cmd.SetArgs(args)
	err = cmd.Execute()
	assert.Error(t, err, "Create command should have failed because tx file does not exist")
	var msgExpr = regexp.MustCompile(`channel create configuration tx file not found.*no such file or directory`)
	assert.True(t, msgExpr.MatchString(err.Error()))

//成功案例：-f选项为空
	args = []string{"-c", mockchannel, "-f", "", "-o", "localhost:7050"}
	cmd.SetArgs(args)
	err = cmd.Execute()
	assert.NoError(t, err)

//成功案例
	args = []string{"-c", mockchannel, "-f", file, "-o", "localhost:7050"}
	cmd.SetArgs(args)
	_, err = createTxFile(file, cb.HeaderType_CONFIG_UPDATE, mockchannel)
	assert.NoError(t, err, "Couldn't create tx file")
	err = cmd.Execute()
	assert.NoError(t, err)
}

func TestCreateChainInvalidTx(t *testing.T) {
	defer resetFlags()

	InitMSP()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchannel := "mockchannel"

	dir, err := ioutil.TempDir("", "createinvaltest-")
	if err != nil {
		t.Fatalf("couldn't create temp dir")
	}

defer os.RemoveAll(dir) //清理

//这是通过创建命令创建的
	defer os.Remove(mockchannel + ".block")

	file := filepath.Join(dir, mockchannel)

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	mockCF := &ChannelCmdFactory{
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
		DeliverClient:    &mockDeliverClient{},
	}

	cmd := createCmd(mockCF)

	AddFlags(cmd)

	args := []string{"-c", mockchannel, "-f", file, "-o", "localhost:7050"}
	cmd.SetArgs(args)

//坏类型配置
	if _, err = createTxFile(file, cb.HeaderType_CONFIG, mockchannel); err != nil {
		t.Fatalf("couldn't create tx file")
	}

	defer os.Remove(file)

	if err = cmd.Execute(); err == nil {
		t.Errorf("expected error")
	} else if _, ok := err.(InvalidCreateTx); !ok {
		t.Errorf("invalid error")
	}

//错误的通道名-与命令中指定的不匹配
	if _, err = createTxFile(file, cb.HeaderType_CONFIG_UPDATE, "different_channel"); err != nil {
		t.Fatalf("couldn't create tx file")
	}

	if err = cmd.Execute(); err == nil {
		t.Errorf("expected error")
	} else if _, ok := err.(InvalidCreateTx); !ok {
		t.Errorf("invalid error")
	}

//空信道
	if _, err = createTxFile(file, cb.HeaderType_CONFIG_UPDATE, ""); err != nil {
		t.Fatalf("couldn't create tx file")
	}

	if err := cmd.Execute(); err == nil {
		t.Errorf("expected error")
	} else if _, ok := err.(InvalidCreateTx); !ok {
		t.Errorf("invalid error")
	}
}

func TestCreateChainNilCF(t *testing.T) {
	defer viper.Reset()
	defer resetFlags()

	InitMSP()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchannel := "mockchannel"
	dir, err := ioutil.TempDir("", "createinvaltest-")
	assert.NoError(t, err, "Couldn't create temp dir")
defer os.RemoveAll(dir) //清理

//这是通过创建命令创建的
	defer os.Remove(mockchannel + ".block")
	file := filepath.Join(dir, mockchannel)

//错误案例：GRPC错误
	viper.Set("orderer.client.connTimeout", 10*time.Millisecond)
	cmd := createCmd(nil)
	AddFlags(cmd)
	args := []string{"-c", mockchannel, "-f", file, "-o", "localhost:7050"}
	cmd.SetArgs(args)
	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create deliver client")

//错误案例：订购服务终结点无效
	args = []string{"-c", mockchannel, "-f", file, "-o", "localhost"}
	cmd.SetArgs(args)
	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ordering service endpoint localhost is not valid or missing")

//错误案例：无效的CA文件
defer os.RemoveAll(dir) //清理
	channelCmd.AddCommand(cmd)
	args = []string{"create", "-c", mockchannel, "-f", file, "-o", "localhost:7050", "--tls", "true", "--cafile", dir + "/ca.pem"}
	channelCmd.SetArgs(args)
	err = channelCmd.Execute()
	assert.Error(t, err)
	t.Log(err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestSanityCheckAndSignChannelCreateTx(t *testing.T) {
	defer resetFlags()

//错误案例1
	env := &cb.Envelope{}
	env.Payload = make([]byte, 10)
	var err error
	env, err = sanityCheckAndSignConfigTx(env)
	assert.Error(t, err, "Error expected for nil payload")
	assert.Contains(t, err.Error(), "bad payload")

//错误案例2
	p := &cb.Payload{Header: nil}
	data, err1 := proto.Marshal(p)
	assert.NoError(t, err1)
	env = &cb.Envelope{Payload: data}
	env, err = sanityCheckAndSignConfigTx(env)
	assert.Error(t, err, "Error expected for bad payload header")
	assert.Contains(t, err.Error(), "bad header")

//错误案例3
	bites := bytes.NewBufferString("foo").Bytes()
	p = &cb.Payload{Header: &cb.Header{ChannelHeader: bites}}
	data, err = proto.Marshal(p)
	assert.NoError(t, err)
	env = &cb.Envelope{Payload: data}
	env, err = sanityCheckAndSignConfigTx(env)
	assert.Error(t, err, "Error expected for bad channel header")
	assert.Contains(t, err.Error(), "could not unmarshall channel header")

//错误案例4
	mockchannel := "mockchannel"
	cid := channelID
	channelID = mockchannel
	defer func() {
		channelID = cid
	}()
	ch := &cb.ChannelHeader{Type: int32(cb.HeaderType_CONFIG_UPDATE), ChannelId: mockchannel}
	data, err = proto.Marshal(ch)
	assert.NoError(t, err)
	p = &cb.Payload{Header: &cb.Header{ChannelHeader: data}, Data: bytes.NewBufferString("foo").Bytes()}
	data, err = proto.Marshal(p)
	assert.NoError(t, err)
	env = &cb.Envelope{Payload: data}
	env, err = sanityCheckAndSignConfigTx(env)
	assert.Error(t, err, "Error expected for bad payload data")
	assert.Contains(t, err.Error(), "Bad config update env")
}
