
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

type QueryResponseGenerator struct {
	MaxResultLimit int
}

//BuildQueryResponse采用迭代器和获取状态来构造QueryResponse
func (q *QueryResponseGenerator) BuildQueryResponse(txContext *TransactionContext, iter commonledger.ResultsIterator,
	iterID string, isPaginated bool, totalReturnLimit int32) (*pb.QueryResponse, error) {

	pendingQueryResults := txContext.GetPendingQueryResult(iterID)
	totalReturnCount := txContext.GetTotalReturnCount(iterID)

	for {
//如果已达到总计数，则返回结果并阻止调用Next（）。
		if *totalReturnCount >= totalReturnLimit {
			return createQueryResponse(txContext, iterID, isPaginated, pendingQueryResults, *totalReturnCount)
		}

		queryResult, err := iter.Next()
		switch {
		case err != nil:
			chaincodeLogger.Errorf("Failed to get query result from iterator")
			txContext.CleanupQueryContext(iterID)
			return nil, err

		case queryResult == nil:

			return createQueryResponse(txContext, iterID, isPaginated, pendingQueryResults, *totalReturnCount)

		case !isPaginated && pendingQueryResults.Size() == q.MaxResultLimit:
//如果不使用显式分页
//如果最大结果数已排队，则剪切批处理，然后将当前结果添加到挂起的批处理中。
//MaxResultLimit用于在链码填充程序和处理程序之间进行批处理
//MaxResultLimit不限制返回到客户端的记录
			batch := pendingQueryResults.Cut()
			if err := pendingQueryResults.Add(queryResult); err != nil {
				txContext.CleanupQueryContext(iterID)
				return nil, err
			}
			*totalReturnCount++
			return &pb.QueryResponse{Results: batch, HasMore: true, Id: iterID}, nil

		default:
			if err := pendingQueryResults.Add(queryResult); err != nil {
				txContext.CleanupQueryContext(iterID)
				return nil, err
			}
			*totalReturnCount++
		}
	}
}

func createQueryResponse(txContext *TransactionContext, iterID string, isPaginated bool, pendingQueryResults *PendingQueryResult, totalReturnCount int32) (*pb.QueryResponse, error) {

	batch := pendingQueryResults.Cut()

	if isPaginated {
//当启用显式分页时，返回带有responseMetadata的批
		bookmark := txContext.CleanupQueryContextWithBookmark(iterID)
		responseMetadata := createResponseMetadata(totalReturnCount, bookmark)
		responseMetadataBytes, err := proto.Marshal(responseMetadata)
		if err != nil {
			return nil, err
		}
		return &pb.QueryResponse{Results: batch, HasMore: false, Id: iterID, Metadata: responseMetadataBytes}, nil
	}

//if explicit pagination is not used, then the end of the resultset has been reached, return the batch
	txContext.CleanupQueryContext(iterID)
	return &pb.QueryResponse{Results: batch, HasMore: false, Id: iterID}, nil

}

func createResponseMetadata(returnCount int32, bookmark string) *pb.QueryResponseMetadata {
	responseMetadata := &pb.QueryResponseMetadata{}
	responseMetadata.Bookmark = bookmark
	responseMetadata.FetchedRecordsCount = int32(returnCount)
	return responseMetadata
}
