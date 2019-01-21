
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


package multichannel

import (
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/orderer/common/blockcutter"
	"github.com/hyperledger/fabric/orderer/common/msgprocessor"
	"github.com/hyperledger/fabric/orderer/consensus"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//ChainSupport保存特定通道的资源。
type ChainSupport struct {
	*ledgerResources
	msgprocessor.Processor
	*BlockWriter
	consensus.Chain
	cutter blockcutter.Receiver
	crypto.LocalSigner
}

func newChainSupport(
	registrar *Registrar,
	ledgerResources *ledgerResources,
	consenters map[string]consensus.Consenter,
	signer crypto.LocalSigner,
	blockcutterMetrics *blockcutter.Metrics,
) *ChainSupport {
//读取通道的最后一个块和元数据
	lastBlock := blockledger.GetBlock(ledgerResources, ledgerResources.Height()-1)

	metadata, err := utils.GetMetadataFromBlock(lastBlock, cb.BlockMetadataIndex_ORDERER)
//假设使用cb.newblock（）创建了一个块，则不应
//即使排序器元数据是空字节片，也会出错
	if err != nil {
		logger.Fatalf("[channel: %s] Error extracting orderer metadata: %s", ledgerResources.ConfigtxValidator().ChainID(), err)
	}

//构造作为附加支持参数所需的有限支持
	cs := &ChainSupport{
		ledgerResources: ledgerResources,
		LocalSigner:     signer,
		cutter: blockcutter.NewReceiverImpl(
			ledgerResources.ConfigtxValidator().ChainID(),
			ledgerResources,
			blockcutterMetrics,
		),
	}

//设置msgprocessor
	cs.Processor = msgprocessor.NewStandardChannel(cs, msgprocessor.CreateStandardChannelFilters(cs))

//设置块编写器
	cs.BlockWriter = newBlockWriter(lastBlock, registrar, cs)

//设置同意人
	consenterType := ledgerResources.SharedConfig().ConsensusType()
	consenter, ok := consenters[consenterType]
	if !ok {
		logger.Panicf("Error retrieving consenter of type: %s", consenterType)
	}

	cs.Chain, err = consenter.HandleChain(cs, metadata)
	if err != nil {
		logger.Panicf("[channel: %s] Error creating consenter: %s", cs.ChainID(), err)
	}

	logger.Debugf("[channel: %s] Done creating channel support resources", cs.ChainID())

	return cs
}

//block返回具有以下数字的块，
//如果不存在这样的块，则为零。
func (cs *ChainSupport) Block(number uint64) *cb.Block {
	if cs.Height() <= number {
		return nil
	}
	return blockledger.GetBlock(cs.Reader(), number)
}

func (cs *ChainSupport) Reader() blockledger.Reader {
	return cs
}

//signer返回此通道的crypto.localsigner。
func (cs *ChainSupport) Signer() crypto.LocalSigner {
	return cs
}

func (cs *ChainSupport) start() {
	cs.Chain.Start()
}

//BlockCutter返回此通道的BlockCutter.Receiver实例。
func (cs *ChainSupport) BlockCutter() blockcutter.Receiver {
	return cs.cutter
}

//验证传递到基础configtx.validator
func (cs *ChainSupport) Validate(configEnv *cb.ConfigEnvelope) error {
	return cs.ConfigtxValidator().Validate(configEnv)
}

//ProposeConfigUpdate传递到基础configtx.validator
func (cs *ChainSupport) ProposeConfigUpdate(configtx *cb.Envelope) (*cb.ConfigEnvelope, error) {
	env, err := cs.ConfigtxValidator().ProposeConfigUpdate(configtx)
	if err != nil {
		return nil, err
	}

	bundle, err := cs.CreateBundle(cs.ChainID(), env.Config)
	if err != nil {
		return nil, err
	}

	if err = checkResources(bundle); err != nil {
		return nil, errors.Wrap(err, "config update is not compatible")
	}

	return env, cs.ValidateNew(bundle)
}

//chainID传递到基础configtx.validator
func (cs *ChainSupport) ChainID() string {
	return cs.ConfigtxValidator().ChainID()
}

//configproto传递到基础configtx.validator
func (cs *ChainSupport) ConfigProto() *cb.Config {
	return cs.ConfigtxValidator().ConfigProto()
}

//序列传递给基础configtx.validator
func (cs *ChainSupport) Sequence() uint64 {
	return cs.ConfigtxValidator().Sequence()
}

//verifyblocksignature验证块的签名。
//它有一个配置信封的可选参数
//这将使块验证使用验证规则
//基于配置开发中的给定配置。
//如果传递的配置信封为零，则使用验证规则
//是在提交前一个块时应用的。
func (cs *ChainSupport) VerifyBlockSignature(sd []*cb.SignedData, envelope *cb.ConfigEnvelope) error {
	policyMgr := cs.PolicyManager()
//如果传递的信封不是空的，我们应该使用另一个策略管理器。
	if envelope != nil {
		bundle, err := channelconfig.NewBundle(cs.ChainID(), envelope.Config)
		if err != nil {
			return err
		}
		policyMgr = bundle.PolicyManager()
	}
	policy, exists := policyMgr.GetPolicy(policies.BlockValidation)
	if !exists {
		return errors.Errorf("policy %s wasn't found", policies.BlockValidation)
	}
	err := policy.Evaluate(sd)
	if err != nil {
		return errors.Wrap(err, "block verification failed")
	}
	return nil
}
