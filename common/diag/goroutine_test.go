
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package diag_test

import (
	"testing"

	"github.com/hyperledger/fabric/common/diag"
	"github.com/hyperledger/fabric/common/flogging/floggingtest"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

func TestCaptureGoRoutines(t *testing.T) {
	gt := NewGomegaWithT(t)
	output, err := diag.CaptureGoRoutines()
	gt.Expect(err).NotTo(HaveOccurred())

	gt.Expect(output).To(MatchRegexp(`goroutine \d+ \[running\]:`))
	gt.Expect(output).To(ContainSubstring("github.com/hyperledger/fabric/common/diag.CaptureGoRoutines"))
}

func TestLogGoRoutines(t *testing.T) {
	gt := NewGomegaWithT(t)
	logger, recorder := floggingtest.NewTestLogger(t, floggingtest.Named("goroutine"))
	diag.LogGoRoutines(logger)

	gt.Expect(recorder).To(gbytes.Say(`goroutine \d+ \[running\]:`))
}
