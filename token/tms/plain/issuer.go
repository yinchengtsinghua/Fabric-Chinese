
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


package plain

import (
	"github.com/hyperledger/fabric/protos/token"
)

//可以导入新令牌的颁发者
type Issuer struct{}

//request import使用tokenstoissue中指定的令牌所有者、类型和数量创建导入请求。
func (i *Issuer) RequestImport(tokensToIssue []*token.TokenToIssue) (*token.TokenTransaction, error) {
	var outputs []*token.PlainOutput
	for _, tti := range tokensToIssue {
		outputs = append(outputs, &token.PlainOutput{
			Owner:    tti.Recipient,
			Type:     tti.Type,
			Quantity: tti.Quantity,
		})
	}

	return &token.TokenTransaction{
		Action: &token.TokenTransaction_PlainAction{
			PlainAction: &token.PlainTokenAction{
				Data: &token.PlainTokenAction_PlainImport{
					PlainImport: &token.PlainImport{
						Outputs: outputs,
					},
				},
			},
		},
	}, nil
}

//RequestExpectation允许基于期望进行间接导入。
//它创建一个具有预期中指定的输出的令牌事务。
func (i *Issuer) RequestExpectation(request *token.ExpectationRequest) (*token.TokenTransaction, error) {
	panic("not implemented yet")
}
