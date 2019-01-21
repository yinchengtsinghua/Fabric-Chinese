
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


package util

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/config/configtest"
	cutil "github.com/hyperledger/fabric/core/container/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

//testHashContentChange更改内容中的随机字节并检查哈希更改
func TestHashContentChange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestHashContentChange")
	}
	b := []byte("firstcontent")
	hash := util.ComputeSHA256(b)

	b2 := []byte("To be, or not to be- that is the question: Whether 'tis nobler in the mind to suffer The slings and arrows of outrageous fortune Or to take arms against a sea of troubles, And by opposing end them. To die- to sleep- No more; and by a sleep to say we end The heartache, and the thousand natural shocks That flesh is heir to. 'Tis a consummation Devoutly to be wish'd.")

	h1 := ComputeHash(b2, hash)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randIndex := (int(r.Uint32())) % len(b2)

	randByte := byte((int(r.Uint32())) % 128)

//确保两个字节不同
	for {
		if randByte != b2[randIndex] {
			break
		}

		randByte = byte((int(r.Uint32())) % 128)
	}

//更改随机字节
	b2[randIndex] = randByte

//这是正在测试的核心哈希函数
	h2 := ComputeHash(b2, hash)

//这两个哈希值应该不同
	if bytes.Compare(h1, h2) == 0 {
		t.Error("Hash expected to be different but is same")
	}
}

//testhashlenchange更改内容的随机长度并检查哈希更改
func TestHashLenChange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestHashLenChange")
	}
	b := []byte("firstcontent")
	hash := util.ComputeSHA256(b)

	b2 := []byte("To be, or not to be-")

	h1 := ComputeHash(b2, hash)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randIndex := (int(r.Uint32())) % len(b2)

	b2 = b2[0:randIndex]

	h2 := ComputeHash(b2, hash)

//哈希值应不同
	if bytes.Compare(h1, h2) == 0 {
		t.Error("Hash expected to be different but is same")
	}
}

//testHashOrderChange更改行列表上哈希计算的顺序并检查哈希更改
func TestHashOrderChange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestHashOrderChange")
	}
	b := []byte("firstcontent")
	hash := util.ComputeSHA256(b)

	b2 := [][]byte{[]byte("To be, or not to be- that is the question:"),
		[]byte("Whether 'tis nobler in the mind to suffer"),
		[]byte("The slings and arrows of outrageous fortune"),
		[]byte("Or to take arms against a sea of troubles,"),
		[]byte("And by opposing end them."),
		[]byte("To die- to sleep- No more; and by a sleep to say we end"),
		[]byte("The heartache, and the thousand natural shocks"),
		[]byte("That flesh is heir to."),
		[]byte("'Tis a consummation Devoutly to be wish'd.")}
	h1 := hash

	for _, l := range b2 {
		h1 = ComputeHash(l, h1)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randIndex1 := (int(r.Uint32())) % len(b2)
	randIndex2 := (int(r.Uint32())) % len(b2)

//确保这两件事不一样
	for {
		if randIndex2 != randIndex1 {
			break
		}

		randIndex2 = (int(r.Uint32())) % len(b2)
	}

//切换任意两条线
	tmp := b2[randIndex2]
	b2[randIndex2] = b2[randIndex1]
	b2[randIndex1] = tmp

	h2 := hash
	for _, l := range b2 {
		h2 = ComputeHash(l, hash)
	}

//哈希值应不同
	if bytes.Compare(h1, h2) == 0 {
		t.Error("Hash expected to be different but is same")
	}
}

//testhashoverfiles计算目录上的哈希值，并确保它与预计算的、硬编码的哈希值匹配。
func TestHashOverFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestHashOverFiles")
	}
	b := []byte("firstcontent")
	hash := util.ComputeSHA256(b)

	hash, err := HashFilesInDir(".", "hashtestfiles1", hash, nil)

	if err != nil {
		t.Fail()
		t.Logf("error : %s", err)
	}

//只要“hashtestfiles1”下没有更改任何文件，hash就应该始终计算为以下值
	expectedHash := "0c92180028200dfabd08d606419737f5cdecfcbab403e3f0d79e8d949f4775bc"

	computedHash := hex.EncodeToString(hash[:])

	if expectedHash != computedHash {
		t.Error("Hash expected to be unchanged")
	}
}

func TestHashDiffDir(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestHashDiffDir")
	}
	b := []byte("firstcontent")
	hash := util.ComputeSHA256(b)

	hash1, err := HashFilesInDir(".", "hashtestfiles1", hash, nil)
	if err != nil {
		t.Errorf("Error getting code %s", err)
	}
	hash2, err := HashFilesInDir(".", "hashtestfiles2", hash, nil)
	if err != nil {
		t.Errorf("Error getting code %s", err)
	}
	if bytes.Compare(hash1, hash2) == 0 {
		t.Error("Hash should be different for 2 different remote repos")
	}
}

func TestHashSameDir(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestHashSameDir")
	}
	assert := assert.New(t)

	b := []byte("firstcontent")
	hash := util.ComputeSHA256(b)
	hash1, err := HashFilesInDir(".", "hashtestfiles1", hash, nil)
	assert.NoError(err, "Error getting code")

	fname := os.TempDir() + "/hash.tar"
	w, err := os.Create(fname)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fname)
	tw := tar.NewWriter(w)
	defer w.Close()
	defer tw.Close()
	hash2, err := HashFilesInDir(".", "hashtestfiles1", hash, tw)
	assert.NoError(err, "Error getting code")

	assert.Equal(bytes.Compare(hash1, hash2), 0,
		"Hash should be same across multiple downloads")
}

func TestHashBadWriter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestHashBadWriter")
	}
	b := []byte("firstcontent")
	hash := util.ComputeSHA256(b)

	fname := os.TempDir() + "/hash.tar"
	w, err := os.Create(fname)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fname)
	tw := tar.NewWriter(w)
	defer w.Close()
	tw.Close()

	_, err = HashFilesInDir(".", "hashtestfiles1", hash, tw)
	assert.Error(t, err,
		"HashFilesInDir invoked with closed writer, should have failed")
}

//testhashnoexistentdir用不存在的目录测试hashfileindir
func TestHashNonExistentDir(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestHashNonExistentDir")
	}
	b := []byte("firstcontent")
	hash := util.ComputeSHA256(b)
	_, err := HashFilesInDir(".", "idontexist", hash, nil)
	assert.Error(t, err, "Expected an error for non existent directory idontexist")
}

//testicodexist测试iscodeexist函数
func TestIsCodeExist(t *testing.T) {
	assert := assert.New(t)
	path := os.TempDir()
	err := IsCodeExist(path)
	assert.NoError(err,
		"%s directory exists, IsCodeExist should not have returned error: %v",
		path, err)

	dir, err := ioutil.TempDir(os.TempDir(), "iscodeexist")
	assert.NoError(err)
	defer os.RemoveAll(dir)
	path = dir + "/blah"
	err = IsCodeExist(path)
	assert.Error(err,
		fmt.Sprintf("%s directory does not exist, IsCodeExist should have returned error", path))

	f := createTempFile(t)
	defer os.Remove(f)
	err = IsCodeExist(f)
	assert.Error(err, fmt.Sprintf("%s is a file, IsCodeExist should have returned error", f))
}

//TestDockerBuild测试DockerBuild函数
func TestDockerBuild(t *testing.T) {
	assert := assert.New(t)
	var err error

	ldflags := "-linkmode external -extldflags '-static'"
	codepackage := bytes.NewReader(getDeploymentPayload())
	binpackage := bytes.NewBuffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	err = DockerBuild(DockerBuildOptions{
		Cmd: fmt.Sprintf("GOPATH=/chaincode/input:$GOPATH go build -ldflags \"%s\" -o /chaincode/output/chaincode helloworld",
			ldflags),
		InputStream:  codepackage,
		OutputStream: binpackage,
	})
	assert.NoError(err, "DockerBuild failed")
}

func getDeploymentPayload() []byte {
	var goprog = `
	package main
	import "fmt"
	func main() {
		fmt.Println("Hello World")
	}
	`
	var zeroTime time.Time
	payload := bytes.NewBufferString(goprog).Bytes()
	inputbuf := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(inputbuf)
	tw := tar.NewWriter(gw)
	tw.WriteHeader(&tar.Header{
		Name:       "src/helloworld/helloworld.go",
		Size:       int64(len(payload)),
		Mode:       0600,
		ModTime:    zeroTime,
		AccessTime: zeroTime,
		ChangeTime: zeroTime,
	})
	tw.Write(payload)
	tw.Close()
	gw.Close()
	return inputbuf.Bytes()
}

func createTempFile(t *testing.T) string {
	tmpfile, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatal(err)
		return ""
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	return tmpfile.Name()
}

//要在建立多拱门后恢复此测试
func TestDockerPull(t *testing.T) {
	codepackage, output := io.Pipe()
	go func() {
		tw := tar.NewWriter(output)

		tw.Close()
		output.Close()
	}()

	binpackage := bytes.NewBuffer(nil)

//
//published and available.  Ideally we could choose something that we know is both multi-arch
//在执行dockerbuild之前删除。这将确保我们练习
//图像拉逻辑。但是，不存在符合所有标准的合适目标。因此
//我们决定使用已知的发布图像。我们不知道图像是否已经
//从本身下载，我们不想先显式删除这个特定的图像，因为
//它可能在其他地方合法使用。相反，我们只知道这应该永远
//工作并称之为“足够近”。
//
//未来的考虑：发布一个已知的虚拟图像，它是多拱门的，并且可以随意地
//删除，并在此使用。
	err := DockerBuild(DockerBuildOptions{
		Image:        cutil.ParseDockerfileTemplate("hyperledger/fabric-ccenv:$(ARCH)-1.1.0"),
		Cmd:          "/bin/true",
		InputStream:  codepackage,
		OutputStream: binpackage,
	})
	if err != nil {
		t.Errorf("Error during build: %s", err)
	}
}

func TestMain(m *testing.M) {
	viper.SetConfigName("core")
	viper.SetEnvPrefix("CORE")
	configtest.AddDevConfigPath(nil)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("could not read config %s\n", err)
		os.Exit(-1)
	}
	os.Exit(m.Run())
}
