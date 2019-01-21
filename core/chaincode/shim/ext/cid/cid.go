
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

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


package cid

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/attrmgr"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

//GetID返回与调用标识关联的ID。这个身份证
//保证在MSP中是唯一的。
func GetID(stub ChaincodeStubInterface) (string, error) {
	c, err := New(stub)
	if err != nil {
		return "", err
	}
	return c.GetID()
}

//GetMSPID返回与以下标识关联的MSP的ID：
//已提交交易记录
func GetMSPID(stub ChaincodeStubInterface) (string, error) {
	c, err := New(stub)
	if err != nil {
		return "", err
	}
	return c.GetMSPID()
}

//getattributeValue返回指定属性的值
func GetAttributeValue(stub ChaincodeStubInterface, attrName string) (value string, found bool, err error) {
	c, err := New(stub)
	if err != nil {
		return "", false, err
	}
	return c.GetAttributeValue(attrName)
}

//AssertAttributeValue检查属性值是否等于指定值
func AssertAttributeValue(stub ChaincodeStubInterface, attrName, attrValue string) error {
	c, err := New(stub)
	if err != nil {
		return err
	}
	return c.AssertAttributeValue(attrName, attrValue)
}

//getx509certificate返回与客户端关联的x509证书，
//如果未通过X509证书识别，则为零。
func GetX509Certificate(stub ChaincodeStubInterface) (*x509.Certificate, error) {
	c, err := New(stub)
	if err != nil {
		return nil, err
	}
	return c.GetX509Certificate()
}

//clientIdentityImpl实现clientIdentity接口
type clientIdentityImpl struct {
	stub  ChaincodeStubInterface
	mspID string
	cert  *x509.Certificate
	attrs *attrmgr.Attributes
}

//new返回clientIdentity的实例
func New(stub ChaincodeStubInterface) (ClientIdentity, error) {
	c := &clientIdentityImpl{stub: stub}
	err := c.init()
	if err != nil {
		return nil, err
	}
	return c, nil
}

//GetID返回与调用标识关联的唯一ID。
func (c *clientIdentityImpl) GetID() (string, error) {
//前面的“x509:：”将此标识为x509证书，并且
//主题和颁发者DNS唯一标识X509证书。
//如果证书被续订，生成的ID将保持不变。
	id := fmt.Sprintf("x509::%s::%s", getDN(&c.cert.Subject), getDN(&c.cert.Issuer))
	return base64.StdEncoding.EncodeToString([]byte(id)), nil
}

//GetMSPID返回与以下标识关联的MSP的ID：
//已提交交易记录
func (c *clientIdentityImpl) GetMSPID() (string, error) {
	return c.mspID, nil
}

//getattributeValue返回指定属性的值
func (c *clientIdentityImpl) GetAttributeValue(attrName string) (value string, found bool, err error) {
	if c.attrs == nil {
		return "", false, nil
	}
	return c.attrs.Value(attrName)
}

//AssertAttributeValue检查属性值是否等于指定值
func (c *clientIdentityImpl) AssertAttributeValue(attrName, attrValue string) error {
	val, ok, err := c.GetAttributeValue(attrName)
	if err != nil {
		return err
	}
	if !ok {
		return errors.Errorf("Attribute '%s' was not found", attrName)
	}
	if val != attrValue {
		return errors.Errorf("Attribute '%s' equals '%s', not '%s'", attrName, val, attrValue)
	}
	return nil
}

//getx509certificate返回与客户端关联的x509证书，
//如果未通过X509证书识别，则为零。
func (c *clientIdentityImpl) GetX509Certificate() (*x509.Certificate, error) {
	return c.cert, nil
}

//初始化客户端
func (c *clientIdentityImpl) init() error {
	signingID, err := c.getIdentity()
	if err != nil {
		return err
	}
	c.mspID = signingID.GetMspid()
	idbytes := signingID.GetIdBytes()
	block, _ := pem.Decode(idbytes)
	if block == nil {
		err := c.getAttributesFromIdemix()
		if err != nil {
			return errors.WithMessage(err, "identity bytes are neither X509 PEM format nor an idemix credential")
		}
		return nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return errors.WithMessage(err, "failed to parse certificate")
	}
	c.cert = cert
	attrs, err := attrmgr.New().GetAttributesFromCert(cert)
	if err != nil {
		return errors.WithMessage(err, "failed to get attributes from the transaction invoker's certificate")
	}
	c.attrs = attrs
	return nil
}

//取消标记chaincodestubinterface.getCreator方法和返回的字节
//返回生成的msp.serialididentity对象
func (c *clientIdentityImpl) getIdentity() (*msp.SerializedIdentity, error) {
	sid := &msp.SerializedIdentity{}
	creator, err := c.stub.GetCreator()
	if err != nil || creator == nil {
		return nil, errors.WithMessage(err, "failed to get transaction invoker's identity from the chaincode stub")
	}
	err = proto.Unmarshal(creator, sid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction invoker's identity")
	}
	return sid, nil
}

func (c *clientIdentityImpl) getAttributesFromIdemix() error {
	creator, err := c.stub.GetCreator()
	attrs, err := attrmgr.New().GetAttributesFromIdemix(creator)
	if err != nil {
		return errors.WithMessage(err, "failed to get attributes from the transaction invoker's idemix credential")
	}
	c.attrs = attrs
	return nil
}

//获取与pkix.name关联的dn（可分辨名称）。
//注意：此代码几乎是中string（）函数的直接副本。
//https://go review.googlesource.com/c/go/+/67270/1/src/crypto/x509/pkix/pkix.go_26
//它返回一个由RFC2253定义的DN。
func getDN(name *pkix.Name) string {
	r := name.ToRDNSequence()
	s := ""
	for i := 0; i < len(r); i++ {
		rdn := r[len(r)-1-i]
		if i > 0 {
			s += ","
		}
		for j, tv := range rdn {
			if j > 0 {
				s += "+"
			}
			typeString := tv.Type.String()
			typeName, ok := attributeTypeNames[typeString]
			if !ok {
				derBytes, err := asn1.Marshal(tv.Value)
				if err == nil {
					s += typeString + "=#" + hex.EncodeToString(derBytes)
continue //不需要值转义。
				}
				typeName = typeString
			}
			valueString := fmt.Sprint(tv.Value)
			escaped := ""
			begin := 0
			for idx, c := range valueString {
				if (idx == 0 && (c == ' ' || c == '#')) ||
					(idx == len(valueString)-1 && c == ' ') {
					escaped += valueString[begin:idx]
					escaped += "\\" + string(c)
					begin = idx + 1
					continue
				}
				switch c {
				case ',', '+', '"', '\\', '<', '>', ';':
					escaped += valueString[begin:idx]
					escaped += "\\" + string(c)
					begin = idx + 1
				}
			}
			escaped += valueString[begin:]
			s += typeName + "=" + escaped
		}
	}
	return s
}

var attributeTypeNames = map[string]string{
	"2.5.4.6":  "C",
	"2.5.4.10": "O",
	"2.5.4.11": "OU",
	"2.5.4.3":  "CN",
	"2.5.4.5":  "SERIALNUMBER",
	"2.5.4.7":  "L",
	"2.5.4.8":  "ST",
	"2.5.4.9":  "STREET",
	"2.5.4.17": "POSTALCODE",
}
