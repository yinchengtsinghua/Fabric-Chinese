
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


package stateleveldb

import (
	proto "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/stateleveldb/msgs"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
)

//编码值对版本化值进行编码。从v1.3开始，编码以nil开始
//字节并包含元数据。
func encodeValue(v *statedb.VersionedValue) ([]byte, error) {
	vvMsg := &msgs.VersionedValueProto{
		VersionBytes: v.Version.ToBytes(),
		Value:        v.Value,
		Metadata:     v.Metadata,
	}
	encodedValue, err := proto.Marshal(vvMsg)
	if err != nil {
		return nil, err
	}
	encodedValue = append([]byte{0}, encodedValue...)
	return encodedValue, nil
}

//decodeValue使用旧（v1.3之前）编码对statedb值字节进行解码
//或者支持元数据的新编码（v1.3及更高版本）。
func decodeValue(encodedValue []byte) (*statedb.VersionedValue, error) {
	if oldFormatEncoding(encodedValue) {
		val, ver := decodeValueOldFormat(encodedValue)
		return &statedb.VersionedValue{Version: ver, Value: val, Metadata: nil}, nil
	}
	msg := &msgs.VersionedValueProto{}
	err := proto.Unmarshal(encodedValue[1:], msg)
	if err != nil {
		return nil, err
	}
	ver, _ := version.NewHeightFromBytes(msg.VersionBytes)
	val := msg.Value
	metadata := msg.Metadata
//protobuf总是将空字节数组设为nil
	if val == nil {
		val = []byte{}
	}
	return &statedb.VersionedValue{Version: ver, Value: val, Metadata: metadata}, nil
}

//encodeValueOldFormat将值追加到版本，允许以二进制形式存储版本和值。
//随着v1.3中元数据功能的引入，我们更改了编码（见下面的函数）。但是，我们保留
//此功能用于测试，以确保我们可以解码旧格式并支持现有的混合格式。
//在一个状态B中。This function should be used only in tests to generate the encoding in old format
func encodeValueOldFormat(value []byte, version *version.Height) []byte {
	encodedValue := version.ToBytes()
	if value != nil {
		encodedValue = append(encodedValue, value...)
	}
	return encodedValue
}

//CuffDeValueOLDFRID将版本和值从二进制值分离
//请参见函数“encodeValueOldFormat”中的注释。我们保留这个功能
//使用它来解码STATEDB中存在的旧格式（PREV1.3）数据。这个函数
//不应直接或在测试中使用。应使用函数“decodeValue”
//对于所有编码-期望检测编码格式并引导调用
//此函数用于解码以旧格式编码的值
func decodeValueOldFormat(encodedValue []byte) ([]byte, *version.Height) {
	height, n := version.NewHeightFromBytes(encodedValue)
	value := encodedValue[n:]
	return value, height
}

//OldFormatEncoding检查值是否使用旧（v1.3之前）格式编码
//或新格式（v1.3及更高版本用于编码元数据）。
func oldFormatEncoding(encodedValue []byte) bool {
	return encodedValue[0] != byte(0) ||
(encodedValue[0]|encodedValue[1]) == byte(0) //这张支票包括一个角箱
//如果旧的格式化值恰好以零字节开头。在这个角落里，
//对于tuple<block 0，tran 0>来说，通道配置恰好是持久的。所以，这个
//假定块0包含单个事务（即事务处理0）
}
