
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
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/hyperledger/fabric/protos/token"
	"github.com/hyperledger/fabric/token/ledger"
	"github.com/pkg/errors"
)

//一种可以转移代币的交易机。
type Transactor struct {
	PublicCredential []byte
	Ledger           ledger.LedgerReader
}

//RequestTransfer创建类型为Transfer Request的令牌事务
//func（t*事务处理程序）requestTransfer（intokens[]*token.inputid，tokenstotransfer[]*token.recipientTransferShare）（*token.tokenTransaction，error）
func (t *Transactor) RequestTransfer(request *token.TransferRequest) (*token.TokenTransaction, error) {
	var outputs []*token.PlainOutput

	inputs, tokenType, _, err := t.getInputsFromTokenIds(request.GetTokenIds())
	if err != nil {
		return nil, err
	}

	for _, ttt := range request.GetShares() {
		outputs = append(outputs, &token.PlainOutput{
			Owner:    ttt.Recipient,
			Type:     tokenType,
			Quantity: ttt.Quantity,
		})
	}

//准备转移请求
	transaction := &token.TokenTransaction{
		Action: &token.TokenTransaction_PlainAction{
			PlainAction: &token.PlainTokenAction{
				Data: &token.PlainTokenAction_PlainTransfer{
					PlainTransfer: &token.PlainTransfer{
						Inputs:  inputs,
						Outputs: outputs,
					},
				},
			},
		},
	}

	return transaction, nil
}

//RequestRecreate创建类型为Recreate Request的令牌事务
func (t *Transactor) RequestRedeem(request *token.RedeemRequest) (*token.TokenTransaction, error) {
	if len(request.GetTokenIds()) == 0 {
		return nil, errors.New("no token ids in RedeemRequest")
	}
	if request.GetQuantityToRedeem() <= 0 {
		return nil, errors.Errorf("quantity to redeem [%d] must be greater than 0", request.GetQuantityToRedeem())
	}

	inputs, tokenType, quantitySum, err := t.getInputsFromTokenIds(request.GetTokenIds())
	if err != nil {
		return nil, err
	}

	if quantitySum < request.QuantityToRedeem {
		return nil, errors.Errorf("total quantity [%d] from TokenIds is less than quantity [%d] to be redeemed", quantitySum, request.QuantityToRedeem)
	}

//为兑换本身添加输出
	var outputs []*token.PlainOutput
	outputs = append(outputs, &token.PlainOutput{
		Type:     tokenType,
		Quantity: request.QuantityToRedeem,
	})

//如果赎回后有剩余数量，则添加另一个输出
	if quantitySum > request.QuantityToRedeem {
		outputs = append(outputs, &token.PlainOutput{
Owner:    t.PublicCredential, //publicCredential是创建者的序列化标识
			Type:     tokenType,
			Quantity: quantitySum - request.QuantityToRedeem,
		})
	}

//Plain赎回共享与PlainTransfer相同的数据结构
	transaction := &token.TokenTransaction{
		Action: &token.TokenTransaction_PlainAction{
			PlainAction: &token.PlainTokenAction{
				Data: &token.PlainTokenAction_PlainRedeem{
					PlainRedeem: &token.PlainTransfer{
						Inputs:  inputs,
						Outputs: outputs,
					},
				},
			},
		},
	}

	return transaction, nil
}

//从分类帐中读取每个令牌ID的令牌数据，并计算所有令牌ID的数量总和
//返回输入、令牌类型、令牌数量之和以及失败时的错误
func (t *Transactor) getInputsFromTokenIds(tokenIds [][]byte) ([]*token.InputId, string, uint64, error) {
	var inputs []*token.InputId
	var tokenType string = ""
	var quantitySum uint64 = 0
	for _, inKeyBytes := range tokenIds {
//将组合键字节解析为字符串
		inKey := parseCompositeKeyBytes(inKeyBytes)

//检查复合键是否符合输出的复合键
		namespace, components, err := splitCompositeKey(inKey)
		if err != nil {
			return nil, "", 0, errors.New(fmt.Sprintf("error splitting input composite key: '%s'", err))
		}
		if namespace != tokenOutput {
			return nil, "", 0, errors.New(fmt.Sprintf("namespace not '%s': '%s'", tokenOutput, namespace))
		}
		if len(components) != 2 {
			return nil, "", 0, errors.New(fmt.Sprintf("not enough components in output ID composite key; expected 2, received '%s'", components))
		}
		txID := components[0]
		index, err := strconv.Atoi(components[1])
		if err != nil {
			return nil, "", 0, errors.New(fmt.Sprintf("error parsing output index '%s': '%s'", components[1], err))
		}

//确保输出存在于分类帐中
		inBytes, err := t.Ledger.GetState(tokenNameSpace, inKey)
		if err != nil {
			return nil, "", 0, err
		}
		if inBytes == nil {
			return nil, "", 0, errors.New(fmt.Sprintf("input '%s' does not exist", inKey))
		}
		input := &token.PlainOutput{}
		err = proto.Unmarshal(inBytes, input)
		if err != nil {
			return nil, "", 0, errors.New(fmt.Sprintf("error unmarshaling input bytes: '%s'", err))
		}

//检查令牌的所有者
		if !bytes.Equal(t.PublicCredential, input.Owner) {
			return nil, "", 0, errors.New(fmt.Sprintf("the requestor does not own inputs"))
		}

//检查令牌类型-每次传输只允许一种类型
		if tokenType == "" {
			tokenType = input.Type
		} else if tokenType != input.Type {
			return nil, "", 0, errors.New(fmt.Sprintf("two or more token types specified in input: '%s', '%s'", tokenType, input.Type))
		}
//将输入添加到输入列表
		inputs = append(inputs, &token.InputId{TxId: txID, Index: uint32(index)})

//数量的总和
		quantitySum += input.Quantity
	}

	return inputs, tokenType, quantitySum, nil
}

//listTokens创建一个标记事务，列出所有者拥有的未使用的标记。
func (t *Transactor) ListTokens() (*token.UnspentTokens, error) {
	iterator, err := t.Ledger.GetStateRangeScanIterator(tokenNameSpace, "", "")
	if err != nil {
		return nil, err
	}

	tokens := make([]*token.TokenOutput, 0)
	prefix, err := createPrefix(tokenOutput)
	if err != nil {
		return nil, err
	}
	for {
		next, err := iterator.Next()

		switch {
		case err != nil:
			return nil, err

		case next == nil:
//来自迭代器的nil响应表示查询结果结束
			return &token.UnspentTokens{Tokens: tokens}, nil

		default:
			result, ok := next.(*queryresult.KV)
			if !ok {
				return nil, errors.New("failed to retrieve unspent tokens: casting error")
			}
			if strings.HasPrefix(result.Key, prefix) {
				output := &token.PlainOutput{}
				err = proto.Unmarshal(result.Value, output)
				if err != nil {
					return nil, errors.New("failed to retrieve unspent tokens: casting error")
				}
				if string(output.Owner) == string(t.PublicCredential) {
					spent, err := t.isSpent(result.Key)
					if err != nil {
						return nil, err
					}
					if !spent {
						tokens = append(tokens,
							&token.TokenOutput{
								Type:     output.Type,
								Quantity: output.Quantity,
								Id:       getCompositeKeyBytes(result.Key),
							})
					}
				}
			}
		}
	}

}

func (t *Transactor) RequestApprove(request *token.ApproveRequest) (*token.TokenTransaction, error) {
	if len(request.GetTokenIds()) == 0 {
		return nil, errors.New("no token ids in ApproveAllowanceRequest")
	}

	if len(request.AllowanceShares) == 0 {
		return nil, errors.New("no recipient shares in ApproveAllowanceRequest")
	}

	var delegatedOutputs []*token.PlainDelegatedOutput

	inputs, tokenType, sumQuantity, err := t.getInputsFromTokenIds(request.GetTokenIds())
	if err != nil {
		return nil, err
	}

//准备批准发送

	delegatedQuantity := uint64(0)
	for _, share := range request.GetAllowanceShares() {
		if len(share.Recipient) == 0 {
			return nil, errors.Errorf("the recipient in approve must be specified")
		}
		if share.Quantity <= 0 {
			return nil, errors.Errorf("the quantity to approve [%d] must be greater than 0", share.GetQuantity())
		}
		delegatedOutputs = append(delegatedOutputs, &token.PlainDelegatedOutput{
			Owner:      []byte(request.Credential),
			Delegatees: [][]byte{share.Recipient},
			Type:       tokenType,
			Quantity:   share.Quantity,
		})
		delegatedQuantity = delegatedQuantity + share.Quantity
	}
	if sumQuantity < delegatedQuantity {
		return nil, errors.Errorf("insufficient funds: %v < %v", sumQuantity, delegatedQuantity)

	}
	var output *token.PlainOutput
	if sumQuantity != delegatedQuantity {
		output = &token.PlainOutput{
			Owner:    request.Credential,
			Type:     tokenType,
			Quantity: sumQuantity - delegatedQuantity,
		}
	}

	transaction := &token.TokenTransaction{
		Action: &token.TokenTransaction_PlainAction{
			PlainAction: &token.PlainTokenAction{
				Data: &token.PlainTokenAction_PlainApprove{
					PlainApprove: &token.PlainApprove{
						Inputs:           inputs,
						DelegatedOutputs: delegatedOutputs,
						Output:           output,
					},
				},
			},
		},
	}

	return transaction, nil
}

func (t *Transactor) RequestTransferFrom(request *token.TransferRequest) (*token.TokenTransaction, error) {
	panic("implement me!")
}

//请求期望允许基于期望进行间接传输。
//它根据预期中指定的输出创建令牌事务。
func (t *Transactor) RequestExpectation(request *token.ExpectationRequest) (*token.TokenTransaction, error) {
	panic("not implemented yet")
}

//完成释放此事务处理程序持有的任何资源
func (t *Transactor) Done() {
	if t.Ledger != nil {
		t.Ledger.Done()
	}
}

//isspent检查是否已使用标识符为outputid的输出令牌。
func (t *Transactor) isSpent(outputID string) (bool, error) {
	key, err := createInputKey(outputID)
	if err != nil {
		return false, err
	}
	result, err := t.Ledger.GetState(tokenNameSpace, key)
	if err != nil {
		return false, err
	}
	if result == nil {
		return false, nil
	}
	return true, nil
}

//为令牌交易中的单个输入创建分类帐密钥，作为
//外宣
func createInputKey(outputID string) (string, error) {
	att := strings.Split(outputID, string(minUnicodeRuneValue))
	return createCompositeKey(tokenInput, att[1:])
}

//创建前缀作为作为参数传递的字符串的函数
func createPrefix(keyword string) (string, error) {
	return createCompositeKey(keyword, nil)
}

//generatekeyfortest仅用于测试目的，稍后将删除。
func GenerateKeyForTest(txID string, index int) (string, error) {
	return createOutputKey(txID, index)
}

func splitCompositeKey(compositeKey string) (string, []string, error) {
	componentIndex := 1
	components := []string{}
	for i := 1; i < len(compositeKey); i++ {
		if compositeKey[i] == minUnicodeRuneValue {
			components = append(components, compositeKey[componentIndex:i])
			componentIndex = i + 1
		}
	}
	if len(components) < 2 {
		return "", nil, errors.New("invalid composite key - no components found")
	}
	return components[0], components[1:], nil
}
