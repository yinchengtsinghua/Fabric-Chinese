
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


package client

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/token"
	tk "github.com/hyperledger/fabric/token"
)

//go：生成伪造者-o mock/prover.go-伪造名称prover。证明者

type Prover interface {

//requestimport允许客户机向验证程序对等服务提交问题请求；
//该函数作为参数tokenstoissue和客户端的签名标识；
//它以字节为单位返回响应，并在请求失败时返回错误消息。
//响应对应于序列化的令牌事务Protobuf消息。
	RequestImport(tokensToIssue []*token.TokenToIssue, signingIdentity tk.SigningIdentity) ([]byte, error)

//RequestTransfer允许客户端向验证程序对等服务提交传输请求；
//函数将fabtoken应用程序凭证（令牌的标识符）作为参数。
//转让人和描述如何分配的股份
//在接收者之间；如果
//请求失败
	RequestTransfer(tokenIDs [][]byte, shares []*token.RecipientTransferShare, signingIdentity tk.SigningIdentity) ([]byte, error)
}

//go：生成仿冒者-o模仿/织物提交者。go-仿冒名称fabric tx submitter。FabricTx提交人

type FabricTxSubmitter interface {

//Submit允许客户为FabToken构建和提交一个结构事务，该FabToken具有
//有效负载序列化的Tx；它将字节数组作为输入
//并返回一个错误，指示Tx提交的成功或失败以及一个错误
//解释原因。
	Submit(tx []byte) error
}

//客户端表示调用prover和txsubmitter的客户端结构
type Client struct {
	SigningIdentity tk.SigningIdentity
	Prover          Prover
	TxSubmitter     FabricTxSubmitter
}

//问题是客户端调用的将令牌引入系统的函数。
//issue将token.tokentoIssue的数组作为参数，该数组定义了哪些token
//将被介绍。

func (c *Client) Issue(tokensToIssue []*token.TokenToIssue) ([]byte, error) {
	serializedTokenTx, err := c.Prover.RequestImport(tokensToIssue, c.SigningIdentity)
	if err != nil {
		return nil, err
	}

	tx, err := c.createTx(serializedTokenTx)
	if err != nil {
		return nil, err
	}

	return tx, c.TxSubmitter.Submit(tx)
}

//Transfer是客户机调用以传输其令牌的函数。
//transfer将token.recipienttransfershare的数组作为参数
//标识谁接收令牌并描述如何分发令牌。
func (c *Client) Transfer(tokenIDs [][]byte, shares []*token.RecipientTransferShare) ([]byte, error) {
	serializedTokenTx, err := c.Prover.RequestTransfer(tokenIDs, shares, c.SigningIdentity)
	if err != nil {
		return nil, err
	}
	tx, err := c.createTx(serializedTokenTx)
	if err != nil {
		return nil, err
	}

	return tx, c.TxSubmitter.Submit(tx)
}

//稍后要更新的TODO，以具有适当的结构头
//createTx是一个函数，它创建一个结构Tx，形成一个字节数组。
func (c *Client) createTx(tokenTx []byte) ([]byte, error) {
	payload := &common.Payload{Data: tokenTx}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}
	signature, err := c.SigningIdentity.Sign(payloadBytes)
	if err != nil {
		return nil, err
	}
	envelope := &common.Envelope{Payload: payloadBytes, Signature: signature}
	return proto.Marshal(envelope)
}
