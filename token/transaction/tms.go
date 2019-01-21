
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
	"github.com/hyperledger/fabric/protos/token"
	"github.com/hyperledger/fabric/token/identity"
	"github.com/hyperledger/fabric/token/ledger"
)

//go：生成仿冒者-o mock/tms_tx_processor.go-forke name tms tx processor。TMSTX处理器
//go：生成伪造者-o mock/tms_manager.go-forke name tms manager。TMS-管理器

//tmstxprocessor用于生成令牌事务的读取依赖项
//（读取集）以及该事务触发的分类帐更新
//（写入集）；读写集通过模拟器对象隐式返回
//在提交函数中作为参数传递的
type TMSTxProcessor interface {
//processtx解析ttx以生成rw集
	ProcessTx(txID string, creator identity.PublicInfo, ttx *token.TokenTransaction, simulator ledger.LedgerWriter) error
}

type TMSManager interface {
//gettxprocessor返回所提供通道的TMS事务的TxProcessor
	GetTxProcessor(channel string) (TMSTxProcessor, error)
}

type TxCreatorInfo struct {
	public []byte
}

func (t *TxCreatorInfo) Public() []byte {
	return t.public
}
