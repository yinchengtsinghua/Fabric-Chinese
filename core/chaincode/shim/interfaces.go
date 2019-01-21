
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


package shim

import (
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//链码接口必须由所有链码实现。织物运行
//按指定调用这些函数的事务。
type Chaincode interface {
//
//已首次建立，允许链码
//初始化其内部数据
	Init(stub ChaincodeStubInterface) pb.Response

//调用Invoke以更新或查询建议交易中的分类帐。
//更新后的状态变量不会提交到分类帐，直到
//事务已提交。
	Invoke(stub ChaincodeStubInterface) pb.Response
}

//chaincodestubinterface被可部署的chaincode应用程序用于访问和
//修改分类帐
type ChaincodeStubInterface interface {
//getargs返回用于chaincode init和invoke的参数
//作为字节数组的数组。
	GetArgs() [][]byte

//GetStringArgs返回用于chaincode init和
//作为字符串数组调用。仅当客户端通过时才使用GetStringArgs
//用作字符串的参数。
	GetStringArgs() []string

//GetFunctionAndParameters返回第一个参数作为函数
//名称和其他参数作为字符串数组中的参数。
//仅当客户端传递预期的参数时才使用GetFunctionAndParameters
//用作字符串。
	GetFunctionAndParameters() (string, []string)

//getargsslice返回用于chaincode init和
//作为字节数组调用
	GetArgsSlice() ([]byte, error)

//gettxid返回事务建议的tx_id，该id对于
//交易和每个客户。参见protos/common/common.proto中的channelheader
//更多详情。
	GetTxID() string

//getchannelid返回建议发送到的要处理链码的通道。
//这将是事务建议的通道ID（请参见通道标题
//在protos/common/common.proto）中，除非链码正在调用另一个
//另一个频道
	GetChannelID() string

//invoke chaincode使用
//相同的事务上下文；也就是说，调用chaincode的chaincode不会
//
//如果被调用的链码在同一个通道上，则只需添加被调用的
//对调用事务进行chaincode读集和写集。
//
//只有响应返回到调用链代码；任何putstate调用
//从被调用的链码对分类帐没有任何影响；也就是说，
//不同通道上被调用的链码将没有其读取集
//以及应用于事务的写入集。只有呼叫链的
//读集和写集将应用于事务。有效地
//
//在随后的提交阶段参与状态验证检查。
//如果“channel”为空，则假定调用方的频道。
	InvokeChaincode(chaincodeName string, args [][]byte, channel string) pb.Response

//GetState从
//分类帐。注意，getstate不从writeset中读取数据，因为
//
//考虑putstate修改的尚未提交的数据。
//如果键在状态数据库中不存在，则返回（nil，nil）。
	GetState(key string) ([]byte, error)

//putstate将指定的“key”和“value”放入事务
//写集为数据写入建议。Putstate不影响分类帐
//直到事务被验证并成功提交。
//简单键不能是空字符串，也不能以null开头
//字符（0x00），以避免范围查询与
//复合键，内部以0x00作为前缀作为复合键
//关键命名空间。
	PutState(key string, value []byte) error

//Delstate记录要在的写入集中删除的指定“key”
//交易建议。“key”及其值将从中删除
//交易验证并成功提交时的分类帐。
	DelState(key string) error

//setstatevalidationparameter设置“key”的密钥级认可策略。
	SetStateValidationParameter(key string, ep []byte) error

//GetStateValidationParameter检索密钥级认可策略
//用于“钥匙”。请注意，这将引入对“key”的读取依赖项。
//事务的readset。
	GetStateValidationParameter(key string) ([]byte, error)

//GetStateByRange返回在
//分类帐。迭代器可用于遍历所有键
//在startkey（包含）和endkey（不包含）之间。
//但是，如果startkey和endkey之间的键数大于
//totalquerylimit（在core.yaml中定义），不能使用此迭代器
//to fetch all keys (results will be capped by the totalQueryLimit).
//迭代器按词法顺序返回键。注释
//startkey和endkey可以是空字符串，这意味着无边界的范围
//开始或结束时查询。
//完成后，对返回的StateQueryIteratorInterface对象调用Close（）。
//在验证阶段重新执行查询以确保结果集
//自事务认可后未更改（检测到幻象读取）。
	GetStateByRange(startKey, endKey string) (StateQueryIteratorInterface, error)

//GetStateByRangeWithPagination返回在
//分类帐。迭代器可用于在startkey（包括startkey）之间提取键。
//和endkey（独占）。
//当空字符串作为值传递给bookmark参数时，返回的
//迭代器可用于在startkey之间提取第一个“pagesize”键
//
//当书签为非空字符串时，迭代器可用于获取
//书签（含）和endkey（不含）之间的第一个“pagesize”键。
//
//可以用作书签参数的值。否则，空字符串必须
//作为书签传递。
//迭代器按词法顺序返回键。注释
//startkey和endkey可以是空字符串，这意味着无边界的范围
//开始或结束时查询。
//完成后，对返回的StateQueryIteratorInterface对象调用Close（）。
//
	GetStateByRangeWithPagination(startKey, endKey string, pageSize int32,
		bookmark string) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error)

//GetStateByPartialCompositeKey根据
//给定的部分复合键。此函数返回迭代器
//它可用于遍历前缀匹配的所有组合键
//给定的部分复合键。但是，如果匹配组合的数目
//键大于totalquerylimit（在core.yaml中定义），此迭代器
//无法用于提取所有匹配的键（结果将受totalquerylimit的限制）。
//“objectType”和属性只应有有效的utf8字符串和
//不应包含U+0000（零字节）和U+10ffff（最大和未分配的代码点）。
//请参见相关函数splitcompositekey和creatcompositekey。
//完成后，对返回的StateQueryIteratorInterface对象调用Close（）。
//在验证阶段重新执行查询以确保结果集
//自事务认可后未更改（检测到幻象读取）。
	GetStateByPartialCompositeKey(objectType string, keys []string) (StateQueryIteratorInterface, error)

//GetStateByPartialCompositeKeyWithPagination根据
//给定的部分复合键。此函数返回迭代器
//它可以用于在组合键上迭代，
//前缀与给定的部分复合键匹配。
//当空字符串作为值传递给bookmark参数时，返回的
//迭代器可用于获取前缀为
//
//当书签为非空字符串时，迭代器可用于获取
//书签（包含）和最后一个匹配项之间的第一个“pagesize”键
//复合键。
//请注意，查询结果（responseMetadata）的前一页中只存在书签。
//可以用作书签参数的值。否则，空字符串必须
//作为书签传递。
//“objectType”和属性只应有有效的utf8字符串
//不应包含U+0000（零字节）和U+10ffff（最大和未分配
//代码点）。请参见相关函数splitcompositekey和creatcompositekey。
//完成后，对返回的StateQueryIteratorInterface对象调用Close（）。
//
	GetStateByPartialCompositeKeyWithPagination(objectType string, keys []string,
		pageSize int32, bookmark string) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error)

//CreateCompositeKey将给定的“attributes”组合成一个组合
//关键。对象类型和属性应该只有有效的utf8
//字符串，不应包含u+0000（零字节）和u+10ffff
//（最大和未分配的代码点）。
//生成的复合键可以用作putstate（）中的键。
	CreateCompositeKey(objectType string, attributes []string) (string, error)

//splitcompositekey将指定的键拆分为
//组合键已形成。在范围查询期间找到复合键
//或者部分组合键查询因此可以拆分为
//复合零件。
	SplitCompositeKey(compositeKey string) (string, []string, error)

//GetQueryResult对状态数据库执行“富”查询。它是
//仅支持富查询的状态数据库，
//CouCHDB。查询字符串采用本机语法
//基础状态数据库的。返回迭代器
//它可以用于遍历查询结果集中的所有键。
//但是，如果查询结果集中的键数大于
//totalquerylimit（在core.yaml中定义），不能使用此迭代器
//获取查询结果集中的所有键（结果将受
//totalquerylimit）。
//在验证阶段不重新执行查询，幻象读取为
//未检测到。也就是说，其他承诺的交易可能已经增加，
//更新或删除了影响结果集的键，但这不会
//在验证/提交时检测。易受此影响的应用程序
//因此，不应将GetQueryResult用作更新的事务的一部分
//并且应该限制使用只读链码操作。
	GetQueryResult(query string) (StateQueryIteratorInterface, error)

//GetQueryResultWithPagination对状态数据库执行“富”查询。
//
//例如，CouCHDB。查询字符串采用本机语法
//基础状态数据库的。返回迭代器
//它可以用来迭代查询结果集中的键。
//当空字符串作为值传递给bookmark参数时，返回的
//迭代器可用于获取查询结果的第一个“pageSize”。
//当书签为非空字符串时，迭代器可用于获取
//书签和查询结果中最后一个键之间的第一个“pagesize”键。
//注意，只有书签出现在查询结果的前一页（responseMetadata）
//可以用作书签参数的值。否则，空字符串
//必须作为书签传递。
//此调用仅在只读事务中受支持。
	GetQueryResultWithPagination(query string, pageSize int32,
		bookmark string) (StateQueryIteratorInterface, *pb.QueryResponseMetadata, error)

//GetHistoryForkey返回键值的历史记录。
//对于每个历史键更新，历史值和关联的
//返回事务ID和时间戳。时间戳是
//timestamp provided by the client in the proposal header.
//GetHistoryForkey需要对等配置
//core.ledger.history.enableHistoryDatabase为true。
//在验证阶段不重新执行查询，幻象读取为
//未检测到。也就是说，其他提交的事务可能已更新
//同时影响结果集的键，而这不会
//在验证/提交时检测到。易受此影响的应用程序
//因此，不应将getHistoryForkey用作
//更新分类帐，并应限制使用只读链码操作。
	GetHistoryForKey(key string) (HistoryQueryIteratorInterface, error)

//getprivatedata从指定的
//‘收藏’。注意，getprivatedata不会从
//private writeset，尚未提交到“collection”。在
//换句话说，getprivatedata不考虑putprivatedata修改的数据
//还没有承诺。
	GetPrivateData(collection, key string) ([]byte, error)

//PutPrivateData puts the specified `key` and `value` into the transaction's
//私有写集。注意，只有私有写集的哈希进入
//交易建议响应（发送给发出
//transaction) and the actual private writeset gets temporarily stored in a
//transient store. PutPrivateData doesn't effect the `collection` until the
//事务已验证并成功提交。简单键不能是
//空字符串，不能以空字符（0x00）开头，以便
//避免与复合键的范围查询冲突，复合键在内部
//prefixed with 0x00 as composite key namespace.
	PutPrivateData(collection string, key string, value []byte) error

//DelState records the specified `key` to be deleted in the private writeset of
//交易。注意，只有私有写集的哈希进入
//交易建议响应（发送给发出
//而实际的私有写集临时存储在
//暂时存储。“key”及其值将从集合中删除
//当事务被验证并成功提交时。
	DelPrivateData(collection, key string) error

//setprivatedatavalidationparameter设置密钥级认可策略
//对于“key”指定的私有数据。
	SetPrivateDataValidationParameter(collection, key string, ep []byte) error

//getprivatedatavalidationparameter检索密钥级别认可
//由'key'指定的私有数据的策略。注意，这介绍了
//对事务的readset中“key”的读取依赖项。
	GetPrivateDataValidationParameter(collection, key string) ([]byte, error)

//GetPrivateDataByRange返回一组键上的范围迭代器
//提供私人收藏。迭代器可用于遍历所有键
//在startkey（包含）和endkey（不包含）之间。
//迭代器按词法顺序返回键。注释
//startkey和endkey可以是空字符串，这意味着无边界的范围
//开始或结束时查询。
//完成后，对返回的StateQueryIteratorInterface对象调用Close（）。
//在验证阶段重新执行查询以确保结果集
//自事务认可后未更改（检测到幻象读取）。
	GetPrivateDataByRange(collection, startKey, endKey string) (StateQueryIteratorInterface, error)

//getprivatedatabypartialcompositekey查询给定private中的状态
//基于给定的部分组合键的集合。此函数返回
//一种迭代器，可用于对前缀为
//匹配给定的部分复合键。“objectType”和属性是
//预期只有有效的utf8字符串，不应包含
//U+0000（零字节）和U+10ffff（最大和未分配的代码点）。
//请参见相关函数splitcompositekey和creatcompositekey。
//完成后，对返回的StateQueryIteratorInterface对象调用Close（）。
//在验证阶段重新执行查询以确保结果集
//自事务认可后未更改（检测到幻象读取）。
	GetPrivateDataByPartialCompositeKey(collection, objectType string, keys []string) (StateQueryIteratorInterface, error)

//getprivatedataqueryresult对给定的private执行“rich”查询
//收集。它只支持支持富查询的状态数据库，
//CouCHDB。查询字符串采用本机语法
//基础状态数据库的。返回迭代器
//它可以用于在查询结果集上迭代（下一步）。
//在验证阶段不重新执行查询，幻象读取为
//未检测到。也就是说，其他承诺的交易可能已经增加，
//更新或删除了影响结果集的键，但这不会
//在验证/提交时检测。易受此影响的应用程序
//因此，不应将GetQueryResult用作更新的事务的一部分
//并且应该限制使用只读链码操作。
	GetPrivateDataQueryResult(collection, query string) (StateQueryIteratorInterface, error)

//getcreator返回“signatureheader.creator”（例如标识）
//“signedProposal”的。这是代理（或用户）的标识
//提交交易记录。
	GetCreator() ([]byte, error)

//getTransient返回“chaincodeProposalPayLoad.Transient”字段。
//
//可能用于实现某种形式的应用程序级别
//保密。此字段的内容，如
//“chaincodeProposalPayLoad”应该始终
//从交易中省略，并从分类帐中排除。
	GetTransient() (map[string][]byte, error)

//GetBinding返回用于强制
//
//以上）投标书本身。这有助于避免可能的重播
//攻击。
	GetBinding() ([]byte, error)

//GetDecorations返回有关建议的其他数据（如果适用）
//
//peer, which append or mutate the chaincode input passed to the chaincode.
	GetDecorations() map[string][]byte

//GetSignedProposal返回SignedProposal对象，该对象包含
//
	GetSignedProposal() (*pb.SignedProposal, error)

//gettexTimeStamp返回创建事务时的时间戳。这个
//从Transaction ChannelHeader中获取，因此它将指示
//客户的时间戳和在所有背书人中具有相同的值。
	GetTxTimestamp() (*timestamp.Timestamp, error)

//setEvent允许chaincode在响应中设置一个事件
//
//在提交块中的事务中可用，无论
//交易的有效性。
	SetEvent(name string, payload []byte) error
}

//CommonIteratorInterface允许链代码检查是否还有其他结果
//从迭代器中提取并在完成时关闭它。
type CommonIteratorInterface interface {
//如果范围查询迭代器包含其他键，则hasNext返回true
//和价值观。
	HasNext() bool

//关闭关闭迭代器。完成后应调用此函数
//从迭代器读取以释放资源。
	Close() error
}

//StateQueryIteratorInterface允许链代码在一组
//范围和执行查询返回的键/值对。
type StateQueryIteratorInterface interface {
//继承hasNext（）并关闭（）
	CommonIteratorInterface

//Next返回范围中的下一个键和值，并执行查询迭代器。
	Next() (*queryresult.KV, error)
}

//HistoryQueryIteratorInterface允许链代码在一组
//历史查询返回的键/值对。
type HistoryQueryIteratorInterface interface {
//继承hasNext（）并关闭（）
	CommonIteratorInterface

//Next返回历史查询迭代器中的下一个键和值。
	Next() (*queryresult.KeyModification, error)
}

//MockQueryIteratorInterface允许链代码在一组
//范围查询返回的键/值对。
//TODO:一旦在mockstub中实现了执行查询和历史查询，
//我们需要更新这个接口
type MockQueryIteratorInterface interface {
	StateQueryIteratorInterface
}
