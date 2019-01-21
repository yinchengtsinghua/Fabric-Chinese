
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
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/container"
	"github.com/hyperledger/fabric/core/container/ccintf"
	cutil "github.com/hyperledger/fabric/core/container/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

//ContainerType是Docker容器类型的字符串
//已在container.vmcontroller中注册
const ContainerType = "DOCKER"

var (
	dockerLogger = flogging.MustGetLogger("dockercontroller")
	hostConfig   *docker.HostConfig
	vmRegExp     = regexp.MustCompile("[^a-zA-Z0-9-_.]")
	imageRegExp  = regexp.MustCompile("^[a-z0-9]+(([._-][a-z0-9]+)+)?$")
)

//getclient返回实现dockerclient接口的实例
type getClient func() (dockerClient, error)

//Dockervm是一个虚拟机。它由图像ID标识
type DockerVM struct {
	getClientFnc getClient
	PeerID       string
	NetworkID    string
	BuildMetrics *BuildMetrics
}

//DockerClient表示Docker客户端
type dockerClient interface {
//CreateContainer创建Docker容器，失败时返回错误
	CreateContainer(opts docker.CreateContainerOptions) (*docker.Container, error)
//uploadToContainer将要提取的tar存档上载到
//容器的文件系统。
	UploadToContainer(id string, opts docker.UploadToContainerOptions) error
//StartContainer启动Docker容器，失败时返回错误
	StartContainer(id string, cfg *docker.HostConfig) error
//AttachToContainer附加到Docker容器，在
//失败
	AttachToContainer(opts docker.AttachToContainerOptions) error
//buildImage从tarball的url或输入中的dockerfile构建图像
//流，失败时返回错误
	BuildImage(opts docker.BuildImageOptions) error
//removeImageExtended按名称或ID删除Docker映像，返回
//故障时出错
	RemoveImageExtended(id string, opts docker.RemoveImageOptions) error
//StopContainer停止Docker容器，在给定的超时后将其杀死
//（以秒为单位）失败时返回错误
	StopContainer(id string, timeout uint) error
//KillContainer向Docker容器发送信号，返回
//失败案例
	KillContainer(opts docker.KillContainerOptions) error
//Rebug维护器移除一个DOCKER容器，在失败的情况下返回一个错误
	RemoveContainer(opts docker.RemoveContainerOptions) error
//使用上下文ping对Docker守护进程执行ping操作。可以使用上下文对象
//to cancel the ping request.
	PingWithContext(context.Context) error
}

//提供程序实现container.vmprovider
type Provider struct {
	PeerID       string
	NetworkID    string
	BuildMetrics *BuildMetrics
}

//NewProvider创建Provider的新实例
func NewProvider(peerID, networkID string, metricsProvider metrics.Provider) *Provider {
	return &Provider{
		PeerID:       peerID,
		NetworkID:    networkID,
		BuildMetrics: NewBuildMetrics(metricsProvider),
	}
}

//newvm创建新的dockervm实例
func (p *Provider) NewVM() container.VM {
	return NewDockerVM(p.PeerID, p.NetworkID, p.BuildMetrics)
}

//new dockervm返回一个新的dockervm实例
func NewDockerVM(peerID, networkID string, buildMetrics *BuildMetrics) *DockerVM {
	return &DockerVM{
		PeerID:       peerID,
		NetworkID:    networkID,
		getClientFnc: getDockerClient,
		BuildMetrics: buildMetrics,
	}
}

func getDockerClient() (dockerClient, error) {
	return cutil.NewDockerClient()
}

func getDockerHostConfig() *docker.HostConfig {
	if hostConfig != nil {
		return hostConfig
	}

	dockerKey := func(key string) string { return "vm.docker.hostConfig." + key }
	getInt64 := func(key string) int64 { return int64(viper.GetInt(dockerKey(key))) }

	var logConfig docker.LogConfig
	err := viper.UnmarshalKey(dockerKey("LogConfig"), &logConfig)
	if err != nil {
		dockerLogger.Warningf("load docker HostConfig.LogConfig failed, error: %s", err.Error())
	}
	networkMode := viper.GetString(dockerKey("NetworkMode"))
	if networkMode == "" {
		networkMode = "host"
	}
	dockerLogger.Debugf("docker container hostconfig NetworkMode: %s", networkMode)

	return &docker.HostConfig{
		CapAdd:  viper.GetStringSlice(dockerKey("CapAdd")),
		CapDrop: viper.GetStringSlice(dockerKey("CapDrop")),

		DNS:         viper.GetStringSlice(dockerKey("Dns")),
		DNSSearch:   viper.GetStringSlice(dockerKey("DnsSearch")),
		ExtraHosts:  viper.GetStringSlice(dockerKey("ExtraHosts")),
		NetworkMode: networkMode,
		IpcMode:     viper.GetString(dockerKey("IpcMode")),
		PidMode:     viper.GetString(dockerKey("PidMode")),
		UTSMode:     viper.GetString(dockerKey("UTSMode")),
		LogConfig:   logConfig,

		ReadonlyRootfs:   viper.GetBool(dockerKey("ReadonlyRootfs")),
		SecurityOpt:      viper.GetStringSlice(dockerKey("SecurityOpt")),
		CgroupParent:     viper.GetString(dockerKey("CgroupParent")),
		Memory:           getInt64("Memory"),
		MemorySwap:       getInt64("MemorySwap"),
		MemorySwappiness: getInt64("MemorySwappiness"),
		OOMKillDisable:   viper.GetBool(dockerKey("OomKillDisable")),
		CPUShares:        getInt64("CpuShares"),
		CPUSet:           viper.GetString(dockerKey("Cpuset")),
		CPUSetCPUs:       viper.GetString(dockerKey("CpusetCPUs")),
		CPUSetMEMs:       viper.GetString(dockerKey("CpusetMEMs")),
		CPUQuota:         getInt64("CpuQuota"),
		CPUPeriod:        getInt64("CpuPeriod"),
		BlkioWeight:      getInt64("BlkioWeight"),
	}
}

func (vm *DockerVM) createContainer(client dockerClient, imageID, containerID string, args, env []string, attachStdout bool) error {
	logger := dockerLogger.With("imageID", imageID, "containerID", containerID)
	logger.Debugw("create container")
	_, err := client.CreateContainer(docker.CreateContainerOptions{
		Name: containerID,
		Config: &docker.Config{
			Cmd:          args,
			Image:        imageID,
			Env:          env,
			AttachStdout: attachStdout,
			AttachStderr: attachStdout,
		},
		HostConfig: getDockerHostConfig(),
	})
	if err != nil {
		return err
	}
	logger.Debugw("created container")
	return nil
}

func (vm *DockerVM) deployImage(client dockerClient, ccid ccintf.CCID, reader io.Reader) error {
	id, err := vm.GetVMNameForDocker(ccid)
	if err != nil {
		return err
	}

	outputbuf := bytes.NewBuffer(nil)
	opts := docker.BuildImageOptions{
		Name:         id,
		Pull:         viper.GetBool("chaincode.pull"),
		InputStream:  reader,
		OutputStream: outputbuf,
	}

	startTime := time.Now()
	err = client.BuildImage(opts)

	vm.BuildMetrics.ChaincodeImageBuildDuration.With(
		"chaincode", ccid.Name+":"+ccid.Version,
		"success", strconv.FormatBool(err == nil),
	).Observe(time.Since(startTime).Seconds())

	if err != nil {
		dockerLogger.Errorf("Error building image: %s", err)
		dockerLogger.Errorf("Build Output:\n********************\n%s\n********************", outputbuf.String())
		return err
	}

	dockerLogger.Debugf("Created image: %s", id)
	return nil
}

//Start使用先前创建的Docker映像启动容器
func (vm *DockerVM) Start(ccid ccintf.CCID, args, env []string, filesToUpload map[string][]byte, builder container.Builder) error {
	imageName, err := vm.GetVMNameForDocker(ccid)
	if err != nil {
		return err
	}

	attachStdout := viper.GetBool("vm.docker.attachStdout")
	containerName := vm.GetVMName(ccid)
	logger := dockerLogger.With("imageName", imageName, "containerName", containerName)

	client, err := vm.getClientFnc()
	if err != nil {
		logger.Debugf("failed to get docker client", "error", err)
		return err
	}

	vm.stopInternal(client, containerName, 0, false, false)

	err = vm.createContainer(client, imageName, containerName, args, env, attachStdout)
	if err == docker.ErrNoSuchImage {
		reader, err := builder.Build()
		if err != nil {
			return errors.Wrapf(err, "failed to generate Dockerfile to build %s", containerName)
		}

		err = vm.deployImage(client, ccid, reader)
		if err != nil {
			return err
		}

		err = vm.createContainer(client, imageName, containerName, args, env, attachStdout)
		if err != nil {
			logger.Errorf("failed to create container: %s", err)
			return err
		}
	} else if err != nil {
		logger.Errorf("create container failed: %s", err)
		return err
	}

//将stdout和stderr流到chaincode记录器
	if attachStdout {
		containerLogger := flogging.MustGetLogger("peer.chaincode." + containerName)
		streamOutput(dockerLogger, client, containerName, containerLogger)
	}

//在启动前将指定的文件上载到容器
//这可用于配置，如TLS密钥和证书
	if len(filesToUpload) != 0 {
//Docker Upload API采用tar文件，因此我们需要首先
//将文件条目合并到tar
		payload := bytes.NewBuffer(nil)
		gw := gzip.NewWriter(payload)
		tw := tar.NewWriter(gw)

		for path, fileToUpload := range filesToUpload {
			cutil.WriteBytesToPackage(path, fileToUpload, tw)
		}

//写出tar文件
		if err := tw.Close(); err != nil {
			return fmt.Errorf("Error writing files to upload to Docker instance into a temporary tar blob: %s", err)
		}

		gw.Close()

		err := client.UploadToContainer(containerName, docker.UploadToContainerOptions{
			InputStream:          bytes.NewReader(payload.Bytes()),
			Path:                 "/",
			NoOverwriteDirNonDir: false,
		})
		if err != nil {
			return fmt.Errorf("Error uploading files to the container instance %s: %s", containerName, err)
		}
	}

//从V1.10中删除带有HostConfig的启动容器，并在V1.2中删除。
	err = client.StartContainer(containerName, nil)
	if err != nil {
		dockerLogger.Errorf("start-could not start container: %s", err)
		return err
	}

	dockerLogger.Debugf("Started container %s", containerName)
	return nil
}

//streamoutput将命名容器的输出镜像到结构记录器。
func streamOutput(logger *flogging.FabricLogger, client dockerClient, containerName string, containerLogger *flogging.FabricLogger) {
//启动一些go例程来管理来自容器的输出流。
//当容器退出时，它们将被自动销毁。
	attached := make(chan struct{})
	r, w := io.Pipe()

	go func() {
//一旦
//附件完成，然后阻塞，直到容器终止。
//返回的错误不在此函数范围之外使用。指派
//局部变量出错，以防止截断函数变量“err”。
		err := client.AttachToContainer(docker.AttachToContainerOptions{
			Container:    containerName,
			OutputStream: w,
			ErrorStream:  w,
			Logs:         true,
			Stdout:       true,
			Stderr:       true,
			Stream:       true,
			Success:      attached,
		})

//如果我们到了这里，容器就终止了。在管道上发送信号
//以便下游可以适当清理
		_ = w.CloseWithError(err)
	}()

	go func() {
defer r.Close() //确保管道读卡器关闭

//在此阻止，直到附件完成或我们超时
		select {
case <-attached: //成功附着
close(attached) //关闭表示现在可以复制流

		case <-time.After(10 * time.Second):
			logger.Errorf("Timeout while attaching to IO channel in container %s", containerName)
			return
		}

		is := bufio.NewReader(r)
		for {
//循环将文本行永久转储到容器记录器中
//直到管道关闭
			line, err := is.ReadString('\n')
			switch err {
			case nil:
				containerLogger.Info(line)
			case io.EOF:
				logger.Infof("Container %s has closed its IO channel", containerName)
				return
			default:
				logger.Errorf("Error reading container output: %s", err)
				return
			}
		}
	}()
}

//停止停止运行链码
func (vm *DockerVM) Stop(ccid ccintf.CCID, timeout uint, dontkill bool, dontremove bool) error {
	client, err := vm.getClientFnc()
	if err != nil {
		dockerLogger.Debugf("stop - cannot create client %s", err)
		return err
	}
	id := strings.Replace(vm.GetVMName(ccid), ":", "_", -1)

	return vm.stopInternal(client, id, timeout, dontkill, dontremove)
}

//healthcheck检查dockervm是否能够与docker通信
//守护进程。
func (vm *DockerVM) HealthCheck(ctx context.Context) error {
	client, err := vm.getClientFnc()
	if err != nil {
		return errors.Wrap(err, "failed to connect to Docker daemon")
	}
	if err := client.PingWithContext(ctx); err != nil {
		return errors.Wrap(err, "failed to ping to Docker daemon")
	}
	return nil
}

func (vm *DockerVM) stopInternal(client dockerClient, id string, timeout uint, dontkill, dontremove bool) error {
	logger := dockerLogger.With("id", id)

	logger.Debugw("stopping container")
	err := client.StopContainer(id, timeout)
	dockerLogger.Debugw("stop container result", "error", err)

	if !dontkill {
		logger.Debugw("killing container")
		err = client.KillContainer(docker.KillContainerOptions{ID: id})
		logger.Debugw("kill container result", "error", err)
	}

	if !dontremove {
		logger.Debugw("removing container")
		err = client.RemoveContainer(docker.RemoveContainerOptions{ID: id, Force: true})
		logger.Debugw("remove container result", "error", err)
	}

	return err
}

//getvmname根据对等信息生成VM名称。它接受一种格式
//function parameter to allow different formatting based on the desired use of
//这个名字。
func (vm *DockerVM) GetVMName(ccid ccintf.CCID) string {
//将任何无效字符替换为“-”（网络ID、对等ID或
//任何格式函数返回的完整名称
	return vmRegExp.ReplaceAllString(vm.preFormatImageName(ccid), "-")
}

//getvmnamefordocker根据对等信息格式化Docker映像。这是
//需要在单个主机、多个对等机中保持映像（存储库）名称的唯一性
//环境（如开发环境）。它计算
//提供的图像名称，然后将其附加到小写图像名称以确保
//唯一性。
func (vm *DockerVM) GetVMNameForDocker(ccid ccintf.CCID) (string, error) {
	name := vm.preFormatImageName(ccid)
	hash := hex.EncodeToString(util.ComputeSHA256([]byte(name)))
	saniName := vmRegExp.ReplaceAllString(name, "-")
	imageName := strings.ToLower(fmt.Sprintf("%s-%s", saniName, hash))

//Check that name complies with Docker's repository naming rules
	if !imageRegExp.MatchString(imageName) {
		dockerLogger.Errorf("Error constructing Docker VM Name. '%s' breaks Docker's repository naming rules", name)
		return "", fmt.Errorf("Error constructing Docker VM Name. '%s' breaks Docker's repository naming rules", imageName)
	}

	return imageName, nil
}

func (vm *DockerVM) preFormatImageName(ccid ccintf.CCID) string {
	name := ccid.GetName()

	if vm.NetworkID != "" && vm.PeerID != "" {
		name = fmt.Sprintf("%s-%s-%s", vm.NetworkID, vm.PeerID, name)
	} else if vm.NetworkID != "" {
		name = fmt.Sprintf("%s-%s", vm.NetworkID, name)
	} else if vm.PeerID != "" {
		name = fmt.Sprintf("%s-%s", vm.PeerID, name)
	}

	return name
}
