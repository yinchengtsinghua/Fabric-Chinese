
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


package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"

	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/peer/common/api"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

//PeerClient表示与对等机通信的客户端
type PeerClient struct {
	commonClient
}

//newpeerclientfromenv从全局创建peerclient的实例
//蝰蛇实例
func NewPeerClientFromEnv() (*PeerClient, error) {
	address, override, clientConfig, err := configFromEnv("peer")
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load config for PeerClient")
	}
	return newPeerClientForClientConfig(address, override, clientConfig)
}

//newpeerclientforaddress使用
//提供对等地址，如果启用了tls，则提供tls根证书文件
func NewPeerClientForAddress(address, tlsRootCertFile string) (*PeerClient, error) {
	if address == "" {
		return nil, errors.New("peer address must be set")
	}

	_, override, clientConfig, err := configFromEnv("peer")
	if clientConfig.SecOpts.UseTLS {
		if tlsRootCertFile == "" {
			return nil, errors.New("tls root cert file must be set")
		}
		caPEM, res := ioutil.ReadFile(tlsRootCertFile)
		if res != nil {
			err = errors.WithMessage(res, fmt.Sprintf("unable to load TLS root cert file from %s", tlsRootCertFile))
			return nil, err
		}
		clientConfig.SecOpts.ServerRootCAs = [][]byte{caPEM}
	}
	return newPeerClientForClientConfig(address, override, clientConfig)
}

func newPeerClientForClientConfig(address, override string, clientConfig comm.ClientConfig) (*PeerClient, error) {
	gClient, err := comm.NewGRPCClient(clientConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create PeerClient from config")
	}
	pClient := &PeerClient{
		commonClient: commonClient{
			GRPCClient: gClient,
			address:    address,
			sn:         override}}
	return pClient, nil
}

//背书人返回背书人服务的客户
func (pc *PeerClient) Endorser() (pb.EndorserClient, error) {
	conn, err := pc.commonClient.NewConnection(pc.address, pc.sn)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("endorser client failed to connect to %s", pc.address))
	}
	return pb.NewEndorserClient(conn), nil
}

//Deliver返回Deliver服务的客户端
func (pc *PeerClient) Deliver() (pb.Deliver_DeliverClient, error) {
	conn, err := pc.commonClient.NewConnection(pc.address, pc.sn)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("deliver client failed to connect to %s", pc.address))
	}
	return pb.NewDeliverClient(conn).Deliver(context.TODO())
}

//PeerDeliver返回用于传递服务的客户端以供对等端特定使用
//案例（即筛选的交付）
func (pc *PeerClient) PeerDeliver() (api.PeerDeliverClient, error) {
	conn, err := pc.commonClient.NewConnection(pc.address, pc.sn)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("deliver client failed to connect to %s", pc.address))
	}
	pbClient := pb.NewDeliverClient(conn)
	return &PeerDeliverClient{Client: pbClient}, nil
}

//admin返回管理服务的客户端
func (pc *PeerClient) Admin() (pb.AdminClient, error) {
	conn, err := pc.commonClient.NewConnection(pc.address, pc.sn)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("admin client failed to connect to %s", pc.address))
	}
	return pb.NewAdminClient(conn), nil
}

//证书返回TLS客户端证书（如果可用）
func (pc *PeerClient) Certificate() tls.Certificate {
	return pc.commonClient.Certificate()
}

//Get背书人客户端返回新的背书人客户端。如果地址和
//未提供tlsrootcertfile，将获取客户端的目标值
//从“peer.address”和的配置设置
//“对等.tls.rootcert.file”
func GetEndorserClient(address, tlsRootCertFile string) (pb.EndorserClient, error) {
	var peerClient *PeerClient
	var err error
	if address != "" {
		peerClient, err = NewPeerClientForAddress(address, tlsRootCertFile)
	} else {
		peerClient, err = NewPeerClientFromEnv()
	}
	if err != nil {
		return nil, err
	}
	return peerClient.Endorser()
}

//getCertificate返回客户端的TLS证书
func GetCertificate() (tls.Certificate, error) {
	peerClient, err := NewPeerClientFromEnv()
	if err != nil {
		return tls.Certificate{}, err
	}
	return peerClient.Certificate(), nil
}

//GetAdminClient返回新的管理客户端。的目标地址
//客户端取自配置设置“peer.address”
func GetAdminClient() (pb.AdminClient, error) {
	peerClient, err := NewPeerClientFromEnv()
	if err != nil {
		return nil, err
	}
	return peerClient.Admin()
}

//GetDeliverClient返回新的传递客户端。如果地址和
//未提供tlsrootcertfile，将获取客户端的目标值
//从“peer.address”和的配置设置
//“对等.tls.rootcert.file”
func GetDeliverClient(address, tlsRootCertFile string) (pb.Deliver_DeliverClient, error) {
	var peerClient *PeerClient
	var err error
	if address != "" {
		peerClient, err = NewPeerClientForAddress(address, tlsRootCertFile)
	} else {
		peerClient, err = NewPeerClientFromEnv()
	}
	if err != nil {
		return nil, err
	}
	return peerClient.Deliver()
}

//GetPeerDeliverClient返回新的传递客户端。如果地址和
//未提供tlsrootcertfile，将获取客户端的目标值
//从“peer.address”和的配置设置
//“对等.tls.rootcert.file”
func GetPeerDeliverClient(address, tlsRootCertFile string) (api.PeerDeliverClient, error) {
	var peerClient *PeerClient
	var err error
	if address != "" {
		peerClient, err = NewPeerClientForAddress(address, tlsRootCertFile)
	} else {
		peerClient, err = NewPeerClientFromEnv()
	}
	if err != nil {
		return nil, err
	}
	return peerClient.PeerDeliver()
}
