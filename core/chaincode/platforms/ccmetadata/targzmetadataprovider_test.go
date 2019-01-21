
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有State Street Corp.保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package ccmetadata

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type tarEntry struct {
	name    string
	content []byte
}

func getCodePackage(code []byte, entries []tarEntry) []byte {
	codePackage := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(codePackage)
	tw := tar.NewWriter(gw)

	var zeroTime time.Time
	for _, e := range entries {
		tw.WriteHeader(&tar.Header{Name: e.name, Size: int64(len(e.content)), ModTime: zeroTime, AccessTime: zeroTime, ChangeTime: zeroTime})
		tw.Write(e.content)
	}

	tw.WriteHeader(&tar.Header{Name: "fake-path", Size: int64(len(code)), ModTime: zeroTime, AccessTime: zeroTime, ChangeTime: zeroTime})
	tw.Write(code)

	tw.Close()
	gw.Close()

	return codePackage.Bytes()
}

func getNumEntries(tarbytes []byte) (int, error) {
	b := bytes.NewReader(tarbytes)
	tr := tar.NewReader(b)

	count := 0
//对于代码包tar中的每个文件，
//如果路径中有“statedb”，则将其添加到statedb artifact tar中
	for {
		_, err := tr.Next()
		if err == io.EOF {
//只有当没有更多的条目需要扫描时，我们才能到达这里。
			break
		}

		if err != nil {
			return -1, err
		}

		count = count + 1
	}

	return count, nil
}

func TestBadDepSpec(t *testing.T) {
	tp := TargzMetadataProvider{}
	_, err := tp.GetMetadataAsTarEntries()
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "nil code package")
}

func TestNoMetadata(t *testing.T) {
	entries := []tarEntry{{"path/to/a/file", []byte("somdata")}}
	cds := getCodePackage([]byte("cc code"), entries)
	tp := TargzMetadataProvider{cds}
	metadata, err := tp.GetMetadataAsTarEntries()
	assert.Nil(t, err)
	assert.NotNil(t, metadata)
	count, err := getNumEntries(metadata)
	assert.Nil(t, err)
	assert.Equal(t, count, 0)
}

func TestMetadata(t *testing.T) {
	entries := []tarEntry{{"path/to/a/file", []byte("somdata")}, {ccPackageStatedbDir + "/m1", []byte("m1data")}, {ccPackageStatedbDir + "/m2", []byte("m2data")}}
	cds := getCodePackage([]byte("cc code"), entries)
	tp := TargzMetadataProvider{cds}
	metadata, err := tp.GetMetadataAsTarEntries()
	assert.Nil(t, err)
	assert.NotNil(t, metadata)
	count, err := getNumEntries(metadata)
	assert.Nil(t, err)
	assert.Equal(t, count, 2)
}
