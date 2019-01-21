
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
*/


package gendoc

import (
	"fmt"
	"go/ast"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/common/metrics"
	"golang.org/x/tools/go/packages"
)

//
//
//
func Options(pkgs []*packages.Package) ([]interface{}, error) {
	var options []interface{}
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			opts, err := FileOptions(f)
			if err != nil {
				return nil, err
			}
			options = append(options, opts...)
		}
	}
	return options, nil
}

//
//
func FileOptions(f *ast.File) ([]interface{}, error) {
	var imports = walkImports(f)
	var options []interface{}
	var errors []error

//
	for _, c := range f.Comments {
		if strings.Index(c.Text(), "gendoc:ignore") != -1 {
			return nil, nil
		}
	}

//
	for i := range f.Decls {
		ast.Inspect(f.Decls[i], func(x ast.Node) bool {
			node, ok := x.(*ast.ValueSpec)
			if !ok {
				return true
			}

			for _, v := range node.Values {
				value, ok := v.(*ast.CompositeLit)
				if !ok {
					continue
				}
				literalType, ok := value.Type.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				ident, ok := literalType.X.(*ast.Ident)
				if !ok {
					continue
				}
				if imports[ident.Name] != "github.com/hyperledger/fabric/common/metrics" {
					continue
				}
				option, err := createOption(literalType)
				if err != nil {
					errors = append(errors, err)
					break
				}
				option, err = populateOption(value, option)
				if err != nil {
					errors = append(errors, err)
					break
				}
				options = append(options, option)
			}
			return false
		})
	}

	if len(errors) != 0 {
		return nil, errors[0]
	}

	return options, nil
}

func walkImports(f *ast.File) map[string]string {
	imports := map[string]string{}

	for i := range f.Imports {
		ast.Inspect(f.Imports[i], func(x ast.Node) bool {
			switch node := x.(type) {
			case *ast.ImportSpec:
				importPath, err := strconv.Unquote(node.Path.Value)
				if err != nil {
					panic(err)
				}
				importName := path.Base(importPath)
				if node.Name != nil {
					importName = node.Name.Name
				}
				imports[importName] = importPath
				return false

			default:
				return true
			}
		})
	}

	return imports
}

func createOption(lit *ast.SelectorExpr) (interface{}, error) {
	optionName := lit.Sel.Name
	switch optionName {
	case "CounterOpts":
		return &metrics.CounterOpts{}, nil
	case "GaugeOpts":
		return &metrics.GaugeOpts{}, nil
	case "HistogramOpts":
		return &metrics.HistogramOpts{}, nil
	default:
		return nil, fmt.Errorf("unknown object type: %s", optionName)
	}
}

func populateOption(lit *ast.CompositeLit, target interface{}) (interface{}, error) {
	val := reflect.ValueOf(target).Elem()
	for _, elem := range lit.Elts {
		if kv, ok := elem.(*ast.KeyValueExpr); ok {
			name := kv.Key.(*ast.Ident).Name
			field := val.FieldByName(name)

			switch name {
//忽略
			case "Buckets":

//
			case "LabelNames":
				labelNames, err := stringSlice(kv.Value.(*ast.CompositeLit))
				if err != nil {
					return nil, err
				}
				labelNamesValue := reflect.ValueOf(labelNames)
				field.Set(labelNamesValue)

//
			case "Namespace", "Subsystem", "Name", "Help", "StatsdFormat":
				basicVal := kv.Value.(*ast.BasicLit)
				val, err := strconv.Unquote(basicVal.Value)
				if err != nil {
					return nil, err
				}
				field.SetString(val)

			default:
				return nil, fmt.Errorf("unknown field name: %s", name)
			}
		}
	}
	return val.Interface(), nil
}

func stringSlice(lit *ast.CompositeLit) ([]string, error) {
	var slice []string

	for _, elem := range lit.Elts {
		val, err := strconv.Unquote(elem.(*ast.BasicLit).Value)
		if err != nil {
			return nil, err
		}
		slice = append(slice, val)
	}

	return slice, nil
}
