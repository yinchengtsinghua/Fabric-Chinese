
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


package car_test

import (
	"bytes"
	"fmt"

	"github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/car"
	"github.com/hyperledger/fabric/core/container"
	cutil "github.com/hyperledger/fabric/core/container/util"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//虚拟机实现虚拟机管理功能。
type VM struct {
	Client *docker.Client
}

//new vm创建一个新的vm实例。
func NewVM() (*VM, error) {
	client, err := cutil.NewDockerClient()
	if err != nil {
		return nil, err
	}
	VM := &VM{Client: client}
	return VM, nil
}

//BuildChainCodeContainer为提供的链代码规范生成容器
func (vm *VM) BuildChaincodeContainer(spec *pb.ChaincodeSpec) error {
	codePackage, err := container.GetChaincodePackageBytes(platforms.NewRegistry(&car.Platform{}), spec)
	if err != nil {
		return fmt.Errorf("Error getting chaincode package bytes: %s", err)
	}

	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: spec, CodePackage: codePackage}
	dockerSpec, err := platforms.NewRegistry(&car.Platform{}).GenerateDockerBuild(
		cds.CCType(),
		cds.Path(),
		cds.Name(),
		cds.Version(),
		cds.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("Error getting chaincode docker image: %s", err)
	}

	output := bytes.NewBuffer(nil)

	err = vm.Client.BuildImage(docker.BuildImageOptions{
		Name:         spec.ChaincodeId.Name,
		InputStream:  dockerSpec,
		OutputStream: output,
	})
	if err != nil {
		return fmt.Errorf("Error building docker: %s (output = %s)", err, output.String())
	}

	return nil
}
