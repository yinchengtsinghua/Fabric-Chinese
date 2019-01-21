
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


package runner

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
	"github.com/tedsuo/ifrit"
)

const CouchDBDefaultImage = "hyperledger/fabric-couchdb:latest"

//couchdb管理Dockerized counchdb实例的执行
//用于测试。
type CouchDB struct {
	Client        *docker.Client
	Image         string
	HostIP        string
	HostPort      int
	ContainerPort docker.Port
	Name          string
	StartTimeout  time.Duration

	ErrorStream  io.Writer
	OutputStream io.Writer

	containerID      string
	hostAddress      string
	containerAddress string
	address          string

	mutex   sync.Mutex
	stopped bool
}

//run运行couchdb容器。它实现ifrit.runner接口
func (c *CouchDB) Run(sigCh <-chan os.Signal, ready chan<- struct{}) error {
	if c.Image == "" {
		c.Image = CouchDBDefaultImage
	}

	if c.Name == "" {
		c.Name = DefaultNamer()
	}

	if c.HostIP == "" {
		c.HostIP = "127.0.0.1"
	}

	if c.ContainerPort == docker.Port("") {
		c.ContainerPort = docker.Port("5984/tcp")
	}

	if c.StartTimeout == 0 {
		c.StartTimeout = DefaultStartTimeout
	}

	if c.Client == nil {
		client, err := docker.NewClientFromEnv()
		if err != nil {
			return err
		}
		c.Client = client
	}

	hostConfig := &docker.HostConfig{
		AutoRemove: true,
		PortBindings: map[docker.Port][]docker.PortBinding{
			c.ContainerPort: {{
				HostIP:   c.HostIP,
				HostPort: strconv.Itoa(c.HostPort),
			}},
		},
	}

	container, err := c.Client.CreateContainer(
		docker.CreateContainerOptions{
			Name:       c.Name,
			Config:     &docker.Config{Image: c.Image},
			HostConfig: hostConfig,
		},
	)
	if err != nil {
		return err
	}
	c.containerID = container.ID

	err = c.Client.StartContainer(container.ID, nil)
	if err != nil {
		return err
	}
	defer c.Stop()

	container, err = c.Client.InspectContainer(container.ID)
	if err != nil {
		return err
	}
	c.hostAddress = net.JoinHostPort(
		container.NetworkSettings.Ports[c.ContainerPort][0].HostIP,
		container.NetworkSettings.Ports[c.ContainerPort][0].HostPort,
	)
	c.containerAddress = net.JoinHostPort(
		container.NetworkSettings.IPAddress,
		c.ContainerPort.Port(),
	)

	streamCtx, streamCancel := context.WithCancel(context.Background())
	defer streamCancel()
	go c.streamLogs(streamCtx)

	containerExit := c.wait()
	ctx, cancel := context.WithTimeout(context.Background(), c.StartTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return errors.Wrapf(ctx.Err(), "database in container %s did not start", c.containerID)
	case <-containerExit:
		return errors.New("container exited before ready")
	case <-c.ready(ctx, c.hostAddress):
		c.address = c.hostAddress
	case <-c.ready(ctx, c.containerAddress):
		c.address = c.containerAddress
	}

	cancel()
	close(ready)

	for {
		select {
		case err := <-containerExit:
			return err
		case <-sigCh:
			if err := c.Stop(); err != nil {
				return err
			}
		}
	}
}

func endpointReady(ctx context.Context, url string) bool {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	return err == nil && resp.StatusCode == http.StatusOK
}

func (c *CouchDB) ready(ctx context.Context, addr string) <-chan struct{} {
	readyCh := make(chan struct{})
url := fmt.Sprintf("http://%S/[，ADDR ]
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			if endpointReady(ctx, url) {
				close(readyCh)
				return
			}
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	return readyCh
}

func (c *CouchDB) wait() <-chan error {
	exitCh := make(chan error)
	go func() {
		exitCode, err := c.Client.WaitContainer(c.containerID)
		if err == nil {
			err = fmt.Errorf("couchdb: process exited with %d", exitCode)
		}
		exitCh <- err
	}()

	return exitCh
}

func (c *CouchDB) streamLogs(ctx context.Context) {
	if c.ErrorStream == nil && c.OutputStream == nil {
		return
	}

	logOptions := docker.LogsOptions{
		Context:      ctx,
		Container:    c.containerID,
		Follow:       true,
		ErrorStream:  c.ErrorStream,
		OutputStream: c.OutputStream,
		Stderr:       c.ErrorStream != nil,
		Stdout:       c.OutputStream != nil,
	}

	err := c.Client.Logs(logOptions)
	if err != nil {
		fmt.Fprintf(c.ErrorStream, "log stream ended with error: %s", err)
	}
}

//地址返回就绪检查成功使用的地址。
func (c *CouchDB) Address() string {
	return c.address
}

//host address返回此couchdb实例可用的主机地址。
func (c *CouchDB) HostAddress() string {
	return c.hostAddress
}

//containerAddress返回此couchdb实例所在的容器地址
//是可用的。
func (c *CouchDB) ContainerAddress() string {
	return c.containerAddress
}

//containerID返回此couchdb的容器ID
func (c *CouchDB) ContainerID() string {
	return c.containerID
}

//start使用ifrit运行程序启动couchdb容器
func (c *CouchDB) Start() error {
	p := ifrit.Invoke(c)

	select {
	case <-p.Ready():
		return nil
	case err := <-p.Wait():
		return err
	}
}

//停止并删除couchdb容器
func (c *CouchDB) Stop() error {
	c.mutex.Lock()
	if c.stopped {
		c.mutex.Unlock()
		return errors.Errorf("container %s already stopped", c.containerID)
	}
	c.stopped = true
	c.mutex.Unlock()

	err := c.Client.StopContainer(c.containerID, 0)
	if err != nil {
		return err
	}

	return nil
}
