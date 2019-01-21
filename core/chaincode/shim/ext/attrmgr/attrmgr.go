
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


/*
 *attrmgr包包含用于管理属性的实用程序。
 *属性作为扩展添加到X509证书中。
 **/


package attrmgr

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

var (
//attroid是ASN.1对象标识符，用于
//X509证书
	AttrOID = asn1.ObjectIdentifier{1, 2, 3, 4, 5, 6, 7, 8, 1}
//AttroidString是Attroid的字符串版本
	AttrOIDString = "1.2.3.4.5.6.7.8.1"
)

//属性是名称/值对
type Attribute interface {
//getname返回属性的名称
	GetName() string
//GetValue返回属性的值
	GetValue() string
}

//attributeRequest是对属性的请求
type AttributeRequest interface {
//getname返回属性的名称
	GetName() string
//如果需要属性，isRequired返回true
	IsRequired() bool
}

//新建构造属性管理器
func New() *Mgr { return &Mgr{} }

//mgr是属性管理器，是此包的主要对象
type Mgr struct{}

//processAttributeRequestsForcert将属性添加到给定的X509证书
//属性请求和属性。
func (mgr *Mgr) ProcessAttributeRequestsForCert(requests []AttributeRequest, attributes []Attribute, cert *x509.Certificate) error {
	attrs, err := mgr.ProcessAttributeRequests(requests, attributes)
	if err != nil {
		return err
	}
	return mgr.AddAttributesToCert(attrs, cert)
}

//processAttributeRequests接受一个属性请求数组和一个标识的属性
//并返回包含请求的属性的属性对象。
func (mgr *Mgr) ProcessAttributeRequests(requests []AttributeRequest, attributes []Attribute) (*Attributes, error) {
	attrsMap := map[string]string{}
	attrs := &Attributes{Attrs: attrsMap}
	missingRequiredAttrs := []string{}
//对于每个属性请求
	for _, req := range requests {
//获取属性
		name := req.GetName()
		attr := getAttrByName(name, attributes)
		if attr == nil {
			if req.IsRequired() {
//找不到属性，它是必需的；返回下面的错误
				missingRequiredAttrs = append(missingRequiredAttrs, name)
			}
//跳过不需要的属性请求
			continue
		}
		attrsMap[name] = attr.GetValue()
	}
	if len(missingRequiredAttrs) > 0 {
		return nil, errors.Errorf("The following required attributes are missing: %+v",
			missingRequiredAttrs)
	}
	return attrs, nil
}

//addattributestocert将公共属性信息添加到X509证书中。
func (mgr *Mgr) AddAttributesToCert(attrs *Attributes, cert *x509.Certificate) error {
	buf, err := json.Marshal(attrs)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal attributes")
	}
	ext := pkix.Extension{
		Id:       AttrOID,
		Critical: false,
		Value:    buf,
	}
	cert.Extensions = append(cert.Extensions, ext)
	return nil
}

//getattributesfromcert从证书获取属性。
func (mgr *Mgr) GetAttributesFromCert(cert *x509.Certificate) (*Attributes, error) {
//从证书中获取证书属性（如果存在）
	buf, err := getAttributesFromCert(cert)
	if err != nil {
		return nil, err
	}
//取消标记为属性对象
	attrs := &Attributes{}
	if buf != nil {
		err := json.Unmarshal(buf, attrs)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to unmarshal attributes from certificate")
		}
	}
	return attrs, nil
}

func (mgr *Mgr) GetAttributesFromIdemix(creator []byte) (*Attributes, error) {
	if creator == nil {
		return nil, errors.New("creator is nil")
	}

	sid := &msp.SerializedIdentity{}
	err := proto.Unmarshal(creator, sid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction invoker's identity")
	}
	idemixID := &msp.SerializedIdemixIdentity{}
	err = proto.Unmarshal(sid.IdBytes, idemixID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction invoker's idemix identity")
	}
//取消标记为属性对象
	attrs := &Attributes{
		Attrs: make(map[string]string),
	}

	ou := &msp.OrganizationUnit{}
	err = proto.Unmarshal(idemixID.Ou, ou)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction invoker's ou")
	}
	attrs.Attrs["ou"] = ou.OrganizationalUnitIdentifier

	role := &msp.MSPRole{}
	err = proto.Unmarshal(idemixID.Role, role)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction invoker's role")
	}
	var roleStr string
	switch role.Role {
	case 0:
		roleStr = "member"
	case 1:
		roleStr = "admin"
	case 2:
		roleStr = "client"
	case 3:
		roleStr = "peer"
	}
	attrs.Attrs["role"] = roleStr

	return attrs, nil
}

//属性包含属性名称和值
type Attributes struct {
	Attrs map[string]string `json:"attrs"`
}

//名称返回属性的名称
func (a *Attributes) Names() []string {
	i := 0
	names := make([]string, len(a.Attrs))
	for name := range a.Attrs {
		names[i] = name
		i++
	}
	return names
}

//如果找到命名属性，则包含返回true
func (a *Attributes) Contains(name string) bool {
	_, ok := a.Attrs[name]
	return ok
}

//值返回属性的值
func (a *Attributes) Value(name string) (string, bool, error) {
	attr, ok := a.Attrs[name]
	return attr, ok, nil
}

//如果属性“name”的值为true，则返回nil；
//否则，将返回适当的错误。
func (a *Attributes) True(name string) error {
	val, ok, err := a.Value(name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Attribute '%s' was not found", name)
	}
	if val != "true" {
		return fmt.Errorf("Attribute '%s' is not true", name)
	}
	return nil
}

//从证书扩展中获取属性信息，如果找不到，则返回nil
func getAttributesFromCert(cert *x509.Certificate) ([]byte, error) {
	for _, ext := range cert.Extensions {
		if isAttrOID(ext.Id) {
			return ext.Value, nil
		}
	}
	return nil, nil
}

//对象ID是否等于属性信息对象ID？
func isAttrOID(oid asn1.ObjectIdentifier) bool {
	if len(oid) != len(AttrOID) {
		return false
	}
	for idx, val := range oid {
		if val != AttrOID[idx] {
			return false
		}
	}
	return true
}

//按名称从“attrs”获取属性，如果未找到，则为nil
func getAttrByName(name string, attrs []Attribute) Attribute {
	for _, attr := range attrs {
		if attr.GetName() == name {
			return attr
		}
	}
	return nil
}
