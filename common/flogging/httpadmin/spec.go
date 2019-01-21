
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


package httpadmin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hyperledger/fabric/common/flogging"
)

//

type Logging interface {
	ActivateSpec(spec string) error
	Spec() string
}

type LogSpec struct {
	Spec string `json:"spec,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewSpecHandler() *SpecHandler {
	return &SpecHandler{
		Logging: flogging.Global,
		Logger:  flogging.MustGetLogger("flogging.httpadmin"),
	}
}

type SpecHandler struct {
	Logging Logging
	Logger  *flogging.FabricLogger
}

func (h *SpecHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPut:
		var logSpec LogSpec
		decoder := json.NewDecoder(req.Body)
		if err := decoder.Decode(&logSpec); err != nil {
			h.sendResponse(resp, http.StatusBadRequest, err)
			return
		}
		req.Body.Close()

		if err := h.Logging.ActivateSpec(logSpec.Spec); err != nil {
			h.sendResponse(resp, http.StatusBadRequest, err)
			return
		}
		resp.WriteHeader(http.StatusNoContent)

	case http.MethodGet:
		h.sendResponse(resp, http.StatusOK, &LogSpec{Spec: h.Logging.Spec()})

	default:
		err := fmt.Errorf("invalid request method: %s", req.Method)
		h.sendResponse(resp, http.StatusBadRequest, err)
	}
}

func (h *SpecHandler) sendResponse(resp http.ResponseWriter, code int, payload interface{}) {
	encoder := json.NewEncoder(resp)
	if err, ok := payload.(error); ok {
		payload = &ErrorResponse{Error: err.Error()}
	}

	resp.WriteHeader(code)

	resp.Header().Set("Content-Type", "application/json")
	if err := encoder.Encode(payload); err != nil {
		h.Logger.Errorw("failed to encode payload", "error", err)
	}
}
