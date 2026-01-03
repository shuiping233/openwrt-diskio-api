package metric

import (
	"openwrt-diskio-api/src/utils"
	"path"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const (
	UtTestDirPath = "./.UtTestTemp"
)

type TestTempFile struct {
	tempFilePath string
}

func (tt *TestTempFile) PrepareTempFile(t *testing.T, data string) (filepath string, length int) {
	fs := afero.NewOsFs()
	tempPath := path.Join(UtTestDirPath, utils.RandHex(32))

	err := fs.MkdirAll(UtTestDirPath, 0666)
	if err != nil {
		t.Fatal(err)
	}
	err = afero.WriteFile(fs, tempPath, []byte(data), 0666)
	if err != nil {
		t.Fatal(err)
	}
	tt.tempFilePath = tempPath
	return tempPath, len(data)
}

func (t *TestTempFile) CleanUp() {
	afero.NewOsFs().RemoveAll(UtTestDirPath)
}

func TestReaderReadFile(t *testing.T) {
	testText := "hello world"
	testTempFile := TestTempFile{}
	testFilePath, _ := testTempFile.PrepareTempFile(t, testText)
	defer testTempFile.CleanUp()
	reader := FsReader{Fs: afero.NewOsFs()}

	readResult, err := reader.ReadFile(testFilePath)
	assert.NoError(t, err)
	assert.Equal(t, testText, readResult)

	readResult, err = reader.ReadFile("/path/notExist")
	assert.Error(t, err)
	assert.Equal(t, "", readResult)

}
func TestReaderOpen(t *testing.T) {
	testText := "hello world"
	testTempFile := TestTempFile{}
	testFilePath, _ := testTempFile.PrepareTempFile(t, testText)
	defer testTempFile.CleanUp()
	reader := FsReader{Fs: afero.NewOsFs()}

	readCloser, err := reader.Open(testFilePath)
	buffer := make([]byte, len(testText))
	assert.NoError(t, err)
	_, err = readCloser.Read(buffer)
	assert.NoError(t, err)
	assert.Equal(t, testText, string(buffer))

	readCloser, err = reader.Open("/path/notExist")
	assert.Error(t, err)
}

func TestReaderExists(t *testing.T) {
	testText := "hello world"
	testTempFile := TestTempFile{}
	testFilePath, _ := testTempFile.PrepareTempFile(t, testText)
	defer testTempFile.CleanUp()
	reader := FsReader{Fs: afero.NewOsFs()}

	exists := reader.Exists(testFilePath)
	assert.Equal(t, true, exists)

	exists = reader.Exists("/path/notExist")
	assert.Equal(t, false, exists)
}
