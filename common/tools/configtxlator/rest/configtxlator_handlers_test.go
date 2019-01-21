
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
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hyperledger/fabric/common/tools/configtxlator/sanitycheck"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func TestProtolatorComputeConfigUpdate(t *testing.T) {
	originalConfig := utils.MarshalOrPanic(&cb.Config{
		ChannelGroup: &cb.ConfigGroup{
			ModPolicy: "foo",
		},
	})

	updatedConfig := utils.MarshalOrPanic(&cb.Config{
		ChannelGroup: &cb.ConfigGroup{
			ModPolicy: "bar",
		},
	})

	buffer := &bytes.Buffer{}
	mpw := multipart.NewWriter(buffer)

	ffw, err := mpw.CreateFormFile("original", "foo")
	assert.NoError(t, err)
	_, err = bytes.NewReader(originalConfig).WriteTo(ffw)
	assert.NoError(t, err)

	ffw, err = mpw.CreateFormFile("updated", "bar")
	assert.NoError(t, err)
	_, err = bytes.NewReader(updatedConfig).WriteTo(ffw)
	assert.NoError(t, err)

	err = mpw.Close()
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/configtxlator/compute/update-from-configs", buffer)
	assert.NoError(t, err)

	req.Header.Set("Content-Type", mpw.FormDataContentType())
	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestProtolatorMissingOriginal(t *testing.T) {
	updatedConfig := utils.MarshalOrPanic(&cb.Config{
		ChannelGroup: &cb.ConfigGroup{
			ModPolicy: "bar",
		},
	})

	buffer := &bytes.Buffer{}
	mpw := multipart.NewWriter(buffer)

	ffw, err := mpw.CreateFormFile("updated", "bar")
	assert.NoError(t, err)
	_, err = bytes.NewReader(updatedConfig).WriteTo(ffw)
	assert.NoError(t, err)

	err = mpw.Close()
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/configtxlator/compute/update-from-configs", buffer)
	assert.NoError(t, err)

	req.Header.Set("Content-Type", mpw.FormDataContentType())
	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProtolatorMissingUpdated(t *testing.T) {
	originalConfig := utils.MarshalOrPanic(&cb.Config{
		ChannelGroup: &cb.ConfigGroup{
			ModPolicy: "bar",
		},
	})

	buffer := &bytes.Buffer{}
	mpw := multipart.NewWriter(buffer)

	ffw, err := mpw.CreateFormFile("original", "bar")
	assert.NoError(t, err)
	_, err = bytes.NewReader(originalConfig).WriteTo(ffw)
	assert.NoError(t, err)

	err = mpw.Close()
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/configtxlator/compute/update-from-configs", buffer)
	assert.NoError(t, err)

	req.Header.Set("Content-Type", mpw.FormDataContentType())
	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProtolatorCorruptProtos(t *testing.T) {
	originalConfig := []byte("Garbage")
	updatedConfig := []byte("MoreGarbage")

	buffer := &bytes.Buffer{}
	mpw := multipart.NewWriter(buffer)

	ffw, err := mpw.CreateFormFile("original", "bar")
	assert.NoError(t, err)
	_, err = bytes.NewReader(originalConfig).WriteTo(ffw)
	assert.NoError(t, err)

	ffw, err = mpw.CreateFormFile("updated", "bar")
	assert.NoError(t, err)
	_, err = bytes.NewReader(updatedConfig).WriteTo(ffw)
	assert.NoError(t, err)

	err = mpw.Close()
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/configtxlator/compute/update-from-configs", buffer)
	assert.NoError(t, err)

	req.Header.Set("Content-Type", mpw.FormDataContentType())
	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestConfigtxlatorSanityCheckConfig(t *testing.T) {
	req, _ := http.NewRequest("POST", "/configtxlator/config/verify", bytes.NewReader(utils.MarshalOrPanic(&cb.Config{})))
	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	outputMsg := &sanitycheck.Messages{}

	err := json.Unmarshal(rec.Body.Bytes(), outputMsg)
	assert.NoError(t, err)
}

func TestConfigtxlatorSanityCheckMalformedConfig(t *testing.T) {
	req, _ := http.NewRequest("POST", "/configtxlator/config/verify", bytes.NewReader([]byte("Garbage")))
	rec := httptest.NewRecorder()
	r := NewRouter()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
