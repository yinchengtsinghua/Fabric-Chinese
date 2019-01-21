
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
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/entities"
	"github.com/pkg/errors"
)

//下面的函数显示了一些关于如何
//使用实体执行加密操作
//在分类帐状态上

//GetStateAndDecrypt检索与键关联的值，
//使用提供的实体对其进行解密并返回结果
//解密的
func getStateAndDecrypt(stub shim.ChaincodeStubInterface, ent entities.Encrypter, key string) ([]byte, error) {
//首先，我们从分类帐中检索密文。
	ciphertext, err := stub.GetState(key)
	if err != nil {
		return nil, err
	}

//GetState will return a nil slice if the key does not exist.
//注意，链码逻辑可能想要区分
//无切片（键在状态db中不存在）和空切片
//（在状态数据库中找到键，但值为空）。我们不
//在这里区分大小写
	if len(ciphertext) == 0 {
		return nil, errors.New("no ciphertext to decrypt")
	}

	return ent.Decrypt(ciphertext)
}

//encryptAndPutState encrypts the supplied value using the
//提供的实体并将其放入与
//提供的kvs密钥
func encryptAndPutState(stub shim.ChaincodeStubInterface, ent entities.Encrypter, key string, value []byte) error {
//at first we use the supplied entity to encrypt the value
	ciphertext, err := ent.Encrypt(value)
	if err != nil {
		return err
	}

	return stub.PutState(key, ciphertext)
}

//GetStateDecryptandVerify检索与密钥关联的值，
//用提供的实体解密，验证签名
//然后返回解密结果
//成功
func getStateDecryptAndVerify(stub shim.ChaincodeStubInterface, ent entities.EncrypterSignerEntity, key string) ([]byte, error) {
//在这里，我们检索和解密与密钥关联的状态
	val, err := getStateAndDecrypt(stub, ent, key)
	if err != nil {
		return nil, err
	}

//我们从解密状态取消签名消息的标记
	msg := &entities.SignedMessage{}
	err = msg.FromBytes(val)
	if err != nil {
		return nil, err
	}

//我们核实签名
	ok, err := msg.Verify(ent)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("invalid signature")
	}

	return msg.Payload, nil
}

//SturnStuttPuttand状态签名提供的值，加密
//提供的值及其签名使用
//提供的实体并将其放入与
//提供的kvs密钥
func signEncryptAndPutState(stub shim.ChaincodeStubInterface, ent entities.EncrypterSignerEntity, key string, value []byte) error {
//在这里，我们创建一个SignedMessage，设置它的有效负载
//到值和实体的ID，以及
//sign it with the entity
	msg := &entities.SignedMessage{Payload: value, ID: []byte(ent.ID())}
	err := msg.Sign(ent)
	if err != nil {
		return err
	}

//在这里，我们序列化已签名的消息
	b, err := msg.ToBytes()
	if err != nil {
		return err
	}

//在这里，我们加密与args[0]关联的序列化版本。
	return encryptAndPutState(stub, ent, key, b)
}

type keyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

//GETStaseByRangeReand DeLyPT从
//使用提供的实体对每个值进行分类和解密；它返回
//一个json编组的keyValuePair切片
func getStateByRangeAndDecrypt(stub shim.ChaincodeStubInterface, ent entities.Encrypter, startKey, endKey string) ([]byte, error) {
//我们称为按范围获取状态，以通过整个范围
	iterator, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

//我们对每个条目进行解密——假设它们都是用同一个密钥加密的。
	keyvalueset := []keyValuePair{}
	for iterator.HasNext() {
		el, err := iterator.Next()
		if err != nil {
			return nil, err
		}

		v, err := ent.Decrypt(el.Value)
		if err != nil {
			return nil, err
		}

		keyvalueset = append(keyvalueset, keyValuePair{el.Key, string(v)})
	}

	bytes, err := json.Marshal(keyvalueset)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
