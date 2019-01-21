
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2018保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package endorser

import (
	"github.com/hyperledger/fabric/core/handlers/endorsement/api/state"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/transientstore"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/pkg/errors"
)

//去：生成mokery-dir。-name querycreator-case underline-输出模拟/
//QueryCreator创建新的QueryExecutor
type QueryCreator interface {
	NewQueryExecutor() (ledger.QueryExecutor, error)
}

//channelstate定义状态操作
type ChannelState struct {
	transientstore.Store
	QueryCreator
}

//FetchState获取状态
func (cs *ChannelState) FetchState() (endorsement.State, error) {
	qe, err := cs.NewQueryExecutor()
	if err != nil {
		return nil, err
	}

	return &StateContext{
		QueryExecutor: qe,
		Store:         cs.Store,
	}, nil
}

//StateContext定义与状态交互的执行上下文
type StateContext struct {
	transientstore.Store
	ledger.QueryExecutor
}

//GetTransientByXid返回与此事务ID关联的私有数据。
func (sc *StateContext) GetTransientByTXID(txID string) ([]*rwset.TxPvtReadWriteSet, error) {
	scanner, err := sc.Store.GetTxPvtRWSetByTxid(txID, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer scanner.Close()
	var data []*rwset.TxPvtReadWriteSet
	for {
		res, err := scanner.NextWithConfig()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if res == nil {
			break
		}
		if res.PvtSimulationResultsWithConfig == nil {
			continue
		}
		data = append(data, res.PvtSimulationResultsWithConfig.PvtRwset)
	}
	return data, nil
}
