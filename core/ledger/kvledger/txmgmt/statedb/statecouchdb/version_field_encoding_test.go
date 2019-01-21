
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
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeOldAndNewFormat(t *testing.T) {
	testdata := []*statedb.VersionedValue{
		{
			Version: version.NewHeight(1, 2),
		},
		{
			Version: version.NewHeight(50, 50),
		},
		{
			Version:  version.NewHeight(50, 50),
			Metadata: []byte("sample-metadata"),
		},
	}

	for i, testdatum := range testdata {
		t.Run(fmt.Sprintf("testcase-newfmt-%d", i),
			func(t *testing.T) { testEncodeDecodeNewFormat(t, testdatum) },
		)
	}

	for i, testdatum := range testdata {
		t.Run(fmt.Sprintf("testcase-oldfmt-%d", i),
			func(t *testing.T) { testEncodeDecodeOldFormat(t, testdatum) },
		)
	}
}

func testEncodeDecodeNewFormat(t *testing.T, v *statedb.VersionedValue) {
	encodedVerField, err := encodeVersionAndMetadata(v.Version, v.Metadata)
	assert.NoError(t, err)

	ver, metadata, err := decodeVersionAndMetadata(encodedVerField)
	assert.NoError(t, err)
	assert.Equal(t, v.Version, ver)
	assert.Equal(t, v.Metadata, metadata)
}

func testEncodeDecodeOldFormat(t *testing.T, v *statedb.VersionedValue) {
	encodedVerField := encodeVersionOldFormat(v.Version)
//函数“decodeversionandmetadata”应该能够处理旧格式
	ver, metadata, err := decodeVersionAndMetadata(encodedVerField)
	assert.NoError(t, err)
	assert.Equal(t, v.Version, ver)
	assert.Nil(t, metadata)
}
