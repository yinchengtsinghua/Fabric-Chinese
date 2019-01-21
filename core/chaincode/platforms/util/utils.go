
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。

许可证限制。
**/


package util

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/util"
	cutil "github.com/hyperledger/fabric/core/container/util"
)

var logger = flogging.MustGetLogger("chaincode.platform.util")

//computehash根据上一个哈希计算内容哈希
func ComputeHash(contents []byte, hash []byte) []byte {
	newSlice := make([]byte, len(hash)+len(contents))

//复制内容
	copy(newSlice[0:len(contents)], contents[:])

//添加上一个哈希
	copy(newSlice[len(contents):], hash[:])

//计算新哈希
	hash = util.ComputeSHA256(newSlice)

	return hash
}

//hashfileindir为目录中的每个文件计算h=hash（h，文件字节）
//目录条目是递归遍历的。最后一个单曲
//为整个目录结构返回哈希值
func HashFilesInDir(rootDir string, dir string, hash []byte, tw *tar.Writer) ([]byte, error) {
	currentDir := filepath.Join(rootDir, dir)
	logger.Debugf("hashFiles %s", currentDir)
//readdir返回dir中已排序的文件列表
	fis, err := ioutil.ReadDir(currentDir)
	if err != nil {
		return hash, fmt.Errorf("ReadDir failed %s\n", err)
	}
	for _, fi := range fis {
		name := filepath.Join(dir, fi.Name())
		if fi.IsDir() {
			var err error
			hash, err = HashFilesInDir(rootDir, name, hash, tw)
			if err != nil {
				return hash, err
			}
			continue
		}
		fqp := filepath.Join(rootDir, name)
		buf, err := ioutil.ReadFile(fqp)
		if err != nil {
			logger.Errorf("Error reading %s\n", err)
			return hash, err
		}

//从文件内容获取新哈希
		hash = ComputeHash(buf, hash)

		if tw != nil {
			is := bytes.NewReader(buf)
			if err = cutil.WriteStreamToPackage(is, fqp, filepath.Join("src", name), tw); err != nil {
				return hash, fmt.Errorf("Error adding file to tar %s", err)
			}
		}
	}
	return hash, nil
}

//
func IsCodeExist(tmppath string) error {
	file, err := os.Open(tmppath)
	if err != nil {
		return fmt.Errorf("Could not open file %s", err)
	}

	fi, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Could not stat file %s", err)
	}

	if !fi.IsDir() {
		return fmt.Errorf("File %s is not dir\n", file.Name())
	}

	return nil
}

type DockerBuildOptions struct {
	Image        string
	Env          []string
	Cmd          string
	InputStream  io.Reader
	OutputStream io.Writer
}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//码头建筑
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//此函数允许在Docker容器中“传递”构建链码，如下所示
//使用标准“docker build”+dockerfile机制的替代方法。普通码头工人
//由于生成的图像是由
//构建时和运行时环境。这个超集可能在几个方面有问题
//
//不需要的应用程序等。
//
//因此，这种机制创建了一个由临时码头工人组成的管道。
//接受源代码作为输入的容器，运行一些函数（例如“go build”），以及
//输出结果。其目的是将这些产出作为
//通过将输出安装到下游Docker构建中的流线型容器
//适当的最小图像。
//
//输入参数相当简单：
//-image：（可选）要使用的生成器图像或“chaincode.builder”
//-env：（可选）构建环境的环境变量。
//-cmd：在容器内执行的命令。
//-inputstream：将扩展为/chaincode/input的文件tarball。
//-outputstream:将从/chaincode/output收集的文件tarball
//成功执行命令后。
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
func DockerBuild(opts DockerBuildOptions) error {
	client, err := cutil.NewDockerClient()
	if err != nil {
		return fmt.Errorf("Error creating docker client: %s", err)
	}
	if opts.Image == "" {
		opts.Image = cutil.GetDockerfileFromConfig("chaincode.builder")
		if opts.Image == "" {
			return fmt.Errorf("No image provided and \"chaincode.builder\" default does not exist")
		}
	}

	logger.Debugf("Attempting build with image %s", opts.Image)

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//确保映像在本地存在，如果不存在则从注册表中提取
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	_, err = client.InspectImage(opts.Image)
	if err != nil {
		logger.Debugf("Image %s does not exist locally, attempt pull", opts.Image)

		err = client.PullImage(docker.PullImageOptions{Repository: opts.Image}, docker.AuthConfiguration{})
		if err != nil {
			return fmt.Errorf("Failed to pull %s: %s", opts.Image, err)
		}
	}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//使用env/cmd创建一个临时容器
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:        opts.Image,
			Env:          opts.Env,
			Cmd:          []string{"/bin/sh", "-c", opts.Cmd},
			AttachStdout: true,
			AttachStderr: true,
		},
	})
	if err != nil {
		return fmt.Errorf("Error creating container: %s", err)
	}
	defer client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//上传我们的输入流
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	err = client.UploadToContainer(container.ID, docker.UploadToContainerOptions{
		Path:        "/chaincode/input",
		InputStream: opts.InputStream,
	})
	if err != nil {
		return fmt.Errorf("Error uploading input to container: %s", err)
	}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//附加stdout缓冲区以捕获可能的编译错误
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	stdout := bytes.NewBuffer(nil)
	cw, err := client.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
		Container:    container.ID,
		OutputStream: stdout,
		ErrorStream:  stdout,
		Logs:         true,
		Stdout:       true,
		Stderr:       true,
		Stream:       true,
	})
	if err != nil {
		return fmt.Errorf("Error attaching to container: %s", err)
	}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//启动实际的构建，实现容器创建时指定的env/cmd
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	err = client.StartContainer(container.ID, nil)
	if err != nil {
		cw.Close()
		return fmt.Errorf("Error executing build: %s \"%s\"", err, stdout.String())
	}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//等待生成完成并收集返回值
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	retval, err := client.WaitContainer(container.ID)
	if err != nil {
		cw.Close()
		return fmt.Errorf("Error waiting for container to complete: %s", err)
	}

//在访问stdout之前，请等待流复制完成。
	cw.Close()
	if err := cw.Wait(); err != nil {
		logger.Errorf("attach wait failed: %s", err)
	}

	if retval > 0 {
		return fmt.Errorf("Error returned from build: %d \"%s\"", retval, stdout.String())
	}

	logger.Debugf("Build output is %s", stdout.String())

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
	err = client.DownloadFromContainer(container.ID, docker.DownloadFromContainerOptions{
		Path:         "/chaincode/output/.",
		OutputStream: opts.OutputStream,
	})
	if err != nil {
		return fmt.Errorf("Error downloading output: %s", err)
	}

	return nil
}
