
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


package statecouchdb

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	proto "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb/msgs"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
)

func encodeVersionAndMetadata(version *version.Height, metadata []byte) (string, error) {
	msg := &msgs.VersionFieldProto{
		VersionBytes: version.ToBytes(),
		Metadata:     metadata,
	}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return "", err
	}
	msgBase64 := base64.StdEncoding.EncodeToString(msgBytes)
	encodedVersionField := append([]byte{byte(0)}, []byte(msgBase64)...)
	return string(encodedVersionField), nil
}

func decodeVersionAndMetadata(encodedstr string) (*version.Height, []byte, error) {
	if oldFormatEncoding(encodedstr) {
		return decodeVersionOldFormat(encodedstr), nil, nil
	}
	versionFieldBytes, err := base64.StdEncoding.DecodeString(encodedstr[1:])
	if err != nil {
		return nil, nil, err
	}
	versionFieldMsg := &msgs.VersionFieldProto{}
	if err = proto.Unmarshal(versionFieldBytes, versionFieldMsg); err != nil {
		return nil, nil, err
	}
	ver, _ := version.NewHeightFromBytes(versionFieldMsg.VersionBytes)
	return ver, versionFieldMsg.Metadata, nil
}

//encodeVersionOldFormat return string representation of version
//随着元数据特性的引入，我们改变了编码（见下面的函数）。但是，我们保留
//此功能用于测试，以确保我们可以解码旧格式并支持现有的混合格式。
//在一个状态B中。This function should be used only in tests to generate the encoding in old format
func encodeVersionOldFormat(version *version.Height) string {
	return fmt.Sprintf("%v:%v", version.BlockNum, version.TxNum)
}

//decodeversionoldformat将版本和值与编码字符串分开
//请参见“encodeversionoldformat”函数中的注释。我们保留这个功能
//to use this for decoding the old format data present in the statedb. 这个函数
//不应直接或在测试中使用。应使用函数“decodeVersionAndMetadata”
//对于所有编码-期望检测编码格式并引导调用
//to this function for decoding the versions encoded in the old format
func decodeVersionOldFormat(encodedVersion string) *version.Height {
	versionArray := strings.Split(fmt.Sprintf("%s", encodedVersion), ":")
//将blocknum从string转换为unsigned int
	blockNum, _ := strconv.ParseUint(versionArray[0], 10, 64)
//将txnum从字符串转换为无符号int
	txNum, _ := strconv.ParseUint(versionArray[1], 10, 64)
	return version.NewHeight(blockNum, txNum)
}

func oldFormatEncoding(encodedstr string) bool {
	return []byte(encodedstr)[0] != byte(0)
}
