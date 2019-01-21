
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


package viperutil

import (
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	version "github.com/hashicorp/go-version"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var logger = flogging.MustGetLogger("viperutil")

type viperGetter func(key string) interface{}

func getKeysRecursively(base string, getKey viperGetter, nodeKeys map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key := range nodeKeys {
		fqKey := base + key
		val := getKey(fqKey)
		if m, ok := val.(map[interface{}]interface{}); ok {
			logger.Debugf("Found map[interface{}]interface{} value for %s", fqKey)
			tmp := make(map[string]interface{})
			for ik, iv := range m {
				cik, ok := ik.(string)
				if !ok {
					panic("Non string key-entry")
				}
				tmp[cik] = iv
			}
			result[key] = getKeysRecursively(fqKey+".", getKey, tmp)
		} else if m, ok := val.(map[string]interface{}); ok {
			logger.Debugf("Found map[string]interface{} value for %s", fqKey)
			result[key] = getKeysRecursively(fqKey+".", getKey, m)
		} else if m, ok := unmarshalJSON(val); ok {
			logger.Debugf("Found real value for %s setting to map[string]string %v", fqKey, m)
			result[key] = m
		} else {
			if val == nil {
				fileSubKey := fqKey + ".File"
				fileVal := getKey(fileSubKey)
				if fileVal != nil {
					result[key] = map[string]interface{}{"File": fileVal}
					continue
				}
			}
			logger.Debugf("Found real value for %s setting to %T %v", fqKey, val, val)
			result[key] = val

		}
	}
	return result
}

func unmarshalJSON(val interface{}) (map[string]string, bool) {
	mp := map[string]string{}
	s, ok := val.(string)
	if !ok {
		logger.Debugf("Unmarshal JSON: value is not a string: %v", val)
		return nil, false
	}
	err := json.Unmarshal([]byte(s), &mp)
	if err != nil {
		logger.Debugf("Unmarshal JSON: value cannot be unmarshalled: %s", err)
		return nil, false
	}
	return mp, true
}

//customdecodehook添加了从字符串分析持续时间的附加函数
//以及将格式为“[thing1，thing2，thing3]”的字符串解析为字符串片
//注意，slice元素周围的空白被删除。
func customDecodeHook() mapstructure.DecodeHookFunc {
	durationHook := mapstructure.StringToTimeDurationHookFunc()
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		dur, err := mapstructure.DecodeHookExec(durationHook, f, t, data)
		if err == nil {
			if _, ok := dur.(time.Duration); ok {
				return dur, nil
			}
		}

		if f.Kind() != reflect.String {
			return data, nil
		}

		raw := data.(string)
		l := len(raw)
		if l > 1 && raw[0] == '[' && raw[l-1] == ']' {
			slice := strings.Split(raw[1:l-1], ",")
			for i, v := range slice {
				slice[i] = strings.TrimSpace(v)
			}
			return slice, nil
		}

		return data, nil
	}
}

func byteSizeDecodeHook() mapstructure.DecodeHookFunc {
	return func(f reflect.Kind, t reflect.Kind, data interface{}) (interface{}, error) {
		if f != reflect.String || t != reflect.Uint32 {
			return data, nil
		}
		raw := data.(string)
		if raw == "" {
			return data, nil
		}
		var re = regexp.MustCompile(`^(?P<size>[0-9]+)\s*(?i)(?P<unit>(k|m|g))b?$`)
		if re.MatchString(raw) {
			size, err := strconv.ParseUint(re.ReplaceAllString(raw, "${size}"), 0, 64)
			if err != nil {
				return data, nil
			}
			unit := re.ReplaceAllString(raw, "${unit}")
			switch strings.ToLower(unit) {
			case "g":
				size = size << 10
				fallthrough
			case "m":
				size = size << 10
				fallthrough
			case "k":
				size = size << 10
			}
			if size > math.MaxUint32 {
				return size, fmt.Errorf("value '%s' overflows uint32", raw)
			}
			return size, nil
		}
		return data, nil
	}
}

func stringFromFileDecodeHook() mapstructure.DecodeHookFunc {
	return func(f reflect.Kind, t reflect.Kind, data interface{}) (interface{}, error) {
//“to”类型应为字符串
		if t != reflect.String {
			return data, nil
		}
//“发件人”类型应为map
		if f != reflect.Map {
			return data, nil
		}
		v := reflect.ValueOf(data)
		switch v.Kind() {
		case reflect.String:
			return data, nil
		case reflect.Map:
			d := data.(map[string]interface{})
			fileName, ok := d["File"]
			if !ok {
				fileName, ok = d["file"]
			}
			switch {
			case ok && fileName != nil:
				bytes, err := ioutil.ReadFile(fileName.(string))
				if err != nil {
					return data, err
				}
				return string(bytes), nil
			case ok:
//文件名为nil
				return nil, fmt.Errorf("Value of File: was nil")
			}
		}
		return data, nil
	}
}

func pemBlocksFromFileDecodeHook() mapstructure.DecodeHookFunc {
	return func(f reflect.Kind, t reflect.Kind, data interface{}) (interface{}, error) {
//“to”类型应为字符串
		if t != reflect.Slice {
			return data, nil
		}
//“发件人”类型应为map
		if f != reflect.Map {
			return data, nil
		}
		v := reflect.ValueOf(data)
		switch v.Kind() {
		case reflect.String:
			return data, nil
		case reflect.Map:
			var fileName string
			var ok bool
			switch d := data.(type) {
			case map[string]string:
				fileName, ok = d["File"]
				if !ok {
					fileName, ok = d["file"]
				}
			case map[string]interface{}:
				var fileI interface{}
				fileI, ok = d["File"]
				if !ok {
					fileI, _ = d["file"]
				}
				fileName, ok = fileI.(string)
			}

			switch {
			case ok && fileName != "":
				var result []string
				bytes, err := ioutil.ReadFile(fileName)
				if err != nil {
					return data, err
				}
				for len(bytes) > 0 {
					var block *pem.Block
					block, bytes = pem.Decode(bytes)
					if block == nil {
						break
					}
					if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
						continue
					}
					result = append(result, string(pem.EncodeToMemory(block)))
				}
				return result, nil
			case ok:
//文件名为nil
				return nil, fmt.Errorf("Value of File: was nil")
			}
		}
		return data, nil
	}
}

var kafkaVersionConstraints map[sarama.KafkaVersion]version.Constraints

func init() {
	kafkaVersionConstraints = make(map[sarama.KafkaVersion]version.Constraints)
	kafkaVersionConstraints[sarama.V0_8_2_0], _ = version.NewConstraint(">=0.8.2,<0.8.2.1")
	kafkaVersionConstraints[sarama.V0_8_2_1], _ = version.NewConstraint(">=0.8.2.1,<0.8.2.2")
	kafkaVersionConstraints[sarama.V0_8_2_2], _ = version.NewConstraint(">=0.8.2.2,<0.9.0.0")
	kafkaVersionConstraints[sarama.V0_9_0_0], _ = version.NewConstraint(">=0.9.0.0,<0.9.0.1")
	kafkaVersionConstraints[sarama.V0_9_0_1], _ = version.NewConstraint(">=0.9.0.1,<0.10.0.0")
	kafkaVersionConstraints[sarama.V0_10_0_0], _ = version.NewConstraint(">=0.10.0.0,<0.10.0.1")
	kafkaVersionConstraints[sarama.V0_10_0_1], _ = version.NewConstraint(">=0.10.0.1,<0.10.1.0")
	kafkaVersionConstraints[sarama.V0_10_1_0], _ = version.NewConstraint(">=0.10.1.0,<0.10.2.0")
	kafkaVersionConstraints[sarama.V0_10_2_0], _ = version.NewConstraint(">=0.10.2.0,<0.11.0.0")
	kafkaVersionConstraints[sarama.V0_11_0_0], _ = version.NewConstraint(">=0.11.0.0,<1.0.0")
	kafkaVersionConstraints[sarama.V1_0_0_0], _ = version.NewConstraint(">=1.0.0")
}

func kafkaVersionDecodeHook() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String || t != reflect.TypeOf(sarama.KafkaVersion{}) {
			return data, nil
		}

		v, err := version.NewVersion(data.(string))
		if err != nil {
			return nil, fmt.Errorf("Unable to parse Kafka version: %s", err)
		}

		for kafkaVersion, constraints := range kafkaVersionConstraints {
			if constraints.Check(v) {
				return kafkaVersion, nil
			}
		}

		return nil, fmt.Errorf("Unsupported Kafka version: '%s'", data)
	}
}

//EnhancedExecutionMarshal旨在将配置文件解组为结构
//引入外来变量时产生误差及支持
//时间。持续时间类型
func EnhancedExactUnmarshal(v *viper.Viper, output interface{}) error {
//all keys并不实际返回所有密钥，它只返回基本密钥
	baseKeys := v.AllSettings()
getterWithClass := func(key string) interface{} { return v.Get(key) } //隐藏接收器
	leafKeys := getKeysRecursively("", getterWithClass, baseKeys)

	logger.Debugf("%+v", leafKeys)
	config := &mapstructure.DecoderConfig{
		ErrorUnused:      true,
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			customDecodeHook(),
			byteSizeDecodeHook(),
			stringFromFileDecodeHook(),
			pemBlocksFromFileDecodeHook(),
			kafkaVersionDecodeHook(),
		),
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(leafKeys)
}

//EnhancedExecutionMarshalKey用于将配置文件子目录解组为结构
func EnhancedExactUnmarshalKey(baseKey string, output interface{}) error {
	m := make(map[string]interface{})
	m[baseKey] = nil
	leafKeys := getKeysRecursively("", viper.Get, m)

	logger.Debugf("%+v", leafKeys)

	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(leafKeys[baseKey])
}

//decode用于解码配置中的不透明字段
func Decode(input interface{}, output interface{}) error {
	return mapstructure.Decode(input, output)
}
