
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


package deliverclient

import (
	"math"

	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
)

type blocksRequester struct {
	tls     bool
	chainID string
	client  blocksprovider.BlocksDeliverer
}

func (b *blocksRequester) RequestBlocks(ledgerInfoProvider blocksprovider.LedgerInfo) error {
	height, err := ledgerInfoProvider.LedgerHeight()
	if err != nil {
		logger.Errorf("Can't get ledger height for channel %s from committer [%s]", b.chainID, err)
		return err
	}

	if height > 0 {
		logger.Debugf("Starting deliver with block [%d] for channel %s", height, b.chainID)
		if err := b.seekLatestFromCommitter(height); err != nil {
			return err
		}
	} else {
		logger.Debugf("Starting deliver with oldest block for channel %s", b.chainID)
		if err := b.seekOldest(); err != nil {
			return err
		}
	}

	return nil
}

func (b *blocksRequester) getTLSCertHash() []byte {
	if b.tls {
		return util.ComputeSHA256(comm.GetCredentialSupport().GetClientCertificate().Certificate[0])
	}
	return nil
}

func (b *blocksRequester) seekOldest() error {
	seekInfo := &orderer.SeekInfo{
		Start:    &orderer.SeekPosition{Type: &orderer.SeekPosition_Oldest{Oldest: &orderer.SeekOldest{}}},
		Stop:     &orderer.SeekPosition{Type: &orderer.SeekPosition_Specified{Specified: &orderer.SeekSpecified{Number: math.MaxUint64}}},
		Behavior: orderer.SeekInfo_BLOCK_UNTIL_READY,
	}

//在order/configupdate/configupdate.go中使用后，可能需要立即获取epoch和msgversion。
	msgVersion := int32(0)
	epoch := uint64(0)
	tlsCertHash := b.getTLSCertHash()
	env, err := utils.CreateSignedEnvelopeWithTLSBinding(common.HeaderType_DELIVER_SEEK_INFO, b.chainID, localmsp.NewSigner(), seekInfo, msgVersion, epoch, tlsCertHash)
	if err != nil {
		return err
	}
	return b.client.Send(env)
}

func (b *blocksRequester) seekLatestFromCommitter(height uint64) error {
	seekInfo := &orderer.SeekInfo{
		Start:    &orderer.SeekPosition{Type: &orderer.SeekPosition_Specified{Specified: &orderer.SeekSpecified{Number: height}}},
		Stop:     &orderer.SeekPosition{Type: &orderer.SeekPosition_Specified{Specified: &orderer.SeekSpecified{Number: math.MaxUint64}}},
		Behavior: orderer.SeekInfo_BLOCK_UNTIL_READY,
	}

//在order/configupdate/configupdate.go中使用后，可能需要立即获取epoch和msgversion。
	msgVersion := int32(0)
	epoch := uint64(0)
	tlsCertHash := b.getTLSCertHash()
	env, err := utils.CreateSignedEnvelopeWithTLSBinding(common.HeaderType_DELIVER_SEEK_INFO, b.chainID, localmsp.NewSigner(), seekInfo, msgVersion, epoch, tlsCertHash)
	if err != nil {
		return err
	}
	return b.client.Send(env)
}
