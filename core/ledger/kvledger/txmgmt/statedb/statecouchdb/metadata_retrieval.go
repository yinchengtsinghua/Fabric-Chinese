
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


package statecouchdb

import (
	"fmt"

	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
)

//nsmetadataretriever实现“batch”接口并包装函数“retrievensmetadata”
//允许对不同命名空间并行执行此函数
type nsMetadataRetriever struct {
	ns              string
	db              *couchdb.CouchDatabase
	keys            []string
	executionResult []*couchdb.DocMetadata
}

//subnsmetadataretriever实现“batch”接口并包装函数“couchdb.batchretrievedocumentmetadata”
//允许对命名空间中的不同键集并行执行此函数。
//根据配置'ledgerconfig.getmaxbatchupdatesize（）'执行不同的键集以创建
type subNsMetadataRetriever nsMetadataRetriever

//retrievedmetadata重试“namespace keys”组合集合的元数据
func (vdb *VersionedDB) retrieveMetadata(nsKeysMap map[string][]string) (map[string][]*couchdb.DocMetadata, error) {
//为每个命名空间构建一个批处理
	nsMetadataRetrievers := []batch{}
	for ns, keys := range nsKeysMap {
		db, err := vdb.getNamespaceDBHandle(ns)
		if err != nil {
			return nil, err
		}
		nsMetadataRetrievers = append(nsMetadataRetrievers, &nsMetadataRetriever{ns: ns, db: db, keys: keys})
	}
	if err := executeBatches(nsMetadataRetrievers); err != nil {
		return nil, err
	}
//从每批中累积结果
	executionResults := make(map[string][]*couchdb.DocMetadata)
	for _, r := range nsMetadataRetrievers {
		nsMetadataRetriever := r.(*nsMetadataRetriever)
		executionResults[nsMetadataRetriever.ns] = nsMetadataRetriever.executionResult
	}
	return executionResults, nil
}

//检索子元数据检索给定命名空间的元数据。
func retrieveNsMetadata(db *couchdb.CouchDatabase, keys []string) ([]*couchdb.DocMetadata, error) {
//根据MaxBachSize为每组密钥构建一个批
	maxBacthSize := ledgerconfig.GetMaxBatchUpdateSize()
	batches := []batch{}
	remainingKeys := keys
	for {
		numKeys := minimum(maxBacthSize, len(remainingKeys))
		if numKeys == 0 {
			break
		}
		batch := &subNsMetadataRetriever{db: db, keys: remainingKeys[:numKeys]}
		batches = append(batches, batch)
		remainingKeys = remainingKeys[numKeys:]
	}
	if err := executeBatches(batches); err != nil {
		return nil, err
	}
//从每批中累积结果
	var executionResults []*couchdb.DocMetadata
	for _, b := range batches {
		executionResults = append(executionResults, b.(*subNsMetadataRetriever).executionResult...)
	}
	return executionResults, nil
}

func (r *nsMetadataRetriever) execute() error {
	var err error
	if r.executionResult, err = retrieveNsMetadata(r.db, r.keys); err != nil {
		return err
	}
	return nil
}

func (r *nsMetadataRetriever) String() string {
	return fmt.Sprintf("nsMetadataRetriever:ns=%s, num keys=%d", r.ns, len(r.keys))
}

func (b *subNsMetadataRetriever) execute() error {
	var err error
	if b.executionResult, err = b.db.BatchRetrieveDocumentMetadata(b.keys); err != nil {
		return err
	}
	return nil
}

func (b *subNsMetadataRetriever) String() string {
	return fmt.Sprintf("subNsMetadataRetriever:ns=%s, num keys=%d", b.ns, len(b.keys))
}

func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}
