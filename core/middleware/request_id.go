
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


package middleware

import (
	"context"
	"net/http"
)

var requestIDKey = requestIDKeyType{}

type requestIDKeyType struct{}

func RequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		return reqID
	}
	return "unknown"
}

type GenerateIDFunc func() string

type requestID struct {
	generateID GenerateIDFunc
	next       http.Handler
}

func WithRequestID(generator GenerateIDFunc) Middleware {
	return func(next http.Handler) http.Handler {
		return &requestID{next: next, generateID: generator}
	}
}

func (r *requestID) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	reqID := req.Header.Get("X-Request-Id")
	if reqID == "" {
		reqID = r.generateID()
		req.Header.Set("X-Request-Id", reqID)
	}

	ctx := context.WithValue(req.Context(), requestIDKey, reqID)
	req = req.WithContext(ctx)

	w.Header().Add("X-Request-Id", reqID)

	r.next.ServeHTTP(w, req)
}
