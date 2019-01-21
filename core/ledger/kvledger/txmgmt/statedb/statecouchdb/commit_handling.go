
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

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/pkg/errors"
)

//nscommittersbuilder实现了“batch”接口。每个批处理在更新和
//生成一个或多个SubnsCommitter类型的批。
type nsCommittersBuilder struct {
	updates         map[string]*statedb.VersionedValue
	db              *couchdb.CouchDatabase
	revisions       map[string]string
	subNsCommitters []batch
}

//subnscommitter实现了“batch”接口。每个批提交分配给它的命名空间中的更新部分
type subNsCommitter struct {
	db             *couchdb.CouchDatabase
	batchUpdateMap map[string]*batchableDocument
}

//BuildCommitters生成SubnsCommitter类型的批。此函数并行处理不同的命名空间
func (vdb *VersionedDB) buildCommitters(updates *statedb.UpdateBatch) ([]batch, error) {
	namespaces := updates.GetUpdatedNamespaces()
	var nsCommitterBuilder []batch
	for _, ns := range namespaces {
		nsUpdates := updates.GetUpdates(ns)
		db, err := vdb.getNamespaceDBHandle(ns)
		if err != nil {
			return nil, err
		}
		nsRevs := vdb.committedDataCache.revs[ns]
		if nsRevs == nil {
			nsRevs = make(nsRevisions)
		}
//对于每个名称空间，使用相应的couchdb句柄和coach修订构造一个生成器。
//已经加载到缓存中的（在验证阶段）
		nsCommitterBuilder = append(nsCommitterBuilder, &nsCommittersBuilder{updates: nsUpdates, db: db, revisions: nsRevs})
	}
	if err := executeBatches(nsCommitterBuilder); err != nil {
		return nil, err
	}
//跨命名空间累积结果（每个生成器中的一个命名空间的一个或多个“subnscommitter”批）
	var combinedSubNsCommitters []batch
	for _, b := range nsCommitterBuilder {
		combinedSubNsCommitters = append(combinedSubNsCommitters, b.(*nsCommittersBuilder).subNsCommitters...)
	}
	return combinedSubNsCommitters, nil
}

//execute在“batch”接口中实现函数。此函数生成一个或多个“subnscommitter”
//覆盖命名空间的更新
func (builder *nsCommittersBuilder) execute() error {
	if err := addRevisionsForMissingKeys(builder.revisions, builder.db, builder.updates); err != nil {
		return err
	}
	maxBacthSize := ledgerconfig.GetMaxBatchUpdateSize()
	batchUpdateMap := make(map[string]*batchableDocument)
	for key, vv := range builder.updates {
		couchDoc, err := keyValToCouchDoc(&keyValue{key: key, VersionedValue: vv}, builder.revisions[key])
		if err != nil {
			return err
		}
		batchUpdateMap[key] = &batchableDocument{CouchDoc: *couchDoc, Deleted: vv.Value == nil}
		if len(batchUpdateMap) == maxBacthSize {
			builder.subNsCommitters = append(builder.subNsCommitters, &subNsCommitter{builder.db, batchUpdateMap})
			batchUpdateMap = make(map[string]*batchableDocument)
		}
	}
	if len(batchUpdateMap) > 0 {
		builder.subNsCommitters = append(builder.subNsCommitters, &subNsCommitter{builder.db, batchUpdateMap})
	}
	return nil
}

//execute在“batch”接口中实现函数。此函数提交由“subnscommitter”管理的更新
func (committer *subNsCommitter) execute() error {
	return commitUpdates(committer.db, committer.batchUpdateMap)
}

//commitupdates将给定的更新提交到couchdb
func commitUpdates(db *couchdb.CouchDatabase, batchUpdateMap map[string]*batchableDocument) error {
//将文档添加到批更新数组
	batchUpdateDocs := []*couchdb.CouchDoc{}
	for _, updateDocument := range batchUpdateMap {
		batchUpdateDocument := updateDocument
		batchUpdateDocs = append(batchUpdateDocs, &batchUpdateDocument.CouchDoc)
	}

//对couchdb进行批量更新。请注意，如果整个批量更新失败或超时，将重试。
	batchUpdateResp, err := db.BatchUpdateDocuments(batchUpdateDocs)
	if err != nil {
		return err
	}
//如果批量更新中的单个文档没有成功，请分别尝试它们。
//按文档遍历couchdb的响应
	for _, respDoc := range batchUpdateResp {
//如果文档返回错误，请重试单个文档
		if respDoc.Ok != true {
			batchUpdateDocument := batchUpdateMap[respDoc.ID]
			var err error
//在保存之前从JSON中删除“_rev”
//这将允许CouchDB重试逻辑在不遇到修改的情况下重试修订。
//“if-match”和JSON中的“rev”标记不匹配
			if batchUpdateDocument.CouchDoc.JSONValue != nil {
				err = removeJSONRevision(&batchUpdateDocument.CouchDoc.JSONValue)
				if err != nil {
					return err
				}
			}
//检查文档是否作为删除类型文档添加到批处理中
			if batchUpdateDocument.Deleted {
				logger.Warningf("CouchDB batch document delete encountered an problem. Retrying delete for document ID:%s", respDoc.ID)
//如果这是一个已删除的文档，请重试删除
//如果由于找不到文档（404错误）导致删除失败，
//文档已被删除，并且DeleteDoc不会返回错误
				err = db.DeleteDoc(respDoc.ID, "")
			} else {
				logger.Warningf("CouchDB batch document update encountered an problem. Retrying update for document ID:%s", respDoc.ID)
//将单个文档保存到couchdb
//请注意，这将根据需要重试
				_, err = db.SaveDoc(respDoc.ID, "", &batchUpdateDocument.CouchDoc)
			}

//如果单个文档更新或删除返回错误，则引发错误
			if err != nil {
				errorString := fmt.Sprintf("error saving document ID: %v. Error: %s,  Reason: %s",
					respDoc.ID, respDoc.Error, respDoc.Reason)

				logger.Errorf(errorString)
				return errors.WithMessage(err, errorString)
			}
		}
	}
	return nil
}

//nsflusher实现了“batch”接口，一个批处理为给定的命名空间执行函数“couchdb.ensurerefullcommit（）”
type nsFlusher struct {
	db *couchdb.CouchDatabase
}

func (vdb *VersionedDB) ensureFullCommit(dbs []*couchdb.CouchDatabase) error {
	var flushers []batch
	for _, db := range dbs {
		flushers = append(flushers, &nsFlusher{db})
	}
	return executeBatches(flushers)
}

func (f *nsFlusher) execute() error {
	dbResponse, err := f.db.EnsureFullCommit()
	if err != nil || dbResponse.Ok != true {
		logger.Errorf("Failed to perform full commit")
		return errors.New("failed to perform full commit")
	}
	return nil
}

func addRevisionsForMissingKeys(revisions map[string]string, db *couchdb.CouchDatabase, nsUpdates map[string]*statedb.VersionedValue) error {
	var missingKeys []string
	for key := range nsUpdates {
		_, ok := revisions[key]
		if !ok {
			missingKeys = append(missingKeys, key)
		}
	}
	logger.Debugf("Pulling revisions for the [%d] keys for namsespace [%s] that were not part of the readset", len(missingKeys), db.DBName)
	retrievedMetadata, err := retrieveNsMetadata(db, missingKeys)
	if err != nil {
		return err
	}
	for _, metadata := range retrievedMetadata {
		revisions[metadata.ID] = metadata.Rev
	}
	return nil
}

//batchabledocument为批定义文档
type batchableDocument struct {
	CouchDoc couchdb.CouchDoc
	Deleted  bool
}
