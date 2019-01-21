
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
*/


package msp

import (
	"time"

	m "github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/msp"
)

type noopmsp struct {
}

//
func NewNoopMsp() m.MSP {
	return &noopmsp{}
}

func (msp *noopmsp) Setup(*msp.MSPConfig) error {
	return nil
}

func (msp *noopmsp) GetVersion() m.MSPVersion {
	return m.MSPv1_0
}

func (msp *noopmsp) GetType() m.ProviderType {
	return 0
}

func (msp *noopmsp) GetIdentifier() (string, error) {
	return "NOOP", nil
}

func (msp *noopmsp) GetSigningIdentity(identifier *m.IdentityIdentifier) (m.SigningIdentity, error) {
	id, _ := newNoopSigningIdentity()
	return id, nil
}

func (msp *noopmsp) GetDefaultSigningIdentity() (m.SigningIdentity, error) {
	id, _ := newNoopSigningIdentity()
	return id, nil
}

//
func (msp *noopmsp) GetRootCerts() []m.Identity {
	return nil
}

//GetIntermediateCerts返回此MSP的中间根证书
func (msp *noopmsp) GetIntermediateCerts() []m.Identity {
	return nil
}

//
func (msp *noopmsp) GetTLSRootCerts() [][]byte {
	return nil
}

//GettlIntermediateCenters返回此MSP的中间根证书
func (msp *noopmsp) GetTLSIntermediateCerts() [][]byte {
	return nil
}

func (msp *noopmsp) DeserializeIdentity(serializedID []byte) (m.Identity, error) {
	id, _ := newNoopIdentity()
	return id, nil
}

func (msp *noopmsp) Validate(id m.Identity) error {
	return nil
}

func (msp *noopmsp) SatisfiesPrincipal(id m.Identity, principal *msp.MSPPrincipal) error {
	return nil
}

//iswell格式检查给定的标识是否可以反序列化为其提供程序特定的形式
func (msp *noopmsp) IsWellFormed(_ *msp.SerializedIdentity) error {
	return nil
}

type noopidentity struct {
}

func newNoopIdentity() (m.Identity, error) {
	return &noopidentity{}, nil
}

func (id *noopidentity) Anonymous() bool {
	panic("implement me")
}

func (id *noopidentity) SatisfiesPrincipal(*msp.MSPPrincipal) error {
	return nil
}

func (id *noopidentity) ExpiresAt() time.Time {
	return time.Time{}
}

func (id *noopidentity) GetIdentifier() *m.IdentityIdentifier {
	return &m.IdentityIdentifier{Mspid: "NOOP", Id: "Bob"}
}

func (id *noopidentity) GetMSPIdentifier() string {
	return "MSPID"
}

func (id *noopidentity) Validate() error {
	return nil
}

func (id *noopidentity) GetOrganizationalUnits() []*m.OUIdentifier {
	return nil
}

func (id *noopidentity) Verify(msg []byte, sig []byte) error {
	return nil
}

func (id *noopidentity) Serialize() ([]byte, error) {
	return []byte("cert"), nil
}

type noopsigningidentity struct {
	noopidentity
}

func newNoopSigningIdentity() (m.SigningIdentity, error) {
	return &noopsigningidentity{}, nil
}

func (id *noopsigningidentity) Sign(msg []byte) ([]byte, error) {
	return []byte("signature"), nil
}

func (id *noopsigningidentity) GetPublicVersion() m.Identity {
	return id
}
