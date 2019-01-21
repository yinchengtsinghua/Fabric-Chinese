
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


package chaincode

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/accesscontrol"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/container"
	"github.com/hyperledger/fabric/core/container/ccintf"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

//处理器处理虚拟机和容器请求。
type Processor interface {
	Process(vmtype string, req container.VMCReq) error
}

//certgenerator为chaincode生成客户端证书。
type CertGenerator interface {
//生成返回证书、私钥和关联项
//具有给定链码名称的证书的哈希
	Generate(ccName string) (*accesscontrol.CertAndPrivKeyPair, error)
}

//ContainerRuntime负责管理容器化的链代码。
type ContainerRuntime struct {
	CertGenerator    CertGenerator
	Processor        Processor
	CACert           []byte
	CommonEnv        []string
	PeerAddress      string
	PlatformRegistry *platforms.Registry
}

//Start在运行时环境中启动链码。
func (c *ContainerRuntime) Start(ccci *ccprovider.ChaincodeContainerInfo, codePackage []byte) error {
	cname := ccci.Name + ":" + ccci.Version

	lc, err := c.LaunchConfig(cname, ccci.Type)
	if err != nil {
		return err
	}

	chaincodeLogger.Debugf("start container: %s", cname)
	chaincodeLogger.Debugf("start container with args: %s", strings.Join(lc.Args, " "))
	chaincodeLogger.Debugf("start container with env:\n\t%s", strings.Join(lc.Envs, "\n\t"))

	scr := container.StartContainerReq{
		Builder: &container.PlatformBuilder{
			Type:             ccci.Type,
			Name:             ccci.Name,
			Version:          ccci.Version,
			Path:             ccci.Path,
			CodePackage:      codePackage,
			PlatformRegistry: c.PlatformRegistry,
		},
		Args:          lc.Args,
		Env:           lc.Envs,
		FilesToUpload: lc.Files,
		CCID: ccintf.CCID{
			Name:    ccci.Name,
			Version: ccci.Version,
		},
	}

	if err := c.Processor.Process(ccci.ContainerType, scr); err != nil {
		return errors.WithMessage(err, "error starting container")
	}

	return nil
}

//stop终止chaincode及其容器运行时环境。
func (c *ContainerRuntime) Stop(ccci *ccprovider.ChaincodeContainerInfo) error {
	scr := container.StopContainerReq{
		CCID: ccintf.CCID{
			Name:    ccci.Name,
			Version: ccci.Version,
		},
		Timeout:    0,
		Dontremove: false,
	}

	if err := c.Processor.Process(ccci.ContainerType, scr); err != nil {
		return errors.WithMessage(err, "error stopping container")
	}

	return nil
}

const (
//链码容器中的相互TLS身份验证客户端密钥和证书路径
	TLSClientKeyPath      string = "/etc/hyperledger/fabric/client.key"
	TLSClientCertPath     string = "/etc/hyperledger/fabric/client.crt"
	TLSClientRootCertPath string = "/etc/hyperledger/fabric/peer.crt"
)

func (c *ContainerRuntime) getTLSFiles(keyPair *accesscontrol.CertAndPrivKeyPair) map[string][]byte {
	if keyPair == nil {
		return nil
	}

	return map[string][]byte{
		TLSClientKeyPath:      []byte(keyPair.Key),
		TLSClientCertPath:     []byte(keyPair.Cert),
		TLSClientRootCertPath: c.CACert,
	}
}

//launchconfig保存链码启动参数、环境变量和文件。
type LaunchConfig struct {
	Args  []string
	Envs  []string
	Files map[string][]byte
}

//
func (c *ContainerRuntime) LaunchConfig(cname string, ccType string) (*LaunchConfig, error) {
	var lc LaunchConfig

//通用环境变量
	lc.Envs = append(c.CommonEnv, "CORE_CHAINCODE_ID_NAME="+cname)

//语言特定参数
	switch ccType {
	case pb.ChaincodeSpec_GOLANG.String(), pb.ChaincodeSpec_CAR.String():
		lc.Args = []string{"chaincode", fmt.Sprintf("-peer.address=%s", c.PeerAddress)}
	case pb.ChaincodeSpec_JAVA.String():
		lc.Args = []string{"/root/chaincode-java/start", "--peerAddress", c.PeerAddress}
	case pb.ChaincodeSpec_NODE.String():
		lc.Args = []string{"/bin/sh", "-c", fmt.Sprintf("cd /usr/local/src; npm start -- --peer.address %s", c.PeerAddress)}
	default:
		return nil, errors.Errorf("unknown chaincodeType: %s", ccType)
	}

//将TLS选项传递到链码
	if c.CertGenerator != nil {
		certKeyPair, err := c.CertGenerator.Generate(cname)
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("failed to generate TLS certificates for %s", cname))
		}
		lc.Files = c.getTLSFiles(certKeyPair)
		if lc.Files == nil {
			return nil, errors.Errorf("failed to acquire TLS certificates for %s", cname)
		}

		lc.Envs = append(lc.Envs, "CORE_PEER_TLS_ENABLED=true")
		lc.Envs = append(lc.Envs, fmt.Sprintf("CORE_TLS_CLIENT_KEY_PATH=%s", TLSClientKeyPath))
		lc.Envs = append(lc.Envs, fmt.Sprintf("CORE_TLS_CLIENT_CERT_PATH=%s", TLSClientCertPath))
		lc.Envs = append(lc.Envs, fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=%s", TLSClientRootCertPath))
	} else {
		lc.Envs = append(lc.Envs, "CORE_PEER_TLS_ENABLED=false")
	}

	chaincodeLogger.Debugf("launchConfig: %s", lc.String())

	return &lc, nil
}

func (lc *LaunchConfig) String() string {
	buf := &bytes.Buffer{}
	if len(lc.Args) > 0 {
		fmt.Fprintf(buf, "executable:%q,", lc.Args[0])
	}

	fileNames := []string{}
	for k := range lc.Files {
		fileNames = append(fileNames, k)
	}
	sort.Strings(fileNames)

	fmt.Fprintf(buf, "Args:[%s],", strings.Join(lc.Args, ","))
	fmt.Fprintf(buf, "Envs:[%s],", strings.Join(lc.Envs, ","))
	fmt.Fprintf(buf, "Files:%v", fileNames)
	return buf.String()
}
