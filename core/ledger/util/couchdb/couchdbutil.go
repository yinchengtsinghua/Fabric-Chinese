
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
	"encoding/hex"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/util"
	"github.com/pkg/errors"
)

var expectedDatabaseNamePattern = `[a-z][a-z0-9.$_()+-]*`
var maxLength = 238

//将couchdb数据库名称的长度限制为
//允许长度249个字符，字符串长度限制
//对于链/通道名称、命名空间/链代码名称，以及
//集合名称，它构成数据库名称，
//定义。
var chainNameAllowedLength = 50
var namespaceNameAllowedLength = 50
var collectionNameAllowedLength = 50

//创建couchinstance创建couchdb实例
func CreateCouchInstance(couchDBConnectURL, id, pw string, maxRetries,
	maxRetriesOnStartup int, connectionTimeout time.Duration, createGlobalChangesDB bool, metricsProvider metrics.Provider) (*CouchInstance, error) {

	couchConf, err := CreateConnectionDefinition(couchDBConnectURL,
		id, pw, maxRetries, maxRetriesOnStartup, connectionTimeout, createGlobalChangesDB)
	if err != nil {
		logger.Errorf("Error calling CouchDB CreateConnectionDefinition(): %s", err)
		return nil, err
	}

//Create the http client once
//客户端和传输对于多个goroutine并发使用是安全的
//为了提高效率，应该只创建一次并重新使用。
	client := &http.Client{Timeout: couchConf.RequestTimeout}

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	transport.DisableCompression = false
	client.Transport = transport

//创建CouCHDB实例
	couchInstance := &CouchInstance{conf: *couchConf, client: client}
	couchInstance.stats = newStats(metricsProvider)
	connectInfo, retVal, verifyErr := couchInstance.VerifyCouchConfig()
	if verifyErr != nil {
		return nil, verifyErr
	}

//如果HTTP返回值不是200，则返回错误
	if retVal.StatusCode != 200 {
		return nil, errors.Errorf("CouchDB connection error, expecting return code of 200, received %v", retVal.StatusCode)
	}

//检查couchdb版本号，如果版本不至少为2.0.0，则返回错误。
	errVersion := checkCouchDBVersion(connectInfo.Version)
	if errVersion != nil {
		return nil, errVersion
	}

	return couchInstance, nil
}

//checkcouchdbversion验证couchdb是否至少为2.0.0
func checkCouchDBVersion(version string) error {

//将版本拆分为多个部分
	majorVersion := strings.Split(version, ".")

//检查主版本号是否至少为2
	majorVersionInt, _ := strconv.Atoi(majorVersion[0])
	if majorVersionInt < 2 {
		return errors.Errorf("CouchDB must be at least version 2.0.0. Detected version %s", version)
	}

	return nil
}

//createCouchDatabase创建一个couchdb数据库对象，以及基础数据库（如果它不存在的话）
func CreateCouchDatabase(couchInstance *CouchInstance, dbName string) (*CouchDatabase, error) {

	databaseName, err := mapAndValidateDatabaseName(dbName)
	if err != nil {
		logger.Errorf("Error calling CouchDB CreateDatabaseIfNotExist() for dbName: %s, error: %s", dbName, err)
		return nil, err
	}

	couchDBDatabase := CouchDatabase{CouchInstance: couchInstance, DBName: databaseName, IndexWarmCounter: 1}

//在分类帐启动时创建couchdb数据库（如果该数据库尚不存在）
	err = couchDBDatabase.CreateDatabaseIfNotExist()
	if err != nil {
		logger.Errorf("Error calling CouchDB CreateDatabaseIfNotExist() for dbName: %s, error: %s", dbName, err)
		return nil, err
	}

	return &couchDBDatabase, nil
}

//CreateSystemDatabaseSifNotexist-如果系统数据库不存在，则创建它们
func CreateSystemDatabasesIfNotExist(couchInstance *CouchInstance) error {

	dbName := "_users"
	systemCouchDBDatabase := CouchDatabase{CouchInstance: couchInstance, DBName: dbName, IndexWarmCounter: 1}
	err := systemCouchDBDatabase.CreateDatabaseIfNotExist()
	if err != nil {
		logger.Errorf("Error calling CouchDB CreateDatabaseIfNotExist() for system dbName: %s, error: %s", dbName, err)
		return err
	}

	dbName = "_replicator"
	systemCouchDBDatabase = CouchDatabase{CouchInstance: couchInstance, DBName: dbName, IndexWarmCounter: 1}
	err = systemCouchDBDatabase.CreateDatabaseIfNotExist()
	if err != nil {
		logger.Errorf("Error calling CouchDB CreateDatabaseIfNotExist() for system dbName: %s, error: %s", dbName, err)
		return err
	}
	if couchInstance.conf.CreateGlobalChangesDB {
		dbName = "_global_changes"
		systemCouchDBDatabase = CouchDatabase{CouchInstance: couchInstance, DBName: dbName, IndexWarmCounter: 1}
		err = systemCouchDBDatabase.CreateDatabaseIfNotExist()
		if err != nil {
			logger.Errorf("Error calling CouchDB CreateDatabaseIfNotExist() for system dbName: %s, error: %s", dbName, err)
			return err
		}
	}
	return nil

}

//constructCouchDBUrl constructs a couchDB url with encoding for the database name
//and all path elements
func constructCouchDBUrl(connectURL *url.URL, dbName string, pathElements ...string) *url.URL {
	var buffer bytes.Buffer
	buffer.WriteString(connectURL.String())
	if dbName != "" {
		buffer.WriteString("/")
		buffer.WriteString(encodePathElement(dbName))
	}
	for _, pathElement := range pathElements {
		buffer.WriteString("/")
		buffer.WriteString(encodePathElement(pathElement))
	}
	return &url.URL{Opaque: buffer.String()}
}

//constructMetadataDBName将数据库名称截断为couchDB允许的长度
//构建元数据库名称
func ConstructMetadataDBName(dbName string) string {
	if len(dbName) > maxLength {
		untruncatedDBName := dbName
//如果长度违反允许的限制，则截断名称
//由于传递的dbname与chain/channel name相同，请使用chain name allowedlength截断
		dbName = dbName[:chainNameAllowedLength]
//For metadataDB (i.e., chain/channel DB), the dbName contains <first 50 chars
//（即chainname allowedlength）的chainname>>（实际chainname的sha256散列）
		dbName = dbName + "(" + hex.EncodeToString(util.ComputeSHA256([]byte(untruncatedDBName))) + ")"
//dbname为50个字符，sha256为1个字符（+64个字符，sha256为1个字符）=116个字符
	}
	return dbName + "_"
}

//constructnamespacedbname将db name截断为couchdb允许的长度
//构造NamespaceDBName
func ConstructNamespaceDBName(chainName, namespace string) string {
//replace upper-case in namespace with a escape sequence '$' and the respective lower-case letter
	escapedNamespace := escapeUpperCase(namespace)
	namespaceDBName := chainName + "_" + escapedNamespace

//对于“chainname_namespace”形式的namespacedbname，在违反长度限制时，截断
//namespacedbname将包含chainname>+“”+
//<命名空间的前50个字符（即，namespacenameallowedlength）字符>+
//（<sha256 hash of[chainname_namespace]>）
//
//对于“chainname_namespace$$collection”形式的namespacedbname，在违反长度限制时，截断
//namespacedbname将包含chainname>+“”+
//<first 50 chars (i.e., namespaceNameAllowedLength) of namespace> + "$$" + <first 50 chars
//（即collectionnameallowedlength）of collection>>（<sha256 hash of[chainname_namespace$$pcollection]>）

	if len(namespaceDBName) > maxLength {
//Compute the hash of untruncated namespaceDBName that needs to be appended to
//为保持唯一性而截断的NamespaceDBName
		hashOfNamespaceDBName := hex.EncodeToString(util.ComputeSHA256([]byte(chainName + "_" + namespace)))

//由于截断的namespacedbname的形式为“chainname”_escapedNamespace“，两个chainname
//和escapedNamespace需要截断到定义的允许长度。
		if len(chainName) > chainNameAllowedLength {
//将chainname截断为chainname allowedlength
			chainName = chainName[0:chainNameAllowedLength]
		}
//因为escapedNamespace可以是“namespace”或“namespace$$collectionName”，
//“namespace”和“collectionname”都需要截断为定义的允许长度。
//“$$”用作命名空间和集合名称之间的连接符。
//将escapedNamespace拆分为转义的命名空间和转义的集合名称（如果存在）。
		names := strings.Split(escapedNamespace, "$$")
		namespace := names[0]
		if len(namespace) > namespaceNameAllowedLength {
//截断命名空间
			namespace = namespace[0:namespaceNameAllowedLength]
		}

		escapedNamespace = namespace

//检查并截断集合名称的长度（如果存在）
		if len(names) == 2 {
			collection := names[1]
			if len(collection) > collectionNameAllowedLength {
//Truncate the escaped collection name
				collection = collection[0:collectionNameAllowedLength]
			}
//将截断的集合名称附加到escapedNamespace
			escapedNamespace = escapedNamespace + "$$" + collection
		}
//构造并返回NamespaceDBName
//50 chars for chainName + 1 char for '_' + 102 chars for escaped namespace + 1 char for '(' + 64 chars
//对于sha256哈希+1个字符用于'）'=219个字符
		return chainName + "_" + escapedNamespace + "(" + hashOfNamespaceDBName + ")"
	}
	return namespaceDBName
}

//mapandvalidatedatabasename检查数据库名称是否包含非法字符
//CouchDB Rules: Only lowercase characters (a-z), digits (0-9), and any of the characters
//允许使用“、$、（、）、+、-”和/等。必须以字母开头。
//
//Restictions have already been applied to the database name from Orderer based on
//Kafka和CouchDB要求的限制（“.”char除外）。数据库名称
//这里传递的是`[a-z][a-z0-9.$模式。
//
//此验证将简单地检查数据库名称是否与上述模式匹配，并将替换
//“$”出现的所有“.”。这不会在已转换的名为
func mapAndValidateDatabaseName(databaseName string) (string, error) {
//测试长度
	if len(databaseName) <= 0 {
		return "", errors.Errorf("database name is illegal, cannot be empty")
	}
	if len(databaseName) > maxLength {
		return "", errors.Errorf("database name is illegal, cannot be longer than %d", maxLength)
	}
	re, err := regexp.Compile(expectedDatabaseNamePattern)
	if err != nil {
		return "", errors.Wrapf(err, "error compiling regexp: %s", expectedDatabaseNamePattern)
	}
	matched := re.FindString(databaseName)
	if len(matched) != len(databaseName) {
		return "", errors.Errorf("databaseName '%s' does not match pattern '%s'", databaseName, expectedDatabaseNamePattern)
	}
//replace all '.' to '$'. The databaseName passed in will never contain an '$'.
//所以，这种转换不会引起碰撞
	databaseName = strings.Replace(databaseName, ".", "$", -1)
	return databaseName, nil
}

//EscapeUppercase将每个大写字母替换为“$”和相应的
//小写字母
func escapeUpperCase(dbName string) string {
	re := regexp.MustCompile(`([A-Z])`)
	dbName = re.ReplaceAllString(dbName, "$$"+"$1")
	return strings.ToLower(dbName)
}
