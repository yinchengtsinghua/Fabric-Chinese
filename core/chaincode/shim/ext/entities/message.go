
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
*/


package entities

import (
	"encoding/json"

	"github.com/pkg/errors"
)

//SignedMessage是一个包含空格的简单结构
//为了有效载荷和它上面的签名，以及方便
//签名、验证、封送和取消封送的功能
type SignedMessage struct {
//ID包含对此消息签名的实体的描述
	ID []byte `json:"id"`

//有效负载包含已签名的消息
	Payload []byte `json:"payload"`

//
	Sig []byte `json:"sig"`
}

//
func (m *SignedMessage) Sign(signer Signer) error {
	if signer == nil {
		return errors.New("nil signer")
	}

	m.Sig = nil
	bytes, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "sign error: json.Marshal returned")
	}
	sig, err := signer.Sign(bytes)
	if err != nil {
		return errors.WithMessage(err, "sign error: signer.Sign returned")
	}
	m.Sig = sig

	return nil
}

//
func (m *SignedMessage) Verify(verifier Signer) (bool, error) {
	if verifier == nil {
		return false, errors.New("nil verifier")
	}

	sig := m.Sig
	m.Sig = nil
	defer func() {
		m.Sig = sig
	}()

	bytes, err := json.Marshal(m)
	if err != nil {
		return false, errors.Wrap(err, "sign error: json.Marshal returned")
	}

	return verifier.Verify(sig, bytes)
}

//
func (m *SignedMessage) ToBytes() ([]byte, error) {
	return json.Marshal(m)
}

//
func (m *SignedMessage) FromBytes(d []byte) error {
	return json.Unmarshal(d, m)
}
