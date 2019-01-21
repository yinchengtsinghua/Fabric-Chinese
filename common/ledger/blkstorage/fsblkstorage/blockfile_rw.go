
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


package fsblkstorage

import (
	"os"

	"github.com/pkg/errors"
)

////作者/／／／
type blockfileWriter struct {
	filePath string
	file     *os.File
}

func newBlockfileWriter(filePath string) (*blockfileWriter, error) {
	writer := &blockfileWriter{filePath: filePath}
	return writer, writer.open()
}

func (w *blockfileWriter) truncateFile(targetSize int) error {
	fileStat, err := w.file.Stat()
	if err != nil {
		return err
	}
	if fileStat.Size() > int64(targetSize) {
		w.file.Truncate(int64(targetSize))
	}
	return nil
}

func (w *blockfileWriter) append(b []byte, sync bool) error {
	_, err := w.file.Write(b)
	if err != nil {
		return err
	}
	if sync {
		return w.file.Sync()
	}
	return nil
}

func (w *blockfileWriter) open() error {
	file, err := os.OpenFile(w.filePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return errors.Wrapf(err, "error opening block file writer for file %s", w.filePath)
	}
	w.file = file
	return nil
}

func (w *blockfileWriter) close() error {
	return errors.WithStack(w.file.Close())
}

////Reader／//／
type blockfileReader struct {
	file *os.File
}

func newBlockfileReader(filePath string) (*blockfileReader, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0600)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening block file reader for file %s", filePath)
	}
	reader := &blockfileReader{file}
	return reader, nil
}

func (r *blockfileReader) read(offset int, length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := r.file.ReadAt(b, int64(offset))
	if err != nil {
		return nil, errors.Wrapf(err, "error reading block file for offset %d and length %d", offset, length)
	}
	return b, nil
}

func (r *blockfileReader) close() error {
	return errors.WithStack(r.file.Close())
}
