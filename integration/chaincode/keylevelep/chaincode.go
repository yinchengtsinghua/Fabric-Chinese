
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


package keylevelep

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/statebased"
	pb "github.com/hyperledger/fabric/protos/peer"
)

/*
背书CC是一个使用基于状态背书的示例链代码。
在in it函数中，它创建两个kvs状态，一个公共状态，一个私有状态，
然后可以通过使用基于状态的链代码函数修改
背书链码便利层。以下链码函数
提供：
-）“addorgs”：提供将添加到
   国家认可政策
-）“delorgs”：提供将从中删除的MSP ID列表
   国家的支持政策
-）“delep”：完全删除国家的关键级认可政策
-）“list orgs”：列出国家认可政策中包含的组织
**/

type EndorsementCC struct {
}

//初始化回调
func (cc *EndorsementCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	err := stub.PutState("pub", []byte("foo"))
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

//调用调度程序
func (cc *EndorsementCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	funcName, _ := stub.GetFunctionAndParameters()
	if function, ok := functions[funcName]; ok {
		return function(stub)
	}
	return shim.Error(fmt.Sprintf("Unknown function %s", funcName))
}

//invoke（）使用的函数调度映射
var functions = map[string]func(stub shim.ChaincodeStubInterface) pb.Response{
	"addorgs":  addOrgs,
	"delorgs":  delOrgs,
	"listorgs": listOrgs,
	"delep":    delEP,
	"setval":   setVal,
	"getval":   getVal,
	"cc2cc":    invokeCC,
}

//addorgs从调用参数添加MSP ID列表
//国家的支持政策
func addOrgs(stub shim.ChaincodeStubInterface) pb.Response {
	_, parameters := stub.GetFunctionAndParameters()
	if len(parameters) < 2 {
		return shim.Error("No orgs to add specified")
	}

//获取密钥的认可策略
	var epBytes []byte
	var err error
	if parameters[0] == "pub" {
		epBytes, err = stub.GetStateValidationParameter("pub")
	} else if parameters[0] == "priv" {
		epBytes, err = stub.GetPrivateDataValidationParameter("col", "priv")
	} else {
		return shim.Error("Unknown key specified")
	}
	if err != nil {
		return shim.Error(err.Error())
	}
	ep, err := statebased.NewStateEP(epBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

//将组织添加到认可策略
	err = ep.AddOrgs(statebased.RoleTypePeer, parameters[1:]...)
	if err != nil {
		return shim.Error(err.Error())
	}
	epBytes, err = ep.Policy()
	if err != nil {
		return shim.Error(err.Error())
	}

//为密钥设置修改的认可策略
	if parameters[0] == "pub" {
		err = stub.SetStateValidationParameter("pub", epBytes)
	} else if parameters[0] == "priv" {
		err = stub.SetPrivateDataValidationParameter("col", "priv", epBytes)
	}
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte{})
}

//delorgs从调用参数中删除MSP ID列表
//从国家的支持政策
func delOrgs(stub shim.ChaincodeStubInterface) pb.Response {
	_, parameters := stub.GetFunctionAndParameters()
	if len(parameters) < 2 {
		return shim.Error("No orgs to delete specified")
	}

//获取密钥的认可策略
	var epBytes []byte
	var err error
	if parameters[0] == "pub" {
		epBytes, err = stub.GetStateValidationParameter("pub")
	} else if parameters[0] == "priv" {
		epBytes, err = stub.GetPrivateDataValidationParameter("col", "priv")
	} else {
		return shim.Error("Unknown key specified")
	}
	ep, err := statebased.NewStateEP(epBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

//从该密钥的认可策略中删除组织
	ep.DelOrgs(parameters...)
	epBytes, err = ep.Policy()
	if err != nil {
		return shim.Error(err.Error())
	}

//为密钥设置修改的认可策略
	if parameters[0] == "pub" {
		err = stub.SetStateValidationParameter("pub", epBytes)
	} else if parameters[0] == "priv" {
		err = stub.SetPrivateDataValidationParameter("col", "priv", epBytes)
	}
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte{})
}

//listorgs返回当前属于
//国家的支持政策
func listOrgs(stub shim.ChaincodeStubInterface) pb.Response {
	_, parameters := stub.GetFunctionAndParameters()
	if len(parameters) < 1 {
		return shim.Error("No key specified")
	}

//获取密钥的认可策略
	var epBytes []byte
	var err error
	if parameters[0] == "pub" {
		epBytes, err = stub.GetStateValidationParameter("pub")
	} else if parameters[0] == "priv" {
		epBytes, err = stub.GetPrivateDataValidationParameter("col", "priv")
	} else {
		return shim.Error("Unknown key specified")
	}
	ep, err := statebased.NewStateEP(epBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

//获取认可政策中的组织列表
	orgs := ep.ListOrgs()
	orgsList, err := json.Marshal(orgs)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(orgsList)
}

//Delep将完全删除基于状态的密钥背书策略
func delEP(stub shim.ChaincodeStubInterface) pb.Response {
	_, parameters := stub.GetFunctionAndParameters()
	if len(parameters) < 1 {
		return shim.Error("No key specified")
	}

//将修改后的密钥背书策略设置为零
	var err error
	if parameters[0] == "pub" {
		err = stub.SetStateValidationParameter("pub", nil)
	} else if parameters[0] == "priv" {
		err = stub.SetPrivateDataValidationParameter("col", "priv", nil)
	} else {
		return shim.Error("Unknown key specified")
	}
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte{})
}

//setval设置kvs键的值
func setVal(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) != 3 {
		return shim.Error("setval expects two arguments")
	}
	var err error
	if string(args[1]) == "pub" {
		err = stub.PutState("pub", args[2])
	} else if string(args[1]) == "priv" {
		err = stub.PutPrivateData("col", "priv", args[2])
	} else {
		return shim.Error("Unknown key specified")
	}
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte{})
}

//getval检索kvs键的值
func getVal(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) != 2 {
		return shim.Error("No key specified")
	}
	var err error
	var val []byte
	if string(args[1]) == "pub" {
		val, err = stub.GetState("pub")
	} else if string(args[1]) == "priv" {
		val, err = stub.GetPrivateData("col", "priv")
	} else {
		return shim.Error("Unknown key specified")
	}
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(val)
}

//invokecc用于在另一个通道上对给定CC进行链码到链码的调用。
func invokeCC(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) < 3 {
		return shim.Error("cc2cc expects at least two arguments (channel and chaincode)")
	}
	channel := string(args[1])
	cc := string(args[2])
	nargs := args[3:]
	resp := stub.InvokeChaincode(cc, nargs, channel)
	return resp
}
