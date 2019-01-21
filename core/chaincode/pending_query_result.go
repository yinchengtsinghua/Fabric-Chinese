
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


package chaincode

import (
	"github.com/golang/protobuf/proto"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type PendingQueryResult struct {
	batch []*pb.QueryResultBytes
}

func (p *PendingQueryResult) Cut() []*pb.QueryResultBytes {
	batch := p.batch
	p.batch = nil
	return batch
}

func (p *PendingQueryResult) Add(queryResult commonledger.QueryResult) error {
	queryResultBytes, err := proto.Marshal(queryResult.(proto.Message))
	if err != nil {
		chaincodeLogger.Errorf("failed to marshal query result: %s", err)
		return err
	}
	p.batch = append(p.batch, &pb.QueryResultBytes{ResultBytes: queryResultBytes})
	return nil
}

func (p *PendingQueryResult) Size() int {
	return len(p.batch)
}
