
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


package tests

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/stretchr/testify/assert"
)

//客户端帮助进行转换模拟。客户机不断地累积每个模拟事务的结果。
//在切片和后期，可以用于剪切测试块进行提交。
//在测试中，对于每个实例化的分类账，客户机的单个实例通常就足够了。
type client struct {
	lgr            ledger.PeerLedger
simulatedTrans []*txAndPvtdata //累积事务模拟的结果
	assert         *assert.Assertions
}

func newClient(lgr ledger.PeerLedger, t *testing.T) *client {
	return &client{lgr, nil, assert.New(t)}
}

//Simulatedatatx采用模拟逻辑并将其包装在
//（a）预模拟任务（如获得新的模拟器）和
//（b）模拟后的任务（如收集（公共和pvt）模拟结果和构建事务）
//因为（a）和（b）都在这个函数中处理，所以只要提供模拟逻辑，测试代码就可以保持简单。
func (c *client) simulateDataTx(txid string, simulationLogic func(s *simulator)) *txAndPvtdata {
	if txid == "" {
		txid = util.GenerateUUID()
	}
	ledgerSimulator, err := c.lgr.NewTxSimulator(txid)
	c.assert.NoError(err)
	sim := &simulator{ledgerSimulator, txid, c.assert}
	simulationLogic(sim)
	txAndPvtdata := sim.done()
	c.simulatedTrans = append(c.simulatedTrans, txAndPvtdata)
	return txAndPvtdata
}

//SimulatedDeployTX模拟部署链代码的转换。这反过来调用函数“Simulatedatatx”
//为分类帐测试提供模拟“lscc”的inoke函数的模拟逻辑
func (c *client) simulateDeployTx(ccName string, collConfs []*collConf) *txAndPvtdata {
	ccData := &ccprovider.ChaincodeData{Name: ccName}
	ccDataBytes, err := proto.Marshal(ccData)
	c.assert.NoError(err)

	psudoLSCCInvokeFunc := func(s *simulator) {
		s.setState("lscc", ccName, string(ccDataBytes))
		if collConfs != nil {
			protoBytes, err := convertToCollConfigProtoBytes(collConfs)
			c.assert.NoError(err)
			s.setState("lscc", privdata.BuildCollectionKVSKey(ccName), string(protoBytes))
		}
	}
	return c.simulateDataTx("", psudoLSCCInvokeFunc)
}

//SimulateUpgradeTx参见功能“SimulatedDeployTx”的注释
func (c *client) simulateUpgradeTx(ccName string, collConfs []*collConf) *txAndPvtdata {
	return c.simulateDeployTx(ccName, collConfs)
}

///////////////////simpulator wrapper函数
type simulator struct {
	ledger.TxSimulator
	txid   string
	assert *assert.Assertions
}

func (s *simulator) getState(ns, key string) string {
	val, err := s.GetState(ns, key)
	s.assert.NoError(err)
	return string(val)
}

func (s *simulator) setState(ns, key string, val string) {
	s.assert.NoError(
		s.SetState(ns, key, []byte(val)),
	)
}

func (s *simulator) delState(ns, key string) {
	s.assert.NoError(
		s.DeleteState(ns, key),
	)
}

func (s *simulator) getPvtdata(ns, coll, key string) {
	_, err := s.GetPrivateData(ns, coll, key)
	s.assert.NoError(err)
}

func (s *simulator) setPvtdata(ns, coll, key string, val string) {
	s.assert.NoError(
		s.SetPrivateData(ns, coll, key, []byte(val)),
	)
}

func (s *simulator) delPvtdata(ns, coll, key string) {
	s.assert.NoError(
		s.DeletePrivateData(ns, coll, key),
	)
}

func (s *simulator) done() *txAndPvtdata {
	s.Done()
	simRes, err := s.GetTxSimulationResults()
	s.assert.NoError(err)
	pubRwsetBytes, err := simRes.GetPubSimulationBytes()
	s.assert.NoError(err)
	envelope, err := constructTransaction(s.txid, pubRwsetBytes)
	s.assert.NoError(err)
	txAndPvtdata := &txAndPvtdata{Txid: s.txid, Envelope: envelope, Pvtws: simRes.PvtSimulationResults}
	return txAndPvtdata
}
