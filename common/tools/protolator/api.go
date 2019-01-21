
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


package protolator

import (
	"github.com/golang/protobuf/proto"
)

////////////////////////////////////////////////////
//
//这组接口和方法的设计允许协议附加Go方法
//对它们而言，这样它们就可以自动编组为人类可读的JSON（其中
//不透明字节字段表示为其扩展的原型内容），然后再次返回
//标准协议消息。
//
//目前有三种不同类型的接口可供Proto实现：
//
//1。staticallyopaque*fieldproto：这些接口应该由具有
//其封送类型在编译时已知的不透明字节字段。这基本上是真的
//用于面向签名的字段，如envelope.payload或header.channelheader
//
//2。variablyopaque*fieldproto：这些接口与staticallyopaque*fieldproto相同
//定义，但在
//staticallyopaque*FieldProto定义。特别是，这允许选择
//一个变量yopaque*fieldproto，依赖于staticallyopaque*fieldproto填充的数据。
//例如，payload.data字段取决于payload.header.channelheader.type字段，
//它沿着一条静态排列的路径。
//
//三。动态*fieldproto：这些接口用于包含其他消息的消息
//无法在编译时确定属性。例如，configValue消息可以计算
//映射字段值[“msp”]在组织上下文中成功，但在通道中完全不成功。
//语境。因为go不是动态语言，所以必须通过
//将基础proto消息包装为另一种类型，在运行时可以使用
//不同的上下文行为。（示例见测试）
//
////////////////////////////////////////////////////

//staticallyopaquefieldproto应该由具有字节字段的proto实现，其中
//是固定类型的封送值
type StaticallyOpaqueFieldProto interface {
//staticallyopaquefields返回包含不透明数据的字段名
	StaticallyOpaqueFields() []string

//staticallyopaquefieldproto返回新分配的正确的proto消息
//键入字段名。
	StaticallyOpaqueFieldProto(name string) (proto.Message, error)
}

//
//
type StaticallyOpaqueMapFieldProto interface {
//staticallyopaquefields返回包含不透明数据的字段名
	StaticallyOpaqueMapFields() []string

//staticallyopaquefieldproto返回新分配的正确的proto消息
//键入字段名。
	StaticallyOpaqueMapFieldProto(name string, key string) (proto.Message, error)
}

//
//它是固定类型的封送值
type StaticallyOpaqueSliceFieldProto interface {
//staticallyopaquefields返回包含不透明数据的字段名
	StaticallyOpaqueSliceFields() []string

//staticallyopaquefieldproto返回新分配的正确的proto消息
//键入字段名。
	StaticallyOpaqueSliceFieldProto(name string, index int) (proto.Message, error)
}

//variablyopaquefieldproto应该由具有字节字段的proto实现，其中
//封送值是否取决于协议的其他内容？
type VariablyOpaqueFieldProto interface {
//variableyopaquefields返回包含不透明数据的字段名
	VariablyOpaqueFields() []string

//variablyopaquefieldproto返回新分配的正确的proto消息
//键入字段名。
	VariablyOpaqueFieldProto(name string) (proto.Message, error)
}

//variablyopaquemapfieldproto应该由映射到字节字段的proto实现。
//它是由协议的其他内容确定的消息类型的封送值。
type VariablyOpaqueMapFieldProto interface {
//variableyopaquefields返回包含不透明数据的字段名
	VariablyOpaqueMapFields() []string

//variablyopaquefieldproto返回新分配的正确的proto消息
//键入字段名。
	VariablyOpaqueMapFieldProto(name string, key string) (proto.Message, error)
}

//variableyopaqueslicefieldproto应该由具有映射到字节字段的协议实现。
//它是由协议的其他内容确定的消息类型的封送值。
type VariablyOpaqueSliceFieldProto interface {
//variableyopaquefields返回包含不透明数据的字段名
	VariablyOpaqueSliceFields() []string

//variablyopaquefieldproto返回新分配的正确的proto消息
//键入字段名。
	VariablyOpaqueSliceFieldProto(name string, index int) (proto.Message, error)
}

//dynamicFieldProto应该由具有其属性的嵌套字段的Proto实现
//（例如它们的不透明类型）在运行时之前无法确定
type DynamicFieldProto interface {
//dynamicFields返回动态字段名
	DynamicFields() []string

//dynamicFieldProto返回新分配的动态消息，修饰底层
//带有运行时确定函数的协议消息
	DynamicFieldProto(name string, underlying proto.Message) (proto.Message, error)
}

//DynamicMapFieldProto应该由具有映射到其属性的消息的Proto实现
//（例如它们的不透明类型）在运行时之前无法确定
type DynamicMapFieldProto interface {
//dynamicFields返回动态字段名
	DynamicMapFields() []string

//dynamicmapfieldproto返回新分配的动态消息，修饰底层
//带有运行时确定函数的协议消息
	DynamicMapFieldProto(name string, key string, underlying proto.Message) (proto.Message, error)
}

//dynamicslicefieldproto应该由具有其属性的消息切片的proto实现。
//（例如它们的不透明类型）在运行时之前无法确定
type DynamicSliceFieldProto interface {
//dynamicFields返回动态字段名
	DynamicSliceFields() []string

//dynamicslicefieldproto返回新分配的动态消息，修饰底层
//带有运行时确定函数的协议消息
	DynamicSliceFieldProto(name string, index int, underlying proto.Message) (proto.Message, error)
}

//deconreatedprot应该由动态*fieldproto接口应用的动态包装器实现。
//
//（而不是按照接口定义（https://github.com/golang/protobuf/issues/291）
type DecoratedProto interface {
//底层返回正在动态修饰的底层协议消息
	Underlying() proto.Message
}
