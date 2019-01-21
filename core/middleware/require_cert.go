
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
	"net/http"
)

type requireCert struct {
	next http.Handler
}

//RequireCert用于确保验证的TLS客户端证书
//用于身份验证。
func RequireCert() Middleware {
	return func(next http.Handler) http.Handler {
		return &requireCert{next: next}
	}
}

func (r *requireCert) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case req.TLS == nil:
		fallthrough
	case len(req.TLS.VerifiedChains) == 0:
		fallthrough
	case len(req.TLS.VerifiedChains[0]) == 0:
		w.WriteHeader(http.StatusUnauthorized)
	default:
		r.next.ServeHTTP(w, req)
	}
}
