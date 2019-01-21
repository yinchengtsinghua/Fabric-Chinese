
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/core/config/configtest"
	config "github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/stretchr/testify/assert"
)

func TestCreateLedgerFactory(t *testing.T) {
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()
	testCases := []struct {
		name            string
		ledgerType      string
		ledgerDir       string
		ledgerDirPrefix string
		expectPanic     bool
	}{
		{"RAM", "ram", "", "", false},
		{"JSONwithPathSet", "json", "test-dir", "", false},
		{"JSONwithPathUnset", "json", "", "test-prefix", false},
		{"FilewithPathSet", "file", filepath.Join(os.TempDir(), "test-dir"), "", false},
		{"FilewithPathUnset", "file", "", "test-prefix", false},
	}

	conf, err := config.Load()
	if err != nil {
		t.Fatal("failed to load config")
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tc.expectPanic && r == nil {
					t.Fatal("Should have panicked")
				}
				if !tc.expectPanic && r != nil {
					t.Fatal("Should not have panicked")
				}
			}()

			conf.General.LedgerType = tc.ledgerType
			conf.FileLedger.Location = tc.ledgerDir
			conf.FileLedger.Prefix = tc.ledgerDirPrefix
			lf, ld := createLedgerFactory(conf)

			defer func() {
				if ld != "" {
					os.RemoveAll(ld)
					t.Log("Removed temp dir:", ld)
				}
			}()
			lf.ChainIDs()
		})
	}
}

func TestCreateSubDir(t *testing.T) {
	testCases := []struct {
		name          string
		count         int
		expectCreated bool
		expectPanic   bool
	}{
		{"CleanDir", 1, true, false},
		{"HasSubDir", 2, false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tc.expectPanic && r == nil {
					t.Fatal("Should have panicked")
				}
				if !tc.expectPanic && r != nil {
					t.Fatal("Should not have panicked")
				}
			}()

			parentDirPath := createTempDir("test-dir")

			var created bool
			for i := 0; i < tc.count; i++ {
				_, created = createSubDir(parentDirPath, "test-sub-dir")
			}

			if created != tc.expectCreated {
				t.Fatalf("Sub dir created = %v, but expectation was = %v", created, tc.expectCreated)
			}
		})
	}
	t.Run("ParentDirNotExists", func(t *testing.T) {
		assert.Panics(t, func() { createSubDir(os.TempDir(), "foo/name") })
	})
}

func TestCreateTempDir(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		tempDir := createTempDir("foo")
		if _, err := os.Stat(tempDir); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Bad", func(t *testing.T) {
		assert.Panics(t, func() {
			createTempDir("foo/bar")
		})
	})

}
