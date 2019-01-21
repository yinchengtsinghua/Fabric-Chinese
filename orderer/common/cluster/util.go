
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package cluster

import (
	"bytes"
	"encoding/hex"
	"encoding/pem"
	"reflect"
	"sync/atomic"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/configtx"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

//connbycertmap将证书表示为字符串
//到GRPC连接
type ConnByCertMap map[string]*grpc.ClientConn

//查找查找证书并返回映射的连接
//到证书，以及是否找到它
func (cbc ConnByCertMap) Lookup(cert []byte) (*grpc.ClientConn, bool) {
	conn, ok := cbc[string(cert)]
	return conn, ok
}

//将给定的连接与证书关联
func (cbc ConnByCertMap) Put(cert []byte, conn *grpc.ClientConn) {
	cbc[string(cert)] = conn
}

//删除删除与给定证书关联的连接
func (cbc ConnByCertMap) Remove(cert []byte) {
	delete(cbc, string(cert))
}

//MemberMapping按ID定义NetworkMembers
type MemberMapping map[uint64]*Stub

//将给定的存根插入到成员映射
func (mp MemberMapping) Put(stub *Stub) {
	mp[stub.ID] = stub
}

//ByID从成员映射中检索具有给定ID的存根
func (mp MemberMapping) ByID(ID uint64) *Stub {
	return mp[ID]
}

//lookupbyclientcert检索具有给定客户端证书的存根
func (mp MemberMapping) LookupByClientCert(cert []byte) *Stub {
	for _, stub := range mp {
		if bytes.Equal(stub.ClientTLSCert, cert) {
			return stub
		}
	}
	return nil
}

//ServerCertificates返回一组服务器证书
//表示为字符串
func (mp MemberMapping) ServerCertificates() StringSet {
	res := make(StringSet)
	for _, member := range mp {
		res[string(member.ServerTLSCert)] = struct{}{}
	}
	return res
}

//字符串集是一组字符串
type StringSet map[string]struct{}

//联合将给定集的元素添加到字符串集
func (ss StringSet) union(set StringSet) {
	for k := range set {
		ss[k] = struct{}{}
	}
}

//减法从字符串集中删除给定集合中的所有元素
func (ss StringSet) subtract(set StringSet) {
	for k := range set {
		delete(ss, k)
	}
}

//谓词拨号程序创建GRPC连接
//只有当给定的谓词
//履行
type PredicateDialer struct {
	Config atomic.Value
}

//newtlspiningdialer创建新的谓词拨号程序
func NewTLSPinningDialer(config comm.ClientConfig) *PredicateDialer {
	d := &PredicateDialer{}
	d.SetConfig(config)
	return d
}

//clientconfig返回comm.clientconfig或错误
//如果无法提取。
func (dialer *PredicateDialer) ClientConfig() (comm.ClientConfig, error) {
	val := dialer.Config.Load()
	if val == nil {
		return comm.ClientConfig{}, errors.New("client config not initialized")
	}
	cc, isClientConfig := val.(comm.ClientConfig)
	if !isClientConfig {
		err := errors.Errorf("value stored is %v, not comm.ClientConfig",
			reflect.TypeOf(val))
		return comm.ClientConfig{}, err
	}
	if cc.SecOpts == nil {
		return comm.ClientConfig{}, errors.New("SecOpts is nil")
	}
//按值复制安全选项
	secOpts := *cc.SecOpts
	return comm.ClientConfig{
		AsyncConnect: cc.AsyncConnect,
		Timeout:      cc.Timeout,
		SecOpts:      &secOpts,
		KaOpts:       cc.KaOpts,
	}, nil
}

//setconfig设置谓词拨号程序的配置
func (dialer *PredicateDialer) SetConfig(config comm.ClientConfig) {
	configCopy := comm.ClientConfig{
		AsyncConnect: config.AsyncConnect,
		Timeout:      config.Timeout,
		SecOpts:      &comm.SecureOptions{},
		KaOpts:       &comm.KeepaliveOptions{},
	}
//显式复制配置
	if config.SecOpts != nil {
		*configCopy.SecOpts = *config.SecOpts
	}
	if config.KaOpts != nil {
		*configCopy.KaOpts = *config.KaOpts
	} else {
		configCopy.KaOpts = nil
	}

	dialer.Config.Store(configCopy)
}

//拨号创建一个新的GRPC连接，只有在远程节点的
//证书链满足VerifyFunc
func (dialer *PredicateDialer) Dial(address string, verifyFunc RemoteVerifier) (*grpc.ClientConn, error) {
	cfg := dialer.Config.Load().(comm.ClientConfig)
	cfg.SecOpts.VerifyCertificate = verifyFunc
	client, err := comm.NewGRPCClient(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return client.NewConnection(address, "")
}

//Dertopem返回一个DER的PEM表示
//编码证书
func DERtoPEM(der []byte) string {
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	}))
}

//标准拨号程序包装谓词拨号程序
//到一个标准的群集。通过nil验证功能的拨号程序
type StandardDialer struct {
	Dialer *PredicateDialer
}

//拨到指定地址
func (bdp *StandardDialer) Dial(address string) (*grpc.ClientConn, error) {
	return bdp.Dialer.Dial(address, nil)
}

//去：生成mokery-dir。-name blockverifier-case underline-output./mocks/

//BlockVerifier验证块签名。
type BlockVerifier interface {
//verifyblocksignature验证块的签名。
//它有一个配置信封的可选参数
//这将使块验证使用验证规则
//基于配置开发中的给定配置。
//如果传递的配置信封为零，则使用验证规则
//是在提交前一个块时应用的。
	VerifyBlockSignature(sd []*common.SignedData, config *common.ConfigEnvelope) error
}

//BlockSequenceVerifier验证给定的连续序列
//的块有效。
type BlockSequenceVerifier func([]*common.Block) error

//拨号程序创建到远程地址的GRPC连接
type Dialer interface {
	Dial(address string) (*grpc.ClientConn, error)
}

//verifyblocks验证给定的连续块序列是否有效，
//如果有效则返回零，否则为错误。
func VerifyBlocks(blockBuff []*common.Block, signatureVerifier BlockVerifier) error {
	if len(blockBuff) == 0 {
		return errors.New("buffer is empty")
	}
//首先，我们验证每个块中的块哈希是：
//等于头中的哈希
//等于后继块中的前一个哈希
	for i := range blockBuff {
		if err := VerifyBlockHash(i, blockBuff); err != nil {
			return err
		}
	}

	var config *common.ConfigEnvelope
//验证在块批处理中找到的所有配置块，
//使用已提交的配置（零）或已提取的配置
//在块批处理的迭代过程中。
	for _, block := range blockBuff {
		configFromBlock, err := ConfigFromBlock(block)
		if err == errNotAConfig {
			continue
		}
		if err != nil {
			return err
		}
//该块是一个配置块，因此请验证它
		if err := VerifyBlockSignature(block, signatureVerifier, config); err != nil {
			return err
		}
		config = configFromBlock
	}

//验证最后一个块的签名
	lastBlock := blockBuff[len(blockBuff)-1]
	return VerifyBlockSignature(lastBlock, signatureVerifier, config)
}

var errNotAConfig = errors.New("not a config block")

//configFromBlock返回configDevelope（如果存在），或者返回*notaConfigBlock错误。
//如果解析失败，它还可能返回一些其他错误。
func ConfigFromBlock(block *common.Block) (*common.ConfigEnvelope, error) {
	if block == nil || block.Data == nil || len(block.Data.Data) == 0 {
		return nil, errors.New("empty block")
	}
	txn := block.Data.Data[0]
	env, err := utils.GetEnvelopeFromBlock(txn)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	payload, err := utils.GetPayload(env)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if payload.Header == nil {
		return nil, errors.New("nil header in payload")
	}
	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if common.HeaderType(chdr.Type) != common.HeaderType_CONFIG {
		return nil, errNotAConfig
	}
	configEnvelope, err := configtx.UnmarshalConfigEnvelope(payload.Data)
	if err != nil {
		return nil, errors.Wrap(err, "invalid config envelope")
	}
	return configEnvelope, nil
}

//verifyblockhash使用给定的索引验证块的哈希链
//在给定块缓冲区的块中。
func VerifyBlockHash(indexInBuffer int, blockBuff []*common.Block) error {
	if len(blockBuff) <= indexInBuffer {
		return errors.Errorf("index %d out of bounds (total %d blocks)", indexInBuffer, len(blockBuff))
	}
	block := blockBuff[indexInBuffer]
	if block.Header == nil {
		return errors.New("missing block header")
	}
	seq := block.Header.Number
	dataHash := block.Data.Hash()
//验证数据哈希是否与头中的哈希匹配
	if !bytes.Equal(dataHash, block.Header.DataHash) {
		computedHash := hex.EncodeToString(dataHash)
		claimedHash := hex.EncodeToString(block.Header.DataHash)
		return errors.Errorf("computed hash of block (%d) (%s) doesn't match claimed hash (%s)",
			seq, computedHash, claimedHash)
	}
//我们在缓冲区中有一个前一个块，确保当前块的前一个哈希与前一个哈希匹配。
	if indexInBuffer > 0 {
		prevBlock := blockBuff[indexInBuffer-1]
		currSeq := block.Header.Number
		if prevBlock.Header == nil {
			return errors.New("previous block header is nil")
		}
		prevSeq := prevBlock.Header.Number
		if prevSeq+1 != currSeq {
			return errors.Errorf("sequences %d and %d were received consecutively", prevSeq, currSeq)
		}
		if !bytes.Equal(block.Header.PreviousHash, prevBlock.Header.Hash()) {
			claimedPrevHash := hex.EncodeToString(block.Header.PreviousHash)
			actualPrevHash := hex.EncodeToString(prevBlock.Header.Hash())
			return errors.Errorf("block %d's hash (%s) mismatches %d's prev block hash (%s)",
				currSeq, actualPrevHash, prevSeq, claimedPrevHash)
		}
	}
	return nil
}

//SignatureSetFromBlock从块中创建一个签名集。
func SignatureSetFromBlock(block *common.Block) ([]*common.SignedData, error) {
	if block.Metadata == nil || len(block.Metadata.Metadata) <= int(common.BlockMetadataIndex_SIGNATURES) {
		return nil, errors.New("no metadata in block")
	}
	metadata, err := utils.GetMetadataFromBlock(block, common.BlockMetadataIndex_SIGNATURES)
	if err != nil {
		return nil, errors.Errorf("failed unmarshaling medatata for signatures: %v", err)
	}

	var signatureSet []*common.SignedData
	for _, metadataSignature := range metadata.Signatures {
		sigHdr, err := utils.GetSignatureHeader(metadataSignature.SignatureHeader)
		if err != nil {
			return nil, errors.Errorf("failed unmarshaling signature header for block with id %d: %v",
				block.Header.Number, err)
		}
		signatureSet = append(signatureSet,
			&common.SignedData{
				Identity: sigHdr.Creator,
				Data: util.ConcatenateBytes(metadata.Value,
					metadataSignature.SignatureHeader, block.Header.Bytes()),
				Signature: metadataSignature.Signature,
			},
		)
	}
	return signatureSet, nil
}

//verifyblocksignature使用给定的blockverifier和给定的配置验证块上的签名。
func VerifyBlockSignature(block *common.Block, verifier BlockVerifier, config *common.ConfigEnvelope) error {
	signatureSet, err := SignatureSetFromBlock(block)
	if err != nil {
		return err
	}
	return verifier.VerifyBlockSignature(signatureSet, config)
}

//endpointconfig定义配置
//对服务节点排序的端点
type EndpointConfig struct {
	TLSRootCAs [][]byte
	Endpoints  []string
}

//EndpointConfigFromConfigBlock检索TLS CA证书和终结点
//来自配置块。
func EndpointconfigFromConfigBlock(block *common.Block) (*EndpointConfig, error) {
	if block == nil {
		return nil, errors.New("nil block")
	}
	envelopeConfig, err := utils.ExtractEnvelope(block, 0)
	if err != nil {
		return nil, err
	}
	var tlsCACerts [][]byte
	bundle, err := channelconfig.NewBundleFromEnvelope(envelopeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed extracting bundle from envelope")
	}
	msps, err := bundle.MSPManager().GetMSPs()
	if err != nil {
		return nil, errors.Wrap(err, "failed obtaining MSPs from MSPManager")
	}
	ordererConfig, ok := bundle.OrdererConfig()
	if !ok {
		return nil, errors.New("failed obtaining orderer config from bundle")
	}
	for _, org := range ordererConfig.Organizations() {
		msp := msps[org.MSPID()]
		if msp == nil {
			return nil, errors.Errorf("no MSP found for MSP with ID of %s", org.MSPID())
		}
		tlsCACerts = append(tlsCACerts, msp.GetTLSRootCerts()...)
	}
	return &EndpointConfig{
		Endpoints:  bundle.ChannelConfig().OrdererAddresses(),
		TLSRootCAs: tlsCACerts,
	}, nil
}
