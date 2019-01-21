
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


package couchdb

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("couchdb")

//重试间隔时间（毫秒）
const retryWaitTime = 125

//DBOperationResponse是成功数据库调用的主体。
type DBOperationResponse struct {
	Ok  bool
	id  string
	rev string
}

//dbinfo是数据库信息的主体。
type DBInfo struct {
	DbName    string `json:"db_name"`
	UpdateSeq string `json:"update_seq"`
	Sizes     struct {
		File     int `json:"file"`
		External int `json:"external"`
		Active   int `json:"active"`
	} `json:"sizes"`
	PurgeSeq int `json:"purge_seq"`
	Other    struct {
		DataSize int `json:"data_size"`
	} `json:"other"`
	DocDelCount       int    `json:"doc_del_count"`
	DocCount          int    `json:"doc_count"`
	DiskSize          int    `json:"disk_size"`
	DiskFormatVersion int    `json:"disk_format_version"`
	DataSize          int    `json:"data_size"`
	CompactRunning    bool   `json:"compact_running"`
	InstanceStartTime string `json:"instance_start_time"`
}

//ConnectionInfo是用于捕获数据库信息和版本的结构
type ConnectionInfo struct {
	Couchdb string `json:"couchdb"`
	Version string `json:"version"`
	Vendor  struct {
		Name string `json:"name"`
	} `json:"vendor"`
}

//RangeQueryResponse is used for processing REST range query responses from CouchDB
type RangeQueryResponse struct {
	TotalRows int32 `json:"total_rows"`
	Offset    int32 `json:"offset"`
	Rows      []struct {
		ID    string `json:"id"`
		Key   string `json:"key"`
		Value struct {
			Rev string `json:"rev"`
		} `json:"value"`
		Doc json.RawMessage `json:"doc"`
	} `json:"rows"`
}

//QueryResponse用于处理来自CouchDB的REST查询响应
type QueryResponse struct {
	Warning  string            `json:"warning"`
	Docs     []json.RawMessage `json:"docs"`
	Bookmark string            `json:"bookmark"`
}

//docmetadata用于捕获couchdb文档头信息，
//用于捕获从CouchDB查询中返回的ID、版本、版本和附件
type DocMetadata struct {
	ID              string                     `json:"_id"`
	Rev             string                     `json:"_rev"`
	Version         string                     `json:"~version"`
	AttachmentsInfo map[string]*AttachmentInfo `json:"_attachments"`
}

//docid是从查询结果中捕获ID的最小结构
type DocID struct {
	ID string `json:"_id"`
}

//queryresult用于从couchdb返回查询结果
type QueryResult struct {
	ID          string
	Value       []byte
	Attachments []*AttachmentInfo
}

//couchconnectiondef包含参数
type CouchConnectionDef struct {
	URL                   string
	Username              string
	Password              string
	MaxRetries            int
	MaxRetriesOnStartup   int
	RequestTimeout        time.Duration
	CreateGlobalChangesDB bool
}

//CouchInstance表示CouchDB实例
type CouchInstance struct {
conf   CouchConnectionDef //连接配置
client *http.Client       //连接到此实例的客户端
	stats  *stats
}

//CouchDatabase represents a database within a CouchDB instance
type CouchDatabase struct {
CouchInstance    *CouchInstance //连接配置
	DBName           string
	IndexWarmCounter int
}

//dbreturn包含couchdb报告的错误
type DBReturn struct {
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
	Reason     string `json:"reason"`
}

//CreateIndexResponse contains an the index creation response from CouchDB
type CreateIndexResponse struct {
	Result string `json:"result"`
	ID     string `json:"id"`
	Name   string `json:"name"`
}

//AttachmentInfo包含CouchDB附加文件的定义
type AttachmentInfo struct {
	Name            string
	ContentType     string `json:"content_type"`
	Length          uint64
	AttachmentBytes []byte `json:"data"`
}

//filedetails定义将附件发送到couchdb所需的结构
type FileDetails struct {
	Follows     bool   `json:"follows"`
	ContentType string `json:"content_type"`
	Length      int    `json:"length"`
}

//CouchDoc定义JSON文档值的结构
type CouchDoc struct {
	JSONValue   []byte
	Attachments []*AttachmentInfo
}

//BatchRetrieveDocMetadataResponse用于处理来自CouchDB的REST批处理响应
type BatchRetrieveDocMetadataResponse struct {
	Rows []struct {
		ID          string `json:"id"`
		DocMetadata struct {
			ID      string `json:"_id"`
			Rev     string `json:"_rev"`
			Version string `json:"~version"`
		} `json:"doc"`
	} `json:"rows"`
}

//batchUpdateResponse为批更新响应定义结构
type BatchUpdateResponse struct {
	ID     string `json:"id"`
	Error  string `json:"error"`
	Reason string `json:"reason"`
	Ok     bool   `json:"ok"`
	Rev    string `json:"rev"`
}

//Base64Attachment contains the definition for an attached file for couchdb
type Base64Attachment struct {
	ContentType    string `json:"content_type"`
	AttachmentData string `json:"data"`
}

//indexresult包含couchdb索引的定义
type IndexResult struct {
	DesignDocument string `json:"designdoc"`
	Name           string `json:"name"`
	Definition     string `json:"definition"`
}

//database security包含couchdb数据库安全性的定义
type DatabaseSecurity struct {
	Admins struct {
		Names []string `json:"names"`
		Roles []string `json:"roles"`
	} `json:"admins"`
	Members struct {
		Names []string `json:"names"`
		Roles []string `json:"roles"`
	} `json:"members"`
}

//closeResponseBody discards the body and then closes it to enable returning it to
//连接池
func closeResponseBody(resp *http.Response) {
	if resp != nil {
io.Copy(ioutil.Discard, resp.Body) //丢弃身体的剩余部分
		resp.Body.Close()
	}
}

//新客户端连接的CreateConnectionDefinition
func CreateConnectionDefinition(couchDBAddress, username, password string, maxRetries,
	maxRetriesOnStartup int, requestTimeout time.Duration, createGlobalChangesDB bool) (*CouchConnectionDef, error) {

	logger.Debugf("Entering CreateConnectionDefinition()")

	connectURL := &url.URL{
		Host:   couchDBAddress,
		Scheme: "http",
	}

//分析构造的URL以验证没有错误
	finalURL, err := url.Parse(connectURL.String())
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, errors.Wrapf(err, "error parsing connect URL: %s", connectURL)
	}

	logger.Debugf("Created database configuration  URL=[%s]", finalURL.String())
	logger.Debugf("Exiting CreateConnectionDefinition()")

//返回包含连接信息的对象
	return &CouchConnectionDef{finalURL.String(), username, password, maxRetries,
		maxRetriesOnStartup, requestTimeout, createGlobalChangesDB}, nil

}

//CreateDatabaseIfNotexist方法提供创建数据库的函数
func (dbclient *CouchDatabase) CreateDatabaseIfNotExist() error {

	logger.Debugf("[%s] Entering CreateDatabaseIfNotExist()", dbclient.DBName)

	dbInfo, couchDBReturn, err := dbclient.GetDatabaseInfo()
	if err != nil {
		if couchDBReturn == nil || couchDBReturn.StatusCode != 404 {
			return err
		}
	}

//如果dbinfo返回populated且状态代码为200，则数据库存在
	if dbInfo != nil && couchDBReturn.StatusCode == 200 {

//如果需要，应用数据库安全性
		errSecurity := dbclient.applyDatabasePermissions()
		if errSecurity != nil {
			return errSecurity
		}

		logger.Debugf("[%s] Database already exists", dbclient.DBName)

		logger.Debugf("[%s] Exiting CreateDatabaseIfNotExist()", dbclient.DBName)

		return nil
	}

	logger.Debugf("[%s] Database does not exist.", dbclient.DBName)

	connectURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

//使用Put处理URL，创建数据库
	resp, _, err := dbclient.handleRequest(http.MethodPut, "CreateDatabaseIfNotExist", connectURL, nil, "", "", maxRetries, true, nil)

	if err != nil {

//检查数据库是否存在
//尽管handlerRequest（）返回了一个错误，但是
//数据库可能已创建并出现错误
//由于超时或争用条件而返回。
//最后检查一下数据库是否真的被创建了。
		dbInfo, couchDBReturn, errDbInfo := dbclient.GetDatabaseInfo()
//如果没有错误，则数据库存在，返回时不出错
		if errDbInfo == nil && dbInfo != nil && couchDBReturn.StatusCode == 200 {

			errSecurity := dbclient.applyDatabasePermissions()
			if errSecurity != nil {
				return errSecurity
			}

			logger.Infof("[%s] Created state database", dbclient.DBName)
			logger.Debugf("[%s] Exiting CreateDatabaseIfNotExist()", dbclient.DBName)
			return nil
		}

		return err

	}
	defer closeResponseBody(resp)

	errSecurity := dbclient.applyDatabasePermissions()
	if errSecurity != nil {
		return errSecurity
	}

	logger.Infof("Created state database %s", dbclient.DBName)

	logger.Debugf("[%s] Exiting CreateDatabaseIfNotExist()", dbclient.DBName)

	return nil

}

//应用数据库安全
func (dbclient *CouchDatabase) applyDatabasePermissions() error {

//If the username and password are not set, then skip applying permissions
	if dbclient.CouchInstance.conf.Username == "" && dbclient.CouchInstance.conf.Password == "" {
		return nil
	}

	securityPermissions := &DatabaseSecurity{}

	securityPermissions.Admins.Names = append(securityPermissions.Admins.Names, dbclient.CouchInstance.conf.Username)
	securityPermissions.Members.Names = append(securityPermissions.Members.Names, dbclient.CouchInstance.conf.Username)

	err := dbclient.ApplyDatabaseSecurity(securityPermissions)
	if err != nil {
		return err
	}

	return nil
}

//getdatabaseinfo方法提供检索数据库信息的函数
func (dbclient *CouchDatabase) GetDatabaseInfo() (*DBInfo, *DBReturn, error) {

	connectURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, nil, errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, couchDBReturn, err := dbclient.handleRequest(http.MethodGet, "GetDatabaseInfo", connectURL, nil, "", "", maxRetries, true, nil)
	if err != nil {
		return nil, couchDBReturn, err
	}
	defer closeResponseBody(resp)

	dbResponse := &DBInfo{}
	decodeErr := json.NewDecoder(resp.Body).Decode(&dbResponse)
	if decodeErr != nil {
		return nil, nil, errors.Wrap(decodeErr, "error decoding response body")
	}

//跟踪数据库信息响应
	logger.Debugw("GetDatabaseInfo()", "dbResponseJSON", dbResponse)

	return dbResponse, couchDBReturn, nil

}

//verifycouchconfig方法提供用于验证连接信息的函数
func (couchInstance *CouchInstance) VerifyCouchConfig() (*ConnectionInfo, *DBReturn, error) {

	logger.Debugf("Entering VerifyCouchConfig()")
	defer logger.Debugf("Exiting VerifyCouchConfig()")

	connectURL, err := url.Parse(couchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, nil, errors.Wrapf(err, "error parsing couch instance URL: %s", couchInstance.conf.URL)
	}
	connectURL.Path = "/"

//获取启动的重试次数
	maxRetriesOnStartup := couchInstance.conf.MaxRetriesOnStartup

	resp, couchDBReturn, err := couchInstance.handleRequest(context.Background(), http.MethodGet, "", "VerifyCouchConfig", connectURL, nil,
		couchInstance.conf.Username, couchInstance.conf.Password, maxRetriesOnStartup, true, nil)

	if err != nil {
		return nil, couchDBReturn, errors.WithMessage(err, "unable to connect to CouchDB, check the hostname and port")
	}
	defer closeResponseBody(resp)

	dbResponse := &ConnectionInfo{}
	decodeErr := json.NewDecoder(resp.Body).Decode(&dbResponse)
	if decodeErr != nil {
		return nil, nil, errors.Wrap(decodeErr, "error decoding response body")
	}

//跟踪数据库信息响应
	logger.Debugw("VerifyConnection() dbResponseJSON: %s", dbResponse)

//检查系统数据库是否存在
//验证系统数据库的存在完成两个步骤
//1。Ensures the system databases are created
//2。验证couchdb配置中提供的用户名密码是否对系统管理员有效
	err = CreateSystemDatabasesIfNotExist(couchInstance)
	if err != nil {
		logger.Errorf("Unable to connect to CouchDB, error: %s. Check the admin username and password.", err)
		return nil, nil, errors.WithMessage(err, "unable to connect to CouchDB. Check the admin username and password")
	}

	return dbResponse, couchDBReturn, nil
}

//健康检查检查对等端是否能够与CouchDB通信
func (couchInstance *CouchInstance) HealthCheck(ctx context.Context) error {
	connectURL, err := url.Parse(couchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return errors.Wrapf(err, "error parsing CouchDB URL: %s", couchInstance.conf.URL)
	}
	_, _, err = couchInstance.handleRequest(ctx, http.MethodHead, "", "HealthCheck", connectURL, nil, "", "", 0, true, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to couch db [%s]", err)
	}
	return nil
}

//DropDatabase提供了删除现有数据库的方法
func (dbclient *CouchDatabase) DropDatabase() (*DBOperationResponse, error) {
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering DropDatabase()", dbName)

	connectURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodDelete, "DropDatabase", connectURL, nil, "", "", maxRetries, true, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	dbResponse := &DBOperationResponse{}
	decodeErr := json.NewDecoder(resp.Body).Decode(&dbResponse)
	if decodeErr != nil {
		return nil, errors.Wrap(decodeErr, "error decoding response body")
	}

	if dbResponse.Ok == true {
		logger.Debugf("[%s] Dropped database", dbclient.DBName)
	}

	logger.Debugf("[%s] Exiting DropDatabase()", dbclient.DBName)

	if dbResponse.Ok == true {

		return dbResponse, nil

	}

	return dbResponse, errors.New("error dropping database")

}

//ensure full commit调用_确保完全提交显式fsync
func (dbclient *CouchDatabase) EnsureFullCommit() (*DBOperationResponse, error) {
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering EnsureFullCommit()", dbName)

	connectURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodPost, "EnsureFullCommit", connectURL, nil, "", "", maxRetries, true, nil, "_ensure_full_commit")
	if err != nil {
		logger.Errorf("Failed to invoke couchdb _ensure_full_commit. Error: %+v", err)
		return nil, err
	}
	defer closeResponseBody(resp)

	dbResponse := &DBOperationResponse{}
	decodeErr := json.NewDecoder(resp.Body).Decode(&dbResponse)
	if decodeErr != nil {
		return nil, errors.Wrap(decodeErr, "error decoding response body")
	}

//检查是否启用了Autowarmindexes
//如果启用了autowarmindexes，索引将在块数之后刷新
//在getwarmindexesafternblocks（）中，已提交到状态数据库
//检查提交的块数是否超过索引预热的阈值
//使用go例程启动warmindexallindexes（），这将作为后台进程执行。
	if ledgerconfig.IsAutoWarmIndexesEnabled() {

		if dbclient.IndexWarmCounter >= ledgerconfig.GetWarmIndexesAfterNBlocks() {
			go dbclient.runWarmIndexAllIndexes()
			dbclient.IndexWarmCounter = 0
		}
		dbclient.IndexWarmCounter++

	}

	logger.Debugf("[%s] Exiting EnsureFullCommit()", dbclient.DBName)

	if dbResponse.Ok == true {

		return dbResponse, nil

	}

	return dbResponse, errors.New("error syncing database")
}

//SaveDoc方法提供了一个保存文档、ID和字节数组的函数
func (dbclient *CouchDatabase) SaveDoc(id string, rev string, couchDoc *CouchDoc) (string, error) {
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering SaveDoc() id=[%s]", dbName, id)

	if !utf8.ValidString(id) {
		return "", errors.Errorf("doc id [%x] not a valid utf8 string", id)
	}

	saveURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return "", errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//为要推送到couchdb的数据设置缓冲区
	data := []byte{}

//设置默认边界，供发送附件时多部分使用
	defaultBoundary := ""

//为共享连接创建标志。对于零长度附件，此设置为false
	keepConnectionOpen := true

//检查附件是否为零，如果是，那么这只是一个JSON
	if couchDoc.Attachments == nil {

//测试以查看这是否是有效的JSON
		if IsJSON(string(couchDoc.JSONValue)) != true {
			return "", errors.New("JSON format is not valid")
		}

//如果没有附件，则使用传入的字节作为JSON
		data = couchDoc.JSONValue

} else { //有附件

//attachments are included, create the multipart definition
		multipartData, multipartBoundary, err3 := createAttachmentPart(couchDoc, defaultBoundary)
		if err3 != nil {
			return "", err3
		}

//如果有零长度附件，请不要保持连接打开
		for _, attach := range couchDoc.Attachments {
			if attach.Length < 1 {
				keepConnectionOpen = false
			}
		}

//将数据缓冲区设置为来自创建多部分数据的数据
		data = multipartData.Bytes()

//Set the default boundary to the value generated in the multipart creation
		defaultBoundary = multipartBoundary

	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

//如果存在修订冲突，请重试处理保存文档的请求
	resp, _, err := dbclient.handleRequestWithRevisionRetry(id, http.MethodPut, dbName, "SaveDoc", saveURL, data, rev, defaultBoundary, maxRetries, keepConnectionOpen, nil)

	if err != nil {
		return "", err
	}
	defer closeResponseBody(resp)

//get the revision and return
	revision, err := getRevisionHeader(resp)
	if err != nil {
		return "", err
	}

	logger.Debugf("[%s] Exiting SaveDoc()", dbclient.DBName)

	return revision, nil

}

//如果文档存在，GetDocumentRevision将返回修订，否则将返回“”。
func (dbclient *CouchDatabase) getDocumentRevision(id string) string {

	var rev = ""

//看看文档是否已经存在，我们需要rev来保存和删除
	_, revdoc, err := dbclient.ReadDoc(id)
	if err == nil {
//将修订版设置为从文档读取返回的修订版
		rev = revdoc
	}
	return rev
}

func createAttachmentPart(couchDoc *CouchDoc, defaultBoundary string) (bytes.Buffer, string, error) {

//为写入结果创建缓冲区
	writeBuffer := new(bytes.Buffer)

//read the attachment and save as an attachment
	writer := multipart.NewWriter(writeBuffer)

//检索多部分的边界
	defaultBoundary = writer.Boundary()

	fileAttachments := map[string]FileDetails{}

	for _, attachment := range couchDoc.Attachments {
		fileAttachments[attachment.Name] = FileDetails{true, attachment.ContentType, len(attachment.AttachmentBytes)}
	}

	attachmentJSONMap := map[string]interface{}{
		"_attachments": fileAttachments}

//添加随文件上载的任何数据
	if couchDoc.JSONValue != nil {

//创建通用映射
		genericMap := make(map[string]interface{})

//将数据解组到通用映射中
		decoder := json.NewDecoder(bytes.NewBuffer(couchDoc.JSONValue))
		decoder.UseNumber()
		decodeErr := decoder.Decode(&genericMap)
		if decodeErr != nil {
			return *writeBuffer, "", errors.Wrap(decodeErr, "error decoding json data")
		}

//将所有键/值添加到附件jsonmap中
		for jsonKey, jsonValue := range genericMap {
			attachmentJSONMap[jsonKey] = jsonValue
		}

	}

	filesForUpload, err := json.Marshal(attachmentJSONMap)
	if err != nil {
		return *writeBuffer, "", errors.Wrap(err, "error marshalling json data")
	}

	logger.Debugf(string(filesForUpload))

//为JSON创建头文件
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "application/json")

	part, err := writer.CreatePart(header)
	if err != nil {
		return *writeBuffer, defaultBoundary, errors.Wrap(err, "error creating multipart")
	}

	part.Write(filesForUpload)

	for _, attachment := range couchDoc.Attachments {

		header := make(textproto.MIMEHeader)
		part, err2 := writer.CreatePart(header)
		if err2 != nil {
			return *writeBuffer, defaultBoundary, errors.Wrap(err2, "error creating multipart")
		}
		part.Write(attachment.AttachmentBytes)

	}

	err = writer.Close()
	if err != nil {
		return *writeBuffer, defaultBoundary, errors.Wrap(err, "error closing multipart writer")
	}

	return *writeBuffer, defaultBoundary, nil

}

func getRevisionHeader(resp *http.Response) (string, error) {

	if resp == nil {
		return "", errors.New("no response received from CouchDB")
	}

	revision := resp.Header.Get("Etag")

	if revision == "" {
		return "", errors.New("no revision tag detected")
	}

	reg := regexp.MustCompile(`"([^"]*)"`)
	revisionNoQuotes := reg.ReplaceAllString(revision, "${1}")
	return revisionNoQuotes, nil

}

//readDoc方法提供了检索文档及其修订的功能
//从数据库中按ID
func (dbclient *CouchDatabase) ReadDoc(id string) (*CouchDoc, string, error) {
	var couchDoc CouchDoc
	attachments := []*AttachmentInfo{}
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering ReadDoc()  id=[%s]", dbName, id)
	if !utf8.ValidString(id) {
		return nil, "", errors.Errorf("doc id [%x] not a valid utf8 string", id)
	}

	readURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, "", errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

	query := readURL.Query()
	query.Add("attachments", "true")

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, couchDBReturn, err := dbclient.handleRequest(http.MethodGet, "ReadDoc", readURL, nil, "", "", maxRetries, true, &query, id)
	if err != nil {
		if couchDBReturn != nil && couchDBReturn.StatusCode == 404 {
			logger.Debugf("[%s] Document not found (404), returning nil value instead of 404 error", dbclient.DBName)
//不存在的文档应返回零值，而不是404错误
//有关详细信息，请参见HTTPS:/GIHUUBCOM/HyLeGeReaGraciSe/Fuff/Strus/936
			return nil, "", nil
		}
		logger.Debugf("[%s] couchDBReturn=%v\n", dbclient.DBName, couchDBReturn)
		return nil, "", err
	}
	defer closeResponseBody(resp)

//从内容类型标题获取媒体类型
	mediaType, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		log.Fatal(err)
	}

//从标题获得修订
	revision, err := getRevisionHeader(resp)
	if err != nil {
		return nil, "", err
	}

//检查是否为多部分，如果检测到多部分，则作为附件处理
	if strings.HasPrefix(mediaType, "multipart/") {
//基于边界设置多部分读卡器
		multipartReader := multipart.NewReader(resp.Body, params["boundary"])
		for {
			p, err := multipartReader.NextPart()
			if err == io.EOF {
break //已处理所有零件
			}
			if err != nil {
				return nil, "", errors.Wrap(err, "error reading next multipart")
			}

			defer p.Close()

			logger.Debugf("[%s] part header=%s", dbclient.DBName, p.Header)
			switch p.Header.Get("Content-Type") {
			case "application/json":
				partdata, err := ioutil.ReadAll(p)
				if err != nil {
					return nil, "", errors.Wrap(err, "error reading multipart data")
				}
				couchDoc.JSONValue = partdata
			default:

//创建附件结构并加载它
				attachment := &AttachmentInfo{}
				attachment.ContentType = p.Header.Get("Content-Type")
				contentDispositionParts := strings.Split(p.Header.Get("Content-Disposition"), ";")
				if strings.TrimSpace(contentDispositionParts[0]) == "attachment" {
					switch p.Header.Get("Content-Encoding") {
case "gzip": //查看部件是否为gzip编码

						var respBody []byte

						gr, err := gzip.NewReader(p)
						if err != nil {
							return nil, "", errors.Wrap(err, "error creating gzip reader")
						}
						respBody, err = ioutil.ReadAll(gr)
						if err != nil {
							return nil, "", errors.Wrap(err, "error reading gzip data")
						}

						logger.Debugf("[%s] Retrieved attachment data", dbclient.DBName)
						attachment.AttachmentBytes = respBody
						attachment.Length = uint64(len(attachment.AttachmentBytes))
						attachment.Name = p.FileName()
						attachments = append(attachments, attachment)

					default:

//retrieve the data,  this is not gzip
						partdata, err := ioutil.ReadAll(p)
						if err != nil {
							return nil, "", errors.Wrap(err, "error reading multipart data")
						}
						logger.Debugf("[%s] Retrieved attachment data", dbclient.DBName)
						attachment.AttachmentBytes = partdata
						attachment.Length = uint64(len(attachment.AttachmentBytes))
						attachment.Name = p.FileName()
						attachments = append(attachments, attachment)

} //结束内容编码开关
} //附件结束
} //end content-type switch
} //对于所有多部分

		couchDoc.Attachments = attachments

		return &couchDoc, revision, nil
	}

//作为JSON文档处理
	couchDoc.JSONValue, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", errors.Wrap(err, "error reading response body")
	}

	logger.Debugf("[%s] Exiting ReadDoc()", dbclient.DBName)
	return &couchDoc, revision, nil
}

//readdocrange方法基于开始键和结束键为一系列文档提供功能
//startkey和endkey也可以是空字符串。如果startkey和endkey为空，则返回所有文档
//此函数提供一个限制选项来指定最大条目数，并由config提供。
//跳过是为将来可能使用而保留的。
func (dbclient *CouchDatabase) ReadDocRange(startKey, endKey string, limit int32) ([]*QueryResult, string, error) {
	dbName := dbclient.DBName
	logger.Debugf("[%s] Entering ReadDocRange()  startKey=%s, endKey=%s", dbName, startKey, endKey)

	var results []*QueryResult

	rangeURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, "", errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

	queryParms := rangeURL.Query()
//将限制增加1以查看是否有更多符合条件的记录
	queryParms.Set("limit", strconv.FormatInt(int64(limit+1), 10))
	queryParms.Add("include_docs", "true")
queryParms.Add("inclusive_end", "false") //endkey应该是独占的，以便与goleveldb一致
queryParms.Add("attachments", "true")    //get the attachments as well

//附加startkey（如果提供）
	if startKey != "" {
		if startKey, err = encodeForJSON(startKey); err != nil {
			return nil, "", err
		}
		queryParms.Add("startkey", "\""+startKey+"\"")
	}

//附加endkey（如果提供）
	if endKey != "" {
		var err error
		if endKey, err = encodeForJSON(endKey); err != nil {
			return nil, "", err
		}
		queryParms.Add("endkey", "\""+endKey+"\"")
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodGet, "RangeDocRange", rangeURL, nil, "", "", maxRetries, true, &queryParms, "_all_docs")
	if err != nil {
		return nil, "", err
	}
	defer closeResponseBody(resp)

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		dump, err2 := httputil.DumpResponse(resp, true)
		if err2 != nil {
			log.Fatal(err2)
		}
		logger.Debugf("[%s] %s", dbclient.DBName, dump)
	}

//作为JSON文档处理
	jsonResponseRaw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", errors.Wrap(err, "error reading response body")
	}

	var jsonResponse = &RangeQueryResponse{}
	err2 := json.Unmarshal(jsonResponseRaw, &jsonResponse)
	if err2 != nil {
		return nil, "", errors.Wrap(err2, "error unmarshalling json data")
	}

//如果找到其他记录，则将计数减少1
//并填充NextStartKey
	if jsonResponse.TotalRows > limit {
		jsonResponse.TotalRows = limit
	}

	logger.Debugf("[%s] Total Rows: %d", dbclient.DBName, jsonResponse.TotalRows)

//使用下一个endkey作为下一个startkey的起始默认值
	nextStartKey := endKey

	for index, row := range jsonResponse.Rows {

		var docMetadata = &DocMetadata{}
		err3 := json.Unmarshal(row.Doc, &docMetadata)
		if err3 != nil {
			return nil, "", errors.Wrap(err3, "error unmarshalling json data")
		}

//如果NEXSTATEKEY有额外的行，那么不要将行添加到结果集
//并填充NextStartKey变量
		if int32(index) >= jsonResponse.TotalRows {
			nextStartKey = docMetadata.ID
			continue
		}

		if docMetadata.AttachmentsInfo != nil {

			logger.Debugf("[%s] Adding JSON document and attachments for id: %s", dbclient.DBName, docMetadata.ID)

			attachments := []*AttachmentInfo{}
			for attachmentName, attachment := range docMetadata.AttachmentsInfo {
				attachment.Name = attachmentName

				attachments = append(attachments, attachment)
			}

			var addDocument = &QueryResult{docMetadata.ID, row.Doc, attachments}
			results = append(results, addDocument)

		} else {

			logger.Debugf("[%s] Adding json docment for id: %s", dbclient.DBName, docMetadata.ID)

			var addDocument = &QueryResult{docMetadata.ID, row.Doc, nil}
			results = append(results, addDocument)

		}

	}

	logger.Debugf("[%s] Exiting ReadDocRange()", dbclient.DBName)

	return results, nextStartKey, nil

}

//DeleteDoc方法提供按ID从数据库中删除文档的功能
func (dbclient *CouchDatabase) DeleteDoc(id, rev string) error {
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering DeleteDoc()  id=%s", dbName, id)

	deleteURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

//如果存在修订冲突，请重试处理保存文档的请求
	resp, couchDBReturn, err := dbclient.handleRequestWithRevisionRetry(id, http.MethodDelete, dbName, "DeleteDoc",
		deleteURL, nil, "", "", maxRetries, true, nil)

	if err != nil {
		if couchDBReturn != nil && couchDBReturn.StatusCode == 404 {
			logger.Debugf("[%s] Document not found (404), returning nil value instead of 404 error", dbclient.DBName)
//不存在的文档应返回零值，而不是404错误
//有关详细信息，请参见HTTPS:/GIHUUBCOM/HyLeGeReaGraciSe/Fuff/Strus/936
			return nil
		}
		return err
	}
	defer closeResponseBody(resp)

	logger.Debugf("[%s] Exiting DeleteDoc()", dbclient.DBName)

	return nil

}

//querydocuments方法提供处理查询的功能
func (dbclient *CouchDatabase) QueryDocuments(query string) ([]*QueryResult, string, error) {
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering QueryDocuments()  query=%s", dbName, query)

	var results []*QueryResult

	queryURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, "", errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodPost, "QueryDocuments", queryURL, []byte(query), "", "", maxRetries, true, nil, "_find")
	if err != nil {
		return nil, "", err
	}
	defer closeResponseBody(resp)

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		dump, err2 := httputil.DumpResponse(resp, true)
		if err2 != nil {
			log.Fatal(err2)
		}
		logger.Debugf("[%s] %s", dbclient.DBName, dump)
	}

//作为JSON文档处理
	jsonResponseRaw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", errors.Wrap(err, "error reading response body")
	}

	var jsonResponse = &QueryResponse{}

	err2 := json.Unmarshal(jsonResponseRaw, &jsonResponse)
	if err2 != nil {
		return nil, "", errors.Wrap(err2, "error unmarshalling json data")
	}

	if jsonResponse.Warning != "" {
		logger.Warnf("The query [%s] caused the following warning: [%s]", query, jsonResponse.Warning)
	}

	for _, row := range jsonResponse.Docs {

		var docMetadata = &DocMetadata{}
		err3 := json.Unmarshal(row, &docMetadata)
		if err3 != nil {
			return nil, "", errors.Wrap(err3, "error unmarshalling json data")
		}

//JSON Query results never have attachments
//下面的if块永远不会执行
		if docMetadata.AttachmentsInfo != nil {

			logger.Debugf("[%s] Adding JSON docment and attachments for id: %s", dbclient.DBName, docMetadata.ID)

			couchDoc, _, err := dbclient.ReadDoc(docMetadata.ID)
			if err != nil {
				return nil, "", err
			}
			var addDocument = &QueryResult{ID: docMetadata.ID, Value: couchDoc.JSONValue, Attachments: couchDoc.Attachments}
			results = append(results, addDocument)

		} else {
			logger.Debugf("[%s] Adding json docment for id: %s", dbclient.DBName, docMetadata.ID)
			var addDocument = &QueryResult{ID: docMetadata.ID, Value: row, Attachments: nil}

			results = append(results, addDocument)

		}
	}

	logger.Debugf("[%s] Exiting QueryDocuments()", dbclient.DBName)

	return results, jsonResponse.Bookmark, nil

}

//listIndex方法列出了为数据库定义的索引
func (dbclient *CouchDatabase) ListIndex() ([]*IndexResult, error) {

//index definition包含couchdb索引的定义
	type indexDefinition struct {
		DesignDocument string          `json:"ddoc"`
		Name           string          `json:"name"`
		Type           string          `json:"type"`
		Definition     json.RawMessage `json:"def"`
	}

//ListIndexResponse包含用于列出CouchDB索引的定义
	type listIndexResponse struct {
		TotalRows int               `json:"total_rows"`
		Indexes   []indexDefinition `json:"indexes"`
	}

	dbName := dbclient.DBName
	logger.Debug("[%s] Entering ListIndex()", dbName)

	indexURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodGet, "ListIndex", indexURL, nil, "", "", maxRetries, true, nil, "_index")
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

//作为JSON文档处理
	jsonResponseRaw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading response body")
	}

	var jsonResponse = &listIndexResponse{}

	err2 := json.Unmarshal(jsonResponseRaw, jsonResponse)
	if err2 != nil {
		return nil, errors.Wrap(err2, "error unmarshalling json data")
	}

	var results []*IndexResult

	for _, row := range jsonResponse.Indexes {

//如果设计文档不是以“_design/”开头，则这是一个系统
//级别索引，没有意义，无法编辑或删除
		designDoc := row.DesignDocument
		s := strings.SplitAfterN(designDoc, "_design/", 2)
		if len(s) > 1 {
			designDoc = s[1]

//将索引定义添加到结果中
			var addIndexResult = &IndexResult{DesignDocument: designDoc, Name: row.Name, Definition: fmt.Sprintf("%s", row.Definition)}
			results = append(results, addIndexResult)
		}

	}

	logger.Debugf("[%s] Exiting ListIndex()", dbclient.DBName)

	return results, nil

}

//CreateIndex method provides a function creating an index
func (dbclient *CouchDatabase) CreateIndex(indexdefinition string) (*CreateIndexResponse, error) {
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering CreateIndex()  indexdefinition=%s", dbName, indexdefinition)

//测试以查看这是否是有效的JSON
	if IsJSON(indexdefinition) != true {
		return nil, errors.New("JSON format is not valid")
	}

	indexURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodPost, "CreateIndex", indexURL, []byte(indexdefinition), "", "", maxRetries, true, nil, "_index")
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	if resp == nil {
		return nil, errors.New("invalid response received from CouchDB")
	}

//读取响应正文
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading response body")
	}

	couchDBReturn := &CreateIndexResponse{}

	jsonBytes := []byte(respBody)

//取消标记响应
	err = json.Unmarshal(jsonBytes, &couchDBReturn)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshalling json data")
	}

	if couchDBReturn.Result == "created" {

		logger.Infof("Created CouchDB index [%s] in state database [%s] using design document [%s]", couchDBReturn.Name, dbclient.DBName, couchDBReturn.ID)

		return couchDBReturn, nil

	}

	logger.Infof("Updated CouchDB index [%s] in state database [%s] using design document [%s]", couchDBReturn.Name, dbclient.DBName, couchDBReturn.ID)

	return couchDBReturn, nil
}

//DeleteIndex方法提供删除索引的函数
func (dbclient *CouchDatabase) DeleteIndex(designdoc, indexname string) error {
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering DeleteIndex()  designdoc=%s  indexname=%s", dbName, designdoc, indexname)

	indexURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodDelete, "DeleteIndex", indexURL, nil, "", "", maxRetries, true, nil, "_index", designdoc, "json", indexname)
	if err != nil {
		return err
	}
	defer closeResponseBody(resp)

	return nil

}

//WarmIndex method provides a function for warming a single index
func (dbclient *CouchDatabase) WarmIndex(designdoc, indexname string) error {
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering WarmIndex()  designdoc=%s  indexname=%s", dbName, designdoc, indexname)

	indexURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

	queryParms := indexURL.Query()
//允许立即返回URL执行的查询参数
//更新后将导致在URL返回后运行索引更新
	queryParms.Add("stale", "update_after")

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodGet, "WarmIndex", indexURL, nil, "", "", maxRetries, true, &queryParms, "_design", designdoc, "_view", indexname)
	if err != nil {
		return err
	}
	defer closeResponseBody(resp)

	return nil

}

//runwarmindexalindexes是warmindexalindexes捕获和报告任何错误的包装器。
func (dbclient *CouchDatabase) runWarmIndexAllIndexes() {

	err := dbclient.WarmIndexAllIndexes()
	if err != nil {
		logger.Errorf("Error detected during WarmIndexAllIndexes(): %+v", err)
	}

}

//warmindexallindexes方法提供了一个函数，用于预热数据库的所有索引。
func (dbclient *CouchDatabase) WarmIndexAllIndexes() error {

	logger.Debugf("[%s] Entering WarmIndexAllIndexes()", dbclient.DBName)

//Retrieve all indexes
	listResult, err := dbclient.ListIndex()
	if err != nil {
		return err
	}

//对于每个索引定义，执行索引刷新
	for _, elem := range listResult {

		err := dbclient.WarmIndex(elem.DesignDocument, elem.Name)
		if err != nil {
			return err
		}

	}

	logger.Debugf("[%s] Exiting WarmIndexAllIndexes()", dbclient.DBName)

	return nil

}

//GetDatabaseSecurity方法提供用于检索数据库安全配置的函数
func (dbclient *CouchDatabase) GetDatabaseSecurity() (*DatabaseSecurity, error) {
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering GetDatabaseSecurity()", dbName)

	securityURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodGet, "GetDatabaseSecurity", securityURL, nil, "", "", maxRetries, true, nil, "_security")

	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

//作为JSON文档处理
	jsonResponseRaw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading response body")
	}

	var jsonResponse = &DatabaseSecurity{}

	err2 := json.Unmarshal(jsonResponseRaw, jsonResponse)
	if err2 != nil {
		return nil, errors.Wrap(err2, "error unmarshalling json data")
	}

	logger.Debugf("[%s] Exiting GetDatabaseSecurity()", dbclient.DBName)

	return jsonResponse, nil

}

//ApplyDatabaseSecurity方法提供更新数据库安全配置的函数
func (dbclient *CouchDatabase) ApplyDatabaseSecurity(databaseSecurity *DatabaseSecurity) error {
	dbName := dbclient.DBName

	logger.Debugf("[%s] Entering ApplyDatabaseSecurity()", dbName)

	securityURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

//确保所有数组都初始化为空数组，而不是nil
	if databaseSecurity.Admins.Names == nil {
		databaseSecurity.Admins.Names = make([]string, 0)
	}
	if databaseSecurity.Admins.Roles == nil {
		databaseSecurity.Admins.Roles = make([]string, 0)
	}
	if databaseSecurity.Members.Names == nil {
		databaseSecurity.Members.Names = make([]string, 0)
	}
	if databaseSecurity.Members.Roles == nil {
		databaseSecurity.Members.Roles = make([]string, 0)
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	databaseSecurityJSON, err := json.Marshal(databaseSecurity)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling json data")
	}

	logger.Debugf("[%s] Applying security to database: %s", dbclient.DBName, string(databaseSecurityJSON))

	resp, _, err := dbclient.handleRequest(http.MethodPut, "ApplyDatabaseSecurity", securityURL, databaseSecurityJSON, "", "", maxRetries, true, nil, "_security")

	if err != nil {
		return err
	}
	defer closeResponseBody(resp)

	logger.Debugf("[%s] Exiting ApplyDatabaseSecurity()", dbclient.DBName)

	return nil

}

//batch retrieve document metadata-用于检索一组键的文档元数据的批处理方法，
//包括ID、CouchDB版本号和分类帐版本
func (dbclient *CouchDatabase) BatchRetrieveDocumentMetadata(keys []string) ([]*DocMetadata, error) {

	logger.Debugf("[%s] Entering BatchRetrieveDocumentMetadata()  keys=%s", dbclient.DBName, keys)

	batchRetrieveURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

	queryParms := batchRetrieveURL.Query()

//虽然batchRetrieveDocumentMetadata（）不返回整个文档，
//for reads/writes, we do need to get document so that we can get the ledger version of the key.
//对于盲写，TODO不需要获取版本，因此当我们批量获取
//未在读取集中表示的写入键的修订号
//（第二次在块处理期间调用BatchRetrieveDocumentMetadata），
//我们可以将include-docs设置为false以优化响应。
	queryParms.Add("include_docs", "true")

	keymap := make(map[string]interface{})

	keymap["keys"] = keys

	jsonKeys, err := json.Marshal(keymap)
	if err != nil {
		return nil, errors.Wrap(err, "error marshalling json data")
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodPost, "BatchRetrieveDocumentMetadata", batchRetrieveURL, jsonKeys, "", "", maxRetries, true, &queryParms, "_all_docs")
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
//compact debug log by replacing carriage return / line feed with dashes to separate http headers
		logger.Debugf("[%s] HTTP Response: %s", dbclient.DBName, bytes.Replace(dump, []byte{0x0d, 0x0a}, []byte{0x20, 0x7c, 0x20}, -1))
	}

//作为JSON文档处理
	jsonResponseRaw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading response body")
	}

	var jsonResponse = &BatchRetrieveDocMetadataResponse{}

	err2 := json.Unmarshal(jsonResponseRaw, &jsonResponse)
	if err2 != nil {
		return nil, errors.Wrap(err2, "error unmarshalling json data")
	}

	docMetadataArray := []*DocMetadata{}

	for _, row := range jsonResponse.Rows {
		docMetadata := &DocMetadata{ID: row.ID, Rev: row.DocMetadata.Rev, Version: row.DocMetadata.Version}
		docMetadataArray = append(docMetadataArray, docMetadata)
	}

	logger.Debugf("[%s] Exiting BatchRetrieveDocumentMetadata()", dbclient.DBName)

	return docMetadataArray, nil

}

//批更新文档-批更新文档的批处理方法
func (dbclient *CouchDatabase) BatchUpdateDocuments(documents []*CouchDoc) ([]*BatchUpdateResponse, error) {
	dbName := dbclient.DBName

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		documentIdsString, err := printDocumentIds(documents)
		if err == nil {
			logger.Debugf("[%s] Entering BatchUpdateDocuments()  document ids=[%s]", dbName, documentIdsString)
		} else {
			logger.Debugf("[%s] Entering BatchUpdateDocuments()  Could not print document ids due to error: %+v", dbName, err)
		}
	}

	batchUpdateURL, err := url.Parse(dbclient.CouchInstance.conf.URL)
	if err != nil {
		logger.Errorf("URL parse error: %s", err)
		return nil, errors.Wrapf(err, "error parsing CouchDB URL: %s", dbclient.CouchInstance.conf.URL)
	}

	documentMap := make(map[string]interface{})

	var jsonDocumentMap []interface{}

	for _, jsonDocument := range documents {

//创建文档映射
		var document = make(map[string]interface{})

//将couchDoc的json组件取消标记到文档中
		err = json.Unmarshal(jsonDocument.JSONValue, &document)
		if err != nil {
			return nil, errors.Wrap(err, "error unmarshalling json data")
		}

//遍历任何附件
		if len(jsonDocument.Attachments) > 0 {

//创建文件附件映射
			fileAttachment := make(map[string]interface{})

//对于每个附件，创建一个base64附件，命名附件，
//添加内容类型，base64编码附件
			for _, attachment := range jsonDocument.Attachments {
				fileAttachment[attachment.Name] = Base64Attachment{attachment.ContentType,
					base64.StdEncoding.EncodeToString(attachment.AttachmentBytes)}
			}

//向文档添加附件
			document["_attachments"] = fileAttachment

		}

//将文档附加到文档图中
		jsonDocumentMap = append(jsonDocumentMap, document)

	}

//将文档添加到“docs”项
	documentMap["docs"] = jsonDocumentMap

	bulkDocsJSON, err := json.Marshal(documentMap)
	if err != nil {
		return nil, errors.Wrap(err, "error marshalling json data")
	}

//获取重试次数
	maxRetries := dbclient.CouchInstance.conf.MaxRetries

	resp, _, err := dbclient.handleRequest(http.MethodPost, "BatchUpdateDocuments", batchUpdateURL, bulkDocsJSON, "", "", maxRetries, true, nil, "_bulk_docs")
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
//通过用破折号替换回车/换行符压缩调试日志以分隔HTTP头
		logger.Debugf("[%s] HTTP Response: %s", dbclient.DBName, bytes.Replace(dump, []byte{0x0d, 0x0a}, []byte{0x20, 0x7c, 0x20}, -1))
	}

//作为JSON文档处理
	jsonResponseRaw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading response body")
	}

	var jsonResponse = []*BatchUpdateResponse{}
	err2 := json.Unmarshal(jsonResponseRaw, &jsonResponse)
	if err2 != nil {
		return nil, errors.Wrap(err2, "error unmarshalling json data")
	}

	logger.Debugf("[%s] Exiting BatchUpdateDocuments() _bulk_docs response=[%s]", dbclient.DBName, string(jsonResponseRaw))

	return jsonResponse, nil

}

//handleRequestWithRevisionRetry方法是一个具有
//文件修订冲突错误的重试，
//可能在保存或删除从客户端HTTP角度超时时检测到，
//但最终还是成功了
func (dbclient *CouchDatabase) handleRequestWithRevisionRetry(id, method, dbName, functionName string, connectURL *url.URL, data []byte, rev string,
	multipartBoundary string, maxRetries int, keepConnectionOpen bool, queryParms *url.Values) (*http.Response, *DBReturn, error) {

//初始化修订冲突的标志
	revisionConflictDetected := false
	var resp *http.Response
	var couchDBReturn *DBReturn
	var errResp error

//尝试HTTP请求的最大重试次数
//In this case, the retry is to catch problems where a client timeout may miss a
//成功的couchdb更新并在handleRequest中重试时导致文档修订冲突
	for attempts := 0; attempts <= maxRetries; attempts++ {

//如果未通过修订，或在之前的尝试中检测到修订冲突，
//查询文档修订的CouchDB
		if rev == "" || revisionConflictDetected {
			rev = dbclient.getDocumentRevision(id)
		}

//处理保存/删除CouchDB数据的请求
		resp, couchDBReturn, errResp = dbclient.CouchInstance.handleRequest(context.Background(), method, dbName, functionName, connectURL,
			data, rev, multipartBoundary, maxRetries, keepConnectionOpen, queryParms, id)

//如果在保存/删除过程中出现409个冲突错误，请记录并重试。
//否则，中断重试循环
		if couchDBReturn != nil && couchDBReturn.StatusCode == 409 {
			logger.Warningf("CouchDB document revision conflict detected, retrying. Attempt:%v", attempts+1)
			revisionConflictDetected = true
		} else {
			break
		}
	}

//返回handleRequest结果
	return resp, couchDBReturn, errResp
}

func (dbclient *CouchDatabase) handleRequest(method, functionName string, connectURL *url.URL, data []byte, rev, multipartBoundary string,
	maxRetries int, keepConnectionOpen bool, queryParms *url.Values, pathElements ...string) (*http.Response, *DBReturn, error) {

	return dbclient.CouchInstance.handleRequest(context.Background(),
		method, dbclient.DBName, functionName, connectURL, data, rev, multipartBoundary,
		maxRetries, keepConnectionOpen, queryParms, pathElements...,
	)
}

//handleRequest方法是一个通用的HTTP请求处理程序。
//如果返回错误，则确保响应主体已关闭，否则
//被叫方正确结束响应的责任。
//任何HTTP错误或CouchDB错误（4xx或500）都将导致返回Golang错误。
func (couchInstance *CouchInstance) handleRequest(ctx context.Context, method, dbName, functionName string, connectURL *url.URL, data []byte, rev string,
	multipartBoundary string, maxRetries int, keepConnectionOpen bool, queryParms *url.Values, pathElements ...string) (*http.Response, *DBReturn, error) {

	logger.Debugf("Entering handleRequest()  method=%s  url=%v  dbName=%s", method, connectURL, dbName)

//为couchdb创建返回对象
	var resp *http.Response
	var errResp error
	couchDBReturn := &DBReturn{}
	defer couchInstance.recordMetric(time.Now(), dbName, functionName, couchDBReturn)

//set initial wait duration for retries
	waitDuration := retryWaitTime * time.Millisecond

	if maxRetries < 0 {
		return nil, nil, errors.New("number of retries must be zero or greater")
	}

	requestURL := constructCouchDBUrl(connectURL, dbName, pathElements...)

	if queryParms != nil {
		requestURL.RawQuery = queryParms.Encode()
	}

	logger.Debugf("Request URL: %s", requestURL)

//尝试HTTP请求的最大重试次数
//如果maxretries为0，将尝试创建一次数据库，并将
//如果失败，则返回错误
//如果maxretries为3（默认），则最多4次尝试（一次尝试，3次重试）
//如果尝试失败，将使用警告条目。
	for attempts := 0; attempts <= maxRetries; attempts++ {

//为有效负载数据设置缓冲区
		payloadData := new(bytes.Buffer)

		payloadData.ReadFrom(bytes.NewReader(data))

//Create request based on URL for couchdb operation
		req, err := http.NewRequest(method, requestURL.String(), payloadData)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error creating http request")
		}
		req.WithContext(ctx)

//如果共享连接不是allowsharedconnection，则将请求设置为完成时关闭
//当前couchdb的零长度附件有问题，不允许重新使用连接。
//couchdb的apache jira项目https://issues.apache.org/jira/browse/couchdb-3394
		if !keepConnectionOpen {
			req.Close = true
		}

//为Put添加内容标题
		if method == http.MethodPut || method == http.MethodPost || method == http.MethodDelete {

//如果未设置multipartboundary，则这是一个JSON，应设置内容类型。
//应用程序/json。否则，它包含一个附件，需要是多部分的
			if multipartBoundary == "" {
				req.Header.Set("Content-Type", "application/json")
			} else {
				req.Header.Set("Content-Type", "multipart/related;boundary=\""+multipartBoundary+"\"")
			}

//检查是否设置了修订，如果设置了，则作为标题传递
			if rev != "" {
				req.Header.Set("If-Match", rev)
			}
		}

//为Put添加内容标题
		if method == http.MethodPut || method == http.MethodPost {
			req.Header.Set("Accept", "application/json")
		}

//为get添加内容头
		if method == http.MethodGet {
			req.Header.Set("Accept", "multipart/related")
		}

//如果设置了用户名和密码，则使用基本身份验证
		if couchInstance.conf.Username != "" && couchInstance.conf.Password != "" {
//REQ.HEADER.SET（“授权”，“基本YWRTAW46YWRTAW5W”）
			req.SetBasicAuth(couchInstance.conf.Username, couchInstance.conf.Password)
		}

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			dump, _ := httputil.DumpRequestOut(req, false)
//通过用破折号替换回车/换行符压缩调试日志以分隔HTTP头
			logger.Debugf("HTTP Request: %s", bytes.Replace(dump, []byte{0x0d, 0x0a}, []byte{0x20, 0x7c, 0x20}, -1))
		}

//执行HTTP请求
		resp, errResp = couchInstance.client.Do(req)

//检查couchdb的返回是否有效
		if invalidCouchDBReturn(resp, errResp) {
			continue
		}

//如果没有golang http错误和couchdb 500错误，则退出重试。
		if errResp == nil && resp != nil && resp.StatusCode < 500 {
//如果这是一个错误，则填充couchdbreturn
			if resp.StatusCode >= 400 {
//读取响应正文并关闭它以进行下一次尝试
				jsonError, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return nil, nil, errors.Wrap(err, "error reading response body")
				}
				defer closeResponseBody(resp)

				errorBytes := []byte(jsonError)
//取消标记响应
				err = json.Unmarshal(errorBytes, &couchDBReturn)
				if err != nil {
					return nil, nil, errors.Wrap(err, "error unmarshalling json data")
				}
			}

			break
		}

//如果maxretries大于0，则记录重试信息
		if maxRetries > 0 {

//如果这是一个意外的golang http错误，请记录该错误并重试。
			if errResp != nil {

//用重试次数记录错误并继续
				logger.Warningf("Retrying couchdb request in %s. Attempt:%v  Error:%v",
					waitDuration.String(), attempts+1, errResp.Error())

//否则，这是来自couchdb的意外500错误。记录错误并重试。
			} else {
//读取响应正文并关闭它以进行下一次尝试
				jsonError, err := ioutil.ReadAll(resp.Body)
				defer closeResponseBody(resp)
				if err != nil {
					return nil, nil, errors.Wrap(err, "error reading response body")
				}

				errorBytes := []byte(jsonError)
//取消标记响应
				err = json.Unmarshal(errorBytes, &couchDBReturn)
				if err != nil {
					return nil, nil, errors.Wrap(err, "error unmarshalling json data")
				}

//用重试计数记录500错误并继续
				logger.Warningf("Retrying couchdb request in %s. Attempt:%v  Couch DB Error:%s,  Status Code:%v  Reason:%v",
					waitDuration.String(), attempts+1, couchDBReturn.Error, resp.Status, couchDBReturn.Reason)

			}
//睡眠指定的睡眠时间，然后重试
			time.Sleep(waitDuration)

//后退，将下次尝试的重试时间加倍
			waitDuration *= 2

		}

} //结束重试循环

//如果在重试结束后仍然存在Golang HTTP错误，则返回该错误。
	if errResp != nil {
		return nil, couchDBReturn, errResp
	}

//根据Golang规范，不应出现这种情况。
//如果此错误从HTTP调用返回（errrep），则resp不应为零，
//这是一个结构，statuscode是一个int
//如果发生这种情况，这将提供更优雅的错误。
	if invalidCouchDBReturn(resp, errResp) {
		return nil, nil, errors.New("unable to connect to CouchDB, check the hostname and port.")
	}

//设置CouchDB请求的返回代码
	couchDBReturn.StatusCode = resp.StatusCode

//检查CouchDB的状态代码是否为400或更高
//响应代码4xx和500将被视为错误-
//golang错误将从couchdbreturn内容中创建，并且两者都将返回
	if resp.StatusCode >= 400 {

//如果状态代码为400或更高，则记录并返回错误
		logger.Debugf("Error handling CouchDB request. Error:%s,  Status Code:%v,  Reason:%s",
			couchDBReturn.Error, resp.StatusCode, couchDBReturn.Reason)

		return nil, couchDBReturn, errors.Errorf("error handling CouchDB request. Error:%s,  Status Code:%v,  Reason:%s",
			couchDBReturn.Error, resp.StatusCode, couchDBReturn.Reason)

	}

	logger.Debugf("Exiting handleRequest()")

//如果没有错误，则返回HTTP响应和couchdb返回对象
	return resp, couchDBReturn, nil
}

func (ci *CouchInstance) recordMetric(startTime time.Time, dbName, api string, couchDBReturn *DBReturn) {
	ci.stats.observeProcessingTime(startTime, dbName, api, strconv.Itoa(couchDBReturn.StatusCode))
}

//InvalidCouchDBResponse检查以确保返回有效的响应或错误
func invalidCouchDBReturn(resp *http.Response, errResp error) bool {
	if resp == nil && errResp == nil {
		return true
	}
	return false
}

//iSjon测试字符串以确定是否有效JSON
func IsJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

//encodePathElement使用golang进行URL路径编码，另外：
//“/”被%2f替换，否则路径编码将被视为路径分隔符并忽略它
//“+”被%2b替换，否则路径编码将忽略它，而couchdb将把加号取消编码为空格。
//请注意，所有其他URL特殊字符都已成功测试，无需特殊处理。
func encodePathElement(str string) string {

	u := &url.URL{}
	u.Path = str
encodedStr := u.EscapedPath() //url encode using golang url path encoding rules
	encodedStr = strings.Replace(encodedStr, "/", "%2F", -1)
	encodedStr = strings.Replace(encodedStr, "+", "%2B", -1)

	return encodedStr
}

func encodeForJSON(str string) (string, error) {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(str); err != nil {
		return "", errors.Wrap(err, "error encoding json data")
	}
//encode向字符串添加双引号，并以\n-将它们作为字节剥离，因为它们都是ASCII（0-127）
	buffer := buf.Bytes()
	return string(buffer[1 : len(buffer)-2]), nil
}

//printDocumentIds是打印指针数组的可读日志项的一种方便方法。
//到沙发文档ID
func printDocumentIds(documentPointers []*CouchDoc) (string, error) {

	documentIds := []string{}

	for _, documentPointer := range documentPointers {
		docMetadata := &DocMetadata{}
		err := json.Unmarshal(documentPointer.JSONValue, &docMetadata)
		if err != nil {
			return "", errors.Wrap(err, "error unmarshalling json data")
		}
		documentIds = append(documentIds, docMetadata.ID)
	}
	return strings.Join(documentIds, ","), nil
}
