
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


package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/cmd/common"
	"github.com/hyperledger/fabric/discovery/client"
	. "github.com/hyperledger/fabric/protos/discovery"
	"github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

//新建背书命令创建新的背书命令
func NewEndorsersCmd(stub Stub, parser ResponseParser) *EndorsersCmd {
	return &EndorsersCmd{
		stub:   stub,
		parser: parser,
	}
}

//背书命令执行一个命令，该命令检索链码调用链的背书人。
type EndorsersCmd struct {
	stub        Stub
	server      *string
	channel     *string
	chaincodes  *[]string
	collections *map[string]string
	parser      ResponseParser
}

//SetCollections sets the collections to be the given collections
func (pc *EndorsersCmd) SetCollections(collections *map[string]string) {
	pc.collections = collections
}

//setchaincodes将链码设置为给定的链码
func (pc *EndorsersCmd) SetChaincodes(chaincodes *[]string) {
	pc.chaincodes = chaincodes
}

//设置服务器设置服务器
func (pc *EndorsersCmd) SetServer(server *string) {
	pc.server = server
}

//setchannel设置通道
func (pc *EndorsersCmd) SetChannel(channel *string) {
	pc.channel = channel
}

//执行执行命令
func (pc *EndorsersCmd) Execute(conf common.Config) error {
	if pc.channel == nil || *pc.channel == "" {
		return errors.New("no channel specified")
	}

	if pc.server == nil || *pc.server == "" {
		return errors.New("no server specified")
	}

	server := *pc.server
	channel := *pc.channel

	ccAndCol := &chaincodesAndCollections{
		Chaincodes:  pc.chaincodes,
		Collections: pc.collections,
	}
	cc2collections, err := ccAndCol.parseInput()
	if err != nil {
		return err
	}

	var ccCalls []*ChaincodeCall

	for _, cc := range *ccAndCol.Chaincodes {
		ccCalls = append(ccCalls, &ChaincodeCall{
			Name:            cc,
			CollectionNames: cc2collections[cc],
		})
	}

	req, err := discovery.NewRequest().OfChannel(channel).AddEndorsersQuery(&ChaincodeInterest{Chaincodes: ccCalls})
	if err != nil {
		return errors.Wrap(err, "failed creating request")
	}

	res, err := pc.stub.Send(server, conf, req)
	if err != nil {
		return err
	}

	return pc.parser.ParseResponse(channel, res)
}

//RemarkerResponseParser解析来自对等方的认可响应
type EndorserResponseParser struct {
	io.Writer
}

//ParseResponse解析给定通道的给定响应
func (parser *EndorserResponseParser) ParseResponse(channel string, res ServiceResponse) error {
	rawResponse := res.Raw()
	if len(rawResponse.Results) == 0 {
		return errors.New("empty results")
	}

	if e := rawResponse.Results[0].GetError(); e != nil {
		return errors.Errorf("server returned: %s", e.Content)
	}

	ccQueryRes := rawResponse.Results[0].GetCcQueryRes()
	if ccQueryRes == nil {
		return errors.Errorf("server returned response of unexpected type: %v", reflect.TypeOf(rawResponse.Results[0]))
	}

	jsonBytes, _ := json.MarshalIndent(parseEndorsementDescriptors(ccQueryRes.Content), "", "\t")
	fmt.Fprintln(parser.Writer, string(jsonBytes))
	return nil
}

type chaincodesAndCollections struct {
	Chaincodes  *[]string
	Collections *map[string]string
}

func (ec *chaincodesAndCollections) existsInChaincodes(chaincodeName string) bool {
	for _, cc := range *ec.Chaincodes {
		if chaincodeName == cc {
			return true
		}
	}
	return false
}

func (ec *chaincodesAndCollections) parseInput() (map[string][]string, error) {
	var emptyChaincodes []string
	if ec.Chaincodes == nil {
		ec.Chaincodes = &emptyChaincodes
	}
	var emptyCollections map[string]string
	if ec.Collections == nil {
		ec.Collections = &emptyCollections
	}

	res := make(map[string][]string)

	for _, cc := range *ec.Chaincodes {
		res[cc] = nil
	}

	for cc, collections := range *ec.Collections {
		if !ec.existsInChaincodes(cc) {
			return nil, errors.Errorf("a collection specified chaincode %s but it wasn't specified with a chaincode flag", cc)
		}
		res[cc] = strings.Split(collections, ",")
	}
	return res, nil
}

func parseEndorsementDescriptors(descriptors []*EndorsementDescriptor) []endorsermentDescriptor {
	var res []endorsermentDescriptor
	for _, desc := range descriptors {
		endorsersByGroups := make(map[string][]endorser)
		for grp, endorsers := range desc.EndorsersByGroups {
			for _, p := range endorsers.Peers {
				endorsersByGroups[grp] = append(endorsersByGroups[grp], endorserFromRaw(p))
			}
		}
		res = append(res, endorsermentDescriptor{
			Chaincode:         desc.Chaincode,
			Layouts:           desc.Layouts,
			EndorsersByGroups: endorsersByGroups,
		})
	}
	return res
}

type endorser struct {
	MSPID        string
	LedgerHeight uint64
	Endpoint     string
	Identity     string
}

type endorsermentDescriptor struct {
	Chaincode         string
	EndorsersByGroups map[string][]endorser
	Layouts           []*Layout
}

func endorserFromRaw(p *Peer) endorser {
	sId := &msp.SerializedIdentity{}
	proto.Unmarshal(p.Identity, sId)
	return endorser{
		MSPID:        sId.Mspid,
		Endpoint:     endpointFromEnvelope(p.MembershipInfo),
		LedgerHeight: ledgerHeightFromEnvelope(p.StateInfo),
		Identity:     string(sId.IdBytes),
	}
}

func endpointFromEnvelope(env *gossip.Envelope) string {
	if env == nil {
		return ""
	}
	aliveMsg, _ := env.ToGossipMessage()
	if aliveMsg == nil {
		return ""
	}
	if !aliveMsg.IsAliveMsg() {
		return ""
	}
	if aliveMsg.GetAliveMsg().Membership == nil {
		return ""
	}
	return aliveMsg.GetAliveMsg().Membership.Endpoint
}

func ledgerHeightFromEnvelope(env *gossip.Envelope) uint64 {
	if env == nil {
		return 0
	}
	stateInfoMsg, _ := env.ToGossipMessage()
	if stateInfoMsg == nil {
		return 0
	}
	if !stateInfoMsg.IsStateInfoMsg() {
		return 0
	}
	if stateInfoMsg.GetStateInfo().Properties == nil {
		return 0
	}
	return stateInfoMsg.GetStateInfo().Properties.LedgerHeight
}
