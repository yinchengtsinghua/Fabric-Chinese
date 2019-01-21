
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

	"github.com/hyperledger/fabric/cmd/common"
	"github.com/hyperledger/fabric/discovery/client"
	"github.com/pkg/errors"
)

//newconfigCmd创建新的configCmd
func NewConfigCmd(stub Stub, parser ResponseParser) *ConfigCmd {
	return &ConfigCmd{
		stub:   stub,
		parser: parser,
	}
}

//configCmd执行检索config的命令
type ConfigCmd struct {
	stub    Stub
	server  *string
	channel *string
	parser  ResponseParser
}

//setserver设置configCmd的服务器
func (pc *ConfigCmd) SetServer(server *string) {
	pc.server = server
}

//setchannel设置configCmd的通道
func (pc *ConfigCmd) SetChannel(channel *string) {
	pc.channel = channel
}

//执行执行命令
func (pc *ConfigCmd) Execute(conf common.Config) error {
	if pc.server == nil || *pc.server == "" {
		return errors.New("no server specified")
	}
	if pc.channel == nil || *pc.channel == "" {
		return errors.New("no channel specified")
	}

	server := *pc.server
	channel := *pc.channel

	req := discovery.NewRequest().OfChannel(channel).AddConfigQuery()
	res, err := pc.stub.Send(server, conf, req)
	if err != nil {
		return err
	}
	return pc.parser.ParseResponse(channel, res)
}

//configresponseparser分析配置响应
type ConfigResponseParser struct {
	io.Writer
}

//ParseResponse解析给定通道的给定响应
func (parser *ConfigResponseParser) ParseResponse(channel string, res ServiceResponse) error {
	chanConf, err := res.ForChannel(channel).Config()
	if err != nil {
		return err
	}
	jsonBytes, _ := json.MarshalIndent(chanConf, "", "\t")
	fmt.Fprintln(parser.Writer, string(jsonBytes))
	return nil
}
