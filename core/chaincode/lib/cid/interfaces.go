
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

import "crypto/x509"

//可部署的chaincode应用程序使用chaincodestubinterface获取标识
//提交事务的代理（或用户）。
type ChaincodeStubInterface interface {
//getcreator返回“signatureheader.creator”（例如标识）
//“signedProposal”的。这是代理（或用户）的标识
//提交交易记录。
	GetCreator() ([]byte, error)
}

//clientIdentity表示有关提交
//交易
type ClientIdentity interface {

//GetID返回与调用标识关联的ID。这个身份证
//保证在MSP中是唯一的。
	GetID() (string, error)

//返回客户端的MSP ID
	GetMSPID() (string, error)

//getattributeValue返回名为“attrname”的客户端属性的值。
//如果客户端拥有该属性，“found”为true，“value”等于
//属性的值。
//如果客户端不具有该属性，“found”为false，“value”
//等于“”。
	GetAttributeValue(attrName string) (value string, found bool, err error)

//断言attributeValue验证客户端是否具有名为“attrname”的属性。
//值为“attrvalue”；否则返回错误。
	AssertAttributeValue(attrName, attrValue string) error

//getx509certificate返回与客户端关联的x509证书，
//如果未通过X509证书识别，则为零。
	GetX509Certificate() (*x509.Certificate, error)
}
