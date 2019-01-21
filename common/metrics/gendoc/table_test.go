
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
*/


package gendoc_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hyperledger/fabric/common/metrics/gendoc"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Table", func() {
	It("generates a markdown document the prometheus metrics", func() {
		var options []interface{}

		filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
			defer GinkgoRecover()
			if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
				f, err := ParseFile(path)
				Expect(err).NotTo(HaveOccurred())
				opts, err := gendoc.FileOptions(f)
				Expect(err).NotTo(HaveOccurred())
				options = append(options, opts...)
			}
			return nil
		})

		cells, err := gendoc.NewCells(options)
		Expect(err).NotTo(HaveOccurred())

		buf := &bytes.Buffer{}
		w := io.MultiWriter(buf, GinkgoWriter)

		gendoc.NewPrometheusTable(cells).Generate(w)
		Expect(buf.String()).To(Equal(strings.TrimPrefix(goldenPromTable, "\n")))
	})

	It("generates a markdown document the statsd metrics", func() {
		var options []interface{}

		filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
			defer GinkgoRecover()
			if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
				f, err := ParseFile(path)
				Expect(err).NotTo(HaveOccurred())
				opts, err := gendoc.FileOptions(f)
				Expect(err).NotTo(HaveOccurred())
				options = append(options, opts...)
			}
			return nil
		})

		cells, err := gendoc.NewCells(options)
		Expect(err).NotTo(HaveOccurred())

		buf := &bytes.Buffer{}
		w := io.MultiWriter(buf, GinkgoWriter)

		gendoc.NewStatsdTable(cells).Generate(w)
		Expect(buf.String()).To(Equal(strings.TrimPrefix(goldenStatsdTable, "\n")))
	})
})

const goldenPromTable = `
+--------------------------+-----------+------------------------------------------------------------+--------------------+
| Name                     | Type      | Description                                                | Labels             |
+==========================+===========+============================================================+====================+
| fixtures_counter         | counter   | This is some help text that is more than a few words long. | label_one          |
|                          |           | It really can be quite long. Really long.                  | label_two          |
+--------------------------+-----------+------------------------------------------------------------+--------------------+
| fixtures_gauge           | gauge     | This is some help text                                     | label_one          |
|                          |           |                                                            | label_two          |
+--------------------------+-----------+------------------------------------------------------------+--------------------+
| fixtures_histogram       | histogram | This is some help text                                     | label_one          |
|                          |           |                                                            | label_two          |
+--------------------------+-----------+------------------------------------------------------------+--------------------+
| namespace_counter_name   | counter   | This is some help text                                     | label_one          |
|                          |           |                                                            | label_two          |
+--------------------------+-----------+------------------------------------------------------------+--------------------+
| namespace_gauge_name     | gauge     | This is some help text                                     | label_one          |
|                          |           |                                                            | label_two          |
+--------------------------+-----------+------------------------------------------------------------+--------------------+
| namespace_histogram_name | histogram | This is some help text                                     | label_one          |
|                          |           |                                                            | label_two          |
+--------------------------+-----------+------------------------------------------------------------+--------------------+
`

const goldenStatsdTable = `
+----------------------------------------------------+-----------+------------------------------------------------------------+
| Bucket                                             | Type      | Description                                                |
+====================================================+===========+============================================================+
| fixtures.counter.%{label_one}.%{label_two}         | counter   | This is some help text that is more than a few words long. |
|                                                    |           | It really can be quite long. Really long.                  |
+----------------------------------------------------+-----------+------------------------------------------------------------+
| fixtures.gauge.%{label_one}.%{label_two}           | gauge     | This is some help text                                     |
+----------------------------------------------------+-----------+------------------------------------------------------------+
| fixtures.histogram.%{label_one}.%{label_two}       | histogram | This is some help text                                     |
+----------------------------------------------------+-----------+------------------------------------------------------------+
| namespace.counter.name.%{label_one}.%{label_two}   | counter   | This is some help text                                     |
+----------------------------------------------------+-----------+------------------------------------------------------------+
| namespace.gauge.name.%{label_one}.%{label_two}     | gauge     | This is some help text                                     |
+----------------------------------------------------+-----------+------------------------------------------------------------+
| namespace.histogram.name.%{label_one}.%{label_two} | histogram | This is some help text                                     |
+----------------------------------------------------+-----------+------------------------------------------------------------+
`
