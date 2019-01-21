
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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	ledgertestutil "github.com/hyperledger/fabric/core/ledger/testutil"
	"github.com/hyperledger/fabric/integration/runner"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const badConnectURL = "couchdb:5990"
const badParseConnectURL = "http://主机5432
const updateDocumentConflictError = "conflict"
const updateDocumentConflictReason = "Document update conflict."

var couchDBDef *CouchDBDef

func cleanup(database string) error {
//创建新连接
	couchInstance, err := CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
	if err != nil {
		fmt.Println("Unexpected error", err)
		return err
	}
	db := CouchDatabase{CouchInstance: couchInstance, DBName: database}
//删除测试数据库
	db.DropDatabase()
	return nil
}

type Asset struct {
	ID        string `json:"_id"`
	Rev       string `json:"_rev"`
	AssetName string `json:"asset_name"`
	Color     string `json:"color"`
	Size      string `json:"size"`
	Owner     string `json:"owner"`
}

var assetJSON = []byte(`{"asset_name":"marble1","color":"blue","size":"35","owner":"jerry"}`)

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
//读取core.yaml文件以获取默认配置。
	ledgertestutil.SetupCoreYAMLConfig()

//切换到CouchDB
	couchAddress, cleanup := couchDBSetup()
	defer cleanup()
	viper.Set("ledger.state.stateDatabase", "CouchDB")
	defer viper.Set("ledger.state.stateDatabase", "goleveldb")

	viper.Set("ledger.state.couchDBConfig.couchDBAddress", couchAddress)
//替换为正确的用户名/密码，例如
//如果在CouchDB上启用了用户安全性，则为admin/admin。
	viper.Set("ledger.state.couchDBConfig.username", "")
	viper.Set("ledger.state.couchDBConfig.password", "")
	viper.Set("ledger.state.couchDBConfig.maxRetries", 3)
	viper.Set("ledger.state.couchDBConfig.maxRetriesOnStartup", 20)
	viper.Set("ledger.state.couchDBConfig.requestTimeout", time.Second*35)
	viper.Set("ledger.state.couchDBConfig.createGlobalChangesDB", true)

//将日志级别设置为“调试”，以仅测试调试代码
	flogging.ActivateSpec("couchdb=debug")

//从配置参数创建couchdb定义
	couchDBDef = GetCouchDBDefinition()

//运行测试
	return m.Run()
}

func couchDBSetup() (addr string, cleanup func()) {
	externalCouch, set := os.LookupEnv("COUCHDB_ADDR")
	if set {
		return externalCouch, func() {}
	}

	couchDB := &runner.CouchDB{}
	if err := couchDB.Start(); err != nil {
		err := fmt.Errorf("failed to start couchDB: %s", err)
		panic(err)
	}
	return couchDB.Address(), func() { couchDB.Stop() }
}

func TestDBConnectionDef(t *testing.T) {

//创建新连接
	_, err := CreateConnectionDefinition(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB)
	assert.NoError(t, err, "Error when trying to create database connection definition")

}

func TestDBBadConnectionDef(t *testing.T) {

//创建新连接
	_, err := CreateConnectionDefinition(badParseConnectURL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB)
	assert.Error(t, err, "Did not receive error when trying to create database connection definition with a bad hostname")

}

func TestEncodePathElement(t *testing.T) {

	encodedString := encodePathElement("testelement")
	assert.Equal(t, "testelement", encodedString)

	encodedString = encodePathElement("test element")
	assert.Equal(t, "test%20element", encodedString)

	encodedString = encodePathElement("/test element")
	assert.Equal(t, "%2Ftest%20element", encodedString)

	encodedString = encodePathElement("/test element:")
	assert.Equal(t, "%2Ftest%20element:", encodedString)

	encodedString = encodePathElement("/test+ element:")
	assert.Equal(t, "%2Ftest%2B%20element:", encodedString)

}

func TestHealthCheck(t *testing.T) {
	client := &http.Client{}

//创建错误的couchdb实例
badURL := fmt.Sprintf("http://%s“，couchdbdef.url+”1”）//此端口上没有可用的coach db服务
	badConnectDef := CouchConnectionDef{URL: badURL, Username: "", Password: "",
		MaxRetries: 1, MaxRetriesOnStartup: 1, RequestTimeout: time.Second * 30}

	badCouchDBInstance := CouchInstance{badConnectDef, client, newStats(&disabled.Provider{})}
	err := badCouchDBInstance.HealthCheck(context.Background())
	assert.Error(t, err, "Health check should result in an error if unable to connect to couch db")
	assert.Contains(t, err.Error(), "failed to connect to couch db")

//创建一个好的couchdb实例
goodURL := fmt.Sprintf("http://%s“，couchdbdef.url）//此端口没有可用的coach db服务
	goodConnectDef := CouchConnectionDef{URL: goodURL, Username: "", Password: "",
		MaxRetries: 1, MaxRetriesOnStartup: 1, RequestTimeout: time.Second * 30}

	goodCouchDBInstance := CouchInstance{goodConnectDef, client, newStats(&disabled.Provider{})}
	err = goodCouchDBInstance.HealthCheck(context.Background())
	assert.NoError(t, err)
}

func TestBadCouchDBInstance(t *testing.T) {

//创建错误的连接定义
	badConnectDef := CouchConnectionDef{URL: badParseConnectURL, Username: "", Password: "",
		MaxRetries: 3, MaxRetriesOnStartup: 10, RequestTimeout: time.Second * 30}

	client := &http.Client{}

//创建错误的couchdb实例
	badCouchDBInstance := CouchInstance{badConnectDef, client, newStats(&disabled.Provider{})}

//创建错误的couchdatabase
	badDB := CouchDatabase{&badCouchDBInstance, "baddb", 1}

//测试连接错误的CreateCouchDatabase
	_, err := CreateCouchDatabase(&badCouchDBInstance, "baddbtest")
	assert.Error(t, err, "Error should have been thrown with CreateCouchDatabase and invalid connection")

//测试连接不正确的CreateSystemDatabaseSifNotexist
	err = CreateSystemDatabasesIfNotExist(&badCouchDBInstance)
	assert.Error(t, err, "Error should have been thrown with CreateSystemDatabasesIfNotExist and invalid connection")

//Test CreateDatabaseIfNotExist with bad connection
	err = badDB.CreateDatabaseIfNotExist()
	assert.Error(t, err, "Error should have been thrown with CreateDatabaseIfNotExist and invalid connection")

//测试连接错误的GetDatabaseInfo
	_, _, err = badDB.GetDatabaseInfo()
	assert.Error(t, err, "Error should have been thrown with GetDatabaseInfo and invalid connection")

//用错误的连接测试verifycouchconfig
	_, _, err = badCouchDBInstance.VerifyCouchConfig()
	assert.Error(t, err, "Error should have been thrown with VerifyCouchConfig and invalid connection")

//测试连接不良连接
	_, err = badDB.EnsureFullCommit()
	assert.Error(t, err, "Error should have been thrown with EnsureFullCommit and invalid connection")

//连接错误的测试DropDatabase
	_, err = badDB.DropDatabase()
	assert.Error(t, err, "Error should have been thrown with DropDatabase and invalid connection")

//连接不良的测试readDoc
	_, _, err = badDB.ReadDoc("1")
	assert.Error(t, err, "Error should have been thrown with ReadDoc and invalid connection")

//测试连接错误的saveDoc
	_, err = badDB.SaveDoc("1", "1", nil)
	assert.Error(t, err, "Error should have been thrown with SaveDoc and invalid connection")

//测试连接不良的deletedoc
	err = badDB.DeleteDoc("1", "1")
	assert.Error(t, err, "Error should have been thrown with DeleteDoc and invalid connection")

//连接不良的测试readdocrange
	_, _, err = badDB.ReadDocRange("1", "2", 1000)
	assert.Error(t, err, "Error should have been thrown with ReadDocRange and invalid connection")

//连接错误的测试QueryDocuments
	_, _, err = badDB.QueryDocuments("1")
	assert.Error(t, err, "Error should have been thrown with QueryDocuments and invalid connection")

//连接错误的测试BatchRetrieveDocumentMetadata
	_, err = badDB.BatchRetrieveDocumentMetadata(nil)
	assert.Error(t, err, "Error should have been thrown with BatchRetrieveDocumentMetadata and invalid connection")

//连接错误的测试批更新文档
	_, err = badDB.BatchUpdateDocuments(nil)
	assert.Error(t, err, "Error should have been thrown with BatchUpdateDocuments and invalid connection")

//连接错误的测试列表索引
	_, err = badDB.ListIndex()
	assert.Error(t, err, "Error should have been thrown with ListIndex and invalid connection")

//测试连接错误的CreateIndex
	_, err = badDB.CreateIndex("")
	assert.Error(t, err, "Error should have been thrown with CreateIndex and invalid connection")

//测试连接错误的DeleteIndex
	err = badDB.DeleteIndex("", "")
	assert.Error(t, err, "Error should have been thrown with DeleteIndex and invalid connection")

}

func TestDBCreateSaveWithoutRevision(t *testing.T) {

	database := "testdbcreatesavewithoutrevision"
	err := cleanup(database)
	assert.NoError(t, err, "Error when trying to cleanup  Error: %s", err)
	defer cleanup(database)

//新建实例和数据库对象
	couchInstance, err := CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
	assert.NoError(t, err, "Error when trying to create couch instance")
	db := CouchDatabase{CouchInstance: couchInstance, DBName: database}

//创建新数据库
	errdb := db.CreateDatabaseIfNotExist()
	assert.NoError(t, errdb, "Error when trying to create database")

//保存测试文档
	_, saveerr := db.SaveDoc("2", "", &CouchDoc{JSONValue: assetJSON, Attachments: nil})
	assert.NoError(t, saveerr, "Error when trying to save a document")

}

func TestDBCreateEnsureFullCommit(t *testing.T) {

	database := "testdbensurefullcommit"
	err := cleanup(database)
	assert.NoError(t, err, "Error when trying to cleanup  Error: %s", err)
	defer cleanup(database)

//新建实例和数据库对象
	couchInstance, err := CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
	assert.NoError(t, err, "Error when trying to create couch instance")
	db := CouchDatabase{CouchInstance: couchInstance, DBName: database}

//创建新数据库
	errdb := db.CreateDatabaseIfNotExist()
	assert.NoError(t, errdb, "Error when trying to create database")

//保存测试文档
	_, saveerr := db.SaveDoc("2", "", &CouchDoc{JSONValue: assetJSON, Attachments: nil})
	assert.NoError(t, saveerr, "Error when trying to save a document")

//确保完全承诺
	_, commiterr := db.EnsureFullCommit()
	assert.NoError(t, commiterr, "Error when trying to ensure a full commit")
}

func TestDBBadDatabaseName(t *testing.T) {

//使用有效的数据库名称混合大小写创建新实例和数据库对象
	couchInstance, err := CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
	assert.NoError(t, err, "Error when trying to create couch instance")
	_, dberr := CreateCouchDatabase(couchInstance, "testDB")
	assert.Error(t, dberr, "Error should have been thrown for an invalid db name")

//使用有效的数据库名称字母和数字创建新的实例和数据库对象
	couchInstance, err = CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
	assert.NoError(t, err, "Error when trying to create couch instance")
	_, dberr = CreateCouchDatabase(couchInstance, "test132")
	assert.NoError(t, dberr, "Error when testing a valid database name")

//使用有效的数据库名称（特殊字符）创建新实例和数据库对象
	couchInstance, err = CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
	assert.NoError(t, err, "Error when trying to create couch instance")
	_, dberr = CreateCouchDatabase(couchInstance, "test1234~!@#$%^&*()[]{}.")
	assert.Error(t, dberr, "Error should have been thrown for an invalid db name")

 /*使用无效的数据库名称创建新实例和数据库对象-太长/*
 couchinstance，err=createcouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 _u，dberr=createCouchDatabase（couchinstance，“a123456789012345678901234567890123456789012345678901234”+
  “567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890”+
  “1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456”+
  “789012345678901234567890123456789012345678901234567890”）。
 assert.error（t，dberr，“对于无效的数据库名称，应该引发错误”）。

}

func testdbbadconnection（t*testing.t）

 //新建实例和数据库对象
 //将maxretriesonStartup限制为3，以减少失败的时间
 {，Err:= CeaCeCouChutStand（BADCONTURTURL，CouCHDBDEF）用户名，CouCHDBDEF，密码，
  couchdbdef.maxretries，3，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.error（t，err，“错误应该是由于错误的连接而引发的”）。
}

func testbaddbcredentials（t*testing.t）

 数据库：=“testdbbadcredentials”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //新建实例和数据库对象
 _u，err=createCouchinstance（couchdbdef.url，“fred”，“fred”，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.error（t，err，“错误应该为错误的凭据引发”）。

}

func testdbcreatedatabaseandpersist（t*testing.t）

 //使用默认配置的maxretries测试create和persist
 testdbcreatedatabaseandpersist（t，couchdbdef.maxretries）

 //测试创建并保持0次重试
 testdbcreatedatabaseandpersist（t，0）

 //使用默认配置的maxretries测试批处理操作
 测试批处理操作（t，couchdbdef.maxretries）

 //用0次重试测试批处理操作
 testBatchBatchOperations(t, 0)

}

func testdbcreatedatabaseandpersist（t*testing.t，maxretries int）

 数据库：=“testdbcreatedatabaseandpersist”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //新建实例和数据库对象
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 //检索新数据库的信息并确保名称匹配
 dbresp，u，errdb：=db.getdatabaseinfo（）。
 assert.noError（t，errdb，“尝试检索数据库信息时出错”）。
 assert.equal（t，数据库，dbresp.dbname）

 //保存测试文档
 _, saveerr := db.SaveDoc("idWith/slash", "", &CouchDoc{JSONValue: assetJSON, Attachments: nil})
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //检索测试文档
 dGETRESP，ReadDoc，GETRR:= DB。
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 //将文档取消标记为资产结构
 资产价值：=&asset
 geterr=json.unmashal（dbgetresp.jsonvalue，&assetresp）
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 //验证所有者检索到的匹配项
 断言相等（t，“Jerry”，assetresp.owner）

 //保存测试文档
 _，saveerr=db.saveDoc（“1”，“，&couchDoc”jsonValue:assetjson，attachments:nil）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //检索测试文档
 dbgetresp，，geterr=db.readdoc（“1”）。
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 //将文档取消标记为资产结构
 assetresp=&asset
 geterr=json.unmashal（dbgetresp.jsonvalue，&assetresp）
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 //验证所有者检索到的匹配项
 断言相等（t，“Jerry”，assetresp.owner）

 //将所有者更改为Bob
 assetresp.owner=“鲍勃”

 //创建JSON的字节数组
 assetdocumented，：=json.marshal（assetresp）

 //保存更新后的测试文档
 _，saveerr=db.saveDoc（“1”，“，&couchDoc”jsonValue:assetDocUpdated，attachments:nil）
 assert.noerror（t，saveerr，“尝试保存更新的文档时出错”）。

 //检索更新后的测试文档
 dbgetresp，，geterr=db.readdoc（“1”）。
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 //将文档取消标记为资产结构
 assetresp=&asset
 json.unmashal（dbgetresp.jsonvalue和assetresp）

 //断言更新已保存和检索
 断言相等（t，“bob”，assetresp.owner）

 testbytes2：=[]字节（`test attachment 2`）

 附件2：=&attachmentinfo
 attachment2.attachmentbytes=测试字节2
 attachment2.contentType=“应用程序/octet流”
 attachment2.name=“数据”
 附件2：=[]*附件信息
 attachments2=附加（attachments2，attachment2）

 //用附件保存测试文档
 _，saveerr=db.saveDoc（“2”，“，&couchDoc”jsonValue:nil，attachments:attachments2）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //检索带有附件的测试文档
 dbgetresp，，geterr=db.readdoc（“2”）。
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 //验证附件中的文本是否正确
 测试附件：=dbgetresp.attachments[0].attachmentbytes
 断言.equal（t，testbytes2，testtach）

 测试字节3：=[]字节

 附件3：=&attachmentinfo
 attachment3.attachmentbytes=测试字节3
 attachment3.contentType=“应用程序/octet流”
 attachment3.name=“数据”
 附件3：=[]*附件信息
 attachments3=附加（attachments3，attachment3）

 //用零长度附件保存测试文档
 _，saveerr=db.saveDoc（“3”，“，&couchDoc”jsonValue:nil，attachments:attachments3）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //检索带有附件的测试文档
 dbgetresp，，geterr=db.readdoc（“3”）。
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 //验证附件中的文本是否正确，零字节
 testatach=dbgetresp.attachments[0].attachmentbytes
 断言.equal（t，testbytes3，testtach）

 testbytes4a：=[]字节（`test attachment 4a`）
 附件4：=&attachmentinfo
 attachment4A.attachmentBytes=测试字节4A
 attachment4a.contenttype=“应用程序/octet流”
 attachment4a.name=“数据1”

 testbytes4b：=[]字节（`test attachment 4b`）
 附件4b：=&attachmentinfo
 attachment4b.attachmentbytes=测试字节4b
 attachment4b.contenttype=“应用程序/octet流”
 attachment4b.name=“数据2”

 附件4：=[]*附件信息
 attachments4=附加（attachments4，attachment4a）
 attachments4 = append(attachments4, attachment4b)

 //用多个附件保存更新的测试文档
 _，saveerr=db.saveDoc（“4”，“，&couchDoc”jsonValue:assetJSon，attachments:attachments4）
 assert.noerror（t，saveerr，“尝试保存更新的文档时出错”）。

 //检索带有附件的测试文档
 dbgetresp，，geterr=db.readdoc（“4”）。
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 对于uu4，附件4：=范围dbgetresp.attachments

  当前名称：=attach4.name
  如果currentname==“数据1”
   断言.equal（t，testbytes4a，attach4.attachmentbytes）
  }
  如果currentname==“数据2”
   断言.equal（t，testbytes4b，attach4.attachmentbytes）
  }

 }

 测试字节5a：=[]字节（`测试附件5a`）
 附件5a：=&attachmentinfo
 attachment5a.attachmentbytes=测试字节5a
 attachment5a.contentType=“应用程序/octet流”
 附件5A.Name =“DATA1”

 测试字节5b：=[]字节
 附件5b：=&attachmentinfo
 attachment5b.attachmentbytes=测试字节5b
 attachment5b.contentType=“应用程序/octet流”
 attachment5b.name=“数据2”

 附件5：=[]*附件信息
 attachments5=附加（attachments5，attachment5a）
 attachments5=附加（attachments5，attachment5b）

 //用多个附件和零长度附件保存更新的测试文档
 _，saveerr=db.saveDoc（“5”，“，&couchDoc”jsonValue:assetJSon，attachments:attachments5）
 assert.noerror（t，saveerr，“尝试保存更新的文档时出错”）。

 //检索带有附件的测试文档
 dbgetresp，，geterr=db.readdoc（“5”）。
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 对于u，附件5：=范围dbgetresp.attachments

  当前名称：=attach5.name
  如果currentname==“数据1”
   断言.equal（t，testbytes5a，attach5.attachmentbytes）
  }
  如果currentname==“数据2”
   断言.equal（t，testbytes5b，attach5.attachmentbytes）
  }

 }

 //试图用无效的ID保存文档
 _，saveerr=db.savedoc（string（[]byte 0xff，0xfe，0xfd），“，&couchdoc jsonvalue:assetjson，attachments:nil）
 assert.error（t，saveerr，“保存具有无效ID的文档时应引发错误”）。

 //试图读取具有无效ID的文档
 _，u，readerr：=db.readdoc（字符串（[]字节0xff，0xfe，0xfd）））
 assert.error（t，readerr，“读取具有无效ID的文档时应引发错误”）。

 //删除数据库
 _u，errddrop：=db.dropdatabase（）。
 assert.noError（t，errdDrop，“删除数据库时出错”）。

 //确保获取丢失数据库的信息时引发错误
 _ux，errdbinfo：=db.getdatabaseinfo（）。
 assert.error（t，errdbinfo，“应为缺少的数据库引发错误”）。

 //试图将文档保存到已删除的数据库中
 _，saveerr=db.saveDoc（“6”，“，&couchDoc”jsonValue:assetjson，attachments:nil）
 assert.error（t，saveerr，“尝试保存到已删除的数据库时应引发错误”）。

 //试图读取已删除的数据库
 _，uuGetErr=db.readDoc（“6”）。
 assert.NoError(t, geterr, "Error should not have been thrown for a missing database, nil value is returned")

}

func testdbrequestTimeout（t*testing.t）

 数据库：=“TestDeBuQuestTimeOutlook”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //创建不可能短的超时
 不可能超时：=time.microsecond*1

 //创建一个超时将失败的新实例和数据库对象
 //同时使用maxretriesonstartup=3减少重试次数
 _u，err=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，，
  couchdbdef.maxretries，3，impossibleTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.error（t，err，“尝试创建具有连接超时的couchdb实例时，应该出现错误”）。

 //新建实例和数据库对象
 _u，err=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，，
  -1，3，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.error（t，err，“尝试创建数据库时应引发错误”）。

}

func testdbtimeoutconflictretry（t*testing.t）

 数据库：=“testDBTimeoutRetry”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //新建实例和数据库对象
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，3，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 //检索新数据库的信息并确保名称匹配
 dbresp，u，errdb：=db.getdatabaseinfo（）。
 assert.noError（t，errdb，“尝试检索数据库信息时出错”）。
 assert.equal（t，数据库，dbresp.dbname）

 //保存测试文档
 _，saveerr：=db.saveDoc（“1”，“，&couchDoc”jsonValue:assetjson，attachments:nil）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //检索测试文档
 _ux，geterr：=db.readdoc（“1”）。
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 //以无效的版本保存测试文档。这将导致重试
 _u，saveerr=db.saveDoc（“1”，“1-11111111111111111111111111”，&couchDoc json值：assetjson，附件：nil）
 assert.noerror（t，saveerr，“试图保存修订冲突的文档时出错”）。

 //删除版本无效的测试文档。这将导致重试
 删除错误：=db.deletedoc（“1”，“1-11111111111111111111111”）
 assert.noError（t，deleteerr，“试图删除修订冲突的文档时出错”）。

}

func testdbbadnumberofretries（t*testing.t）

 数据库：=“testDBBadRetries”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //新建实例和数据库对象
 _u，err=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，，
  -1，3，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.error（t，err，“尝试创建数据库时应引发错误”）。

}

func测试dbbadjson（t*testing.t）

 数据库：=“TestBdBdjson”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //新建实例和数据库对象
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 //检索新数据库的信息并确保名称匹配
 dbresp，u，errdb：=db.getdatabaseinfo（）。
 assert.noError（t，errdb，“尝试检索数据库信息时出错”）。
 assert.equal（t，数据库，dbresp.dbname）

 badjson：=[]字节（`“资产名称”`）

 //保存测试文档
 _，saveerr：=db.savedoc（“1”，“，&couchdoc”jsonvalue:badjson，attachments:nil）
 assert.error（t，saveerr，“应该为错误的JSON引发错误”）。

}

func测试前缀扫描（t*testing.t）

 数据库：=“testprifixscan”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //新建实例和数据库对象
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 //检索新数据库的信息并确保名称匹配
 dbresp，u，errdb：=db.getdatabaseinfo（）。
 assert.noError（t，errdb，“尝试检索数据库信息时出错”）。
 assert.equal（t，数据库，dbresp.dbname）

 //保存文档
 对于i：=0；i<20；i++
  ID1：=字符串（0）+字符串（i）+字符串（0）
  ID2：=字符串（0）+字符串（i）+字符串（1）
  ID3：=字符串（0）+字符串（i）+字符串（utf8.maxrune-1）
  _u，saveerr：=db.saveDoc（id1，“”，&couchDoc jsonValue:assetjson，attachments:nil）
  assert.noError（t，saveerr，“尝试保存文档时出错”）。
  _，saveerr=db.saveDoc（id2，“”，&couchDoc jsonValue:assetjson，attachments:nil）
  assert.noError（t，saveerr，“尝试保存文档时出错”）。
  _，saveerr=db.saveDoc（id3，“”，&couchDoc jsonValue:assetjson，attachments:nil）
  assert.noError（t，saveerr，“尝试保存文档时出错”）。

 }
 开始键：=string（0）+string（10）
 endkey：=startkey+string（utf8.maxrune）
 _ux，geterr：=db.readdoc（endkey）
 assert.noError（t，geterr，“尝试获取lastkey时出错”）。

 resultsptr，u，geterr：=db.readdocrange（开始键，结束键，1000）
 assert.noError（t，geterr，“尝试执行范围扫描时出错”）。
 断言.notnil（t，resultsptr）
 结果：=resultsptr
 assert.equal（t，3，len（results））。
 assert.equal（t，string（0）+string（10）+string（0），结果[0].id）
 assert.equal（t，string（0）+string（10）+string（1），结果[1].id）
 assert.equal（t，string（0）+string（10）+string（utf8.maxrune-1），结果[2].id）

 //删除数据库
 _u，errddrop：=db.dropdatabase（）。
 assert.noError（t，errdDrop，“删除数据库时出错”）。

 //检索新数据库的信息并确保名称匹配
 _ux，errdbinfo：=db.getdatabaseinfo（）。
 assert.error（t，errdbinfo，“应为缺少的数据库引发错误”）。

}

func testdbsaveattachment（t*testing.t）

 数据库：=“testdbsaveattachment”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 byte文本：=[]byte（`这是一个测试文档。这只是一个测试`）

 附件：=&attachmentinfo
 attachment.attachmentbytes=字节文本
 attachment.contentType=“文本/普通”
 attachment.length=uint64（长度（字节文本））
 attachment.name=“值字节”

 附件：=[]*attachmentinfo
 附件=附加（附件，附件）

 //新建实例和数据库对象
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 //保存测试文档
 _，saveerr：=db.saveDoc（“10”，“，&couchDoc”jsonValue:nil，attachments:attachments）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //尝试检索包含附件的更新测试文档
 couchDoc，u，geterr2：=db.readDoc（“10”）。
 assert.noError（t，geterr2，“尝试检索带附件的文档时出错”）。
 断言.notnil（t，couchDoc.附件）
 assert.equal（t，byteText，couchDoc.attachments[0].attachmentBytes）
 assert.equal（t，attachment.length，couchDoc.attachments[0].长度）

}

func testdbdeleteddocument（t*testing.t）

 数据库：=“testDBDeleteDocument”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //新建实例和数据库对象
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 //保存测试文档
 {，SaveRr:= Db. SaveDoc（“2”，“and”和CouCHDOC {JSONVals:AsSertJeon，附件：nIL}）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //Attempt to retrieve the test document
 _，u，readerr：=db.readdoc（“2”）。
 assert.noError（t，readerr，“尝试检索带有附件的文档时出错”）。

 //删除测试文档
 删除错误：=db.deletedoc（“2”，“）
 assert.noError（t，deleteerr，“尝试删除文档时出错”）。

 //尝试检索测试文档
 readValue，，：=db.readDoc（“2”）。
 断言.nil（t，readvalue）

}

func testdbdeletenonexistingdocument（t*testing.t）

 数据库：=“testdbdeletenonexistingdocument”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //新建实例和数据库对象
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 //保存测试文档
 删除错误：=db.deletedoc（“2”，“）
 assert.noError（t，deleteerr，“尝试删除不存在的文档时出错”）。
}

func testcouchdbversion（t*testing.t）

 错误：=checkcouchdbversion（“2.0.0”）。
 assert.noError（t，err，“对于有效版本，不应引发错误”）。

 err=checkcouchdbversion（“4.5.0”）。
 assert.noError（t，err，“对于有效版本，不应引发错误”）。

 err=checkcouchdbversion（“1.6.5.4”）。
 assert.error（t，err，“对于无效版本，应该引发错误”）。

 err=checkcouchdbversion（“0.0.0.0”）。
 assert.error（t，err，“对于无效版本，应该引发错误”）。

}

功能测试操作（t*testing.t）

 数据库：=“testindexoperations”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 bytejson1：=[]字节（`“”id“：”1“，”asset_name“：”marble1“，”color“：”blue“，”size“：1，”owner“：”jerry“`）
 bytejson2：=[]字节（`“”id“：”2“，”asset_name“：”marble2“，”color“：”red“，”size“：2，”owner“：”tom“`）
 bytejson3：=[]字节（`“”id“：”3“，”asset_name“：”marble3“，”color“：”green“，”size“：3，”owner“：”jerry“`）
 bytejson4：=[]字节（`“”id“：”4“，”asset_name“：”marble4“，”color“：”purple“，”size“：4，”owner“：”tom“`）
 bytejson5：=[]字节（`“”id“：”5“，”asset_name“：”marble5“，”color“：”blue“，”size“：5，”owner“：”jerry“`）
 bytejson6：=[]字节（`“”id“：”6“，”asset_name“：”marble6“，”color“：”white“，”size“：6，”owner“：”tom“`）
 bytejson7：=[]字节（`“”id“：”7“，”asset_name“：”marble7“，”color“：”white“，”size“：7，”owner“：”tom“`）
 bytejson8：=[]字节（`“”id“：”8“，”asset_name“：”marble8“，”color“：”white“，”size“：8，”owner“：”tom“`）
 bytejson9：=[]字节（`“”id“：”9“，”asset_name“：”marble9“，”color“：”white“，”size“：9，”owner“：”tom“`）
 bytejson10：=[]字节（`“”id“：”10“，”asset_name“：”marble10“，”color“：”white“，”size“：10，”owner“：”tom“`）

 //创建新的实例和数据库对象---------------------------------------------------
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 批更新文档：=[]*couchDoc

 batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:bytejson1，attachments:nil）
 batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:bytejson2，attachments:nil）
 batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:bytejson3，attachments:nil）
 batchUpdateDocs = append(batchUpdateDocs, &CouchDoc{JSONValue: byteJSON4, Attachments: nil})
 batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:bytejson5，attachments:nil）
 batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:bytejson6，attachments:nil）
 batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:bytejson7，attachments:nil）
 batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:bytejson8，attachments:nil）
 batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:bytejson9，attachments:nil）
 batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:bytejson10，attachments:nil）

 _uuErr=db.batchUpdateDocuments（批更新文档）
 assert.noError（t，err，“添加批文档时出错”）。

 //创建索引定义
 indexdefsize：=`“index”：“fields”：[“size”：“desc”]，“ddoc”：“indexsizesortdoc”，“name”：“indexsizesortname”，“type”：“json”`

 //Create the index
 _u，err=db.createindex（indexdefsize）
 assert.noError（t，err，“创建索引时引发错误”）。

 //检索索引列表
 //索引创建/删除后，由于couchdb索引列表被异步更新，因此延迟100毫秒
 时间.睡眠（100*时间.毫秒）
 ListResult，错误：=db.ListIndex（）。
 assert.noError（t，err，“检索索引时引发错误”）。

 //只应返回一个项
 assert.equal（t，1，len（listreult））。

 //验证返回的定义
 对于uu，elem：=范围列表结果
  assert.equal（t，“indexsizesortdoc”，elem.designdocument）
  assert.equal（t，“indexsizesortname”，elem.name）
  //确保索引定义正确，couchdb 2.1.1还将返回“partial_filter_selector”：
  assert.equal（t，true，strings.contains（elem.definition，`“fields”：[“size”：“desc”]`））
 }

 //创建没有设计文档或名称的索引定义
 indexdefcolor：=`“index”：“fields”：[“color”：“desc”]`

 //创建索引
 _u，err=db.createindex（indexdefcolor）
 assert.noError（t，err，“创建索引时引发错误”）。

 //检索索引列表
 //索引创建/删除后，由于couchdb索引列表被异步更新，因此延迟100毫秒
 时间.睡眠（100*时间.毫秒）
 listResult，err=db.listIndex（）。
 assert.noError（t，err，“检索索引时引发错误”）。

 //返回两个索引
 assert.equal（t，2，len（listreult））。

 //Delete the named index
 err=db.deleteIndex（“indexSizeSortDoc”，“indexSizeSortName”）。
 assert.noError（t，err，“删除索引时引发错误”）。

 //检索索引列表
 //索引创建/删除后，由于couchdb索引列表被异步更新，因此延迟100毫秒
 时间.睡眠（100*时间.毫秒）
 listResult，err=db.listIndex（）。
 assert.noError（t，err，“检索索引时引发错误”）。

 //应该返回一个索引
 assert.equal（t，1，len（listreult））。

 //删除未命名索引
 对于uu，elem：=范围列表结果
  错误=db.deleteindex（elem.designdocument，elem.name）
  assert.noError（t，err，“删除索引时引发错误”）。
 }

 //检索索引列表
 //索引创建/删除后，由于couchdb索引列表被异步更新，因此延迟100毫秒
 时间.睡眠（100*时间.毫秒）
 listResult，err=db.listIndex（）。
 assert.noError（t，err，“检索索引时引发错误”）。
 assert.equal（t，0，len（listreult））。

 //创建一个按降序排序的查询字符串，这将需要一个索引
 querystring：=`“selector”：“size”：“$gt”：0，”“fields”：[“id”，“rev”，“owner”，“asset_name”，“color”，“size”]，“sort”：[“size”：“desc”]，“limit”：10，“skip”：0 `

 //执行带有排序的查询，这将引发异常
 _x，u，err=db.querydocuments（querystring）
 assert.error（t，err，“在没有有效索引的情况下查询时应引发错误”）。

 //创建索引
 _u，err=db.createindex（indexdefsize）
 assert.noError（t，err，“创建索引时引发错误”）。

 //索引创建/删除后，由于couchdb索引列表被异步更新，因此延迟100毫秒
 时间.睡眠（100*时间.毫秒）

 //使用索引执行查询，该操作应成功
 _x，u，err=db.querydocuments（querystring）
 assert.noerror（t，err，“使用索引查询时引发错误”）。

 //创建另一个索引定义
 indexDefSize = `{"index":{"fields":[{"data.size":"desc"},{"data.owner":"desc"}]},"ddoc":"indexSizeOwnerSortDoc", "name":"indexSizeOwnerSortName","type":"json"}`

 //创建索引
 dbresp，错误：=db.createindex（indexdefsize）
 assert.noError（t，err，“创建索引时引发错误”）。

 //验证是否为创建索引而“创建”响应
 assert.equal（t，“已创建”，dbresp.result）

 //索引创建/删除后，由于couchdb索引列表被异步更新，因此延迟100毫秒
 时间.睡眠（100*时间.毫秒）

 //更新索引
 dbresp，err=db.createindex（indexdefsize）
 assert.noError（t，err，“创建索引时引发错误”）。

 //验证更新的响应是否“存在”
 assert.equal（t，“存在”，dbresp.result）

 //检索索引列表
 //索引创建/删除后，由于couchdb索引列表被异步更新，因此延迟100毫秒
 时间.睡眠（100*时间.毫秒）
 listResult，err=db.listIndex（）。
 assert.noError（t，err，“检索索引时引发错误”）。

 //应该只有两个定义
 assert.equal（t，2，len（listreult））。

 //使用无效的JSON创建无效的索引定义
 indexDefSize=`“index”“fields”：[“data.size”：“desc”，“data.owner”：“desc”]，“ddoc”：“indexSizeOwnerSortDoc”，“name”：“indexSizeOwnerSortName”，“type”：“json”`

 //创建索引
 _u，err=db.createindex（indexdefsize）
 错误（T，Err，“错误的JSON应该被抛出错误”）

 //使用有效的JSON和无效的索引定义创建无效的索引定义
 indexdefsize=`“index”：“fields2”：[“data.size”：“desc”，“data.owner”：“desc”]，“ddoc”：“indexsizeOwnerSortDoc”，“name”：“indexsizeOwnerSortName”，“type”：“json”`

 //创建索引
 _u，err=db.createindex（indexdefsize）
 错误（t，rr，“错误的索引定义应该已经抛出错误”）

}

func testrichquery（t*testing.t）

 ByTEJSON01::[]字节（{“AsSeTyNe”）：“MalBe01”，“颜色”：“蓝色”，“Stand”：1，“所有者”：“JeRy”}）
 bytejson02：=[]字节（`“资产名称”：“marble02”，“color”：“red”，“size”：2，“owner”：“tom”`）
 bytejson03：=[]字节（`“资产名称”：“Marble03”，“颜色”：“绿色”，“大小”：3，“所有者”：“Jerry”`）
 byteJSON04 := []byte(`{"asset_name":"marble04","color":"purple","size":4,"owner":"tom"}`)
 bytejson05：=[]字节（`“资产名称”：“Marble05”，“颜色”：“蓝色”，“大小”：5，“所有者”：“Jerry”`）
 bytejson06：=[]字节（`“资产名称”：“marble06”，“color”：“white”，“size”：6，“owner”：“tom”`）
 bytejson07：=[]字节（`“资产名称”：“marble07”，“color”：“white”，“size”：7，“owner”：“tom”`）
 bytejson08：=[]字节（`“资产名称”：“marble08”，“color”：“white”，“size”：8，“owner”：“tom”`）
 bytejson09：=[]字节（`“资产名称”：“marble09”，“color”：“white”，“size”：9，“owner”：“tom”`）
 bytejson10：=[]字节（`“资产名称”：“大理石10”，“颜色”：“白色”，“大小”：10，“所有者”：“汤姆”`）
 bytejson11：=[]字节（`“资产名称”：“Marble11”，“颜色”：“绿色”，“大小”：11，“所有者”：“Tom”`）
 bytejson12：=[]字节（`“资产名称”：“大理石12”，“颜色”：“绿色”，“大小”：12，“所有者”：“Frank”`）

 附件1：=&attachmentinfo
 attachment1.attachmentbytes=[]字节（`marble01-测试附件`）
 attachment1.contentType=“应用程序/octet流”
 attachment1.Name = "data"
 附件1：=[]*附件信息
 attachments1 = append(attachments1, attachment1)

 附件2：=&attachmentinfo
 attachment2.attachmentbytes=[]字节（`marble02-测试附件`）
 attachment2.contentType=“应用程序/octet流”
 attachment2.name=“数据”
 附件2：=[]*附件信息
 attachments2=附加（attachments2，attachment2）

 附件3：=&attachmentinfo
 attachment3.attachmentbytes=[]字节（`marble03-测试附件`）
 attachment3.contentType=“应用程序/octet流”
 attachment3.name=“数据”
 附件3：=[]*附件信息
 attachments3=附加（attachments3，attachment3）

 附件4：=&attachmentinfo
 attachment4.attachmentbytes=[]字节（`marble04-测试附件`）
 attachment4.contentType=“应用程序/octet流”
 attachment4.name=“数据”
 附件4：=[]*附件信息
 attachments4=附加（attachments4，attachment4）

 附件5：=&attachmentinfo
 attachment5.attachmentbytes=[]字节（`marble05-测试附件`）
 attachment5.contentType=“应用程序/octet流”
 attachment5.name=“数据”
 附件5：=[]*附件信息
 attachments5=附加（attachments5，attachment5）

 附件6：=&attachmentinfo
 attachment6.attachmentbytes=[]字节（`marble06-测试附件`）
 attachment6.contentType=“应用程序/octet流”
 attachment6.name=“数据”
 附件6：=[]*附件信息
 attachments6=附加（attachments6，attachment6）

 附件7：=&attachmentinfo
 AtthChan7.7 AtthCuxBythys= []字节（'MalBuff07-测试附件）
 attachment7.contentType=“应用程序/octet流”
 attachment7.Name = "data"
 附件7：=[]*附件信息
 attachments7=附加（attachments7，attachment7）

 附件8：=&attachmentinfo
 attachment8.attachmentbytes=[]字节（`marble08-测试附件`）
 attachment8.ContentType = "application/octet-stream"
 attachment7.name=“数据”
 附件8：=[]*附件信息
 attachments8=附加（attachments8，attachment8）

 附件9：=&attachmentinfo
 attachment9.attachmentbytes=[]字节（`marble09-测试附件`）
 attachment9.contentType=“应用程序/octet流”
 attachment9.name=“数据”
 attachments9 := []*AttachmentInfo{}
 attachments9=附加（attachments9，attachment9）

 附件10：=&attachmentinfo
 AtthChans10AtjChansBythy[]]字节（“MalLab10--测试附件”）
 attachment10.contentType=“应用程序/octet流”
 attachment10.Name = "data"
 附件10：=[]*附件信息
 attachments10=附加（attachments10，attachment10）

 附件11：=&attachmentinfo
 attachment11.AttachmentBytes = []byte(`marble11 - test attachment`)
 attachment11.ContentType = "application/octet-stream"
 attachment11.name=“数据”
 附件11：=[]*附件信息
 attachments11=附加（attachments11，attachment11）

 附件12：=&attachmentinfo
 attachment12.attachmentbytes=[]byte（`marble12-测试附件`）
 attachment12.contentType=“应用程序/octet流”
 attachment12.name=“数据”
 附件12：=[]*附件信息
 attachment12=附加（attachment12，attachment12）

 数据库：=“testrichquery”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //创建新的实例和数据库对象---------------------------------------------------
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 //保存测试文档
 _，saveerr：=db.savedoc（“marble01”，“”，&couchdoc jsonvalue:bytejson01，attachments:attachments1）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 _，saveerr=db.saveDoc（“marble02”，“”，&couchDoc jsonValue:bytejson02，attachments:attachments2）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 _，saveerr=db.savedoc（“marble03”，“”，&couchdoc jsonvalue:bytejson03，attachments:attachments3）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 _，saveerr=db.saveDoc（“marble04”，“”，&couchDoc jsonValue:bytejson04，attachments:attachments4）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 _，saveerr=db.savedoc（“marble05”，“”，&couchdoc jsonvalue:bytejson05，attachments:attachments5）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 _，saveerr=db.savedoc（“marble06”，“”，&couchdoc jsonvalue:bytejson06，attachments:attachment6）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 _，saveerr=db.savedoc（“marble07”，“”，&couchdoc jsonvalue:bytejson07，attachments:attachments7）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 SaveDoc，“SaveReR= db”（“MalBe08”，“and CoucDoc {jSunValue:ByTEJSON08，附件：AutoSerts8}”）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 _，saveerr=db.savedoc（“marble09”，“”，&couchdoc jsonvalue:bytejson09，attachments:attachments9）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 _，saveerr=db.savedoc（“marble10”，“”，&couchdoc jsonvalue:bytejson10，attachments:attachments10）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 _，saveerr=db.savedoc（“marble11”，“”，&couchdoc jsonvalue:bytejson11，attachments:attachments11）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //保存测试文档
 _，saveerr=db.savedoc（“marble12”，“”，&couchdoc jsonvalue:bytejson12，attachments:attachments12）
 assert.noError（t，saveerr，“尝试保存文档时出错”）。

 //带有无效JSON的测试查询----------------------------------------------------------------
 查询字符串：=`选择器“：”所有者“：”

 _x，u，err=db.querydocuments（querystring）
 assert.error（t，err，“错误应该为错误的JSON引发”）。

 //用对象测试查询---------------------------------------------------------------------
 querystring=`“selector”：“owner”：“$eq”：“jerry”`

 queryResult, _, err := db.QueryDocuments(queryString)
 assert.noError（t，err，“尝试执行查询时出错”）。

 //owner=“jerry”应该有3个结果
 assert.Equal(t, 3, len(queryResult))

 //使用隐式运算符测试查询------------------------------------------------------------
 querystring=`“selector”：“owner”：“jerry”`

 queryresult，，err=db.querydocuments（querystring）
 assert.noError（t，err，“尝试执行查询时出错”）。

 //owner=“jerry”应该有3个结果
 assert.equal（t，3，len（queryresult））。

 //用指定字段测试查询---------------------------------------------------------------------
 querystring=`“selector”：“owner”：“$eq”：“jerry”，“fields”：[“owner”，“asset_name”，“color”，“size”]`

 queryresult，，err=db.querydocuments（querystring）
 assert.noError（t，err，“尝试执行查询时出错”）。

 //owner=“jerry”应该有3个结果
 assert.equal（t，3，len（queryresult））。

 //Test query with a leading operator   -------------------------------------------------------------------
 querystring=`“selector”：“$or”：[”owner“：”$eq“：”jerry“”，“owner”：“$eq”：“frank”]`

 queryresult，，err=db.querydocuments（querystring）
 assert.noError（t，err，“尝试执行查询时出错”）。

 //owner=“jerry”或owner=“frank”应该有4个结果
 assert.equal（t，4，len（queryresult））。

 //测试查询隐式和显式运算符-------------------------------------------------------------
 querystring=`“selector”：“color”：“green”，“$or”：[“owner”：“tom”，“owner”：“frank”]`

 queryresult，，err=db.querydocuments（querystring）
 assert.noError（t，err，“尝试执行查询时出错”）。

 //对于color=“green”和（owner=“jerry”或owner=“frank”），应该有2个结果。
 assert.equal（t，2，len（queryresult））。

 //使用前导运算符测试查询---------------------------------------------------------------------
 querystring=`“selector”：“$and”：[”size“：”$gte“：2，”size“：”$lte“：5]`

 queryresult，，err=db.querydocuments（querystring）
 assert.noError（t，err，“尝试执行查询时出错”）。

 //对于大小＞2和大小<＝5，应该有4个结果。
 assert.equal（t，4，len（queryresult））。

 //使用前导和嵌入运算符测试查询------------------------------------------------------
 querystring=`“selector”：“$and”：[”size“：”$gte“：3，”size“：”$lte“：10，”$not“：”size“：7]`

 queryresult，，err=db.querydocuments（querystring）
 assert.noError（t，err，“尝试执行查询时出错”）。

 //对于大小大于等于3且大小小于等于10而不是7，应该有7个结果
 assert.equal（t，7，len（queryresult））。

 //使用前导运算符和对象数组测试查询-----------------------------------------------------------
 querystring=`“selector”：“$and”：[”size“：”$gte“：2，”size“：”$lte“：10，”$nor“：[”size“：3，”size“：5，”size“：7]”`

 queryresult，，err=db.querydocuments（querystring）
 assert.noError（t，err，“尝试执行查询时出错”）。

 //对于大小大于等于2且大小小于等于10而不是3、5或7，应该有6个结果
 assert.equal（t，6，len（queryresult））。

 //测试范围查询------------------------------------------------------------------------------------
 QueRebug结果，ReadDocRange，（MARBLU02），“MARBL06 06”，10000）
 assert.noError（t，err，“尝试执行范围查询时出错”）。

 //应该有4个结果
 assert.equal（t，4，len（queryresult））。

 //检索到的附件应正确
 assert.equal（t，attachment2.attachmentbytes，queryresult[0].attachments[0].attachmentbytes）
 assert.equal（t，attachment3.attachmentbytes，queryresult[1].attachments[0].attachmentbytes）
 assert.equal（t，attachment4.attachmentbytes，queryresult[2].attachments[0].attachmentbytes）
 assert.equal（t，attachment5.attachmentbytes，queryresult[3].attachments[0].attachmentbytes）

 //测试tom的查询----------------------------------------------------------------
 querystring=`“selector”：“owner”：“$eq”：“tom”`

 queryresult，，err=db.querydocuments（querystring）
 assert.noError（t，err，“尝试执行查询时出错”）。

 //所有者应该有8个结果=“汤姆”
 assert.equal（t，8，len（queryresult））。

 //测试带有限制的tom的查询-----------------------------------------------------------
 querystring=`“selector”：“owner”：“$eq”：“tom”，“limit”：2 `

 queryresult，，err=db.querydocuments（querystring）
 assert.noError（t，err，“尝试执行查询时出错”）。

 //owner=“tom”应该有2个结果，限制为2
 assert.equal（t，2，len（queryresult））。

 //创建索引定义
 indexdefsize：=`“index”：“fields”：[“size”：“desc”]，“ddoc”：“indexsizesortdoc”，“name”：“indexsizesortname”，“type”：“json”`

 //创建索引
 _u，err=db.createindex（indexdefsize）
 assert.noError（t，err，“创建索引时引发错误”）。

 //索引创建/删除后，由于couchdb索引列表被异步更新，因此延迟100毫秒
 时间.睡眠（100*时间.毫秒）

 //使用有效索引测试查询----------------------------------------------------------------
 querystring=`“selector”：“size”：“$gt”：0，”“使用index”：[“indexsizesortdoc”，“indexsizesortname”]`

 _x，u，err=db.querydocuments（querystring）
 assert.noError（t，err，“尝试使用有效索引执行查询时出错”）。

}

func testbatchbatchoperations（t*testing.t，maxretries int）

 bytejson01：=[]byte（`“”id“：”marble01“，”asset\u name“：”marble01“，”color“：”blue“，”size“：”1“，”owner“：”jerry“`）
 ByTEJSON02:= []字节（{“{ ID”）：“MalBe02”，“AsSeTyNe”）：“MalBe02”，“颜色”：“红色”，“大小”：“2”，“所有者”：“Tom”}）
 bytejson03：=[]字节（`“_id”：“marble03”，“asset_name”：“marble03”，“color”：“green”，“size”：“3”，“owner”：“jerry”`）
 bytejson04：=[]字节（`“_id”：“marble04”，“asset_name”：“marble04”，“color”：“purple”，“size”：“4”，“owner”：“tom”`）
 bytejson05：=[]字节（`“_id”：“marble05”，“asset_name”：“marble05”，“color”：“blue”，“size”：“5”，“owner”：“jerry”`）
 byteJSON06 := []byte(`{"_id":"marble06#$&'()*+,/:;=?@[]","asset_name":"marble06#$&'()*+,/:;=?@[]","color":"blue","size":"6","owner":"jerry"}`)

 附件1：=&attachmentinfo
 attachment1.attachmentbytes=[]字节（`marble01-测试附件`）
 attachment1.contentType=“应用程序/octet流”
 附件=名称“数据”
 附件1：=[]*附件信息
 attachments1=附加（attachments1，attachment1）

 附件2：=&attachmentinfo
 attachment2.attachmentbytes=[]字节（`marble02-测试附件`）
 attachment2.contentType=“应用程序/octet流”
 attachment2.name=“数据”
 附件2：=[]*附件信息
 attachments2=附加（attachments2，attachment2）

 附件3：=&attachmentinfo
 attachment3.attachmentbytes=[]字节（`marble03-测试附件`）
 attachment3.contentType=“应用程序/octet流”
 attachment3.name=“数据”
 附件3：=[]*附件信息
 attachments3=附加（attachments3，attachment3）

 附件4：=&attachmentinfo
 attachment4.attachmentbytes=[]字节（`marble04-测试附件`）
 attachment4.contentType=“应用程序/octet流”
 attachment4.name=“数据”
 附件4：=[]*附件信息
 attachments4=附加（attachments4，attachment4）

 附件5：=&attachmentinfo
 attachment5.attachmentbytes=[]字节（`marble05-测试附件`）
 attachment5.contentType=“应用程序/octet流”
 attachment5.name=“数据”
 附件5：=[]*附件信息
 attachments5=附加（attachments5，attachment5）

 附件6：=&attachmentinfo
 attachment6.attachmentbytes=[]字节（`marble06$&'（）*+，/：；=？@[]-测试附件`）
 attachment6.contentType=“应用程序/octet流”
 attachment6.name=“数据”
 附件6：=[]*附件信息
 attachments6=附加（attachments6，attachment6）

 数据库：=“测试批处理”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //创建新的实例和数据库对象---------------------------------------------------
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 批更新文档：=[]*couchDoc

 值1：=&couchDoc json值：bytejson01，附件：附件1
 值2：=&couchDoc json值：bytejson02，附件：附件2
 值3：=&couchDoc json值：bytejson03，附件：附件3
 值4：=&couchDoc json值：bytejson04，附件：附件4
 值5：=&couchDoc json值：bytejson05，附件：附件5
 值6：=&couchDoc json值：bytejson06，附件：附件6

 batchUpdateDocs=append（batchUpdateDocs，值1）
 batchUpdateDocs=append（batchUpdateDocs，值2）
 batchUpdateDocs=append（batchUpdateDocs，值3）
 batchUpdateDocs=append（batchUpdateDocs，值4）
 batchUpdateDocs=append（batchUpdateDocs，值5）
 batchUpdateDocs=append（batchUpdateDocs，值6）

 batchUpdateRep，错误：=db.batchUpdateDocuments（batchUpdateDocs）
 assert.noError（t，err，“尝试更新一批文档时出错”）。

 //检查以确保每个批更新响应都成功
 对于uu，updatedDoc：=range batchupdateresp
  断言.equal（t，true，updatedoc.ok）
 }

 //——————————————————————————————————————————————————————————————————————————————————————————————————————————————
 //测试检索json
 dbgetresp，u，geterr：=db.readdoc（“marble01”）。
 assert.noError（t，geterr，“尝试读取文档时出错”）。

 资产价值：=&asset
 geterr=json.unmashal（dbgetresp.jsonvalue，&assetresp）
 assert.noError（t，geterr，“尝试检索文档时出错”）。
 //验证所有者检索到的匹配项
 断言相等（t，“Jerry”，assetresp.owner）

 //——————————————————————————————————————————————————————————————————————————————————————————————————————————————
 //使用带有URL特殊字符的ID测试检索JSON，
 //这将确认批处理文档ID和URL ID是一致的，即使它们包含特殊字符
 dbgetresp，，geterr=db.readdoc（“marble06$&'（）*+，/：；=？”@ [ ] ]
 assert.noError（t，geterr，“尝试读取文档时出错”）。

 assetresp=&asset
 geterr=json.unmashal（dbgetresp.jsonvalue，&assetresp）
 assert.noError（t，geterr，“尝试检索文档时出错”）。
 //验证所有者检索到的匹配项
 断言相等（t，“Jerry”，assetresp.owner）

 //——————————————————————————————————————————————————————————————————————————————————————————————————————————————
 //测试检索二进制
 dbgetresp，，geterr=db.readdoc（“marble03”）。
 assert.noError（t，geterr，“尝试读取文档时出错”）。
 //检索附件
 附件：=dbgetresp.attachments
 //只保存了一个，所以取第一个
 检索附件：=附件[0]
 //验证文本匹配
 assert.equal（t，retrievedAttachment.AttachmentBytes，Attachment3.AttachmentBytes）
 //——————————————————————————————————————————————————————————————————————————————————————————————————————————————
 //测试错误更新
 batchUpdateDocs=[]*couchDoc
 batchUpdateDocs=append（batchUpdateDocs，值1）
 batchUpdateDocs=append（batchUpdateDocs，值2）
 batchUpdateRep，err=db.batchUpdateDocuments（batchUpdateDocs）
 assert.noError（t，err，“尝试更新一批文档时出错”）。
 //没有提供修订，因此这两个更新应该失败
 //验证“OK”字段是否返回为false
 对于uu，updatedDoc：=range batchupdateresp
  断言.equal（t，false，updatedoc.ok）
  assert.equal（t，updateDocumentConflictError，updateDoc.Error）
  assert.equal（t，updateDocumentConflictReason，updateDoc.Reason）
 }

 //——————————————————————————————————————————————————————————————————————————————————————————————————————————————
 //测试批检索密钥并更新

 var键[]字符串

 keys=append（keys，“marble01”）。
 键=附加（键，“Marble03”）

 batchRevs，错误：=db.batchRetrieveDocumentMetadata（键）
 assert.noError（t，err，“尝试检索修订时出错”）。

 batchUpdateDocs=[]*couchDoc

 //遍历修订文档
 对于，revDoc：=range batchrevs
  如果revDoc.id=“Marble01”
   //用rev更新json并添加到批处理中
   marble01文档：=addRevisionAndDeleteStatus（revDoc.rev，bytejson01，false）
   batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:marble01Doc，attachments:attachments1）
  }

  如果revDoc.id=“Marble03”
   //用rev更新json并添加到批处理中
   marble03doc：=addRevisionAndDeleteStatus（revDoc.rev，bytejson03，false）
   batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:marble03Doc，attachments:attachments3）
  }
 }

 //用批更新couchdb
 batchUpdateRep，err=db.batchUpdateDocuments（batchUpdateDocs）
 assert.noError（t，err，“尝试更新一批文档时出错”）。
 //检查以确保每个批更新响应都成功
 对于uu，updatedDoc：=range batchupdateresp
  断言.equal（t，true，updatedoc.ok）
 }

 //——————————————————————————————————————————————————————————————————————————————————————————————————————————————
 //测试批删除

 键=[]字符串

 键=附加（键，“Marble02”）
 键=附加（键，“Marble04”）

 batchRevs，err=db.batchRetrieveDocumentMetadata（键）
 assert.noError（t，err，“尝试检索修订时出错”）。

 batchUpdateDocs=[]*couchDoc

 //遍历修订文档
 对于，revDoc：=range batchrevs
  如果revDoc.id=“Marble02”
   //用rev更新json并添加到批处理中
   marble02doc：=addRevisionAndDeleteStatus（revDoc.rev，bytejson02，真）
   batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:marble02Doc，attachments:attachments1）
  }
  如果revDoc.id=“Marble04”
   //用rev更新json并添加到批处理中
   marble04doc：=addRevisionAndDeleteStatus（RevDoc.Rev，BytejSon04，真）
   batchUpdateDocs=append（batchUpdateDocs，&couchDoc jsonValue:marble04Doc，attachments:attachments3）
  }
 }

 //用批更新couchdb
 batchUpdateRep，err=db.batchUpdateDocuments（batchUpdateDocs）
 assert.noError（t，err，“尝试更新一批文档时出错”）。

 //检查以确保每个批更新响应都成功
 对于uu，updatedDoc：=range batchupdateresp
  断言.equal（t，true，updatedoc.ok）
 }

 //检索测试文档
 dbgetresp，，geterr=db.readdoc（“marble02”）。
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 //断言值已被删除
 断言.nil（t，dbgetresp）

 //检索测试文档
 dbgetresp，，geterr=db.readdoc（“marble04”）。
 assert.noError（t，geterr，“尝试检索文档时出错”）。

 //断言值已被删除
 断言.nil（t，dbgetresp）

}

//addRevisionAndDeleteStatus将version和chaincodeID的键添加到json值
func addRevisionAndDeleteStatus（修订字符串，值为[]字节，已删除bool）[]字节

 //创建版本映射
 jsonmap：=make（map[string]接口）

 json.unmashal（值和jsonmap）

 //添加修订
 如有修改！=“{”
  jsonmap[“版次”]=版次
 }

 //如果要删除此记录，请将“ou deleted”属性设置为true
 如果已删除{
  jsonmap[“删除”]=真
 }
 //将数据封送到字节数组
 返回json，：=json.marshal（jsonmap）

 返回返回

}

func测试数据库安全设置（t*testing.t）

 数据库：=“testDBSecuritySettings”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //创建新的实例和数据库对象---------------------------------------------------
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 //创建数据库安全对象
 安全类型配置：=&databasesecurity
 securityPermissions.admins.names=append（securityPermissions.admins.names，“admin”）。
 securityPermissions.members.names=append（securityPermissions.members.names，“admin”）。

 //应用安全性
 错误=db.applyDatabaseSecurity（SecurityPermissions）
 assert.noerror（t，err，“尝试应用数据库安全性时出错”）。

 //检索数据库安全性
 databasesecurity，错误：=db.getdatabasesecurity（）
 assert.noerror（t，err，“检索数据库安全性时出错”）。

 //验证对管理员的检索
 assert.equal（t，“admin”，databasesecurity.admins.name[0]）

 //验证成员的检索
 assert.equal（t，“admin”，databasesecurity.members.names[0]）

 //创建空的数据库安全对象
 securityPermissions=&databaseSecurity

 //应用安全性
 错误=db.applyDatabaseSecurity（SecurityPermissions）
 assert.noerror（t，err，“尝试应用数据库安全性时出错”）。

 //检索数据库安全性
 databasesecurity，err=db.getdatabasesecurity（）。
 assert.noerror（t，err，“检索数据库安全性时出错”）。

 //验证对管理员的检索，应为空数组
 assert.equal（t，0，len（databasesecurity.admins.name））。

 //验证成员的检索，应为空数组
 assert.equal（t，0，len（databasesecurity.members.name））。

}

带特殊字符的func testurlw（t*testing.t）

 数据库：=“testdb+，带+加号”
 错误：=清理（数据库）
 assert.noerror（t，err，“尝试清除错误时出错：%s”，err）
 延迟清理（数据库）

 //分析构造的URL
 finalURL，错误：=url.parse（“http://127.0.0.1:5984”）
 NoError（t，rr，“解析CouCHDB URL时抛出的错误”）

 //用多个路径元素测试constructcouchdburl函数
 couchdburl：=constructcouchdburl（finalurl，database，“_index”，“designdoc”，“json”，“indexname”）。
 assert.equal（t，“http://127.0.0.1:5984/testdb%2bwith%2plus\u sign/\u index/designdoc/json/indexname”，couchdburl.string（））

 //创建新的实例和数据库对象---------------------------------------------------
 couchinstance，错误：=createCouchinstance（couchdbdef.url，couchdbdef.username，couchdbdef.password，
  couchdbdef.maxretries，couchdbdef.maxretriesonstartup，couchdbdef.requestTimeout，couchdbdef.createGlobalChangesDB，&disabled.provider）
 assert.noerror（t，err，“尝试创建coach实例时出错”）。
 数据库：=couchdatabase couchinstance:couchinstance，dbname:database

 //新建数据库
 errdb：=db.createdatabaseifnotexist（）。
 assert.noerror（t，errdb，“尝试创建数据库时出错”）。

 dbinfo，u，errinfo：=db.getdatabaseinfo（）。
 assert.noerror（t，errinfo，“尝试获取数据库信息时出错”）。

 assert.equal（t，数据库，dbinfo.dbname）

}
