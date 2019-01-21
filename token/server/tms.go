
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

package server

import (
	"github.com/hyperledger/fabric/protos/token"
)

//go：生成伪造者-o mock/issuer.go-伪造姓名的发行者。发行人

//颁发者创建令牌导入请求。
type Issuer interface {
//问题创建导入请求事务。
	RequestImport(tokensToIssue []*token.TokenToIssue) (*token.TokenTransaction, error)

//RequestExpectation允许基于期望进行间接导入。
//它创建一个具有预期中指定的输出的令牌事务。
	RequestExpectation(request *token.ExpectationRequest) (*token.TokenTransaction, error)
}

//go：生成伪造者-o mock/transactor.go-forke name transactor。交易人

//交易人允许使用已发行的代币
type Transactor interface {
//RequestTransfer创建与令牌传输相关联的数据，假设
//应用程序级标识。Intokens字节是标识符
//在输出中，需要从分类帐中查找其详细信息。
	RequestTransfer(request *token.TransferRequest) (*token.TokenTransaction, error)

//RequestReduce允许赎回输入tokenids中的令牌
//它查询分类账以读取每个令牌ID的详细信息。
//它创建了一个令牌事务，并为兑换的令牌和
//可能是另一个输出，用于将剩余的令牌（如果有）传输给创建者
	RequestRedeem(request *token.RedeemRequest) (*token.TokenTransaction, error)

//listTokens返回此事务处理程序拥有的未使用令牌的一部分
	ListTokens() (*token.UnspentTokens, error)

//RequestApprove创建一个包含必要数据的令牌事务
//赞成
	RequestApprove(request *token.ApproveRequest) (*token.TokenTransaction, error)

//RequestTransferFrom创建一个包含必要数据的令牌事务
//转让先前委托转让的第三方的代币
//通过批准请求
	RequestTransferFrom(request *token.TransferRequest) (*token.TokenTransaction, error)

//请求期望允许基于期望进行间接传输。
//它创建一个具有预期中指定的输出的令牌事务。
	RequestExpectation(request *token.ExpectationRequest) (*token.TokenTransaction, error)

//完成释放此事务处理程序持有的任何资源
	Done()
}

//go：生成伪造者-o mock/tms_manager.go-forke name tms manager。TMS-管理器

type TMSManager interface {
//GetIssuer返回绑定到已传递通道的颁发者及其凭据
//是元组（privatecredential、publiccredential）。
	GetIssuer(channel string, privateCredential, publicCredential []byte) (Issuer, error)

//GetTransactor返回绑定到已传递通道的事务处理程序及其凭据
//是元组（privatecredential、publiccredential）。
	GetTransactor(channel string, privateCredential, publicCredential []byte) (Transactor, error)
}
