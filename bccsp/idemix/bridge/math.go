
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

package bridge

import (
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/hyperledger/fabric/idemix"
)

//大封装AMCL大整数
type Big struct {
	E *FP256BN.BIG
}

func (b *Big) Bytes() ([]byte, error) {
	return idemix.BigToBytes(b.E), nil
}

//ECP封装AMCL椭圆曲线点
type Ecp struct {
	E *FP256BN.ECP
}

func (o *Ecp) Bytes() ([]byte, error) {
	var res []byte
	res = append(res, idemix.BigToBytes(o.E.GetX())...)
	res = append(res, idemix.BigToBytes(o.E.GetY())...)

	return res, nil
}
