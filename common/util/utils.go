
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package util

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/metadata"
)

type alg struct {
	hashFun func([]byte) string
}

const defaultAlg = "sha256"

var availableIDgenAlgs = map[string]alg{
	defaultAlg: {GenerateIDfromTxSHAHash},
}

//computesha256返回sha2-256数据
func ComputeSHA256(data []byte) (hash []byte) {
	hash, err := factory.GetDefault().Hash(data, &bccsp.SHA256Opts{})
	if err != nil {
		panic(fmt.Errorf("Failed computing SHA256 on [% x]", data))
	}
	return
}

//computesha3256返回sha3-256数据
func ComputeSHA3256(data []byte) (hash []byte) {
	hash, err := factory.GetDefault().Hash(data, &bccsp.SHA3_256Opts{})
	if err != nil {
		panic(fmt.Errorf("Failed computing SHA3_256 on [% x]", data))
	}
	return
}

//GenerateBytesUuid返回基于RFC 4122的Uuid，返回生成的字节
func GenerateBytesUUID() []byte {
	uuid := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, uuid)
	if err != nil {
		panic(fmt.Sprintf("Error generating UUID: %s", err))
	}

//变量位；见第4.1.1节
	uuid[8] = uuid[8]&^0xc0 | 0x80

//第4版（伪随机）；见第4.1.3节
	uuid[6] = uuid[6]&^0xf0 | 0x40

	return uuid
}

//generateintuuid返回基于rfc 4122的uuid，返回big.int
func GenerateIntUUID() *big.Int {
	uuid := GenerateBytesUUID()
	z := big.NewInt(0)
	return z.SetBytes(uuid)
}

//GenerateUid返回基于RFC 4122的UUID
func GenerateUUID() string {
	uuid := GenerateBytesUUID()
	return idBytesToStr(uuid)
}

//
func CreateUtcTimestamp() *timestamp.Timestamp {
	now := time.Now().UTC()
	secs := now.Unix()
	nanos := int32(now.UnixNano() - (secs * 1000000000))
	return &(timestamp.Timestamp{Seconds: secs, Nanos: nanos})
}

//GenerateHashFromSignature返回组合参数的哈希
func GenerateHashFromSignature(path string, args []byte) []byte {
	return ComputeSHA256(args)
}

//generateidfromtxshash使用tx有效负载生成sha256哈希
func GenerateIDfromTxSHAHash(payload []byte) string {
	return fmt.Sprintf("%x", ComputeSHA256(payload))
}

//GenerateIDWithAlg使用自定义算法生成ID
func GenerateIDWithAlg(customIDgenAlg string, payload []byte) (string, error) {
	if customIDgenAlg == "" {
		customIDgenAlg = defaultAlg
	}
	var alg = availableIDgenAlgs[customIDgenAlg]
	if alg.hashFun != nil {
		return alg.hashFun(payload), nil
	}
	return "", fmt.Errorf("Wrong ID generation algorithm was given: %s", customIDgenAlg)
}

func idBytesToStr(id []byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:])
}

//
//第二个切片应是第一个切片的子集。
func FindMissingElements(all []string, some []string) (delta []string) {
all:
	for _, v1 := range all {
		for _, v2 := range some {
			if strings.Compare(v1, v2) == 0 {
				continue all
			}
		}
		delta = append(delta, v1)
	}
	return
}

//
func ToChaincodeArgs(args ...string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}

//arrayToChainCodeArgs将字符串参数数组转换为[]字节参数数组
func ArrayToChaincodeArgs(args []string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}

const testchainid = "testchainid"
const testorgid = "**TEST_ORGID**"

//GetTestChainID返回排序器使用的chainID常量
func GetTestChainID() string {
	return testchainid
}

//
func GetTestOrgID() string {
	return testorgid
}

//GetSysCCVersion返回所有系统链代码的版本
//这需要在围绕系统链码的策略上重新讨论
//用户的“升级”以及与“结构”升级的关系。为了
//现在保持简单，使用织物的版本标记
func GetSysCCVersion() string {
	return metadata.Version
}

//ConcatenateBytes对于组合多个字节数组非常有用，特别是对于
//多个字段上的签名或摘要
func ConcatenateBytes(data ...[]byte) []byte {
	finalLength := 0
	for _, slice := range data {
		finalLength += len(slice)
	}
	result := make([]byte, finalLength)
	last := 0
	for _, slice := range data {
		for i := range slice {
			result[i+last] = slice[i]
		}
		last += len(slice)
	}
	return result
}

//`flatten`以深度优先的方式递归检索结构中的每个叶节点
//并将结果聚合到给定的字符串切片中，格式为：“path.to.leaf=value”
//按照定义的顺序。根名称在路径中被忽略。此助手函数是
//用于漂亮地打印结构，如configs。
//例如，给定的数据结构：
//{
//B{
//C:“Foo”，
//D：42，
//}
//女：
//}
//它应该生成一个包含以下项的字符串切片：
//[
//“b.c=\”foo\“”，
//“B.D＝42”，
//“E＝”
//]
func Flatten(i interface{}) []string {
	var res []string
	flatten("", &res, reflect.ValueOf(i))
	return res
}

const DELIMITER = "."

func flatten(k string, m *[]string, v reflect.Value) {
	delimiter := DELIMITER
	if k == "" {
		delimiter = ""
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			*m = append(*m, fmt.Sprintf("%s =", k))
			return
		}
		flatten(k, m, v.Elem())
	case reflect.Struct:
		if x, ok := v.Interface().(fmt.Stringer); ok {
			*m = append(*m, fmt.Sprintf("%s = %v", k, x))
			return
		}

		for i := 0; i < v.NumField(); i++ {
			flatten(k+delimiter+v.Type().Field(i).Name, m, v.Field(i))
		}
	case reflect.String:
//引用字符串值很有用
		*m = append(*m, fmt.Sprintf("%s = \"%s\"", k, v))
	default:
		*m = append(*m, fmt.Sprintf("%s = %v", k, v))
	}
}
