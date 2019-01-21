
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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
)

//测试事务对私有数据的调用。
func TestQueriesPrivateData(t *testing.T) {
//跳过此测试，因为此测试要求设置应用程序配置，以便将私有数据功能设置为“true”。
//然而，随着一些软件包的最新重组，不可能用所需的配置注册系统链码进行测试。
//请参见文件“fabric/core/scc/register.go”中的函数寄存器ysccs。如果没有这个lscc，则在部署具有集合配置的链代码时返回错误。
//这个测试应该作为一个集成测试移到chaincode包之外。
	t.Skip()
	chainID := util.GetTestChainID()
	_, chaincodeSupport, cleanup, err := initPeer(chainID)
	if err != nil {
		t.Fail()
		t.Logf("Error creating peer: %s", err)
	}

	defer cleanup()

	url := "github.com/hyperledger/fabric/examples/chaincode/go/map"
	cID := &pb.ChaincodeID{Name: "tmap", Path: url, Version: "0"}

	f := "init"
	args := util.ToChaincodeArgs(f)

	spec := &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}

	cccid := &ccprovider.CCContext{
		Name:    "tmap",
		Version: "0",
	}

	var nextBlockNumber uint64 = 1
//此测试假设有四个集合
	collectionConfig := []*common.StaticCollectionConfig{{Name: "c1"}, {Name: "c2"}, {Name: "c3"}, {Name: "c4"}}
	collectionConfigPkg := constructCollectionConfigPkg(collectionConfig)
	defer chaincodeSupport.Stop(&ccprovider.ChaincodeContainerInfo{
		Name:          cID.Name,
		Version:       cID.Version,
		Path:          cID.Path,
		Type:          "GOLANG",
		ContainerType: "DOCKER",
	})
	_, err = deployWithCollectionConfigs(chainID, cccid, spec, collectionConfigPkg, nextBlockNumber, chaincodeSupport)
	nextBlockNumber++
	ccID := spec.ChaincodeId.Name
	if err != nil {
		t.Fail()
		t.Logf("Error initializing chaincode %s(%s)", ccID, err)
		return
	}

//添加101个大理石用于测试范围查询和丰富的查询（对于有能力的分类账）
//公共和私人数据。测试将测试范围查询和富查询
//和具有查询限制的查询
	for i := 1; i <= 101; i++ {
		f = "put"

//51归汤姆所有，50归杰瑞所有
		owner := "tom"
		if i%2 == 0 {
			owner = "jerry"
		}

//一个大理石颜色是红色，100个是蓝色
		color := "blue"
		if i == 12 {
			color = "red"
		}

		key := fmt.Sprintf("marble%03d", i)
		argsString := fmt.Sprintf("{\"docType\":\"marble\",\"name\":\"%s\",\"color\":\"%s\",\"size\":35,\"owner\":\"%s\"}", key, color, owner)
		args = util.ToChaincodeArgs(f, key, argsString)
		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++

		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

		f = "putPrivate"

		key = fmt.Sprintf("pmarble%03d", i)
		args = util.ToChaincodeArgs(f, "c1", key, argsString)
		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++

		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

	}

//在3个私人收藏中插入大理石
	for i := 2; i <= 4; i++ {
		collection := fmt.Sprintf("c%d", i)
		value := fmt.Sprintf("value_c%d", i)

		f = "putPrivate"
		t.Logf("invoking PutPrivateData with collection:<%s> key:%s", collection, "marble001")
		args = util.ToChaincodeArgs(f, collection, "pmarble001", value)
		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++

		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}
	}

//从集合C3中读取大理石
	f = "getPrivate"
	args = util.ToChaincodeArgs(f, "c3", "pmarble001")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err := invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++

	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

	var val string
	err = json.Unmarshal(retval, &val)
	expectedValue := fmt.Sprintf("value_c%d", 3)
	if val != expectedValue {
		t.Fail()
		t.Logf("Error detected with the GetPrivateData: expected '%s' but got '%s'", expectedValue, val)
		return
	}

//从集合c3中删除大理石
	f = "removePrivate"
	args = util.ToChaincodeArgs(f, "c3", "pmarble001")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++

	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

//从集合C4中删除大理石
	f = "removePrivate"
	args = util.ToChaincodeArgs(f, "c4", "pmarble001")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++

	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

//从集合c3读取已删除的大理石，以验证是否正确执行了删除操作。
	f = "getPrivate"
	args = util.ToChaincodeArgs(f, "c3", "pmarble001")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++

	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

	err = json.Unmarshal(retval, &val)
	if val != "" {
		t.Fail()
		t.Logf("Error detected with the GetPrivateData")
		return
	}

//尝试从公共状态读取插入到集合C2中的大理石以进行检查
//是否返回大理石（为正确操作，不应返回）
	f = "get"
	args = util.ToChaincodeArgs(f, "pmarble001")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++

	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

	err = json.Unmarshal(retval, &val)
	if val != "" {
		t.Fail()
		t.Logf("Error detected with the GetState: %s", val)
		return
	}
//“Marble001”到“Marble011”的以下范围查询应返回10个Marbles
	f = "keysPrivate"
	args = util.ToChaincodeArgs(f, "c1", "pmarble001", "pmarble011")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}
	var keys []interface{}
	err = json.Unmarshal(retval, &keys)
	if len(keys) != 10 {
		t.Fail()
		t.Logf("Error detected with the range query, should have returned 10 but returned %v", len(keys))
		return
	}

//“Marble001”到“Marble011”的以下范围查询应返回10个Marbles
	f = "keys"
	args = util.ToChaincodeArgs(f, "marble001", "marble011")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

	err = json.Unmarshal(retval, &keys)
	if len(keys) != 10 {
		t.Fail()
		t.Logf("Error detected with the range query, should have returned 10 but returned %v", len(keys))
		return
	}

//FAB-1163-以下范围查询应超时并产生错误
//同伴应该优雅地处理这个问题，而不是死去。

//保存原始超时并将新超时设置为1秒
	origTimeout := chaincodeSupport.ExecuteTimeout
	chaincodeSupport.ExecuteTimeout = time.Duration(1) * time.Second

//链码休眠2秒，超时1
	args = util.ToChaincodeArgs(f, "marble001", "marble002", "2000")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	if err == nil {
		t.Fail()
		t.Logf("expected timeout error but succeeded")
		return
	}

//恢复超时
	chaincodeSupport.ExecuteTimeout = origTimeout

//查询所有大理石将返回101个大理石
//此查询应返回正好101个结果（一个对next（）的调用）
//“marble001”到“marble102”的以下范围查询应返回101个marbles。
	f = "keys"
	args = util.ToChaincodeArgs(f, "marble001", "marble102")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

//取消标记结果
	err = json.Unmarshal(retval, &keys)

//检查是否有101个值
//使用默认查询限制10000，此查询实际上是不受限制的
	if len(keys) != 101 {
		t.Fail()
		t.Logf("Error detected with the range query, should have returned 101 but returned %v", len(keys))
		return
	}

//正在查询所有简单键。此查询应返回正好101个简单键（一个
//调用next（））没有复合键。
//以下“to”的开放式范围查询应返回101个大理石
	f = "keys"
	args = util.ToChaincodeArgs(f, "", "")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

//取消标记结果
	err = json.Unmarshal(retval, &keys)

//检查是否有101个值
//使用默认查询限制10000，此查询实际上是不受限制的
	if len(keys) != 101 {
		t.Fail()
		t.Logf("Error detected with the range query, should have returned 101 but returned %v", len(keys))
		return
	}

//只有CouchDB和
//查询限制仅适用于CouchDB范围和富查询
	if ledgerconfig.IsCouchDBEnabled() == true {

//用于垫片配料的角箱。电流垫片批量为100
//此查询应返回100个结果（不调用next（））
		f = "query"
		args = util.ToChaincodeArgs(f, "{\"selector\":{\"color\":\"blue\"}}")

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++

		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

//取消标记结果
		err = json.Unmarshal(retval, &keys)

//检查是否有100个值
		if len(keys) != 100 {
			t.Fail()
			t.Logf("Error detected with the rich query, should have returned 100 but returned %v %s", len(keys), keys)
			return
		}
		f = "queryPrivate"
		args = util.ToChaincodeArgs(f, "c1", "{\"selector\":{\"color\":\"blue\"}}")

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++
		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

//取消标记结果
		err = json.Unmarshal(retval, &keys)

//检查是否有100个值
		if len(keys) != 100 {
			t.Fail()
			t.Logf("Error detected with the rich query, should have returned 100 but returned %v %s", len(keys), keys)
			return
		}
//将查询限制重置为5
		viper.Set("ledger.state.queryLimit", 5)

//由于查询限制，“Marble01”到“Marble11”的以下范围查询应返回5个Marbles。
		f = "keys"
		args = util.ToChaincodeArgs(f, "marble001", "marble011")

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, retval, err := invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++
		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

//取消标记结果
		err = json.Unmarshal(retval, &keys)
//检查是否有5个值
		if len(keys) != 5 {
			t.Fail()
			t.Logf("Error detected with the range query, should have returned 5 but returned %v", len(keys))
			return
		}

//将查询限制重置为10000
		viper.Set("ledger.state.queryLimit", 10000)

//以下丰富的查询应返回50个大理石
		f = "query"
		args = util.ToChaincodeArgs(f, "{\"selector\":{\"owner\":\"jerry\"}}")

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++

		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

//取消标记结果
		err = json.Unmarshal(retval, &keys)

//检查是否有50个值
//使用默认查询限制10000，此查询实际上是不受限制的
		if len(keys) != 50 {
			t.Fail()
			t.Logf("Error detected with the rich query, should have returned 50 but returned %v", len(keys))
			return
		}

//将查询限制重置为5
		viper.Set("ledger.state.queryLimit", 5)

//由于查询限制，下面的富查询应返回5个大理石。
		f = "query"
		args = util.ToChaincodeArgs(f, "{\"selector\":{\"owner\":\"jerry\"}}")

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++
		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

//取消标记结果
		err = json.Unmarshal(retval, &keys)

//检查是否有5个值
		if len(keys) != 5 {
			t.Fail()
			t.Logf("Error detected with the rich query, should have returned 5 but returned %v", len(keys))
			return
		}

	}

//历史查询修改
	f = "put"
	args = util.ToChaincodeArgs(f, "marble012", "{\"docType\":\"marble\",\"name\":\"marble012\",\"color\":\"red\",\"size\":30,\"owner\":\"jerry\"}")
	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

	f = "put"
	args = util.ToChaincodeArgs(f, "marble012", "{\"docType\":\"marble\",\"name\":\"marble012\",\"color\":\"red\",\"size\":30,\"owner\":\"jerry\"}")
	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

//“Marble12”的以下历史查询应返回3条记录
	f = "history"
	args = util.ToChaincodeArgs(f, "marble012")
	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

	var history []interface{}
	err = json.Unmarshal(retval, &history)
	if len(history) != 3 {
		t.Fail()
		t.Logf("Error detected with the history query, should have returned 3 but returned %v", len(history))
		return
	}
}

func constructCollectionConfigPkg(staticCollectionConfigs []*common.StaticCollectionConfig) *common.CollectionConfigPackage {
	var cc []*common.CollectionConfig
	for _, sc := range staticCollectionConfigs {
		cc = append(cc, &common.CollectionConfig{
			Payload: &common.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: sc}})
	}
	return &common.CollectionConfigPackage{Config: cc}
}
