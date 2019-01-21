
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


package main

import (
	"fmt"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/entities"
	pb "github.com/hyperledger/fabric/protos/peer"
)

const DECKEY = "DECKEY"
const VERKEY = "VERKEY"
const ENCKEY = "ENCKEY"
const SIGKEY = "SIGKEY"
const IV = "IV"

//enccc示例使用加密/签名的链代码的简单链代码实现
type EncCC struct {
	bccspInst bccsp.BCCSP
}

//init对此cc不起任何作用
func (t *EncCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

//加密程序公开如何在
//使用已通过
//瞬态场
func (t *EncCC) Encrypter(stub shim.ChaincodeStubInterface, args []string, encKey, IV []byte) pb.Response {
//创建加密程序实体-我们给它一个ID、bccsp实例、密钥和（可选）iv
	ent, err := entities.NewAES256EncrypterEntity("ID", t.bccspInst, encKey, IV)
	if err != nil {
		return shim.Error(fmt.Sprintf("entities.NewAES256EncrypterEntity failed, err %s", err))
	}

	if len(args) != 2 {
		return shim.Error("Expected 2 parameters to function Encrypter")
	}

	key := args[0]
	cleartextValue := []byte(args[1])

//在这里，我们加密clearTextValue并将其分配给key
	err = encryptAndPutState(stub, ent, key, cleartextValue)
	if err != nil {
		return shim.Error(fmt.Sprintf("encryptAndPutState failed, err %+v", err))
	}
	return shim.Success(nil)
}

//解密器公开如何从分类帐中读取数据并使用AES256进行解密。
//通过瞬态字段提供给链码的位键。
func (t *EncCC) Decrypter(stub shim.ChaincodeStubInterface, args []string, decKey, IV []byte) pb.Response {
//创建加密程序实体-我们给它一个ID、bccsp实例、密钥和（可选）iv
	ent, err := entities.NewAES256EncrypterEntity("ID", t.bccspInst, decKey, IV)
	if err != nil {
		return shim.Error(fmt.Sprintf("entities.NewAES256EncrypterEntity failed, err %s", err))
	}

	if len(args) != 1 {
		return shim.Error("Expected 1 parameters to function Decrypter")
	}

	key := args[0]

//在这里，我们解密与密钥关联的状态
	cleartextValue, err := getStateAndDecrypt(stub, ent, key)
	if err != nil {
		return shim.Error(fmt.Sprintf("getStateAndDecrypt failed, err %+v", err))
	}

//在这里，我们返回解密后的值
	return shim.Success(cleartextValue)
}

//EncrypterSigner将在收到的键之后如何将状态写入分类帐
//加密（AES 256位密钥）和签名（X9.62/secg曲线在256位主字段上）已通过
//瞬态场
func (t *EncCC) EncrypterSigner(stub shim.ChaincodeStubInterface, args []string, encKey, sigKey []byte) pb.Response {
//创建加密程序/签名者实体-我们给它一个ID、bccsp实例和密钥
	ent, err := entities.NewAES256EncrypterECDSASignerEntity("ID", t.bccspInst, encKey, sigKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("entities.NewAES256EncrypterEntity failed, err %s", err))
	}

	if len(args) != 2 {
		return shim.Error("Expected 2 parameters to function EncrypterSigner")
	}

	key := args[0]
	cleartextValue := []byte(args[1])

//在这里，我们对clearTextValue进行签名、加密并将其分配给key
	err = signEncryptAndPutState(stub, ent, key, cleartextValue)
	if err != nil {
		return shim.Error(fmt.Sprintf("signEncryptAndPutState failed, err %+v", err))
	}

	return shim.Success(nil)
}

//DelpRealValueT公开了如何在收到密钥后获取状态到分类帐中。
//解密（AES 256位密钥）和验证（X9.62/secg曲线在256位主字段上）已通过
//瞬态场
func (t *EncCC) DecrypterVerify(stub shim.ChaincodeStubInterface, args []string, decKey, verKey []byte) pb.Response {
//创建解密器/验证实体-我们给它一个ID、bccsp实例和密钥
	ent, err := entities.NewAES256EncrypterECDSASignerEntity("ID", t.bccspInst, decKey, verKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("entities.NewAES256DecrypterEntity failed, err %s", err))
	}

	if len(args) != 1 {
		return shim.Error("Expected 1 parameters to function DecrypterVerify")
	}
	key := args[0]

//在这里，我们解密与密钥相关联的状态并验证它
	cleartextValue, err := getStateDecryptAndVerify(stub, ent, key)
	if err != nil {
		return shim.Error(fmt.Sprintf("getStateDecryptAndVerify failed, err %+v", err))
	}

//这里，我们将解密和验证的值作为结果返回。
	return shim.Success(cleartextValue)
}

//RangeDecrypter显示使用解密器可以满足范围查询的方式。
//直接解密以前加密的键值对的实体
func (t *EncCC) RangeDecrypter(stub shim.ChaincodeStubInterface, decKey []byte) pb.Response {
//创建加密程序实体-我们给它一个ID、bccsp实例和密钥
	ent, err := entities.NewAES256EncrypterEntity("ID", t.bccspInst, decKey, nil)
	if err != nil {
		return shim.Error(fmt.Sprintf("entities.NewAES256EncrypterEntity failed, err %s", err))
	}

	bytes, err := getStateByRangeAndDecrypt(stub, ent, "", "")
	if err != nil {
		return shim.Error(fmt.Sprintf("getStateByRangeAndDecrypt failed, err %+v", err))
	}

	return shim.Success(bytes)
}

//调用此链代码将公开要加密、解密事务性的函数
//数据。它还支持一个加密、签名和解密的示例，
//验证。初始化向量（iv）可以作为parm传递给
//确保对等机具有确定性数据。
func (t *EncCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
//获取参数和瞬态
	f, args := stub.GetFunctionAndParameters()
	tMap, err := stub.GetTransient()
	if err != nil {
		return shim.Error(fmt.Sprintf("Could not retrieve transient, err %s", err))
	}

	switch f {
	case "ENCRYPT":
//确保有一个瞬变的关键-假设是
//它与字符串“enckey”关联
		if _, in := tMap[ENCKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient encryption key %s", ENCKEY))
		}

		return t.Encrypter(stub, args[0:], tMap[ENCKEY], tMap[IV])
	case "DECRYPT":

//确保有一个瞬变的关键-假设是
//它与绳子“deckey”有关
		if _, in := tMap[DECKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient decryption key %s", DECKEY))
		}

		return t.Decrypter(stub, args[0:], tMap[DECKEY], tMap[IV])
	case "ENCRYPTSIGN":
//确保键在瞬态图中-假设它们
//与字符串“EnKEY”和“SigKy”关联。
		if _, in := tMap[ENCKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient key %s", ENCKEY))
		} else if _, in := tMap[SIGKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient key %s", SIGKEY))
		}

		return t.EncrypterSigner(stub, args[0:], tMap[ENCKEY], tMap[SIGKEY])
	case "DECRYPTVERIFY":
//确保键在瞬态图中-假设它们
//与字符串“deckey”和“verkey”关联
		if _, in := tMap[DECKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient key %s", DECKEY))
		} else if _, in := tMap[VERKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient key %s", VERKEY))
		}

		return t.DecrypterVerify(stub, args[0:], tMap[DECKEY], tMap[VERKEY])
	case "RANGEQUERY":
//确保有一个瞬变的关键-假设是
//它与字符串“enckey”关联
		if _, in := tMap[DECKEY]; !in {
			return shim.Error(fmt.Sprintf("Expected transient key %s", DECKEY))
		}

		return t.RangeDecrypter(stub, tMap[DECKEY])
	default:
		return shim.Error(fmt.Sprintf("Unsupported function %s", f))
	}
}

func main() {
	factory.InitFactories(nil)

	err := shim.Start(&EncCC{factory.GetDefault()})
	if err != nil {
		fmt.Printf("Error starting EncCC chaincode: %s", err)
	}
}
