
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


package rest

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

var (
	testProto = &cb.Block{
		Header: &cb.BlockHeader{
			PreviousHash: []byte("foo"),
		},
		Data: &cb.BlockData{
			Data: [][]byte{
				utils.MarshalOrPanic(&cb.Envelope{
					Payload: utils.MarshalOrPanic(&cb.Payload{
						Header: &cb.Header{
							ChannelHeader: utils.MarshalOrPanic(&cb.ChannelHeader{
								Type: int32(cb.HeaderType_CONFIG),
							}),
						},
					}),
					Signature: []byte("bar"),
				}),
			},
		},
	}

	testOutput = `{"data":{"data":[{"payload":{"data":null,"header":{"channel_header":{"channel_id":"","epoch":"0","extension":null,"timestamp":null,"tls_cert_hash":null,"tx_id":"","type":1,"version":0},"signature_header":null}},"signature":"YmFy"}]},"header":{"data_hash":null,"number":"0","previous_hash":"Zm9v"},"metadata":null}`
)

func TestProtolatorDecode(t *testing.T) {
	data, err := proto.Marshal(testProto)
	assert.NoError(t, err)

	url := fmt.Sprintf("/protolator/decode/%s", proto.MessageName(testProto))

	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

//删除所有空白
	compactJSON := strings.Replace(strings.Replace(strings.Replace(rec.Body.String(), "\n", "", -1), "\t", "", -1), " ", "", -1)

	assert.Equal(t, testOutput, compactJSON)
}

func TestProtolatorEncode(t *testing.T) {

	url := fmt.Sprintf("/protolator/encode/%s", proto.MessageName(testProto))

	req, _ := http.NewRequest("POST", url, bytes.NewReader([]byte(testOutput)))
	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	outputMsg := &cb.Block{}

	err := proto.Unmarshal(rec.Body.Bytes(), outputMsg)
	assert.NoError(t, err)
	assert.True(t, proto.Equal(testProto, outputMsg))
}

func TestProtolatorDecodeNonExistantProto(t *testing.T) {
	req, _ := http.NewRequest("POST", "/protolator/decode/NonExistantMsg", bytes.NewReader([]byte{}))
	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestProtolatorEncodeNonExistantProto(t *testing.T) {
	req, _ := http.NewRequest("POST", "/protolator/encode/NonExistantMsg", bytes.NewReader([]byte{}))
	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestProtolatorDecodeBadData(t *testing.T) {
	url := fmt.Sprintf("/protolator/decode/%s", proto.MessageName(testProto))

	req, _ := http.NewRequest("POST", url, bytes.NewReader([]byte("Garbage")))

	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProtolatorEncodeBadData(t *testing.T) {
	url := fmt.Sprintf("/protolator/encode/%s", proto.MessageName(testProto))

	req, _ := http.NewRequest("POST", url, bytes.NewReader([]byte("Garbage")))

	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
