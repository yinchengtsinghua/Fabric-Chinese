
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


package pkcs11

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"

	"github.com/miekg/pkcs11"
	"go.uber.org/zap/zapcore"
)

func loadLib(lib, pin, label string) (*pkcs11.Ctx, uint, *pkcs11.SessionHandle, error) {
	var slot uint
	logger.Debugf("Loading pkcs11 library [%s]\n", lib)
	if lib == "" {
		return nil, slot, nil, fmt.Errorf("No PKCS11 library default")
	}

	ctx := pkcs11.New(lib)
	if ctx == nil {
		return nil, slot, nil, fmt.Errorf("Instantiate failed [%s]", lib)
	}

	ctx.Initialize()
	slots, err := ctx.GetSlotList(true)
	if err != nil {
		return nil, slot, nil, fmt.Errorf("Could not get Slot List [%s]", err)
	}
	found := false
	for _, s := range slots {
		info, errToken := ctx.GetTokenInfo(s)
		if errToken != nil {
			continue
		}
		logger.Debugf("Looking for %s, found label %s\n", label, info.Label)
		if label == info.Label {
			found = true
			slot = s
			break
		}
	}
	if !found {
		return nil, slot, nil, fmt.Errorf("Could not find token with label %s", label)
	}

	var session pkcs11.SessionHandle
	for i := 0; i < 10; i++ {
		session, err = ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
		if err != nil {
			logger.Warningf("OpenSession failed, retrying [%s]\n", err)
		} else {
			break
		}
	}
	if err != nil {
		logger.Fatalf("OpenSession [%s]\n", err)
	}
	logger.Debugf("Created new pkcs11 session %+v on slot %d\n", session, slot)

	if pin == "" {
		return nil, slot, nil, fmt.Errorf("No PIN set")
	}
	err = ctx.Login(session, pkcs11.CKU_USER, pin)
	if err != nil {
		if err != pkcs11.Error(pkcs11.CKR_USER_ALREADY_LOGGED_IN) {
			return nil, slot, nil, fmt.Errorf("Login failed [%s]", err)
		}
	}

	return ctx, slot, &session, nil
}

func (csp *impl) getSession() (session pkcs11.SessionHandle) {
	select {
	case session = <-csp.sessions:
		logger.Debugf("Reusing existing pkcs11 session %+v on slot %d\n", session, csp.slot)

	default:
//缓存为空（或完全在使用中），请创建新会话
		var s pkcs11.SessionHandle
		var err error
		for i := 0; i < 10; i++ {
			s, err = csp.ctx.OpenSession(csp.slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
			if err != nil {
				logger.Warningf("OpenSession failed, retrying [%s]\n", err)
			} else {
				break
			}
		}
		if err != nil {
			panic(fmt.Errorf("OpenSession failed [%s]", err))
		}
		logger.Debugf("Created new pkcs11 session %+v on slot %d\n", s, csp.slot)
		session = s
	}
	return session
}

func (csp *impl) returnSession(session pkcs11.SessionHandle) {
	select {
	case csp.sessions <- session:
//将会话返回到会话缓存
	default:
//缓存中有大量会话，正在删除
		csp.ctx.CloseSession(session)
	}
}

//通过ski查找存储在cka-id中的EC密钥
//此功能可能适用于EC和RSA密钥。
func (csp *impl) getECKey(ski []byte) (pubKey *ecdsa.PublicKey, isPriv bool, err error) {
	p11lib := csp.ctx
	session := csp.getSession()
	defer csp.returnSession(session)
	isPriv = true
	_, err = findKeyPairFromSKI(p11lib, session, ski, privateKeyFlag)
	if err != nil {
		isPriv = false
		logger.Debugf("Private key not found [%s] for SKI [%s], looking for Public key", err, hex.EncodeToString(ski))
	}

	publicKey, err := findKeyPairFromSKI(p11lib, session, ski, publicKeyFlag)
	if err != nil {
		return nil, false, fmt.Errorf("Public key not found [%s] for SKI [%s]", err, hex.EncodeToString(ski))
	}

	ecpt, marshaledOid, err := ecPoint(p11lib, session, *publicKey)
	if err != nil {
		return nil, false, fmt.Errorf("Public key not found [%s] for SKI [%s]", err, hex.EncodeToString(ski))
	}

	curveOid := new(asn1.ObjectIdentifier)
	_, err = asn1.Unmarshal(marshaledOid, curveOid)
	if err != nil {
		return nil, false, fmt.Errorf("Failed Unmarshaling Curve OID [%s]\n%s", err.Error(), hex.EncodeToString(marshaledOid))
	}

	curve := namedCurveFromOID(*curveOid)
	if curve == nil {
		return nil, false, fmt.Errorf("Cound not recognize Curve from OID")
	}
	x, y := elliptic.Unmarshal(curve, ecpt)
	if x == nil {
		return nil, false, fmt.Errorf("Failed Unmarshaling Public Key")
	}

	pubKey = &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
	return pubKey, isPriv, nil
}

//RFC 5480，2.1.1.1.1。命名曲线
//
//secp24r1对象标识符：：=
//ISO（1）认证组织（3）认证（132）曲线（0）33
//
//secp256r1对象标识符：：=
//ISO（1）构件主体（2）US（840）ANSI-X9-62（10045）曲线（3）
//素数（1）7 }
//
//secp384r1对象标识符：：=
//ISO（1）认证组织（3）认证（132）曲线（0）34
//
//secp521r1对象标识符：：=
//ISO（1）认证组织（3）认证（132）曲线（0）35
//
var (
	oidNamedCurveP224 = asn1.ObjectIdentifier{1, 3, 132, 0, 33}
	oidNamedCurveP256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}
	oidNamedCurveP384 = asn1.ObjectIdentifier{1, 3, 132, 0, 34}
	oidNamedCurveP521 = asn1.ObjectIdentifier{1, 3, 132, 0, 35}
)

func namedCurveFromOID(oid asn1.ObjectIdentifier) elliptic.Curve {
	switch {
	case oid.Equal(oidNamedCurveP224):
		return elliptic.P224()
	case oid.Equal(oidNamedCurveP256):
		return elliptic.P256()
	case oid.Equal(oidNamedCurveP384):
		return elliptic.P384()
	case oid.Equal(oidNamedCurveP521):
		return elliptic.P521()
	}
	return nil
}

func oidFromNamedCurve(curve elliptic.Curve) (asn1.ObjectIdentifier, bool) {
	switch curve {
	case elliptic.P224():
		return oidNamedCurveP224, true
	case elliptic.P256():
		return oidNamedCurveP256, true
	case elliptic.P384():
		return oidNamedCurveP384, true
	case elliptic.P521():
		return oidNamedCurveP521, true
	}

	return nil, false
}

func (csp *impl) generateECKey(curve asn1.ObjectIdentifier, ephemeral bool) (ski []byte, pubKey *ecdsa.PublicKey, err error) {
	p11lib := csp.ctx
	session := csp.getSession()
	defer csp.returnSession(session)

	id := nextIDCtr()
	publabel := fmt.Sprintf("BCPUB%s", id.Text(16))
	prvlabel := fmt.Sprintf("BCPRV%s", id.Text(16))

	marshaledOID, err := asn1.Marshal(curve)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not marshal OID [%s]", err.Error())
	}

	pubkeyT := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_EC),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, !ephemeral),
		pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),
		pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, marshaledOID),
		pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, false),

		pkcs11.NewAttribute(pkcs11.CKA_ID, publabel),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, publabel),
	}

	prvkeyT := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_EC),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, !ephemeral),
		pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
		pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),

		pkcs11.NewAttribute(pkcs11.CKA_ID, prvlabel),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, prvlabel),

		pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, false),
		pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
	}

	pub, prv, err := p11lib.GenerateKeyPair(session,
		[]*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_EC_KEY_PAIR_GEN, nil)},
		pubkeyT, prvkeyT)

	if err != nil {
		return nil, nil, fmt.Errorf("P11: keypair generate failed [%s]", err)
	}

	ecpt, _, _ := ecPoint(p11lib, session, pub)
	hash := sha256.Sum256(ecpt)
	ski = hash[:]

//将滑雪板两个键的cka_id（公钥）和cka_标签设置为滑雪板的十六进制字符串
	setskiT := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_ID, ski),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, hex.EncodeToString(ski)),
	}

	logger.Infof("Generated new P11 key, SKI %x\n", ski)
	err = p11lib.SetAttributeValue(session, pub, setskiT)
	if err != nil {
		return nil, nil, fmt.Errorf("P11: set-ID-to-SKI[public] failed [%s]", err)
	}

	err = p11lib.SetAttributeValue(session, prv, setskiT)
	if err != nil {
		return nil, nil, fmt.Errorf("P11: set-ID-to-SKI[private] failed [%s]", err)
	}

//对于公钥和私钥，将cka-modifible设置为false
	if csp.immutable {
		setCKAModifiable := []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_MODIFIABLE, false),
		}

		_, pubCopyerror := p11lib.CopyObject(session, pub, setCKAModifiable)
		if pubCopyerror != nil {
			return nil, nil, fmt.Errorf("P11: Public Key copy failed with error [%s] . Please contact your HSM vendor", pubCopyerror)
		}

		pubKeyDestroyError := p11lib.DestroyObject(session, pub)
		if pubKeyDestroyError != nil {
			return nil, nil, fmt.Errorf("P11: Public Key destroy failed with error [%s]. Please contact your HSM vendor", pubCopyerror)
		}

		_, prvCopyerror := p11lib.CopyObject(session, prv, setCKAModifiable)
		if prvCopyerror != nil {
			return nil, nil, fmt.Errorf("P11: Private Key copy failed with error [%s]. Please contact your HSM vendor", prvCopyerror)
		}
		prvKeyDestroyError := p11lib.DestroyObject(session, prv)
		if pubKeyDestroyError != nil {
			return nil, nil, fmt.Errorf("P11: Private Key destroy failed with error [%s]. Please contact your HSM vendor", prvKeyDestroyError)
		}
	}

	nistCurve := namedCurveFromOID(curve)
	if curve == nil {
		return nil, nil, fmt.Errorf("Cound not recognize Curve from OID")
	}
	x, y := elliptic.Unmarshal(nistCurve, ecpt)
	if x == nil {
		return nil, nil, fmt.Errorf("Failed Unmarshaling Public Key")
	}

	pubGoKey := &ecdsa.PublicKey{Curve: nistCurve, X: x, Y: y}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		listAttrs(p11lib, session, prv)
		listAttrs(p11lib, session, pub)
	}

	return ski, pubGoKey, nil
}

func (csp *impl) signP11ECDSA(ski []byte, msg []byte) (R, S *big.Int, err error) {
	p11lib := csp.ctx
	session := csp.getSession()
	defer csp.returnSession(session)

	privateKey, err := findKeyPairFromSKI(p11lib, session, ski, privateKeyFlag)
	if err != nil {
		return nil, nil, fmt.Errorf("Private key not found [%s]", err)
	}

	err = p11lib.SignInit(session, []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil)}, *privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("Sign-initialize  failed [%s]", err)
	}

	var sig []byte

	sig, err = p11lib.Sign(session, msg)
	if err != nil {
		return nil, nil, fmt.Errorf("P11: sign failed [%s]", err)
	}

	R = new(big.Int)
	S = new(big.Int)
	R.SetBytes(sig[0 : len(sig)/2])
	S.SetBytes(sig[len(sig)/2:])

	return R, S, nil
}

func (csp *impl) verifyP11ECDSA(ski []byte, msg []byte, R, S *big.Int, byteSize int) (bool, error) {
	p11lib := csp.ctx
	session := csp.getSession()
	defer csp.returnSession(session)

	logger.Debugf("Verify ECDSA\n")

	publicKey, err := findKeyPairFromSKI(p11lib, session, ski, publicKeyFlag)
	if err != nil {
		return false, fmt.Errorf("Public key not found [%s]", err)
	}

	r := R.Bytes()
	s := S.Bytes()

//如果需要，用零填充R和S的前面
	sig := make([]byte, 2*byteSize)
	copy(sig[byteSize-len(r):byteSize], r)
	copy(sig[2*byteSize-len(s):], s)

	err = p11lib.VerifyInit(session, []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil)},
		*publicKey)
	if err != nil {
		return false, fmt.Errorf("PKCS11: Verify-initialize [%s]", err)
	}
	err = p11lib.Verify(session, msg, sig)
	if err == pkcs11.Error(pkcs11.CKR_SIGNATURE_INVALID) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("PKCS11: Verify failed [%s]", err)
	}

	return true, nil
}

const (
	privateKeyFlag = true
	publicKeyFlag  = false
)

func findKeyPairFromSKI(mod *pkcs11.Ctx, session pkcs11.SessionHandle, ski []byte, keyType bool) (*pkcs11.ObjectHandle, error) {
	ktype := pkcs11.CKO_PUBLIC_KEY
	if keyType == privateKeyFlag {
		ktype = pkcs11.CKO_PRIVATE_KEY
	}

	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, ktype),
		pkcs11.NewAttribute(pkcs11.CKA_ID, ski),
	}
	if err := mod.FindObjectsInit(session, template); err != nil {
		return nil, err
	}

//单会话实例，假设只命中一次
	objs, _, err := mod.FindObjects(session, 1)
	if err != nil {
		return nil, err
	}
	if err = mod.FindObjectsFinal(session); err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		return nil, fmt.Errorf("Key not found [%s]", hex.Dump(ski))
	}

	return &objs[0], nil
}

//相当简单的EC点查询，而不是OpenCryptoki
//错误报告长度，包括以下字段的04标记
//EP11中的SPKI返回了MACED公钥：
//
//ATTR 385/x181型，长度66 B——应为1+64
//电子商务点：
//00000000 04 CE 30 31 6D 5a fd d3 53 2d 54 9a 27 54 d8 7c
//000000 10 D9 80 35 91 09 2d 6f 06 5a 8e e3 cb c0 01 b7 c9
//0000000 20 13 5D 70 D4 E5 62 F2 1B 10 93 F7 D5 77 41 BA 9D
//00000030933E 18 3E 00 C6 0A 0E D2 36 CC 7F BE 50 16 EF
//000000 40 06 06
//
//cf.正确字段：
//0 89：序列
//2 19：顺序
//4 7：对象标识符ECPublicKey（1 2 840 10045 2 1）
//13 8：对象标识符prime256v1（1 2 840 10045 3 1 7）
//：}
//23 66：位串
//：04 CE 30 31 6D 5a fd d3 53 2d 54 9a 27 54 d8 7c
//：D9 80 35 91 09 2d 6f 06 5a 8e e3 cb c0 01 b7 c9
//：13 5D 70 D4 E5 62 F2 1B 10 93 F7 D5 77 41 BA 9D
//：93 3E 18 3E 00 C6 0A 0E D2 36 CC 7F BE 50 16 EF
//：06
//：}
//
//作为短期解决方案，如果出现以下情况，请删除尾随字节：
//-接收偶数字节==2*主坐标+2字节
//-起始字节为04：未压缩EC点
//-尾随字节为04：假定它属于下一个八位字节字符串
//
//[2016年10月22日v3.5.1遇到错误分析]
//
//Softhsm在未压缩点之前报告额外的两个字节
//0x04<length*2+1>
//Vv<实际起点
//00000000 04 41 04 6C C8 57 32 13 02 12 6A 19 23 1D 5A 64.A.L.W2…J..ZD
//000000 10 33 0C eb 75 4d e8 99 22 92 35 96 b2 39 58 14 1e 3..um..5..9x..
//0000000 20 19 de ef 32 46 50 68 02 24 62 36 db ed b1 84 7b…2ph.$B6…..
//0000003 93 D8 40 C3 D5 A6 B7 38 16 D2 35 0A 53 11 F9 51….@…..8..5.S..Q
//000000 40 FC A7 16…
func ecPoint(p11lib *pkcs11.Ctx, session pkcs11.SessionHandle, key pkcs11.ObjectHandle) (ecpt, oid []byte, err error) {
	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_EC_POINT, nil),
		pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, nil),
	}

	attr, err := p11lib.GetAttributeValue(session, key, template)
	if err != nil {
		return nil, nil, fmt.Errorf("PKCS11: get(EC point) [%s]", err)
	}

	for _, a := range attr {
		if a.Type == pkcs11.CKA_EC_POINT {
			logger.Debugf("EC point: attr type %d/0x%x, len %d\n%s\n", a.Type, a.Type, len(a.Value), hex.Dump(a.Value))

//解决方法，见上文
			if (0 == (len(a.Value) % 2)) &&
				(byte(0x04) == a.Value[0]) &&
				(byte(0x04) == a.Value[len(a.Value)-1]) {
				logger.Debugf("Detected opencryptoki bug, trimming trailing 0x04")
ecpt = a.Value[0 : len(a.Value)-1] //修剪尾随0x04
			} else if byte(0x04) == a.Value[0] && byte(0x04) == a.Value[2] {
				logger.Debugf("Detected SoftHSM bug, trimming leading 0x04 0xXX")
				ecpt = a.Value[2:len(a.Value)]
			} else {
				ecpt = a.Value
			}
		} else if a.Type == pkcs11.CKA_EC_PARAMS {
			logger.Debugf("EC point: attr type %d/0x%x, len %d\n%s\n", a.Type, a.Type, len(a.Value), hex.Dump(a.Value))

			oid = a.Value
		}
	}
	if oid == nil || ecpt == nil {
		return nil, nil, fmt.Errorf("CKA_EC_POINT not found, perhaps not an EC Key?")
	}

	return ecpt, oid, nil
}

func listAttrs(p11lib *pkcs11.Ctx, session pkcs11.SessionHandle, obj pkcs11.ObjectHandle) {
	var cktype, ckclass uint
	var ckaid, cklabel []byte

	if p11lib == nil {
		return
	}

	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, ckclass),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, cktype),
		pkcs11.NewAttribute(pkcs11.CKA_ID, ckaid),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, cklabel),
	}

//如果缺少值，则允许某些错误
	attr, err := p11lib.GetAttributeValue(session, obj, template)
	if err != nil {
		logger.Debugf("P11: get(attrlist) [%s]\n", err)
	}

	for _, a := range attr {
//如果绑定提供了将属性十六进制转换为字符串的方法，则更友好
		logger.Debugf("ListAttr: type %d/0x%x, length %d\n%s", a.Type, a.Type, len(a.Value), hex.Dump(a.Value))
	}
}

func (csp *impl) getSecretValue(ski []byte) []byte {
	p11lib := csp.ctx
	session := csp.getSession()
	defer csp.returnSession(session)

	keyHandle, err := findKeyPairFromSKI(p11lib, session, ski, privateKeyFlag)
	if err != nil {
		logger.Warningf("P11: findKeyPairFromSKI [%s]\n", err)
	}
	var privKey []byte
	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, privKey),
	}

//如果缺少值，则允许某些错误
	attr, err := p11lib.GetAttributeValue(session, *keyHandle, template)
	if err != nil {
		logger.Warningf("P11: get(attrlist) [%s]\n", err)
	}

	for _, a := range attr {
//如果绑定提供了将属性十六进制转换为字符串的方法，则更友好
		logger.Debugf("ListAttr: type %d/0x%x, length %d\n%s", a.Type, a.Type, len(a.Value), hex.Dump(a.Value))
		return a.Value
	}
	logger.Warningf("No Key Value found: %v", err)
	return nil
}

var (
	bigone  = new(big.Int).SetInt64(1)
	idCtr   = new(big.Int)
	idMutex sync.Mutex
)

func nextIDCtr() *big.Int {
	idMutex.Lock()
	idCtr = new(big.Int).Add(idCtr, bigone)
	idMutex.Unlock()
	return idCtr
}
