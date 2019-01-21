
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


package example04

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/examples/chaincode/go/example02"
)

//这是对chaincode上任何成功的invoke（）的响应示例04
var eventResponse = "{\"Name\":\"Event\",\"Amount\":\"1\"}"

func checkInit(t *testing.T, stub *shim.MockStub, args [][]byte) {
	res := stub.MockInit("1", args)
	if res.Status != shim.OK {
		fmt.Println("Init failed", string(res.Message))
		t.FailNow()
	}
}

func checkState(t *testing.T, stub *shim.MockStub, name string, value string) {
	bytes := stub.State[name]
	if bytes == nil {
		fmt.Println("State", name, "failed to get value")
		t.FailNow()
	}
	if string(bytes) != value {
		fmt.Println("State value", name, "was not", value, "as expected")
		t.FailNow()
	}
}

func checkQuery(t *testing.T, stub *shim.MockStub, name string, value string) {
	res := stub.MockInvoke("1", [][]byte{[]byte("query"), []byte(name)})
	if res.Status != shim.OK {
		fmt.Println("Query", name, "failed", string(res.Message))
		t.FailNow()
	}
	if res.Payload == nil {
		fmt.Println("Query", name, "failed to get value")
		t.FailNow()
	}
	if string(res.Payload) != value {
		fmt.Println("Query value", name, "was not", value, "as expected")
		t.FailNow()
	}
}

func checkInvoke(t *testing.T, stub *shim.MockStub, args [][]byte) {
	res := stub.MockInvoke("1", args)
	if res.Status != shim.OK {
		fmt.Println("Invoke", args, "failed", string(res.Message))
		t.FailNow()
	}
}

func TestExample04_Init(t *testing.T) {
	scc := new(SimpleChaincode)
	stub := shim.NewMockStub("ex04", scc)

//初始A=123 B=234
	checkInit(t, stub, [][]byte{[]byte("init"), []byte("Event"), []byte("123")})

	checkState(t, stub, "Event", "123")
}

func TestExample04_Query(t *testing.T) {
	scc := new(SimpleChaincode)
	stub := shim.NewMockStub("ex04", scc)

//初始A=345 B=456
	checkInit(t, stub, [][]byte{[]byte("init"), []byte("Event"), []byte("1")})

//查询A
	checkQuery(t, stub, "Event", eventResponse)
}

func TestExample04_Invoke(t *testing.T) {
	scc := new(SimpleChaincode)
	stub := shim.NewMockStub("ex04", scc)

	chaincodeToInvoke := "ex02"

	ccEx2 := new(example02.SimpleChaincode)
	stubEx2 := shim.NewMockStub(chaincodeToInvoke, ccEx2)
	checkInit(t, stubEx2, [][]byte{[]byte("init"), []byte("a"), []byte("111"), []byte("b"), []byte("222")})
	stub.MockPeerChaincode(chaincodeToInvoke, stubEx2)

//初始A=567 B=678
	checkInit(t, stub, [][]byte{[]byte("init"), []byte("Event"), []byte("1")})

//通过example04的chaincode调用a->b 10
	checkInvoke(t, stub, [][]byte{[]byte("invoke"), []byte(chaincodeToInvoke), []byte("Event"), []byte("1")})
	checkQuery(t, stub, "Event", eventResponse)
	checkQuery(t, stubEx2, "a", "101")
	checkQuery(t, stubEx2, "b", "232")

//通过example04的chaincode调用a->b 10
	checkInvoke(t, stub, [][]byte{[]byte("invoke"), []byte(chaincodeToInvoke), []byte("Event"), []byte("1")})
	checkQuery(t, stub, "Event", eventResponse)
	checkQuery(t, stubEx2, "a", "91")
	checkQuery(t, stubEx2, "b", "242")
}
