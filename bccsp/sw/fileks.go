
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

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

package sw

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/utils"
)

//NewFileBasedKeyStore在给定位置实例化了一个基于文件的密钥存储。
//如果指定了非空密码，则可以加密密钥存储。
//它也可以设置为只读。在这种情况下，任何存储操作
//将被禁止
func NewFileBasedKeyStore(pwd []byte, path string, readOnly bool) (bccsp.KeyStore, error) {
	ks := &fileBasedKeyStore{}
	return ks, ks.Init(pwd, path, readOnly)
}

//FileBasedKeyStore是一个基于文件夹的密钥库。
//每个密钥都存储在一个单独的文件中，该文件的名称包含密钥的ski
//以及标识密钥类型的标志。所有钥匙都存储在
//在初始化时提供其路径的文件夹。
//密钥库可以用密码初始化，这个密码
//用于加密和解密存储密钥的文件。
//密钥库可以是只读的，以避免覆盖密钥。
type fileBasedKeyStore struct {
	path string

	readOnly bool
	isOpen   bool

	pwd []byte

//同步
	m sync.Mutex
}

//init用密码（文件夹路径）初始化此密钥库
//存储密钥的位置和只读标志。
//每个密钥都存储在一个单独的文件中，该文件的名称包含密钥的ski
//以及标识密钥类型的标志。
//如果密钥库是用密码初始化的，则此密码
//用于加密和解密存储密钥的文件。
//对于非加密密钥存储库，pwd可以为零。如果是加密的
//在没有密码的情况下初始化密钥存储，然后从
//密钥库将失败。
//密钥库可以是只读的，以避免覆盖密钥。
func (ks *fileBasedKeyStore) Init(pwd []byte, path string, readOnly bool) error {
//验证输入
//随钻测井可以为零

	if len(path) == 0 {
		return errors.New("An invalid KeyStore path provided. Path cannot be an empty string.")
	}

	ks.m.Lock()
	defer ks.m.Unlock()

	if ks.isOpen {
		return errors.New("KeyStore already initilized.")
	}

	ks.path = path
	ks.pwd = utils.Clone(pwd)

	err := ks.createKeyStoreIfNotExists()
	if err != nil {
		return err
	}

	err = ks.openKeyStore()
	if err != nil {
		return err
	}

	ks.readOnly = readOnly

	return nil
}

//如果此密钥库是只读的，则read only返回true，否则返回false。
//如果readonly为true，则storekey将失败。
func (ks *fileBasedKeyStore) ReadOnly() bool {
	return ks.readOnly
}

//getkey返回一个key对象，其ski是通过的。
func (ks *fileBasedKeyStore) GetKey(ski []byte) (bccsp.Key, error) {
//验证参数
	if len(ski) == 0 {
		return nil, errors.New("Invalid SKI. Cannot be of zero length.")
	}

	suffix := ks.getSuffix(hex.EncodeToString(ski))

	switch suffix {
	case "key":
//加载密钥
		key, err := ks.loadKey(hex.EncodeToString(ski))
		if err != nil {
			return nil, fmt.Errorf("Failed loading key [%x] [%s]", ski, err)
		}

		return &aesPrivateKey{key, false}, nil
	case "sk":
//加载私钥
		key, err := ks.loadPrivateKey(hex.EncodeToString(ski))
		if err != nil {
			return nil, fmt.Errorf("Failed loading secret key [%x] [%s]", ski, err)
		}

		switch key.(type) {
		case *ecdsa.PrivateKey:
			return &ecdsaPrivateKey{key.(*ecdsa.PrivateKey)}, nil
		case *rsa.PrivateKey:
			return &rsaPrivateKey{key.(*rsa.PrivateKey)}, nil
		default:
			return nil, errors.New("Secret key type not recognized")
		}
	case "pk":
//加载公钥
		key, err := ks.loadPublicKey(hex.EncodeToString(ski))
		if err != nil {
			return nil, fmt.Errorf("Failed loading public key [%x] [%s]", ski, err)
		}

		switch key.(type) {
		case *ecdsa.PublicKey:
			return &ecdsaPublicKey{key.(*ecdsa.PublicKey)}, nil
		case *rsa.PublicKey:
			return &rsaPublicKey{key.(*rsa.PublicKey)}, nil
		default:
			return nil, errors.New("Public key type not recognized")
		}
	default:
		return ks.searchKeystoreForSKI(ski)
	}
}

//storekey将密钥k存储在此密钥库中。
//如果此密钥库是只读的，则该方法将失败。
func (ks *fileBasedKeyStore) StoreKey(k bccsp.Key) (err error) {
	if ks.readOnly {
		return errors.New("Read only KeyStore.")
	}

	if k == nil {
		return errors.New("Invalid key. It must be different from nil.")
	}
	switch k.(type) {
	case *ecdsaPrivateKey:
		kk := k.(*ecdsaPrivateKey)

		err = ks.storePrivateKey(hex.EncodeToString(k.SKI()), kk.privKey)
		if err != nil {
			return fmt.Errorf("Failed storing ECDSA private key [%s]", err)
		}

	case *ecdsaPublicKey:
		kk := k.(*ecdsaPublicKey)

		err = ks.storePublicKey(hex.EncodeToString(k.SKI()), kk.pubKey)
		if err != nil {
			return fmt.Errorf("Failed storing ECDSA public key [%s]", err)
		}

	case *rsaPrivateKey:
		kk := k.(*rsaPrivateKey)

		err = ks.storePrivateKey(hex.EncodeToString(k.SKI()), kk.privKey)
		if err != nil {
			return fmt.Errorf("Failed storing RSA private key [%s]", err)
		}

	case *rsaPublicKey:
		kk := k.(*rsaPublicKey)

		err = ks.storePublicKey(hex.EncodeToString(k.SKI()), kk.pubKey)
		if err != nil {
			return fmt.Errorf("Failed storing RSA public key [%s]", err)
		}

	case *aesPrivateKey:
		kk := k.(*aesPrivateKey)

		err = ks.storeKey(hex.EncodeToString(k.SKI()), kk.privKey)
		if err != nil {
			return fmt.Errorf("Failed storing AES key [%s]", err)
		}

	default:
		return fmt.Errorf("Key type not reconigned [%s]", k)
	}

	return
}

func (ks *fileBasedKeyStore) searchKeystoreForSKI(ski []byte) (k bccsp.Key, err error) {

	files, _ := ioutil.ReadDir(ks.path)
	for _, f := range files {
		if f.IsDir() {
			continue
		}

if f.Size() > (1 << 16) { //64K，有点任意限制，甚至考虑到大型RSA密钥
			continue
		}

		raw, err := ioutil.ReadFile(filepath.Join(ks.path, f.Name()))
		if err != nil {
			continue
		}

		key, err := utils.PEMtoPrivateKey(raw, ks.pwd)
		if err != nil {
			continue
		}

		switch key.(type) {
		case *ecdsa.PrivateKey:
			k = &ecdsaPrivateKey{key.(*ecdsa.PrivateKey)}
		case *rsa.PrivateKey:
			k = &rsaPrivateKey{key.(*rsa.PrivateKey)}
		default:
			continue
		}

		if !bytes.Equal(k.SKI(), ski) {
			continue
		}

		return k, nil
	}
	return nil, fmt.Errorf("Key with SKI %s not found in %s", hex.EncodeToString(ski), ks.path)
}

func (ks *fileBasedKeyStore) getSuffix(alias string) string {
	files, _ := ioutil.ReadDir(ks.path)
	for _, f := range files {
		if strings.HasPrefix(f.Name(), alias) {
			if strings.HasSuffix(f.Name(), "sk") {
				return "sk"
			}
			if strings.HasSuffix(f.Name(), "pk") {
				return "pk"
			}
			if strings.HasSuffix(f.Name(), "key") {
				return "key"
			}
			break
		}
	}
	return ""
}

func (ks *fileBasedKeyStore) storePrivateKey(alias string, privateKey interface{}) error {
	rawKey, err := utils.PrivateKeyToPEM(privateKey, ks.pwd)
	if err != nil {
		logger.Errorf("Failed converting private key to PEM [%s]: [%s]", alias, err)
		return err
	}

	err = ioutil.WriteFile(ks.getPathForAlias(alias, "sk"), rawKey, 0600)
	if err != nil {
		logger.Errorf("Failed storing private key [%s]: [%s]", alias, err)
		return err
	}

	return nil
}

func (ks *fileBasedKeyStore) storePublicKey(alias string, publicKey interface{}) error {
	rawKey, err := utils.PublicKeyToPEM(publicKey, ks.pwd)
	if err != nil {
		logger.Errorf("Failed converting public key to PEM [%s]: [%s]", alias, err)
		return err
	}

	err = ioutil.WriteFile(ks.getPathForAlias(alias, "pk"), rawKey, 0600)
	if err != nil {
		logger.Errorf("Failed storing private key [%s]: [%s]", alias, err)
		return err
	}

	return nil
}

func (ks *fileBasedKeyStore) storeKey(alias string, key []byte) error {
	pem, err := utils.AEStoEncryptedPEM(key, ks.pwd)
	if err != nil {
		logger.Errorf("Failed converting key to PEM [%s]: [%s]", alias, err)
		return err
	}

	err = ioutil.WriteFile(ks.getPathForAlias(alias, "key"), pem, 0600)
	if err != nil {
		logger.Errorf("Failed storing key [%s]: [%s]", alias, err)
		return err
	}

	return nil
}

func (ks *fileBasedKeyStore) loadPrivateKey(alias string) (interface{}, error) {
	path := ks.getPathForAlias(alias, "sk")
	logger.Debugf("Loading private key [%s] at [%s]...", alias, path)

	raw, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Errorf("Failed loading private key [%s]: [%s].", alias, err.Error())

		return nil, err
	}

	privateKey, err := utils.PEMtoPrivateKey(raw, ks.pwd)
	if err != nil {
		logger.Errorf("Failed parsing private key [%s]: [%s].", alias, err.Error())

		return nil, err
	}

	return privateKey, nil
}

func (ks *fileBasedKeyStore) loadPublicKey(alias string) (interface{}, error) {
	path := ks.getPathForAlias(alias, "pk")
	logger.Debugf("Loading public key [%s] at [%s]...", alias, path)

	raw, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Errorf("Failed loading public key [%s]: [%s].", alias, err.Error())

		return nil, err
	}

	privateKey, err := utils.PEMtoPublicKey(raw, ks.pwd)
	if err != nil {
		logger.Errorf("Failed parsing private key [%s]: [%s].", alias, err.Error())

		return nil, err
	}

	return privateKey, nil
}

func (ks *fileBasedKeyStore) loadKey(alias string) ([]byte, error) {
	path := ks.getPathForAlias(alias, "key")
	logger.Debugf("Loading key [%s] at [%s]...", alias, path)

	pem, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Errorf("Failed loading key [%s]: [%s].", alias, err.Error())

		return nil, err
	}

	key, err := utils.PEMtoAES(pem, ks.pwd)
	if err != nil {
		logger.Errorf("Failed parsing key [%s]: [%s]", alias, err)

		return nil, err
	}

	return key, nil
}

func (ks *fileBasedKeyStore) createKeyStoreIfNotExists() error {
//检查密钥库目录
	ksPath := ks.path
	missing, err := utils.DirMissingOrEmpty(ksPath)

	if missing {
		logger.Debugf("KeyStore path [%s] missing [%t]: [%s]", ksPath, missing, utils.ErrToString(err))

		err := ks.createKeyStore()
		if err != nil {
			logger.Errorf("Failed creating KeyStore At [%s]: [%s]", ksPath, err.Error())
			return nil
		}
	}

	return nil
}

func (ks *fileBasedKeyStore) createKeyStore() error {
//如果密钥库目录根目录尚不存在，则创建它
	ksPath := ks.path
	logger.Debugf("Creating KeyStore at [%s]...", ksPath)

	os.MkdirAll(ksPath, 0755)

	logger.Debugf("KeyStore created at [%s].", ksPath)
	return nil
}

func (ks *fileBasedKeyStore) openKeyStore() error {
	if ks.isOpen {
		return nil
	}
	ks.isOpen = true
	logger.Debugf("KeyStore opened at [%s]...done", ks.path)

	return nil
}

func (ks *fileBasedKeyStore) getPathForAlias(alias, suffix string) string {
	return filepath.Join(ks.path, alias+"_"+suffix)
}
