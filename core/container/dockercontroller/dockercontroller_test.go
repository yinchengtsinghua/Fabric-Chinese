
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


package dockercontroller

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/common/flogging/floggingtest"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/common/metrics/metricsfakes"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/container/ccintf"
	coreutil "github.com/hyperledger/fabric/core/testutil"
	pb "github.com/hyperledger/fabric/protos/peer"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//此测试以前是核心/容器中集成样式测试的一部分，已移至此处
func TestIntegrationPath(t *testing.T) {
	coreutil.SetupTestConfig()
	dc := NewDockerVM("", util.GenerateUUID(), NewBuildMetrics(&disabled.Provider{}))
	ccid := ccintf.CCID{Name: "simple"}

	err := dc.Start(ccid, nil, nil, nil, InMemBuilder{})
	require.NoError(t, err)

//停止、杀死和删除
	err = dc.Stop(ccid, 0, true, true)
	require.NoError(t, err)

	err = dc.Start(ccid, nil, nil, nil, nil)
	require.NoError(t, err)

//停止，杀死，但不删除
	_ = dc.Stop(ccid, 0, false, true)
}

func TestHostConfig(t *testing.T) {
	coreutil.SetupTestConfig()
	var hostConfig = new(docker.HostConfig)
	err := viper.UnmarshalKey("vm.docker.hostConfig", hostConfig)
	if err != nil {
		t.Fatalf("Load docker HostConfig wrong, error: %s", err.Error())
	}
	assert.NotNil(t, hostConfig.LogConfig)
	assert.Equal(t, "json-file", hostConfig.LogConfig.Type)
	assert.Equal(t, "50m", hostConfig.LogConfig.Config["max-size"])
	assert.Equal(t, "5", hostConfig.LogConfig.Config["max-file"])
}

func TestGetDockerHostConfig(t *testing.T) {
	coreutil.SetupTestConfig()
hostConfig = nil //Docker主机配置存在缓存的全局单例，其他测试可能会与
	hostConfig := getDockerHostConfig()
	assert.NotNil(t, hostConfig)
	assert.Equal(t, "host", hostConfig.NetworkMode)
	assert.Equal(t, "json-file", hostConfig.LogConfig.Type)
	assert.Equal(t, "50m", hostConfig.LogConfig.Config["max-size"])
	assert.Equal(t, "5", hostConfig.LogConfig.Config["max-file"])
	assert.Equal(t, int64(1024*1024*1024*2), hostConfig.Memory)
	assert.Equal(t, int64(0), hostConfig.CPUShares)
}

func Test_Start(t *testing.T) {
	gt := NewGomegaWithT(t)
	dvm := DockerVM{
		BuildMetrics: NewBuildMetrics(&disabled.Provider{}),
	}
	ccid := ccintf.CCID{
		Name:    "simple",
		Version: "1.0",
	}
	args := make([]string, 1)
	env := make([]string, 1)
	files := map[string][]byte{
		"hello": []byte("world"),
	}

//失败案例
//案例1:GetMockClient返回错误
	dvm.getClientFnc = getMockClient
	getClientErr = true
	err := dvm.Start(ccid, args, env, files, nil)
	gt.Expect(err).To(HaveOccurred())
	getClientErr = false

//案例2:DockerClient.CreateContainer返回错误
	createErr = true
	err = dvm.Start(ccid, args, env, files, nil)
	gt.Expect(err).To(HaveOccurred())
	createErr = false

//案例3:DockerClient.UploadToContainer返回错误
	uploadErr = true
	err = dvm.Start(ccid, args, env, files, nil)
	gt.Expect(err).To(HaveOccurred())
	uploadErr = false

//案例4:dockerclient.startcontainer返回docker.nosuchimgerr，buildImage失败
	noSuchImgErr = true
	buildErr = true
	err = dvm.Start(ccid, args, env, files, &mockBuilder{buildFunc: func() (io.Reader, error) { return &bytes.Buffer{}, nil }})
	gt.Expect(err).To(HaveOccurred())
	buildErr = false

	chaincodePath := "github.com/hyperledger/fabric/examples/chaincode/go/example01/cmd"
	spec := &pb.ChaincodeSpec{
		Type:        pb.ChaincodeSpec_GOLANG,
		ChaincodeId: &pb.ChaincodeID{Name: "ex01", Path: chaincodePath},
		Input:       &pb.ChaincodeInput{Args: util.ToChaincodeArgs("f")},
	}
	codePackage, err := platforms.NewRegistry(&golang.Platform{}).GetDeploymentPayload(spec.CCType(), spec.Path())
	if err != nil {
		t.Fatal()
	}
	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: spec, CodePackage: codePackage}
	bldr := &mockBuilder{
		buildFunc: func() (io.Reader, error) {
			return platforms.NewRegistry(&golang.Platform{}).GenerateDockerBuild(
				cds.CCType(),
				cds.Path(),
				cds.Name(),
				cds.Version(),
				cds.Bytes(),
			)
		},
	}

//案例5：开始调用和Dokcliclit.CurrActoEnter返回
//docker.nosuchimgerr和dockerclient.start返回错误
	viper.Set("vm.docker.attachStdout", true)
	startErr = true
	err = dvm.Start(ccid, args, env, files, bldr)
	gt.Expect(err).To(HaveOccurred())
	startErr = false

//成功案例
	err = dvm.Start(ccid, args, env, files, bldr)
	gt.Expect(err).NotTo(HaveOccurred())
	noSuchImgErr = false

//返回错误容器
	stopErr = true
	err = dvm.Start(ccid, args, env, files, nil)
	gt.Expect(err).NotTo(HaveOccurred())
	stopErr = false

//dockerclient.killcontainer返回错误
	killErr = true
	err = dvm.Start(ccid, args, env, files, nil)
	gt.Expect(err).NotTo(HaveOccurred())
	killErr = false

//dockerClient.RemoveContainer returns error
	removeErr = true
	err = dvm.Start(ccid, args, env, files, nil)
	gt.Expect(err).NotTo(HaveOccurred())
	removeErr = false

	err = dvm.Start(ccid, args, env, files, nil)
	gt.Expect(err).NotTo(HaveOccurred())
}

func Test_streamOutput(t *testing.T) {
	gt := NewGomegaWithT(t)

	logger, recorder := floggingtest.NewTestLogger(t)
	containerLogger, containerRecorder := floggingtest.NewTestLogger(t)

	client := &mockClient{}
	errCh := make(chan error, 1)
	optsCh := make(chan docker.AttachToContainerOptions, 1)
	client.attachToContainerStub = func(opts docker.AttachToContainerOptions) error {
		optsCh <- opts
		return <-errCh
	}

	streamOutput(logger, client, "container-name", containerLogger)

	var opts docker.AttachToContainerOptions
	gt.Eventually(optsCh).Should(Receive(&opts))
	gt.Eventually(opts.Success).Should(BeSent(struct{}{}))
	gt.Eventually(opts.Success).Should(BeClosed())

	fmt.Fprintf(opts.OutputStream, "message-one\n")
fmt.Fprintf(opts.OutputStream, "message-two") //不会被写下来
	gt.Eventually(containerRecorder).Should(gbytes.Say("message-one"))
	gt.Consistently(containerRecorder.Entries).Should(HaveLen(1))

	close(errCh)
	gt.Eventually(recorder).Should(gbytes.Say("Container container-name has closed its IO channel"))
	gt.Consistently(recorder.Entries).Should(HaveLen(1))
	gt.Consistently(containerRecorder.Entries).Should(HaveLen(1))
}

func Test_BuildMetric(t *testing.T) {
	ccid := ccintf.CCID{Name: "simple", Version: "1.0"}
	client := &mockClient{}

	tests := []struct {
		desc           string
		buildErr       bool
		expectedLabels []string
	}{
		{desc: "success", buildErr: false, expectedLabels: []string{"chaincode", "simple:1.0", "success", "true"}},
		{desc: "failure", buildErr: true, expectedLabels: []string{"chaincode", "simple:1.0", "success", "false"}},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			gt := NewGomegaWithT(t)
			fakeChaincodeImageBuildDuration := &metricsfakes.Histogram{}
			fakeChaincodeImageBuildDuration.WithReturns(fakeChaincodeImageBuildDuration)
			dvm := DockerVM{
				BuildMetrics: &BuildMetrics{
					ChaincodeImageBuildDuration: fakeChaincodeImageBuildDuration,
				},
			}

			buildErr = tt.buildErr
			dvm.deployImage(client, ccid, &bytes.Buffer{})

			gt.Expect(fakeChaincodeImageBuildDuration.WithCallCount()).To(Equal(1))
			gt.Expect(fakeChaincodeImageBuildDuration.WithArgsForCall(0)).To(Equal(tt.expectedLabels))
			gt.Expect(fakeChaincodeImageBuildDuration.ObserveArgsForCall(0)).NotTo(BeZero())
			gt.Expect(fakeChaincodeImageBuildDuration.ObserveArgsForCall(0)).To(BeNumerically("<", 1.0))
		})
	}

	buildErr = false
}

func Test_Stop(t *testing.T) {
	dvm := DockerVM{}
	ccid := ccintf.CCID{Name: "simple"}

//失败案例：GetMockClient返回错误
	getClientErr = true
	dvm.getClientFnc = getMockClient
	err := dvm.Stop(ccid, 10, true, true)
	assert.Error(t, err)
	getClientErr = false

//成功案例
	err = dvm.Stop(ccid, 10, true, true)
	assert.NoError(t, err)
}

func Test_HealthCheck(t *testing.T) {
	dvm := DockerVM{}

	dvm.getClientFnc = func() (dockerClient, error) {
		client := &mockClient{
			pingErr: false,
		}
		return client, nil
	}
	err := dvm.HealthCheck(context.Background())
	assert.NoError(t, err)

	dvm.getClientFnc = func() (dockerClient, error) {
		client := &mockClient{
			pingErr: true,
		}
		return client, nil
	}
	err = dvm.HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Error pinging daemon")
}

type testCase struct {
	name           string
	vm             *DockerVM
	ccid           ccintf.CCID
	expectedOutput string
}

func TestGetVMNameForDocker(t *testing.T) {
	tc := []testCase{
		{
			name:           "mycc",
			vm:             &DockerVM{NetworkID: "dev", PeerID: "peer0"},
			ccid:           ccintf.CCID{Name: "mycc", Version: "1.0"},
			expectedOutput: fmt.Sprintf("%s-%s", "dev-peer0-mycc-1.0", hex.EncodeToString(util.ComputeSHA256([]byte("dev-peer0-mycc-1.0")))),
		},
		{
			name:           "mycc-nonetworkid",
			vm:             &DockerVM{PeerID: "peer1"},
			ccid:           ccintf.CCID{Name: "mycc", Version: "1.0"},
			expectedOutput: fmt.Sprintf("%s-%s", "peer1-mycc-1.0", hex.EncodeToString(util.ComputeSHA256([]byte("peer1-mycc-1.0")))),
		},
		{
			name:           "myCC-UCids",
			vm:             &DockerVM{NetworkID: "Dev", PeerID: "Peer0"},
			ccid:           ccintf.CCID{Name: "myCC", Version: "1.0"},
			expectedOutput: fmt.Sprintf("%s-%s", "dev-peer0-mycc-1.0", hex.EncodeToString(util.ComputeSHA256([]byte("Dev-Peer0-myCC-1.0")))),
		},
		{
			name:           "myCC-idsWithSpecialChars",
			vm:             &DockerVM{NetworkID: "Dev$dev", PeerID: "Peer*0"},
			ccid:           ccintf.CCID{Name: "myCC", Version: "1.0"},
			expectedOutput: fmt.Sprintf("%s-%s", "dev-dev-peer-0-mycc-1.0", hex.EncodeToString(util.ComputeSHA256([]byte("Dev$dev-Peer*0-myCC-1.0")))),
		},
		{
			name:           "mycc-nopeerid",
			vm:             &DockerVM{NetworkID: "dev"},
			ccid:           ccintf.CCID{Name: "mycc", Version: "1.0"},
			expectedOutput: fmt.Sprintf("%s-%s", "dev-mycc-1.0", hex.EncodeToString(util.ComputeSHA256([]byte("dev-mycc-1.0")))),
		},
		{
			name:           "myCC-LCids",
			vm:             &DockerVM{NetworkID: "dev", PeerID: "peer0"},
			ccid:           ccintf.CCID{Name: "myCC", Version: "1.0"},
			expectedOutput: fmt.Sprintf("%s-%s", "dev-peer0-mycc-1.0", hex.EncodeToString(util.ComputeSHA256([]byte("dev-peer0-myCC-1.0")))),
		},
	}

	for _, test := range tc {
		name, err := test.vm.GetVMNameForDocker(test.ccid)
		assert.Nil(t, err, "Expected nil error")
		assert.Equal(t, test.expectedOutput, name, "Unexpected output for test case name: %s", test.name)
	}

}

func TestGetVMName(t *testing.T) {
	tc := []testCase{
		{
			name:           "myCC-preserveCase",
			vm:             &DockerVM{NetworkID: "Dev", PeerID: "Peer0"},
			ccid:           ccintf.CCID{Name: "myCC", Version: "1.0"},
			expectedOutput: fmt.Sprintf("%s", "Dev-Peer0-myCC-1.0"),
		},
	}

	for _, test := range tc {
		name := test.vm.GetVMName(test.ccid)
		assert.Equal(t, test.expectedOutput, name, "Unexpected output for test case name: %s", test.name)
	}

}

/*unc testformimagename_invalidchars（t*testing.t）
 _uuErr：=formatImageName（“无效*字符”）
 assert.notnil（t，err，“预期错误”）。
*/


type InMemBuilder struct{}

func (imb InMemBuilder) Build() (io.Reader, error) {
	buf := &bytes.Buffer{}
	fmt.Fprintln(buf, "FROM busybox:latest")
	fmt.Fprintln(buf, `CMD ["tail", "-f", "/dev/null"]`)

	startTime := time.Now()
	inputbuf := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(inputbuf)
	tr := tar.NewWriter(gw)
	tr.WriteHeader(&tar.Header{
		Name:       "Dockerfile",
		Size:       int64(buf.Len()),
		ModTime:    startTime,
		AccessTime: startTime,
		ChangeTime: startTime,
	})
	tr.Write(buf.Bytes())
	tr.Close()
	gw.Close()
	return inputbuf, nil
}

func getMockClient() (dockerClient, error) {
	if getClientErr {
		return nil, errors.New("Failed to get client")
	}
	return &mockClient{noSuchImgErrReturned: false}, nil
}

type mockBuilder struct {
	buildFunc func() (io.Reader, error)
}

func (m *mockBuilder) Build() (io.Reader, error) {
	return m.buildFunc()
}

type mockClient struct {
	noSuchImgErrReturned bool
	pingErr              bool

	attachToContainerStub func(docker.AttachToContainerOptions) error
}

var getClientErr, createErr, uploadErr, noSuchImgErr, buildErr, removeImgErr,
	startErr, stopErr, killErr, removeErr bool

func (c *mockClient) CreateContainer(options docker.CreateContainerOptions) (*docker.Container, error) {
	if createErr {
		return nil, errors.New("Error creating the container")
	}
	if noSuchImgErr && !c.noSuchImgErrReturned {
		c.noSuchImgErrReturned = true
		return nil, docker.ErrNoSuchImage
	}
	return &docker.Container{}, nil
}

func (c *mockClient) StartContainer(id string, cfg *docker.HostConfig) error {
	if startErr {
		return errors.New("Error starting the container")
	}
	return nil
}

func (c *mockClient) UploadToContainer(id string, opts docker.UploadToContainerOptions) error {
	if uploadErr {
		return errors.New("Error uploading archive to the container")
	}
	return nil
}

func (c *mockClient) AttachToContainer(opts docker.AttachToContainerOptions) error {
	if c.attachToContainerStub != nil {
		return c.attachToContainerStub(opts)
	}
	if opts.Success != nil {
		opts.Success <- struct{}{}
	}
	return nil
}

func (c *mockClient) BuildImage(opts docker.BuildImageOptions) error {
	if buildErr {
		return errors.New("Error building image")
	}
	return nil
}

func (c *mockClient) RemoveImageExtended(id string, opts docker.RemoveImageOptions) error {
	if removeImgErr {
		return errors.New("Error removing extended image")
	}
	return nil
}

func (c *mockClient) StopContainer(id string, timeout uint) error {
	if stopErr {
		return errors.New("Error stopping container")
	}
	return nil
}

func (c *mockClient) KillContainer(opts docker.KillContainerOptions) error {
	if killErr {
		return errors.New("Error killing container")
	}
	return nil
}

func (c *mockClient) RemoveContainer(opts docker.RemoveContainerOptions) error {
	if removeErr {
		return errors.New("Error removing container")
	}
	return nil
}

func (c *mockClient) PingWithContext(context.Context) error {
	if c.pingErr {
		return errors.New("Error pinging daemon")
	}
	return nil
}
