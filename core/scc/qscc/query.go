
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


package qscc

import (
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/aclmgmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/peer"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

//new返回qsc的实例。
//通常每个对等机调用一次。
func New(aclProvider aclmgmt.ACLProvider) *LedgerQuerier {
	return &LedgerQuerier{
		aclProvider: aclProvider,
	}
}

func (e *LedgerQuerier) Name() string              { return "qscc" }
func (e *LedgerQuerier) Path() string              { return "github.com/hyperledger/fabric/core/scc/qscc" }
func (e *LedgerQuerier) InitArgs() [][]byte        { return nil }
func (e *LedgerQuerier) Chaincode() shim.Chaincode { return e }
func (e *LedgerQuerier) InvokableExternal() bool   { return true }
func (e *LedgerQuerier) InvokableCC2CC() bool      { return true }
func (e *LedgerQuerier) Enabled() bool             { return true }

//LedgerQueryer实现了分类账查询功能，包括：
//GETCHANIN信息返回BuffChanIn信息
//-getblockbynumber返回一个块
//GETBaseByHASH返回块
//-getTransactionByID返回事务
type LedgerQuerier struct {
	aclProvider aclmgmt.ACLProvider
}

var qscclogger = flogging.MustGetLogger("qscc")

//这些是来自invoke first参数的函数名
const (
	GetChainInfo       string = "GetChainInfo"
	GetBlockByNumber   string = "GetBlockByNumber"
	GetBlockByHash     string = "GetBlockByHash"
	GetTransactionByID string = "GetTransactionByID"
	GetBlockByTxID     string = "GetBlockByTxID"
)

//创建链时，每个链调用一次init。
//这允许chaincode在
//链上的任何事务执行。
func (e *LedgerQuerier) Init(stub shim.ChaincodeStubInterface) pb.Response {
	qscclogger.Info("Init QSCC")

	return shim.Success(nil)
}

//调用invoke时使用args[0]包含查询函数名args[1]
//包含链ID，暂时是临时的，直到它是存根的一部分。
//每个函数都需要如下所述的其他参数：
//getchaininfo：返回以字节为单位封送的blockchaininfo对象
//getBlockByNumber：返回由args[2]中的块号指定的块。
//# GetBlockByHash: Return the block specified by block hash in args[2]
//getTransactionByID：返回参数[2]中ID指定的事务
func (e *LedgerQuerier) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()

	if len(args) < 2 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments, %d", len(args)))
	}
	fname := string(args[0])
	cid := string(args[1])

	if fname != GetChainInfo && len(args) < 3 {
		return shim.Error(fmt.Sprintf("missing 3rd argument for %s", fname))
	}

	targetLedger := peer.GetLedger(cid)
	if targetLedger == nil {
		return shim.Error(fmt.Sprintf("Invalid chain ID, %s", cid))
	}

	qscclogger.Debugf("Invoke function: %s on chain: %s", fname, cid)

//处理ACL：
//1。获取已签署的建议
	sp, err := stub.GetSignedProposal()
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed getting signed proposal from stub, %s: %s", cid, err))
	}

//2。检查通道读卡器策略
	res := getACLResource(fname)
	if err = e.aclProvider.CheckACL(res, cid, sp); err != nil {
		return shim.Error(fmt.Sprintf("access denied for [%s][%s]: [%s]", fname, cid, err))
	}

	switch fname {
	case GetTransactionByID:
		return getTransactionByID(targetLedger, args[2])
	case GetBlockByNumber:
		return getBlockByNumber(targetLedger, args[2])
	case GetBlockByHash:
		return getBlockByHash(targetLedger, args[2])
	case GetChainInfo:
		return getChainInfo(targetLedger)
	case GetBlockByTxID:
		return getBlockByTxID(targetLedger, args[2])
	}

	return shim.Error(fmt.Sprintf("Requested function %s not found.", fname))
}

func getTransactionByID(vledger ledger.PeerLedger, tid []byte) pb.Response {
	if tid == nil {
		return shim.Error("Transaction ID must not be nil.")
	}

	processedTran, err := vledger.GetTransactionByID(string(tid))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get transaction with id %s, error %s", string(tid), err))
	}

	bytes, err := utils.Marshal(processedTran)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
}

func getBlockByNumber(vledger ledger.PeerLedger, number []byte) pb.Response {
	if number == nil {
		return shim.Error("Block number must not be nil.")
	}
	bnum, err := strconv.ParseUint(string(number), 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to parse block number with error %s", err))
	}
	block, err := vledger.GetBlockByNumber(bnum)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get block number %d, error %s", bnum, err))
	}
//TODO:返回前考虑修剪块内容
//具体来说，将事务“数据”从事务数组有效负载中去掉
//这将保留事务有效负载头，
//如果需要完整的事务详细信息，客户端可以执行getTransactionByID（）。

	bytes, err := utils.Marshal(block)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
}

func getBlockByHash(vledger ledger.PeerLedger, hash []byte) pb.Response {
	if hash == nil {
		return shim.Error("Block hash must not be nil.")
	}
	block, err := vledger.GetBlockByHash(hash)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get block hash %s, error %s", string(hash), err))
	}
//TODO:返回前考虑修剪块内容
//具体来说，将事务“数据”从事务数组有效负载中去掉
//这将保留事务有效负载头，
//如果需要完整的事务详细信息，客户端可以执行getTransactionByID（）。

	bytes, err := utils.Marshal(block)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
}

func getChainInfo(vledger ledger.PeerLedger) pb.Response {
	binfo, err := vledger.GetBlockchainInfo()
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get block info with error %s", err))
	}
	bytes, err := utils.Marshal(binfo)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
}

func getBlockByTxID(vledger ledger.PeerLedger, rawTxID []byte) pb.Response {
	txID := string(rawTxID)
	block, err := vledger.GetBlockByTxID(txID)

	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get block for txID %s, error %s", txID, err))
	}

	bytes, err := utils.Marshal(block)

	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
}

func getACLResource(fname string) string {
	return "qscc/" + fname
}
