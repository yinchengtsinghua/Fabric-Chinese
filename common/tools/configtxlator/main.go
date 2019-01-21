
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


package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/tools/configtxlator/metadata"
	"github.com/hyperledger/fabric/common/tools/configtxlator/rest"
	"github.com/hyperledger/fabric/common/tools/configtxlator/update"
	"github.com/hyperledger/fabric/common/tools/protolator"
	_ "github.com/hyperledger/fabric/protos/common"
cb "github.com/hyperledger/fabric/protos/common" //导入这些以注册原型类型
	_ "github.com/hyperledger/fabric/protos/msp"
	_ "github.com/hyperledger/fabric/protos/orderer"
	_ "github.com/hyperledger/fabric/protos/orderer/etcdraft"
	_ "github.com/hyperledger/fabric/protos/peer"

	"github.com/gorilla/handlers"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

//命令行标志
var (
	app = kingpin.New("configtxlator", "Utility for generating Hyperledger Fabric channel configurations")

	start    = app.Command("start", "Start the configtxlator REST server")
	hostname = start.Flag("hostname", "The hostname or IP on which the REST server will listen").Default("0.0.0.0").String()
	port     = start.Flag("port", "The port on which the REST server will listen").Default("7059").Int()
	cors     = start.Flag("CORS", "Allowable CORS domains, e.g. '*' or 'www.example.com' (may be repeated).").Strings()

	protoEncode       = app.Command("proto_encode", "Converts a JSON document to protobuf.")
	protoEncodeType   = protoEncode.Flag("type", "The type of protobuf structure to encode to.  For example, 'common.Config'.").Required().String()
	protoEncodeSource = protoEncode.Flag("input", "A file containing the JSON document.").Default(os.Stdin.Name()).File()
	protoEncodeDest   = protoEncode.Flag("output", "A file to write the output to.").Default(os.Stdout.Name()).OpenFile(os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

	protoDecode       = app.Command("proto_decode", "Converts a proto message to JSON.")
	protoDecodeType   = protoDecode.Flag("type", "The type of protobuf structure to decode from.  For example, 'common.Config'.").Required().String()
	protoDecodeSource = protoDecode.Flag("input", "A file containing the proto message.").Default(os.Stdin.Name()).File()
	protoDecodeDest   = protoDecode.Flag("output", "A file to write the JSON document to.").Default(os.Stdout.Name()).OpenFile(os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

	computeUpdate          = app.Command("compute_update", "Takes two marshaled common.Config messages and computes the config update which transitions between the two.")
	computeUpdateOriginal  = computeUpdate.Flag("original", "The original config message.").File()
	computeUpdateUpdated   = computeUpdate.Flag("updated", "The updated config message.").File()
	computeUpdateChannelID = computeUpdate.Flag("channel_id", "The name of the channel for this update.").Required().String()
	computeUpdateDest      = computeUpdate.Flag("output", "A file to write the JSON document to.").Default(os.Stdout.Name()).OpenFile(os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

	version = app.Command("version", "Show version information")
)

var logger = flogging.MustGetLogger("configtxlator")

func main() {
	kingpin.Version("0.0.1")
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
//“启动”命令
	case start.FullCommand():
		startServer(fmt.Sprintf("%s:%d", *hostname, *port), *cors)
//“proto_encode”命令
	case protoEncode.FullCommand():
		defer (*protoEncodeSource).Close()
		defer (*protoEncodeDest).Close()
		err := encodeProto(*protoEncodeType, *protoEncodeSource, *protoEncodeDest)
		if err != nil {
			app.Fatalf("Error decoding: %s", err)
		}
	case protoDecode.FullCommand():
		defer (*protoDecodeSource).Close()
		defer (*protoDecodeDest).Close()
		err := decodeProto(*protoDecodeType, *protoDecodeSource, *protoDecodeDest)
		if err != nil {
			app.Fatalf("Error decoding: %s", err)
		}
	case computeUpdate.FullCommand():
		defer (*computeUpdateOriginal).Close()
		defer (*computeUpdateUpdated).Close()
		defer (*computeUpdateDest).Close()
		err := computeUpdt(*computeUpdateOriginal, *computeUpdateUpdated, *computeUpdateDest, *computeUpdateChannelID)
		if err != nil {
			app.Fatalf("Error computing update: %s", err)
		}
//“版本”命令
	case version.FullCommand():
		printVersion()
	}

}

func startServer(address string, cors []string) {
	var err error

	listener, err := net.Listen("tcp", address)
	if err != nil {
		app.Fatalf("Could not bind to address '%s': %s", address, err)
	}

	if len(cors) > 0 {
		origins := handlers.AllowedOrigins(cors)
//注意，configtxlator目前只公开post API，这是
//
		methods := handlers.AllowedMethods([]string{http.MethodPost})
		headers := handlers.AllowedHeaders([]string{"Content-Type"})
		logger.Infof("Serving HTTP requests on %s with CORS %v", listener.Addr(), cors)
		err = http.Serve(listener, handlers.CORS(origins, methods, headers)(rest.NewRouter()))
	} else {
		logger.Infof("Serving HTTP requests on %s", listener.Addr())
		err = http.Serve(listener, rest.NewRouter())
	}

	app.Fatalf("Error starting server:[%s]\n", err)
}

func printVersion() {
	fmt.Println(metadata.GetVersionInfo())
}

func encodeProto(msgName string, input, output *os.File) error {
	msgType := proto.MessageType(msgName)
	if msgType == nil {
		return errors.Errorf("message of type %s unknown", msgType)
	}
	msg := reflect.New(msgType.Elem()).Interface().(proto.Message)

	err := protolator.DeepUnmarshalJSON(input, msg)
	if err != nil {
		return errors.Wrapf(err, "error decoding input")
	}

	out, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrapf(err, "error marshaling")
	}

	_, err = output.Write(out)
	if err != nil {
		return errors.Wrapf(err, "error writing output")
	}

	return nil
}

func decodeProto(msgName string, input, output *os.File) error {
	msgType := proto.MessageType(msgName)
	if msgType == nil {
		return errors.Errorf("message of type %s unknown", msgType)
	}
	msg := reflect.New(msgType.Elem()).Interface().(proto.Message)

	in, err := ioutil.ReadAll(input)
	if err != nil {
		return errors.Wrapf(err, "error reading input")
	}

	err = proto.Unmarshal(in, msg)
	if err != nil {
		return errors.Wrapf(err, "error unmarshaling")
	}

	err = protolator.DeepMarshalJSON(output, msg)
	if err != nil {
		return errors.Wrapf(err, "error encoding output")
	}

	return nil
}

func computeUpdt(original, updated, output *os.File, channelID string) error {
	origIn, err := ioutil.ReadAll(original)
	if err != nil {
		return errors.Wrapf(err, "error reading original config")
	}

	origConf := &cb.Config{}
	err = proto.Unmarshal(origIn, origConf)
	if err != nil {
		return errors.Wrapf(err, "error unmarshaling original config")
	}

	updtIn, err := ioutil.ReadAll(updated)
	if err != nil {
		return errors.Wrapf(err, "error reading updated config")
	}

	updtConf := &cb.Config{}
	err = proto.Unmarshal(updtIn, updtConf)
	if err != nil {
		return errors.Wrapf(err, "error unmarshaling updated config")
	}

	cu, err := update.Compute(origConf, updtConf)
	if err != nil {
		return errors.Wrapf(err, "error computing config update")
	}

	cu.ChannelId = channelID

	outBytes, err := proto.Marshal(cu)
	if err != nil {
		return errors.Wrapf(err, "error marshaling computed config update")
	}

	_, err = output.Write(outBytes)
	if err != nil {
		return errors.Wrapf(err, "error writing config update to output")
	}

	return nil
}
