
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
	"encoding/hex"
	"testing"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/common/util"
	"github.com/stretchr/testify/assert"
)

//coach db util功能的单元测试
func TestCreateCouchDBConnectionAndDB(t *testing.T) {

	database := "testcreatecouchdbconnectionanddb"
	cleanup(database)
	defer cleanup(database)
//创建新连接
	couchInstance, err := CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
	assert.NoError(t, err, "Error when trying to CreateCouchInstance")

	_, err = CreateCouchDatabase(couchInstance, database)
	assert.NoError(t, err, "Error when trying to CreateCouchDatabase")

}

//coach db util功能的单元测试
func TestNotCreateCouchGlobalChangesDB(t *testing.T) {
	value := couchDBDef.CreateGlobalChangesDB
	couchDBDef.CreateGlobalChangesDB = false
	defer resetCreateGlobalChangesDBValue(value)
	database := "_global_changes"
	cleanup(database)
	defer cleanup(database)

//创建新连接
	couchInstance, err := CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
	assert.NoError(t, err, "Error when trying to CreateCouchInstance")

	db := CouchDatabase{CouchInstance: couchInstance, DBName: database}

//检索新数据库的信息并确保名称匹配
	_, _, errdb := db.GetDatabaseInfo()
	assert.NotNil(t, errdb)
}

func resetCreateGlobalChangesDBValue(value bool) {
	couchDBDef.CreateGlobalChangesDB = value
}

//coach db util功能的单元测试
func TestCreateCouchDBSystemDBs(t *testing.T) {

	database := "testcreatecouchdbsystemdb"
	cleanup(database)
	defer cleanup(database)

//创建新连接
	couchInstance, err := CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})

	assert.NoError(t, err, "Error when trying to CreateCouchInstance")

	err = CreateSystemDatabasesIfNotExist(couchInstance)
	assert.NoError(t, err, "Error when trying to create system databases")

	db := CouchDatabase{CouchInstance: couchInstance, DBName: "_users"}

//检索新数据库的信息并确保名称匹配
	dbResp, _, errdb := db.GetDatabaseInfo()
	assert.NoError(t, errdb, "Error when trying to retrieve _users database information")
	assert.Equal(t, "_users", dbResp.DbName)

	db = CouchDatabase{CouchInstance: couchInstance, DBName: "_replicator"}

//检索新数据库的信息并确保名称匹配
	dbResp, _, errdb = db.GetDatabaseInfo()
	assert.NoError(t, errdb, "Error when trying to retrieve _replicator database information")
	assert.Equal(t, "_replicator", dbResp.DbName)

	db = CouchDatabase{CouchInstance: couchInstance, DBName: "_global_changes"}

//检索新数据库的信息并确保名称匹配
	dbResp, _, errdb = db.GetDatabaseInfo()
	assert.NoError(t, errdb, "Error when trying to retrieve _global_changes database information")
	assert.Equal(t, "_global_changes", dbResp.DbName)

}

func TestDatabaseMapping(t *testing.T) {
//使用数据库名称混合大小写创建新实例和数据库对象
	_, err := mapAndValidateDatabaseName("testDB")
	assert.Error(t, err, "Error expected because the name contains capital letters")

//使用具有特殊字符的数据库名称创建新实例和数据库对象
	_, err = mapAndValidateDatabaseName("test1234/1")
	assert.Error(t, err, "Error expected because the name contains illegal chars")

//使用具有特殊字符的数据库名称创建新实例和数据库对象
	_, err = mapAndValidateDatabaseName("5test1234")
	assert.Error(t, err, "Error expected because the name starts with a number")

//使用空字符串创建新的实例和数据库对象
	_, err = mapAndValidateDatabaseName("")
	assert.Error(t, err, "Error should have been thrown for an invalid name")

	_, err = mapAndValidateDatabaseName("a12345678901234567890123456789012345678901234" +
		"56789012345678901234567890123456789012345678901234567890123456789012345678901234567890" +
		"12345678901234567890123456789012345678901234567890123456789012345678901234567890123456" +
		"78901234567890123456789012345678901234567890")
	assert.Error(t, err, "Error should have been thrown for an invalid name")

	transformedName, err := mapAndValidateDatabaseName("test.my.db-1")
	assert.NoError(t, err, "")
	assert.Equal(t, "test$my$db-1", transformedName)
}

func TestConstructMetadataDBName(t *testing.T) {
//链名称的允许模式：[A-Z][A-Z0-9.-]
	chainName := "tob2g.y-z0f.qwp-rq5g4-ogid5g6oucyryg9sc16mz0t4vuake5q557esz7sn493nf0ghch0xih6dwuirokyoi4jvs67gh6r5v6mhz3-292un2-9egdcs88cstg3f7xa9m1i8v4gj0t3jedsm-woh3kgiqehwej6h93hdy5tr4v.1qmmqjzz0ox62k.507sh3fkw3-mfqh.ukfvxlm5szfbwtpfkd1r4j.cy8oft5obvwqpzjxb27xuw6"

	truncatedChainName := "tob2g.y-z0f.qwp-rq5g4-ogid5g6oucyryg9sc16mz0t4vuak"
	assert.Equal(t, chainNameAllowedLength, len(truncatedChainName))

//<chainname的前50个字符（即chainname allowedlength）>+1个字符用于'（'+<64个字符用于sha256哈希
//（十六进制编码）的非加密链名>+1个字符用于'）'+1个字符用于''=117个字符
	hash := hex.EncodeToString(util.ComputeSHA256([]byte(chainName)))
	expectedDBName := truncatedChainName + "(" + hash + ")" + "_"
	expectedDBNameLength := 117

	constructedDBName := ConstructMetadataDBName(chainName)
	assert.Equal(t, expectedDBNameLength, len(constructedDBName))
	assert.Equal(t, expectedDBName, constructedDBName)
}

func TestConstructedNamespaceDBName(t *testing.T) {
//= =场景1：ChannNeNeNS $ $ COLL＝=

//链名称的允许模式：[A-Z][A-Z0-9.-]
	chainName := "tob2g.y-z0f.qwp-rq5g4-ogid5g6oucyryg9sc16mz0t4vuake5q557esz7sn493nf0ghch0xih6dwuirokyoi4jvs67gh6r5v6mhz3-292un2-9egdcs88cstg3f7xa9m1i8v4gj0t3jedsm-woh3kgiqehwej6h93hdy5tr4v.1qmmqjzz0ox62k.507sh3fkw3-mfqh.ukfvxlm5szfbwtpfkd1r4j.cy8oft5obvwqpzjxb27xuw6"

//命名空间和集合的允许模式：[a-za-z0-9_u-]
	ns := "wMCnSXiV9YoIqNQyNvFVTdM8XnUtvrOFFIWsKelmP5NEszmNLl8YhtOKbFu3P_NgwgsYF8PsfwjYCD8f1XRpANQLoErDHwLlweryqXeJ6vzT2x0pS_GwSx0m6tBI0zOmHQOq_2De8A87x6zUOPwufC2T6dkidFxiuq8Sey2-5vUo_iNKCij3WTeCnKx78PUIg_U1gp4_0KTvYVtRBRvH0kz5usizBxPaiFu3TPhB9XLviScvdUVSbSYJ0Z"
//第一个字母“p”表示私有数据命名空间。我们可以使用“h”来表示在中定义的哈希数据命名空间。
//私有状态/公用存储\u db.go
	coll := "pvWjtfSTXVK8WJus5s6zWoMIciXd7qHRZIusF9SkOS6m8XuHCiJDE9cCRuVerq22Na8qBL2ywDGFpVMIuzfyEXLjeJb0mMuH4cwewT6r1INOTOSYwrikwOLlT_fl0V1L7IQEwUBB8WCvRqSdj6j5-E5aGul_pv_0UeCdwWiyA_GrZmP7ocLzfj2vP8btigrajqdH-irLO2ydEjQUAvf8fiuxru9la402KmKRy457GgI98UHoUdqV3f3FCdR"

	truncatedChainName := "tob2g.y-z0f.qwp-rq5g4-ogid5g6oucyryg9sc16mz0t4vuak"
	truncatedEscapedNs := "w$m$cn$s$xi$v9$yo$iq$n$qy$nv$f$v$td$m8$xn$utvr$o$f"
	truncatedEscapedColl := "pv$wjtf$s$t$x$v$k8$w$jus5s6z$wo$m$ici$xd7q$h$r$z$i"
	assert.Equal(t, chainNameAllowedLength, len(truncatedChainName))
	assert.Equal(t, namespaceNameAllowedLength, len(truncatedEscapedNs))
	assert.Equal(t, collectionNameAllowedLength, len(truncatedEscapedColl))

	untruncatedDBName := chainName + "_" + ns + "$$" + coll
	hash := hex.EncodeToString(util.ComputeSHA256([]byte(untruncatedDBName)))
	expectedDBName := truncatedChainName + "_" + truncatedEscapedNs + "$$" + truncatedEscapedColl + "(" + hash + ")"
//<chainname的前50个字符（即chainname allowedlength）-+1个字符用于“'+<first 50个字符
//(i.e., namespaceNameAllowedLength) of escaped namespace> + 2 chars for '$$' + <first 50 chars
//（即collectionnameallowedlength）的转义集合>+1个字符用于'（'+<64个字符用于sha256哈希）
//（十六进制编码）的未加密链名“$coll>+1 char for”）'=219个字符
	expectedDBNameLength := 219

	namespace := ns + "$$" + coll
	constructedDBName := ConstructNamespaceDBName(chainName, namespace)
	assert.Equal(t, expectedDBNameLength, len(constructedDBName))
	assert.Equal(t, expectedDBName, constructedDBName)

//==方案2:链名称

	untruncatedDBName = chainName + "_" + ns
	hash = hex.EncodeToString(util.ComputeSHA256([]byte(untruncatedDBName)))
	expectedDBName = truncatedChainName + "_" + truncatedEscapedNs + "(" + hash + ")"
//<chainname的前50个字符（即chainname allowedlength）-+1个字符用于“'+<first 50个字符
//（即，namespacename allowedlength）的转义命名空间>+1个字符用于'（'+<64个字符用于sha256哈希）
//(hex encoding) of untruncated chainName_ns> + 1 char for ')' = 167 chars
	expectedDBNameLength = 167

	namespace = ns
	constructedDBName = ConstructNamespaceDBName(chainName, namespace)
	assert.Equal(t, expectedDBNameLength, len(constructedDBName))
	assert.Equal(t, expectedDBName, constructedDBName)
}
