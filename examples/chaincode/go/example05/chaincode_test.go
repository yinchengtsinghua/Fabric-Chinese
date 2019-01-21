
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


package example05

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/examples/chaincode/go/example02"
)

var chaincodeName = "ex02"

//chaincode示例05似乎想要返回对查询（）的JSON响应
//但它实际上并不这样做，它只是返回和值
func jsonResponse(name string, value string) string {
	return fmt.Sprintf("jsonResponse = \"{\"Name\":\"%v\",\"Value\":\"%v\"}", name, value)
}

func checkInit(t *testing.T, stub *shim.MockStub, args [][]byte) {
	res := stub.MockInit("1", args)
	if res.Status != shim.OK {
		fmt.Println("Init failed", string(res.Message))
		t.FailNow()
	}
}

func checkState(t *testing.T, stub *shim.MockStub, name string, expect string) {
	bytes := stub.State[name]
	if bytes == nil {
		fmt.Println("State", name, "failed to get value")
		t.FailNow()
	}
	if string(bytes) != expect {
		fmt.Println("State value", name, "was not", expect, "as expected")
		t.FailNow()
	}
}

func checkQuery(t *testing.T, stub *shim.MockStub, args [][]byte, expect string) {
	res := stub.MockInvoke("1", args)
	if res.Status != shim.OK {
		fmt.Println("Query", args, "failed", string(res.Message))
		t.FailNow()
	}
	if res.Payload == nil {
		fmt.Println("Query", args, "failed to get result")
		t.FailNow()
	}
	if string(res.Payload) != expect {
		fmt.Println("Query result ", string(res.Payload), "was not", expect, "as expected")
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

func TestExample05_Init(t *testing.T) {
	scc := new(SimpleChaincode)
	stub := shim.NewMockStub("ex05", scc)

//初始A=123 B=234
	checkInit(t, stub, [][]byte{[]byte("init"), []byte("sumStoreName"), []byte("432")})

	checkState(t, stub, "sumStoreName", "432")
}

func TestExample05_Query(t *testing.T) {
	scc := new(SimpleChaincode)
	stub := shim.NewMockStub("ex05", scc)

	ccEx2 := new(example02.SimpleChaincode)
	stubEx2 := shim.NewMockStub(chaincodeName, ccEx2)
	checkInit(t, stubEx2, [][]byte{[]byte("init"), []byte("a"), []byte("111"), []byte("b"), []byte("222")})
	stub.MockPeerChaincode(chaincodeName, stubEx2)

	checkInit(t, stub, [][]byte{[]byte("init"), []byte("sumStoreName"), []byte("0")})

//A+B=111+222=333
checkQuery(t, stub, [][]byte{[]byte("query"), []byte(chaincodeName), []byte("sumStoreName"), []byte("")}, "333") //示例05不返回JSON？
}

func TestExample05_Invoke(t *testing.T) {
	scc := new(SimpleChaincode)
	stub := shim.NewMockStub("ex05", scc)

	ccEx2 := new(example02.SimpleChaincode)
	stubEx2 := shim.NewMockStub(chaincodeName, ccEx2)
	checkInit(t, stubEx2, [][]byte{[]byte("init"), []byte("a"), []byte("222"), []byte("b"), []byte("333")})
	stub.MockPeerChaincode(chaincodeName, stubEx2)

	checkInit(t, stub, [][]byte{[]byte("init"), []byte("sumStoreName"), []byte("0")})

//A+B=222+333=555
	checkInvoke(t, stub, [][]byte{[]byte("invoke"), []byte(chaincodeName), []byte("sumStoreName"), []byte("")})
checkQuery(t, stub, [][]byte{[]byte("query"), []byte(chaincodeName), []byte("sumStoreName"), []byte("")}, "555") //示例05不返回JSON？
	checkQuery(t, stubEx2, [][]byte{[]byte("query"), []byte("a")}, "222")
	checkQuery(t, stubEx2, [][]byte{[]byte("query"), []byte("b")}, "333")

//更新a-=10和b+=10
	checkInvoke(t, stubEx2, [][]byte{[]byte("invoke"), []byte("a"), []byte("b"), []byte("10")})

//A+B=212+343=555
	checkInvoke(t, stub, [][]byte{[]byte("invoke"), []byte(chaincodeName), []byte("sumStoreName"), []byte("")})
checkQuery(t, stub, [][]byte{[]byte("query"), []byte(chaincodeName), []byte("sumStoreName"), []byte("")}, "555") //示例05不返回JSON？
	checkQuery(t, stubEx2, [][]byte{[]byte("query"), []byte("a")}, "212")
	checkQuery(t, stubEx2, [][]byte{[]byte("query"), []byte("b")}, "343")
}
