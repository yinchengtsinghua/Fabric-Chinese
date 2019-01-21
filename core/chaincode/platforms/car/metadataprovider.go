
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有，State Street Corp.保留所有权利。
γ
SPDX许可证标识符：Apache-2.0
**/

package car

import (
	"archive/tar"
	"bytes"
)

//元数据提供程序提供元数据
type MetadataProvider struct {
}

//getmetadataastarentries从chaincodedeploymentspec中提取metata数据
func (carMetadataProv *MetadataProvider) GetMetadataAsTarEntries() ([]byte, error) {
//这将转换汽车生成的元数据
//与生成的tar条目相同的tar条目
//其他平台。
//
//当前未实现，用户不能通过汽车指定元数据

	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
