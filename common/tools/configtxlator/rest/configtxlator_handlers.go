
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


package rest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/tools/configtxlator/sanitycheck"
	"github.com/hyperledger/fabric/common/tools/configtxlator/update"
	cb "github.com/hyperledger/fabric/protos/common"
)

func fieldBytes(fieldName string, r *http.Request) ([]byte, error) {
	fieldFile, _, err := r.FormFile(fieldName)
	if err != nil {
		return nil, err
	}
	defer fieldFile.Close()

	return ioutil.ReadAll(fieldFile)
}

func fieldConfigProto(fieldName string, r *http.Request) (*cb.Config, error) {
	fieldBytes, err := fieldBytes(fieldName, r)
	if err != nil {
		return nil, fmt.Errorf("error reading field bytes: %s", err)
	}

	config := &cb.Config{}
	err = proto.Unmarshal(fieldBytes, config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling field bytes: %s", err)
	}

	return config, nil
}

func ComputeUpdateFromConfigs(w http.ResponseWriter, r *http.Request) {
	originalConfig, err := fieldConfigProto("original", r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error with field 'original': %s\n", err)
		return
	}

	updatedConfig, err := fieldConfigProto("updated", r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error with field 'updated': %s\n", err)
		return
	}

	configUpdate, err := update.Compute(originalConfig, updatedConfig)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error computing update: %s\n", err)
		return
	}

	configUpdate.ChannelId = r.FormValue("channel")

	encoded, err := proto.Marshal(configUpdate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error marshaling config update: %s\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(encoded)
}

func SanityCheckConfig(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}

	config := &cb.Config{}
	err = proto.Unmarshal(buf, config)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error unmarshaling data to common.Config': %s\n", err)
		return
	}

	fmt.Printf("Sanity checking %+v\n", config)
	sanityCheckMessages, err := sanitycheck.Check(config)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error performing sanity check: %s\n", err)
		return
	}

	resBytes, err := json.Marshal(sanityCheckMessages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error marshaling result to JSON: %s\n", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}
