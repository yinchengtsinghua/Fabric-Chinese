
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


package msp

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/signer"
	m "github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

//mspsetupfunctype是安装函数的原型
type mspSetupFuncType func(config *m.FabricMSPConfig) error

//validateIdentityousFunctype是用于验证标识的OU的函数原型
type validateIdentityOUsFuncType func(id *identity) error

//satisfiesPrincipalInternalFuncType是用于检查是否满足主体的函数原型
type satisfiesPrincipalInternalFuncType func(id Identity, principal *m.MSPPrincipal) error

//这是MSP的一个实例，
//对其加密原语使用bccsp。
type bccspmsp struct {
//版本指定此MSP的行为
	version MSPVersion
//以下函数指针用于更改行为
//这取决于它的版本。
//InternalSetupFunc是指向设置函数的指针
	internalSetupFunc mspSetupFuncType

//InternalValidateIdentityousFunc是指向函数的指针，用于验证标识的OU
	internalValidateIdentityOusFunc validateIdentityOUsFuncType

//InternalSatisfiesPrincipaLinInternalUnc是指向函数的指针，用于检查是否满足主体
	internalSatisfiesPrincipalInternalFunc satisfiesPrincipalInternalFuncType

//我们信任的CA证书列表
	rootCerts []Identity

//我们信任的中间证书列表
	intermediateCerts []Identity

//我们信任的CA TLS证书列表
	tlsRootCerts [][]byte

//我们信任的中间TLS证书列表
	tlsIntermediateCerts [][]byte

//其键对应于原材料的证书树内部节点映射
//（顺序表示）被转换成字符串的证书，其值
//是布尔型的。true表示证书是证书树的内部节点。
//错误意味着证书对应于证书树的叶子。
	certificationTreeInternalNodesMap map[string]bool

//签名身份列表
	signer SigningIdentity

//管理标识列表
	admins []Identity

//加密提供者
	bccsp bccsp.BCCSP

//此MSP的提供程序标识符
	name string

//MSP成员的验证选项
	opts *x509.VerifyOptions

//证书吊销列表
	CRL []*pkix.CertificateList

//名单
	ouIdentifiers map[string][][]byte

//CryptoConfig包含
	cryptoConfig *m.FabricCryptoConfig

//结节状构造
	ouEnforcement bool
//这些是客户机、对等机和订购者的OU标识符。
//它们用来区分这些实体
	clientOU, peerOU *OUIdentifier
}

//newbccspmsp返回由bccsp备份的msp实例
//加密提供程序。它可以处理X.509证书和CAN
//生成标识并签名由支持的标识
//证书和密钥对
func newBccspMsp(version MSPVersion) (MSP, error) {
	mspLogger.Debugf("Creating BCCSP-based MSP instance")

	bccsp := factory.GetDefault()
	theMsp := &bccspmsp{}
	theMsp.version = version
	theMsp.bccsp = bccsp
	switch version {
	case MSPv1_0:
		theMsp.internalSetupFunc = theMsp.setupV1
		theMsp.internalValidateIdentityOusFunc = theMsp.validateIdentityOUsV1
		theMsp.internalSatisfiesPrincipalInternalFunc = theMsp.satisfiesPrincipalInternalPreV13
	case MSPv1_1:
		theMsp.internalSetupFunc = theMsp.setupV11
		theMsp.internalValidateIdentityOusFunc = theMsp.validateIdentityOUsV11
		theMsp.internalSatisfiesPrincipalInternalFunc = theMsp.satisfiesPrincipalInternalPreV13
	case MSPv1_3:
		theMsp.internalSetupFunc = theMsp.setupV11
		theMsp.internalValidateIdentityOusFunc = theMsp.validateIdentityOUsV11
		theMsp.internalSatisfiesPrincipalInternalFunc = theMsp.satisfiesPrincipalInternalV13
	default:
		return nil, errors.Errorf("Invalid MSP version [%v]", version)
	}

	return theMsp, nil
}

func (msp *bccspmsp) getCertFromPem(idBytes []byte) (*x509.Certificate, error) {
	if idBytes == nil {
		return nil, errors.New("getCertFromPem error: nil idBytes")
	}

//解码PEM字节
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, errors.Errorf("getCertFromPem error: could not decode pem bytes [%v]", idBytes)
	}

//获得证书
	var cert *x509.Certificate
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "getCertFromPem error: failed to parse x509 cert")
	}

	return cert, nil
}

func (msp *bccspmsp) getIdentityFromConf(idBytes []byte) (Identity, bccsp.Key, error) {
//获得证书
	cert, err := msp.getCertFromPem(idBytes)
	if err != nil {
		return nil, nil, err
	}

//以正确的格式获取公钥
	certPubK, err := msp.bccsp.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: true})

	mspId, err := newIdentity(cert, certPubK, msp)
	if err != nil {
		return nil, nil, err
	}

	return mspId, certPubK, nil
}

func (msp *bccspmsp) getSigningIdentityFromConf(sidInfo *m.SigningIdentityInfo) (SigningIdentity, error) {
	if sidInfo == nil {
		return nil, errors.New("getIdentityFromBytes error: nil sidInfo")
	}

//提取身份的公共部分
	idPub, pubKey, err := msp.getIdentityFromConf(sidInfo.PublicSigner)
	if err != nil {
		return nil, err
	}

//在bccsp密钥库中查找匹配的私钥
	privKey, err := msp.bccsp.GetKey(pubKey.SKI())
//不太安全：如果bccsp找不到密钥，则尝试从keyinfo导入私钥。
	if err != nil {
		mspLogger.Debugf("Could not find SKI [%s], trying KeyMaterial field: %+v\n", hex.EncodeToString(pubKey.SKI()), err)
		if sidInfo.PrivateSigner == nil || sidInfo.PrivateSigner.KeyMaterial == nil {
			return nil, errors.New("KeyMaterial not found in SigningIdentityInfo")
		}

		pemKey, _ := pem.Decode(sidInfo.PrivateSigner.KeyMaterial)
		privKey, err = msp.bccsp.KeyImport(pemKey.Bytes, &bccsp.ECDSAPrivateKeyImportOpts{Temporary: true})
		if err != nil {
			return nil, errors.WithMessage(err, "getIdentityFromBytes error: Failed to import EC private key")
		}
	}

//获取对等签名者
	peerSigner, err := signer.New(msp.bccsp, privKey)
	if err != nil {
		return nil, errors.WithMessage(err, "getIdentityFromBytes error: Failed initializing bccspCryptoSigner")
	}

	return newSigningIdentity(idPub.(*identity).cert, idPub.(*identity).pk, peerSigner, msp)
}

//安装程序设置内部数据结构
//对于这个MSP，给定一个MSPConfig引用；它
//如果成功或出错，则返回nil
func (msp *bccspmsp) Setup(conf1 *m.MSPConfig) error {
	if conf1 == nil {
		return errors.New("Setup error: nil conf reference")
	}

//如果它是fabric类型的msp，则提取mspconfig实例
	conf := &m.FabricMSPConfig{}
	err := proto.Unmarshal(conf1.Config, conf)
	if err != nil {
		return errors.Wrap(err, "failed unmarshalling fabric msp config")
	}

//设置此MSP的名称
	msp.name = conf.Name
	mspLogger.Debugf("Setting up MSP instance %s", msp.name)

//设置
	return msp.internalSetupFunc(conf)
}

//GetVersion返回此MSP的版本
func (msp *bccspmsp) GetVersion() MSPVersion {
	return msp.version
}

//GetType返回此MSP的类型
func (msp *bccspmsp) GetType() ProviderType {
	return FABRIC
}

//GetIdentifier返回此实例的MSP标识符
func (msp *bccspmsp) GetIdentifier() (string, error) {
	return msp.name, nil
}

//GettlsRootCerts返回此MSP的根证书
func (msp *bccspmsp) GetTLSRootCerts() [][]byte {
	return msp.tlsRootCerts
}

//GettlIntermediateCenters返回此MSP的中间根证书
func (msp *bccspmsp) GetTLSIntermediateCerts() [][]byte {
	return msp.tlsIntermediateCerts
}

//GetDefaultSigningIdentity返回
//此MSP的默认签名标识（如果有）
func (msp *bccspmsp) GetDefaultSigningIdentity() (SigningIdentity, error) {
	mspLogger.Debugf("Obtaining default signing identity")

	if msp.signer == nil {
		return nil, errors.New("this MSP does not possess a valid default signing identity")
	}

	return msp.signer, nil
}

//GetSigningIdentity返回特定签名
//由提供的标识符标识的标识
func (msp *bccspmsp) GetSigningIdentity(identifier *IdentityIdentifier) (SigningIdentity, error) {
//托多
	return nil, errors.Errorf("no signing identity for %#v", identifier)
}

//验证确定是否
//提供的标识根据
//回到这个MSP的信任之根；它返回
//如果身份有效或
//否则出错
func (msp *bccspmsp) Validate(id Identity) error {
	mspLogger.Debugf("MSP %s validating identity", msp.name)

	switch id := id.(type) {
//如果此标识属于此特定类型，
//这就是我如何在
//此MSP的信任根目录
	case *identity:
		return msp.validateIdentity(id)
	default:
		return errors.New("identity type not recognized")
	}
}

//hasorole检查标识是否属于组织单位
//与指定的msprole关联。
//此函数不检查证明者标识符。
//之前需要执行适当的验证。
func (msp *bccspmsp) hasOURole(id Identity, mspRole m.MSPRole_MSPRoleType) error {
//检查可疑
	if !msp.ouEnforcement {
		return errors.New("NodeOUs not activated. Cannot tell apart identities.")
	}

	mspLogger.Debugf("MSP %s checking if the identity is a client", msp.name)

	switch id := id.(type) {
//如果此标识属于此特定类型，
//这就是我如何在
//此MSP的信任根目录
	case *identity:
		return msp.hasOURoleInternal(id, mspRole)
	default:
		return errors.New("Identity type not recognized")
	}
}

func (msp *bccspmsp) hasOURoleInternal(id *identity, mspRole m.MSPRole_MSPRoleType) error {
	var nodeOUValue string
	switch mspRole {
	case m.MSPRole_CLIENT:
		nodeOUValue = msp.clientOU.OrganizationalUnitIdentifier
	case m.MSPRole_PEER:
		nodeOUValue = msp.peerOU.OrganizationalUnitIdentifier
	default:
		return fmt.Errorf("Invalid MSPRoleType. It must be CLIENT, PEER or ORDERER")
	}

	for _, OU := range id.GetOrganizationalUnits() {
		if OU.OrganizationalUnitIdentifier == nodeOUValue {
			return nil
		}
	}

	return fmt.Errorf("The identity does not contain OU [%s], MSP: [%s]", mspRole, msp.name)
}

//DeserializeIDentity返回给定字节级别的标识
//序列化实体结构的表示形式
func (msp *bccspmsp) DeserializeIdentity(serializedID []byte) (Identity, error) {
	mspLogger.Debug("Obtaining identity")

//我们首先反序列化到一个序列化的实体以获取MSP ID
	sId := &m.SerializedIdentity{}
	err := proto.Unmarshal(serializedID, sId)
	if err != nil {
		return nil, errors.Wrap(err, "could not deserialize a SerializedIdentity")
	}

	if sId.Mspid != msp.name {
		return nil, errors.Errorf("expected MSP ID %s, received %s", msp.name, sId.Mspid)
	}

	return msp.deserializeIdentityInternal(sId.IdBytes)
}

//DeserializeIdentityInternal返回给定字节级表示的标识
func (msp *bccspmsp) deserializeIdentityInternal(serializedIdentity []byte) (Identity, error) {
//此MSP将始终以这种方式反序列化证书
	bl, _ := pem.Decode(serializedIdentity)
	if bl == nil {
		return nil, errors.New("could not decode the PEM structure")
	}
	cert, err := x509.ParseCertificate(bl.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "parseCertificate failed")
	}

//现在我们有了证书；请确保它的字段
//（例如，issuer.ou或subject.ou）与
//此MSP具有的MSP ID；否则可能是攻击
//待办事项！
//我们还不能做，因为没有标准化的方法
//（然而）将MSP ID编码到证书的X.509主体中

	pub, err := msp.bccsp.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: true})
	if err != nil {
		return nil, errors.WithMessage(err, "failed to import certificate's public key")
	}

	return newIdentity(cert, pub, msp)
}

//如果标识与主体匹配，则satisfiesprincipal返回空值，否则返回错误。
func (msp *bccspmsp) SatisfiesPrincipal(id Identity, principal *m.MSPPrincipal) error {
	principals, err := collectPrincipals(principal, msp.GetVersion())
	if err != nil {
		return err
	}
	for _, principal := range principals {
		err = msp.internalSatisfiesPrincipalInternalFunc(id, principal)
		if err != nil {
			return err
		}
	}
	return nil
}

//CollectPrincipals将来自合并主体的主体收集到单个mspprincipal切片中。
func collectPrincipals(principal *m.MSPPrincipal, mspVersion MSPVersion) ([]*m.MSPPrincipal, error) {
	switch principal.PrincipalClassification {
	case m.MSPPrincipal_COMBINED:
//MSP v1.0或v1.1不支持组合主体
		if mspVersion <= MSPv1_1 {
			return nil, errors.Errorf("invalid principal type %d", int32(principal.PrincipalClassification))
		}
//主体是多个主体的组合。
		principals := &m.CombinedPrincipal{}
		err := proto.Unmarshal(principal.Principal, principals)
		if err != nil {
			return nil, errors.Wrap(err, "could not unmarshal CombinedPrincipal from principal")
		}
//如果合并主体中没有主体，则返回错误。
		if len(principals.Principals) == 0 {
			return nil, errors.New("No principals in CombinedPrincipal")
		}
//对所有组合的主体递归调用msp.collectprincipals。
//组合主体的嵌套级别没有限制。
		var principalsSlice []*m.MSPPrincipal
		for _, cp := range principals.Principals {
			internalSlice, err := collectPrincipals(cp, mspVersion)
			if err != nil {
				return nil, err
			}
			principalsSlice = append(principalsSlice, internalSlice...)
		}
//所有合并的主体都被收集到PrincipalsSlice中
		return principalsSlice, nil
	default:
		return []*m.MSPPrincipal{principal}, nil
	}
}

//satisfiesPrincipaLinternalPrev13将标识和主体作为参数。
//如果出现错误，函数将返回一个错误。
//该函数实现MSP在v1.1之前（包括v1.1）的行为。
func (msp *bccspmsp) satisfiesPrincipalInternalPreV13(id Identity, principal *m.MSPPrincipal) error {
	switch principal.PrincipalClassification {
//在这种情况下，我们必须检查
//标识在MSP中具有角色-成员或管理员
	case m.MSPPrincipal_ROLE:
//主体包含MSP角色
		mspRole := &m.MSPRole{}
		err := proto.Unmarshal(principal.Principal, mspRole)
		if err != nil {
			return errors.Wrap(err, "could not unmarshal MSPRole from principal")
		}

//首先，我们检查MSP
//标识符与标识相同
		if mspRole.MspIdentifier != msp.name {
			return errors.Errorf("the identity is a member of a different MSP (expected %s, got %s)", mspRole.MspIdentifier, id.GetMSPIdentifier())
		}

//现在我们验证不同的MSP角色
		switch mspRole.Role {
		case m.MSPRole_MEMBER:
//对于会员，我们只需检查
//此标识是否对MSP有效
			mspLogger.Debugf("Checking if identity satisfies MEMBER role for %s", msp.name)
			return msp.Validate(id)
		case m.MSPRole_ADMIN:
			mspLogger.Debugf("Checking if identity satisfies ADMIN role for %s", msp.name)
//对于管理员，我们检查
//ID正是我们的管理员之一
			for _, admincert := range msp.admins {
				if bytes.Equal(id.(*identity).cert.Raw, admincert.(*identity).cert.Raw) {
//我们不需要检查管理员是否是有效身份
//根据这个MSP，因为我们已经在设置时检查过了
//如果有火柴，我们就可以回去了
					return nil
				}
			}
			return errors.New("This identity is not an admin")
		case m.MSPRole_CLIENT:
			fallthrough
		case m.MSPRole_PEER:
			mspLogger.Debugf("Checking if identity satisfies role [%s] for %s", m.MSPRole_MSPRoleType_name[int32(mspRole.Role)], msp.name)
			if err := msp.Validate(id); err != nil {
				return errors.Wrapf(err, "The identity is not valid under this MSP [%s]", msp.name)
			}

			if err := msp.hasOURole(id, mspRole.Role); err != nil {
				return errors.Wrapf(err, "The identity is not a [%s] under this MSP [%s]", m.MSPRole_MSPRoleType_name[int32(mspRole.Role)], msp.name)
			}
			return nil
		default:
			return errors.Errorf("invalid MSP role type %d", int32(mspRole.Role))
		}
	case m.MSPPrincipal_IDENTITY:
//在这种情况下，我们必须反序列化主体的标识
//并与我们的证书逐字节比较
		principalId, err := msp.DeserializeIdentity(principal.Principal)
		if err != nil {
			return errors.WithMessage(err, "invalid identity principal, not a certificate")
		}

		if bytes.Equal(id.(*identity).cert.Raw, principalId.(*identity).cert.Raw) {
			return principalId.Validate()
		}

		return errors.New("The identities do not match")
	case m.MSPPrincipal_ORGANIZATION_UNIT:
//负责人包含组织单位
		OU := &m.OrganizationUnit{}
		err := proto.Unmarshal(principal.Principal, OU)
		if err != nil {
			return errors.Wrap(err, "could not unmarshal OrganizationUnit from principal")
		}

//首先，我们检查MSP
//标识符与标识相同
		if OU.MspIdentifier != msp.name {
			return errors.Errorf("the identity is a member of a different MSP (expected %s, got %s)", OU.MspIdentifier, id.GetMSPIdentifier())
		}

//然后我们检查这个MSP的身份是否有效
//如果不是的话就失败了
		err = msp.Validate(id)
		if err != nil {
			return err
		}

//现在我们检查这个身份是否与请求的身份匹配
		for _, ou := range id.GetOrganizationalUnits() {
			if ou.OrganizationalUnitIdentifier == OU.OrganizationalUnitIdentifier &&
				bytes.Equal(ou.CertifiersIdentifier, OU.CertifiersIdentifier) {
				return nil
			}
		}

//如果我们在这里，找不到匹配项，返回一个错误
		return errors.New("The identities do not match")
	default:
		return errors.Errorf("invalid principal type %d", int32(principal.PrincipalClassification))
	}
}

//satisfiesprincipalinternalv13将身份和主体作为参数。
//如果出现错误，函数将返回一个错误。
//该函数实现了从v1.3开始的MSP预期的其他行为。
//对于1.3版之前的功能，函数调用satisfiesprincipalinternalprev13。
func (msp *bccspmsp) satisfiesPrincipalInternalV13(id Identity, principal *m.MSPPrincipal) error {
	switch principal.PrincipalClassification {
	case m.MSPPrincipal_COMBINED:
		return errors.New("SatisfiesPrincipalInternal shall not be called with a CombinedPrincipal")
	case m.MSPPrincipal_ANONYMITY:
		anon := &m.MSPIdentityAnonymity{}
		err := proto.Unmarshal(principal.Principal, anon)
		if err != nil {
			return errors.Wrap(err, "could not unmarshal MSPIdentityAnonymity from principal")
		}
		switch anon.AnonymityType {
		case m.MSPIdentityAnonymity_ANONYMOUS:
			return errors.New("Principal is anonymous, but X.509 MSP does not support anonymous identities")
		case m.MSPIdentityAnonymity_NOMINAL:
			return nil
		default:
			return errors.Errorf("Unknown principal anonymity type: %d", anon.AnonymityType)
		}

	default:
//使用pre-v1.3函数检查其他主体类型
		return msp.satisfiesPrincipalInternalPreV13(id, principal)
	}
}

//getCertificationChain返回此MSP中传递的标识的证书链
func (msp *bccspmsp) getCertificationChain(id Identity) ([]*x509.Certificate, error) {
	mspLogger.Debugf("MSP %s getting certification chain", msp.name)

	switch id := id.(type) {
//如果此标识属于此特定类型，
//这就是我如何在
//此MSP的信任根目录
	case *identity:
		return msp.getCertificationChainForBCCSPIdentity(id)
	default:
		return nil, errors.New("identity type not recognized")
	}
}

//getCertificationChainForBCSPIdentity返回此MSP中传递的BCCSP标识的证书链
func (msp *bccspmsp) getCertificationChainForBCCSPIdentity(id *identity) ([]*x509.Certificate, error) {
	if id == nil {
		return nil, errors.New("Invalid bccsp identity. Must be different from nil.")
	}

//我们希望有一个有效的VerifyOptions实例
	if msp.opts == nil {
		return nil, errors.New("Invalid msp instance")
	}

//CA不能直接用作标识。
	if id.cert.IsCA {
		return nil, errors.New("An X509 certificate with Basic Constraint: " +
			"Certificate Authority equals true cannot be used as an identity")
	}

	return msp.getValidationChain(id.cert, false)
}

func (msp *bccspmsp) getUniqueValidationChain(cert *x509.Certificate, opts x509.VerifyOptions) ([]*x509.Certificate, error) {
//请Golang根据设置时生成的选项为我们验证证书
	if msp.opts == nil {
		return nil, errors.New("the supplied identity has no verify options")
	}
	validationChains, err := cert.Verify(opts)
	if err != nil {
		return nil, errors.WithMessage(err, "the supplied identity is not valid")
	}

//我们只支持单一的验证链；
//如果不止一个，那么可能
//不知道谁拥有身份
	if len(validationChains) != 1 {
		return nil, errors.Errorf("this MSP only supports a single validation chain, got %d", len(validationChains))
	}

	return validationChains[0], nil
}

func (msp *bccspmsp) getValidationChain(cert *x509.Certificate, isIntermediateChain bool) ([]*x509.Certificate, error) {
	validationChain, err := msp.getUniqueValidationChain(cert, msp.getValidityOptsForCert(cert))
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting validation chain")
	}

//我们希望链条长度至少为2
	if len(validationChain) < 2 {
		return nil, errors.Errorf("expected a chain of length at least 2, got %d", len(validationChain))
	}

//检查父级是否为证书树的叶级
//如果验证中间链，第一个证书将
	parentPosition := 1
	if isIntermediateChain {
		parentPosition = 0
	}
	if msp.certificationTreeInternalNodesMap[string(validationChain[parentPosition].Raw)] {
		return nil, errors.Errorf("invalid validation chain. Parent certificate should be a leaf of the certification tree [%v]", cert.Raw)
	}
	return validationChain, nil
}

//getCertificationChainIdentifier返回此MSP中传递的标识的证书链标识符。
//标识符被计算为链中证书串联的sha256。
func (msp *bccspmsp) getCertificationChainIdentifier(id Identity) ([]byte, error) {
	chain, err := msp.getCertificationChain(id)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed getting certification chain for [%v]", id))
	}

//链[0]是表示标识的证书。
//它将被丢弃
	return msp.getCertificationChainIdentifierFromChain(chain[1:])
}

func (msp *bccspmsp) getCertificationChainIdentifierFromChain(chain []*x509.Certificate) ([]byte, error) {
//散列链
//将标识证书的哈希用作标识标识符中的ID
	hashOpt, err := bccsp.GetHashOpt(msp.cryptoConfig.IdentityIdentifierHashFunction)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting hash function options")
	}

	hf, err := msp.bccsp.GetHash(hashOpt)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting hash function when computing certification chain identifier")
	}
	for i := 0; i < len(chain); i++ {
		hf.Write(chain[i].Raw)
	}
	return hf.Sum(nil), nil
}

//SaniticeStart确保使用ECDSA签名的X509证书
//在低S中有签名。如果不是这样，则证书
//重新生成以具有低S签名。
func (msp *bccspmsp) sanitizeCert(cert *x509.Certificate) (*x509.Certificate, error) {
	if isECDSASignedCert(cert) {
//查找父证书以执行清理
		var parentCert *x509.Certificate
		chain, err := msp.getUniqueValidationChain(cert, msp.getValidityOptsForCert(cert))
		if err != nil {
			return nil, err
		}

//此时，cert可能是根CA证书
//或中级CA证书
		if cert.IsCA && len(chain) == 1 {
//证书是根CA证书
			parentCert = cert
		} else {
			parentCert = chain[1]
		}

//消毒
		cert, err = sanitizeECDSASignedCert(cert, parentCert)
		if err != nil {
			return nil, err
		}
	}
	return cert, nil
}

//iswell格式检查给定的标识是否可以反序列化为其提供程序特定的形式。
//在这个MSP实现中，格式良好意味着PEM的类型是
//字符串“certificate”或类型完全丢失。
func (msp *bccspmsp) IsWellFormed(identity *m.SerializedIdentity) error {
	bl, _ := pem.Decode(identity.IdBytes)
	if bl == nil {
		return errors.New("PEM decoding resulted in an empty block")
	}
//重要提示：此方法看起来非常类似于getcertfrompem（idbytes[]byte）（*x509.certificate，error）
//但是我们：
//1）必须确保PEM块为类型证书或为空。
//2）不得用此方法替换getcertfrompem，否则我们将介绍
//验证逻辑中的一种变化，它将导致一个链叉。
	if bl.Type != "CERTIFICATE" && bl.Type != "" {
		return errors.Errorf("pem type is %s, should be 'CERTIFICATE' or missing", bl.Type)
	}
	_, err := x509.ParseCertificate(bl.Bytes)
	return err
}
