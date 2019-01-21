
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
	"encoding/hex"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/customtx"
	"github.com/hyperledger/fabric/protos/token"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/hyperledger/fabric/token/identity"
	"github.com/hyperledger/fabric/token/ledger"
	"github.com/pkg/errors"
)

const (
minUnicodeRuneValue   = 0            //U+ 0000
maxUnicodeRuneValue   = utf8.MaxRune //U+10ffff-最大（和未分配）码位
	compositeKeyNamespace = "\x00"
	tokenOutput           = "tokenOutput"
	tokenRedeem           = "tokenRedeem"
	tokenTx               = "tokenTx"
	tokenDelegatedOutput  = "tokenDelegatedOutput"
	tokenInput            = "tokenInput"
	tokenDelegatedInput   = "tokenDelegateInput"
	tokenNameSpace        = "tms"
)

var verifierLogger = flogging.MustGetLogger("token.tms.plain.verifier")

//验证程序验证并提交令牌事务。
type Verifier struct {
	IssuingValidator identity.IssuingValidator
}

//processtx检查事务是否正确。最近的分类帐状态。
//processTx检查是应按顺序进行的检查，因为块内的事务可能引入依赖项。
func (v *Verifier) ProcessTx(txID string, creator identity.PublicInfo, ttx *token.TokenTransaction, simulator ledger.LedgerWriter) error {
	verifierLogger.Debugf("checking transaction with txID '%s'", txID)
	err := v.checkProcess(txID, creator, ttx, simulator)
	if err != nil {
		return err
	}

	verifierLogger.Debugf("committing transaction with txID '%s'", txID)
	err = v.commitProcess(txID, creator, ttx, simulator)
	if err != nil {
		verifierLogger.Errorf("error committing transaction with txID '%s': %s", txID, err)
		return err
	}
	verifierLogger.Debugf("successfully processed transaction with txID '%s'", txID)
	return nil
}

func (v *Verifier) checkProcess(txID string, creator identity.PublicInfo, ttx *token.TokenTransaction, simulator ledger.LedgerReader) error {
	action := ttx.GetPlainAction()
	if action == nil {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("check process failed for transaction '%s': missing token action", txID)}
	}

	err := v.checkAction(creator, action, txID, simulator)
	if err != nil {
		return err
	}

	err = v.checkTxDoesNotExist(txID, simulator)
	if err != nil {
		return err
	}
	return nil
}

func (v *Verifier) checkAction(creator identity.PublicInfo, plainAction *token.PlainTokenAction, txID string, simulator ledger.LedgerReader) error {
	switch action := plainAction.Data.(type) {
	case *token.PlainTokenAction_PlainImport:
		return v.checkImportAction(creator, action.PlainImport, txID, simulator)
	case *token.PlainTokenAction_PlainTransfer:
		return v.checkTransferAction(creator, action.PlainTransfer, txID, simulator)
	case *token.PlainTokenAction_PlainRedeem:
		return v.checkRedeemAction(creator, action.PlainRedeem, txID, simulator)
	case *token.PlainTokenAction_PlainApprove:
		return v.checkApproveAction(creator, action.PlainApprove, txID, simulator)
	default:
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("unknown plain token action: %T", action)}
	}
}

func (v *Verifier) checkImportAction(creator identity.PublicInfo, importAction *token.PlainImport, txID string, simulator ledger.LedgerReader) error {
	err := v.checkImportOutputs(importAction.GetOutputs(), txID, simulator)
	if err != nil {
		return err
	}
	return v.checkImportPolicy(creator, txID, importAction)
}

func (v *Verifier) checkImportOutputs(outputs []*token.PlainOutput, txID string, simulator ledger.LedgerReader) error {
	if len(outputs) == 0 {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("no outputs in transaction: %s", txID)}
	}
	for i, output := range outputs {
		err := v.checkOutputDoesNotExist(i, txID, simulator)
		if err != nil {
			return err
		}

		if output.Quantity == 0 {
			return &customtx.InvalidTxError{Msg: fmt.Sprintf("output %d quantity is 0 in transaction: %s", i, txID)}
		}
	}
	return nil
}

func (v *Verifier) checkTransferAction(creator identity.PublicInfo, transferAction *token.PlainTransfer, txID string, simulator ledger.LedgerReader) error {
	outputType, outputSum, err := v.checkTransferOutputs(transferAction.GetOutputs(), txID, simulator)
	if err != nil {
		return err
	}
	inputType, inputSum, err := v.checkTransferInputs(creator, transferAction.GetInputs(), txID, simulator)
	if err != nil {
		return err
	}
	if outputType != inputType {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("token type mismatch in inputs and outputs for transfer with ID %s (%s vs %s)", txID, outputType, inputType)}
	}
	if outputSum != inputSum {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("token sum mismatch in inputs and outputs for transfer with ID %s (%d vs %d)", txID, outputSum, inputSum)}
	}
	return nil
}

func (v *Verifier) checkRedeemAction(creator identity.PublicInfo, redeemAction *token.PlainTransfer, txID string, simulator ledger.LedgerReader) error {
//首先执行与传输相同的检查
	err := v.checkTransferAction(creator, redeemAction, txID, simulator)
	if err != nil {
		return err
	}

//然后对兑换输出执行附加检查
//兑换交易的输出不应超过2个。
	outputs := redeemAction.GetOutputs()
	if len(outputs) > 2 {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("too many outputs (%d) in a redeem transaction", len(outputs))}
	}

//输出[0]应始终为兑现输出-即所有者应为零
	if outputs[0].Owner != nil {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("owner should be nil in a redeem output")}
	}

//如果输出[1]存在，则其所有者必须与创建者相同。
	if len(outputs) == 2 && !bytes.Equal(creator.Public(), outputs[1].Owner) {
		println(hex.EncodeToString(creator.Public()))
		println(hex.EncodeToString(outputs[1].Owner))
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("wrong owner for remaining tokens, should be original owner %s, but got %s", creator.Public(), outputs[1].Owner)}
	}

	return nil
}

func (v *Verifier) checkOutputDoesNotExist(index int, txID string, simulator ledger.LedgerReader) error {
	outputID, err := createOutputKey(txID, index)
	if err != nil {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("error creating output ID: %s", err)}
	}

	existingOutputBytes, err := simulator.GetState(tokenNameSpace, outputID)
	if err != nil {
		return err
	}

	if existingOutputBytes != nil {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("output already exists: %s", outputID)}
	}
	return nil
}

func (v *Verifier) checkTransferOutputs(outputs []*token.PlainOutput, txID string, simulator ledger.LedgerReader) (string, uint64, error) {
	tokenType := ""
	tokenSum := uint64(0)
	for i, output := range outputs {
		err := v.checkOutputDoesNotExist(i, txID, simulator)
		if err != nil {
			return "", 0, err
		}
		if tokenType == "" {
			tokenType = output.GetType()
		} else if tokenType != output.GetType() {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("multiple token types ('%s', '%s') in transfer output for txID '%s'", tokenType, output.GetType(), txID)}
		}
		tokenSum += output.GetQuantity()
	}
	return tokenType, tokenSum, nil
}

func (v *Verifier) checkTransferInputs(creator identity.PublicInfo, inputIDs []*token.InputId, txID string, simulator ledger.LedgerReader) (string, uint64, error) {
	tokenType := ""
	inputSum := uint64(0)
	processedIDs := make(map[string]bool)
	for _, id := range inputIDs {
		inputKey, err := createOutputKey(id.TxId, int(id.Index))
		if err != nil {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("error creating output ID for transfer input: %s", err)}
		}
		input, err := v.getOutput(inputKey, simulator)
		if err != nil {
			return "", 0, err
		}
		err = v.checkInputOwner(creator, input, inputKey)
		if err != nil {
			return "", 0, err
		}
		if tokenType == "" {
			tokenType = input.GetType()
		} else if tokenType != input.GetType() {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("multiple token types in transfer input for txID: %s (%s, %s)", txID, tokenType, input.GetType())}
		}
		if processedIDs[inputKey] {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("token input '%s' spent more than once in single transfer with txID '%s'", inputKey, txID)}
		}
		processedIDs[inputKey] = true
		inputSum += input.GetQuantity()
		spentKey, err := createSpentKey(id.TxId, int(id.Index))
		if err != nil {
			return "", 0, err
		}
		spent, err := v.isSpent(spentKey, simulator)
		if err != nil {
			return "", 0, err
		}
		if spent {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("input with ID %s for transfer has already been spent", inputKey)}
		}
	}
	return tokenType, inputSum, nil
}

func (v *Verifier) checkInputOwner(creator identity.PublicInfo, input *token.PlainOutput, inputID string) error {
	if !bytes.Equal(creator.Public(), input.Owner) {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("transfer input with ID %s not owned by creator", inputID)}
	}
	return nil
}

func (v *Verifier) checkTxDoesNotExist(txID string, simulator ledger.LedgerReader) error {
	txKey, err := createTxKey(txID)
	if err != nil {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("error creating txID: %s", err)}
	}

	existingTx, err := simulator.GetState(tokenNameSpace, txKey)
	if err != nil {
		return err
	}

	if existingTx != nil {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("transaction already exists: %s", txID)}
	}
	return nil
}

func (v *Verifier) checkImportPolicy(creator identity.PublicInfo, txID string, importData *token.PlainImport) error {
	for _, output := range importData.Outputs {
		err := v.IssuingValidator.Validate(creator, output.Type)
		if err != nil {
			return &customtx.InvalidTxError{Msg: fmt.Sprintf("import policy check failed: %s", err)}
		}
	}
	return nil
}

func (v *Verifier) commitProcess(txID string, creator identity.PublicInfo, ttx *token.TokenTransaction, simulator ledger.LedgerWriter) error {
	verifierLogger.Debugf("committing action with txID '%s'", txID)
	err := v.commitAction(ttx.GetPlainAction(), txID, simulator)
	if err != nil {
		verifierLogger.Errorf("error committing action with txID '%s': %s", txID, err)
		return err
	}

	verifierLogger.Debugf("adding transaction with txID '%s'", txID)
	err = v.addTransaction(txID, ttx, simulator)
	if err != nil {
		verifierLogger.Debugf("error adding transaction with txID '%s': %s", txID, err)
		return err
	}

	verifierLogger.Debugf("action with txID '%s' committed successfully", txID)
	return nil
}

func (v *Verifier) commitAction(plainAction *token.PlainTokenAction, txID string, simulator ledger.LedgerWriter) (err error) {
	switch action := plainAction.Data.(type) {
	case *token.PlainTokenAction_PlainImport:
		err = v.commitImportAction(action.PlainImport, txID, simulator)
	case *token.PlainTokenAction_PlainTransfer:
		err = v.commitTransferAction(action.PlainTransfer, txID, simulator)
	case *token.PlainTokenAction_PlainRedeem:
//调用与transfer相同的commit方法，因为plainReduce指向与transfer相同的输出类型
		err = v.commitTransferAction(action.PlainRedeem, txID, simulator)
	case *token.PlainTokenAction_PlainApprove:
		err = v.commitApproveAction(action.PlainApprove, txID, simulator)
	}
	return
}

func (v *Verifier) commitImportAction(importAction *token.PlainImport, txID string, simulator ledger.LedgerWriter) error {
	for i, output := range importAction.GetOutputs() {
		outputID, err := createOutputKey(txID, i)
		if err != nil {
			return &customtx.InvalidTxError{Msg: fmt.Sprintf("error creating output ID: %s", err)}
		}

		err = v.addOutput(outputID, output, simulator)
		if err != nil {
			return err
		}
	}
	return nil
}

//对转账和赎回交易都调用CommittTransferAction
//检查每个输出的所有者以确定如何生成密钥
func (v *Verifier) commitTransferAction(transferAction *token.PlainTransfer, txID string, simulator ledger.LedgerWriter) error {
	var outputID string
	var err error
	for i, output := range transferAction.GetOutputs() {
		if output.Owner != nil {
			outputID, err = createOutputKey(txID, i)
		} else {
			outputID, err = createRedeemKey(txID, i)
		}
		if err != nil {
			return &customtx.InvalidTxError{Msg: fmt.Sprintf("error creating output ID: %s", err)}
		}

		err = v.addOutput(outputID, output, simulator)
		if err != nil {
			return err
		}
	}
	return v.markInputsSpent(txID, transferAction.GetInputs(), simulator)
}

func (v *Verifier) checkApproveAction(creator identity.PublicInfo, approveAction *token.PlainApprove, txID string, simulator ledger.LedgerReader) error {
	outputType, outputSum, err := v.checkApproveOutputs(creator, approveAction.GetOutput(), approveAction.GetDelegatedOutputs(), txID, simulator)
	if err != nil {
		return err
	}
	inputType, inputSum, err := v.checkTransferInputs(creator, approveAction.GetInputs(), txID, simulator)
	if err != nil {
		return err
	}
	if outputType != inputType {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("token type mismatch in inputs and outputs for approve with ID %s (%s vs %s)", txID, outputType, inputType)}
	}
	if outputSum != inputSum {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("token sum mismatch in inputs and outputs for approve with ID %s (%d vs %d)", txID, outputSum, inputSum)}
	}
	return nil
}

func (v *Verifier) commitApproveAction(approveAction *token.PlainApprove, txID string, simulator ledger.LedgerWriter) error {
	if approveAction.GetOutput() != nil {
		outputID, err := createOutputKey(txID, 0)
		if err != nil {
			return err
		}
		err = v.addOutput(outputID, approveAction.GetOutput(), simulator)
		if err != nil {
			return err
		}
	}
	for i, delegatedOutput := range approveAction.GetDelegatedOutputs() {
//CreateDelegatedOutputKey（）错误已签入checkDelegatedOutputsdoes不存在
		outputID, _ := createDelegatedOutputKey(txID, i)
		err := v.addDelegatedOutput(outputID, delegatedOutput, simulator)
		if err != nil {
			return err
		}
	}
	return v.markInputsSpent(txID, approveAction.GetInputs(), simulator)
}

func (v *Verifier) checkApproveOutputs(creator identity.PublicInfo, output *token.PlainOutput, delegatedOutputs []*token.PlainDelegatedOutput, txID string, simulator ledger.LedgerReader) (string, uint64, error) {
	tokenType := ""
	tokenSum := uint64(0)

//在Approve Tx中输出是可选的
//检查此Tx是否包含输出
	if output != nil {
//检查所有者不是空切片
		if !bytes.Equal(output.Owner, creator.Public()) {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("the owner of the output is not valid")}
		}
		tokenType = output.GetType()
		err := v.checkOutputDoesNotExist(0, txID, simulator)
		if err != nil {
			return "", 0, err
		}
//检查是否可以为将来的输出创建已用密钥
		spentKey, err := createSpentKey(txID, 0)
		if err != nil {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("the output for approve txID '%s' is invalid: cannot create a spent key", txID)}
		}
		spent, err := v.isSpent(spentKey, simulator)
		if err != nil {
			return "", 0, errors.New(fmt.Sprintf("the output for approve txID '%s' is invalid: cannot check spent status", txID))
		}
		if spent {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("the output for approve txID '%s' is invalid: cannot create a spent key", txID)}
		}
		tokenSum = output.GetQuantity()
	}

//检查委托输出的一致性
	for i, delegatedOutput := range delegatedOutputs {
//检查委托输出是否将创建者作为所有者之一
		if !bytes.Equal(delegatedOutput.Owner, creator.Public()) {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("the owner of the delegated output is invalid")}
		}
//检查类型的一致性
		if tokenType == "" {
			tokenType = delegatedOutput.GetType()
		} else if tokenType != delegatedOutput.GetType() {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("multiple token types ('%s', '%s') in approve outputs for txID '%s'", tokenType, delegatedOutput.GetType(), txID)}
		}
//每个委托输出应该有一个委托者
		if len(delegatedOutput.Delegatees) != 1 {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("the number of delegates in approve txID '%s' is not 1, it is [%d]", txID, len(delegatedOutput.Delegatees))}
		}
//检查委托人不是空切片
		if len(delegatedOutput.Delegatees[0]) == 0 {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("the delegated output for approve txID '%s' does not have a delegatee", txID)}
		}

		err := v.checkDelegatedOutputDoesNotExist(i, txID, simulator)
		if err != nil {
			return "", 0, errors.New(fmt.Sprintf("the delegated output for approve txID '%s' is invalid: the ID exists", txID))
		}
//检查将来是否可以为委托输出创建已用密钥
		spentKey, err := createSpentDelegatedOutputKey(txID, int(i))
		if err != nil {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("the delegated output for approve txID '%s' is invalid: cannot create a spent key", txID)}
		}
		spent, err := v.isSpent(spentKey, simulator)
		if err != nil {
			return "", 0, errors.New(fmt.Sprintf("the delegated output for approve txID '%s' is invalid: cannot check spent status", txID))
		}
		if spent {
			return "", 0, &customtx.InvalidTxError{Msg: fmt.Sprintf("the delegated output for approve txID '%s' is invalid: cannot create a spent key", txID)}
		}
		tokenSum += delegatedOutput.GetQuantity()
	}

	return tokenType, tokenSum, nil
}

func (v *Verifier) addOutput(outputID string, output *token.PlainOutput, simulator ledger.LedgerWriter) error {
	outputBytes := utils.MarshalOrPanic(output)

	return simulator.SetState(tokenNameSpace, outputID, outputBytes)
}

func (v *Verifier) addDelegatedOutput(outputID string, delegatedOutput *token.PlainDelegatedOutput, simulator ledger.LedgerWriter) error {
	outputBytes := utils.MarshalOrPanic(delegatedOutput)

	return simulator.SetState(tokenNameSpace, outputID, outputBytes)
}

func (v *Verifier) addTransaction(txID string, ttx *token.TokenTransaction, simulator ledger.LedgerWriter) error {
	ttxBytes := utils.MarshalOrPanic(ttx)

	ttxID, err := createTxKey(txID)
	if err != nil {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("error creating txID: %s", err)}
	}

	return simulator.SetState(tokenNameSpace, ttxID, ttxBytes)
}

var TokenInputSpentMarker = []byte{1}

func (v *Verifier) markInputsSpent(txID string, inputs []*token.InputId, simulator ledger.LedgerWriter) error {
	for _, id := range inputs {
		inputID, err := createSpentKey(id.TxId, int(id.Index))
		if err != nil {
			return &customtx.InvalidTxError{Msg: fmt.Sprintf("error creating spent key: %s", err)}
		}
		verifierLogger.Debugf("marking input '%s' as spent", inputID)
		err = simulator.SetState(tokenNameSpace, inputID, TokenInputSpentMarker)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *Verifier) getOutput(outputID string, simulator ledger.LedgerReader) (*token.PlainOutput, error) {
	outputBytes, err := simulator.GetState(tokenNameSpace, outputID)
	if err != nil {
		return nil, err
	}
	if outputBytes == nil {
		return nil, &customtx.InvalidTxError{Msg: fmt.Sprintf("input with ID %s for transfer does not exist", outputID)}
	}
	if len(outputBytes) == 0 {
		return nil, &customtx.InvalidTxError{Msg: fmt.Sprintf("input with ID %s for transfer does not exist", outputID)}
	}
	output := &token.PlainOutput{}
	err = proto.Unmarshal(outputBytes, output)
	if err != nil {
		return nil, &customtx.InvalidTxError{Msg: fmt.Sprintf("unmarshaling error: %s", err)}
	}
	return output, nil
}

//isspent检查是否已使用标识符为outputid的输出令牌。
func (v *Verifier) isSpent(spentKey string, simulator ledger.LedgerReader) (bool, error) {
	verifierLogger.Debugf("checking if input with ID '%s' has been spent", spentKey)
	result, err := simulator.GetState(tokenNameSpace, spentKey)
	return result != nil, err
}

//为令牌事务中的单个输出创建分类帐键，作为
//事务ID和输出的索引
func createOutputKey(txID string, index int) (string, error) {
	return createCompositeKey(tokenOutput, []string{txID, strconv.Itoa(index)})
}

//为令牌事务中的兑现输出创建分类键，作为
//事务ID和输出的索引
func createRedeemKey(txID string, index int) (string, error) {
	return createCompositeKey(tokenRedeem, []string{txID, strconv.Itoa(index)})
}

//为令牌交易创建分类帐密钥，作为交易ID的函数
func createTxKey(txID string) (string, error) {
	return createCompositeKey(tokenTx, []string{txID})
}

//为令牌事务中已用的单个输出创建分类键，作为
//事务ID和输出的索引
func createSpentKey(txID string, index int) (string, error) {
	return createCompositeKey("tokenInput", []string{txID, strconv.Itoa(index)})
}

//从core/chaincode/shim/chaincode.go复制的createCompositeKey及其相关函数和常量。
func createCompositeKey(objectType string, attributes []string) (string, error) {
	if err := validateCompositeKeyAttribute(objectType); err != nil {
		return "", err
	}
	ck := compositeKeyNamespace + objectType + string(minUnicodeRuneValue)
	for _, att := range attributes {
		if err := validateCompositeKeyAttribute(att); err != nil {
			return "", err
		}
		ck += att + string(minUnicodeRuneValue)
	}
	return ck, nil
}

func validateCompositeKeyAttribute(str string) error {
	if !utf8.ValidString(str) {
		return errors.Errorf("not a valid utf8 string: [%x]", str)
	}
	for index, runeValue := range str {
		if runeValue == minUnicodeRuneValue || runeValue == maxUnicodeRuneValue {
			return errors.Errorf(`input contain unicode %#U starting at position [%d]. %#U and %#U are not allowed in the input attribute of a composite key`,
				runeValue, index, minUnicodeRuneValue, maxUnicodeRuneValue)
		}
	}
	return nil
}

func parseCompositeKeyBytes(keyBytes []byte) string {
	return string(keyBytes)
}

func getCompositeKeyBytes(compositeKey string) []byte {
	return []byte(compositeKey)
}

//为令牌事务中的单个委托输出创建分类帐密钥，作为
//事务ID和输出的索引
func createDelegatedOutputKey(txID string, index int) (string, error) {
	return createCompositeKey(tokenDelegatedOutput, []string{txID, strconv.Itoa(index)})
}

//为令牌事务中已用的单个委托输出创建分类帐密钥，作为
//事务ID和委托输出的索引
func createSpentDelegatedOutputKey(txID string, index int) (string, error) {
	return createCompositeKey(tokenDelegatedInput, []string{txID, strconv.Itoa(index)})
}

func (v *Verifier) checkDelegatedOutputDoesNotExist(index int, txID string, simulator ledger.LedgerReader) error {
	outputID, err := createDelegatedOutputKey(txID, index)
	if err != nil {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("error creating output ID: %s", err)}
	}

	existingOutputBytes, err := simulator.GetState(tokenNameSpace, outputID)
	if err != nil {
		return err
	}

	if existingOutputBytes != nil {
		return &customtx.InvalidTxError{Msg: fmt.Sprintf("output already exists: %s", outputID)}
	}
	return nil
}
