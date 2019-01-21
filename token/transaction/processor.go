
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


package transaction

import (
	"fmt"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("fabtoken-processor")

//处理器实现接口“github.com/hyperledger/fabric/core/ledger/customtx/processor”
//用于FabToken交易
type Processor struct {
	TMSManager TMSManager
}

func (p *Processor) GenerateSimulationResults(txEnv *common.Envelope, simulator ledger.TxSimulator, initializingLedger bool) error {
//提取通道头和令牌事务
	ch, ttx, ci, err := UnmarshalTokenTransaction(txEnv.Payload)
	if err != nil {
		return errors.WithMessage(err, "failed unmarshalling token transaction")
	}

//获取对应于通道的tmstxprocessor
	txProcessor, err := p.TMSManager.GetTxProcessor(ch.ChannelId)
	if err != nil {
		return errors.WithMessage(err, "failed getting committer")
	}

//使用模拟器提取与事务相关联的读取依赖项和分类帐更新
	err = txProcessor.ProcessTx(ch.TxId, ci, ttx, simulator)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("failed committing transaction for channel %s", ch.ChannelId))
	}

	return err
}
