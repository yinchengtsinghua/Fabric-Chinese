
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


package common

import (
	"io/ioutil"

	"github.com/hyperledger/fabric/cmd/common/comm"
	"github.com/hyperledger/fabric/cmd/common/signer"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	v12 = iota
)

//配置聚合TLS和签名的配置
type Config struct {
	Version      int
	TLSConfig    comm.Config
	SignerConfig signer.Config
}

//configFromFile加载给定的文件并将其转换为config
func ConfigFromFile(file string) (Config, error) {
	configData, err := ioutil.ReadFile(file)
	if err != nil {
		return Config{}, errors.WithStack(err)
	}
	config := Config{}

	if err := yaml.Unmarshal([]byte(configData), &config); err != nil {
		return Config{}, errors.Errorf("error unmarshaling YAML file %s: %s", file, err)
	}

	return config, validateConfig(config)
}

//tofile将配置写入文件
func (c Config) ToFile(file string) error {
	if err := validateConfig(c); err != nil {
		return errors.Wrap(err, "config isn't valid")
	}
	b, _ := yaml.Marshal(c)
	if err := ioutil.WriteFile(file, b, 0600); err != nil {
		return errors.Errorf("failed writing file %s: %v", file, err)
	}
	return nil
}

func validateConfig(conf Config) error {
	nonEmptyStrings := []string{
		conf.SignerConfig.MSPID,
		conf.SignerConfig.IdentityPath,
		conf.SignerConfig.KeyPath,
	}

	for _, s := range nonEmptyStrings {
		if s == "" {
			return errors.New("empty string that is mandatory")
		}
	}
	return nil
}
