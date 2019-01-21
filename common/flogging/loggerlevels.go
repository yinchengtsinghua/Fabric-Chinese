
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


package flogging

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

//loggerlevels跟踪已命名记录器的日志级别。
type LoggerLevels struct {
	defaultLevel zapcore.Level

	mutex      sync.RWMutex
	levelCache map[string]zapcore.Level
	specs      map[string]zapcore.Level
}

//DefaultLevel返回不具有的记录器的默认日志记录级别
//显式级别集。
func (l *LoggerLevels) DefaultLevel() zapcore.Level {
	l.mutex.RLock()
	lvl := l.defaultLevel
	l.mutex.RUnlock()
	return lvl
}

//activatespec用于修改日志记录级别。
//
//日志规范的格式如下：
//[<logger>[，<logger>…]=]<level>[：[<logger>[，<logger>…]=]<level>…]
func (l *LoggerLevels) ActivateSpec(spec string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	defaultLevel := zapcore.InfoLevel
	specs := map[string]zapcore.Level{}
	for _, field := range strings.Split(spec, ":") {
		split := strings.Split(field, "=")
		switch len(split) {
case 1: //水平
			if field != "" && !IsValidLevel(field) {
				return errors.Errorf("invalid logging specification '%s': bad segment '%s'", spec, field)
			}
			defaultLevel = NameToLevel(field)

case 2: //<logger>[，<logger>…]=<level>
			if split[0] == "" {
				return errors.Errorf("invalid logging specification '%s': no logger specified in segment '%s'", spec, field)
			}
			if field != "" && !IsValidLevel(split[1]) {
				return errors.Errorf("invalid logging specification '%s': bad segment '%s'", spec, field)
			}

			level := NameToLevel(split[1])
			loggers := strings.Split(split[0], ",")
			for _, logger := range loggers {
//检查规范中的记录器名称是否有效。这个
//尾随句点在规范中作为记录器名称进行修剪。
//以句号结尾表示
//spec指的是确切的记录器名称（即不是前缀）
				if !isValidLoggerName(strings.TrimSuffix(logger, ".")) {
					return errors.Errorf("invalid logging specification '%s': bad logger name '%s'", spec, logger)
				}
				specs[logger] = level
			}

		default:
			return errors.Errorf("invalid logging specification '%s': bad segment '%s'", spec, field)
		}
	}

	l.defaultLevel = defaultLevel
	l.specs = specs
	l.levelCache = map[string]zapcore.Level{}

	return nil
}

//logggernameregexp定义有效的记录器名称
var loggerNameRegexp = regexp.MustCompile(`^[[:alnum:]_#:-]+(\.[[:alnum:]_#:-]+)*$`)

//IsvalidLoggerName检查记录器名称是否仅包含有效的
//字符。以句点开头/结尾或包含特殊字符的名称
//字符（句点、下划线、磅符号、冒号除外）
//
func isValidLoggerName(loggerName string) bool {
	return loggerNameRegexp.MatchString(loggerName)
}

//LEVEL返回记录器的有效日志记录级别。如果一个级别没有
//已为记录器显式设置，默认日志级别将为
//返回。
func (l *LoggerLevels) Level(loggerName string) zapcore.Level {
	if level, ok := l.cachedLevel(loggerName); ok {
		return level
	}

	l.mutex.Lock()
	level := l.calculateLevel(loggerName)
	l.levelCache[loggerName] = level
	l.mutex.Unlock()

	return level
}

//CalculateLevel返回记录器名称以查找适当的
//当前规格的日志级别。
func (l *LoggerLevels) calculateLevel(loggerName string) zapcore.Level {
	candidate := loggerName + "."
	for {
		if lvl, ok := l.specs[candidate]; ok {
			return lvl
		}

		idx := strings.LastIndex(candidate, ".")
		if idx <= 0 {
			return l.defaultLevel
		}
		candidate = candidate[:idx]
	}
}

//cachedLevel尝试从
//隐藏物。如果没有找到记录器，则OK将为false。
func (l *LoggerLevels) cachedLevel(loggerName string) (lvl zapcore.Level, ok bool) {
	l.mutex.RLock()
	level, ok := l.levelCache[loggerName]
	l.mutex.RUnlock()
	return level, ok
}

//spec返回活动日志规范的规范化版本。
func (l *LoggerLevels) Spec() string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	var fields []string
	for k, v := range l.specs {
		fields = append(fields, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(fields)
	fields = append(fields, l.defaultLevel.String())

	return strings.Join(fields, ":")
}
