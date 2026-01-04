package metric

import (
	"errors"
	"io"
	"openwrt-diskio-api/backend/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var testProcPaths = model.ProcfsPaths{}

type TestReader struct {
	mock.Mock
}

func (c *TestReader) ReadFile(path string) (string, error) {
	result := c.Called(path)
	// 第 0 个返回值是 string，第 1 个是 error
	return result.String(0), result.Error(1)
}
func (c *TestReader) Exists(path string) bool {
	return true
}
func (c *TestReader) Open(path string) (io.ReadCloser, error) {
	return nil, nil
}

func NewMockReader(readData string, mockError error, path string) *TestReader {
	reader := &TestReader{}
	reader.
		On("ReadFile", path).
		Return(readData, mockError)
	return reader
}

type TestCommandRunner struct {
	mock.Mock
}

func (c *TestCommandRunner) Run(name string, args ...string) (string, error) {
	result := c.Called(name, args)
	// 第 0 个返回值是 string，第 1 个是 error
	return result.String(0), result.Error(1)
}

func NewMockRunner(mockData string, mockError error, commands []string) *TestCommandRunner {
	reader := &TestCommandRunner{}
	reader.
		On("Run", commands[0], commands[1:]).
		Return(mockData, mockError)
	return reader
}

func TestReadCpuTemperature(t *testing.T) {

	testCases := []struct {
		testName  string
		readData  string
		mockError error
		expected  float64
	}{
		{
			testName:  "load proc file success",
			readData:  "29444",
			mockError: nil,
			expected:  29.444,
		},
		{
			testName:  "load proc file failed",
			readData:  "",
			mockError: errors.New("Test error"),
			expected:  float64(-1),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			reader := NewMockReader(testCase.readData, testCase.mockError, testProcPaths.CpuTemp())
			temperature, unit := readCpuTemperature(reader)
			assert.Equal(t, testCase.expected, temperature)
			assert.Equal(t, model.Celsius, unit)

		})
	}
}

func TestReadCpuIdle(t *testing.T) {
	testCases := []struct {
		testName  string
		readData  string
		mockError error
		expected1 uint64
		expected2 uint64
		expected3 []model.CpuSnapUnit
		expected4 error
	}{
		{
			testName: "load proc file success",
			readData: `cpu  1043791 0 1339539 265632208 325 0 1644933 0 0 0
cpu0 259672 0 330535 66794754 147 0 230147 0 0 0
cpu1 263110 0 336276 66415272 64 0 368819 0 0 0
cpu2 268302 0 346996 66375303 55 0 460430 0 0 0
cpu3 252705 0 325730 66046877 58 0 585535 0 0 0
intr 703399859 0 40388647 102208287 0 0 52118872 50625529 0 0 0 0 87500381 0 0 169338861 0 0 0 0 8615289 0 1 1 1 0 0 0 0 0 0 0 0 0 0 0 0 0 70716787 0 0 0 15412 0 0 0 0 0 0 41101 0 0 0 0 0 18 0 0 0 1 0 0 0 0 0 0 0 6 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 121830665 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
ctxt 430686250
btime 1766670786
processes 677083
procs_running 1
procs_blocked 0
softirq 709050685 1337984 167961916 259144 400626814 18849 0 2523414 104303354 175 32019035`,
			mockError: nil,
			expected1: 269660796,
			expected2: 265632208,
			expected3: []model.CpuSnapUnit{
				{Cycles: 67615255, Idle: 66794754},
				{Cycles: 67383541, Idle: 66415272},
				{Cycles: 67451086, Idle: 66375303},
				{Cycles: 67210905, Idle: 66046877},
			},
			expected4: nil,
		},
		{
			testName:  "load proc file failed",
			readData:  "",
			mockError: errors.New("Test error"),
			expected1: 0,
			expected2: 0,
			expected3: nil,
			expected4: errors.New("Test error"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			reader := NewMockReader(testCase.readData, testCase.mockError, testProcPaths.CpuUsage())
			allCoreCycles, allCoreIdle, coresIdle, err := readCpuIdle(reader)
			assert.Equal(t, testCase.expected1, allCoreCycles)
			assert.Equal(t, testCase.expected2, allCoreIdle)
			assert.Equal(t, testCase.expected3, coresIdle)
			assert.Equal(t, testCase.expected4, err)
		})
	}
}

func TestReadTotalCpuUsage(t *testing.T) {
	testCases := []struct {
		testName  string
		readData  string
		mockError error
		lastSnap  *model.CpuSnap
		expected1 uint64
		expected2 uint64
		expected3 []model.CpuSnapUnit
		expected4 error
	}{
		{
			testName: "Correct cpu idle",
			readData: `cpu  1043791 0 1339539 265632208 325 0 1644933 0 0 0
cpu0 259672 0 330535 66794754 147 0 230147 0 0 0
cpu1 263110 0 336276 66415272 64 0 368819 0 0 0
cpu2 268302 0 346996 66375303 55 0 460430 0 0 0
cpu3 252705 0 325730 66046877 58 0 585535 0 0 0
intr 703399859 0 40388647 102208287 0 0 52118872 50625529 0 0 0 0 87500381 0 0 169338861 0 0 0 0 8615289 0 1 1 1 0 0 0 0 0 0 0 0 0 0 0 0 0 70716787 0 0 0 15412 0 0 0 0 0 0 41101 0 0 0 0 0 18 0 0 0 1 0 0 0 0 0 0 0 6 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 121830665 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
ctxt 430686250
btime 1766670786
processes 677083
procs_running 1
procs_blocked 0
softirq 709050685 1337984 167961916 259144 400626814 18849 0 2523414 104303354 175 32019035`,
			mockError: nil,
			expected1: 269660796,
			expected2: 265632208,
			expected3: []model.CpuSnapUnit{
				{Cycles: 67615255, Idle: 66794754},
				{Cycles: 67383541, Idle: 66415272},
				{Cycles: 67451086, Idle: 66375303},
				{Cycles: 67210905, Idle: 66046877},
			},
			expected4: nil,
		},
		{
			testName:  "load temperature failed",
			readData:  "",
			mockError: errors.New("Test error"),
			expected1: 0,
			expected2: 0,
			expected3: nil,
			expected4: errors.New("Test error"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			reader := NewMockReader(testCase.readData, testCase.mockError, testProcPaths.CpuUsage())
			allCoreCycles, allCoreIdle, coresIdle, err := readCpuIdle(reader)
			assert.Equal(t, testCase.expected1, allCoreCycles)
			assert.Equal(t, testCase.expected2, allCoreIdle)
			assert.Equal(t, testCase.expected3, coresIdle)
			assert.Equal(t, testCase.expected4, err)
		})
	}

}

func TestReadLocalTimeZone(t *testing.T) {
	testCases := []struct {
		testName        string
		readerReturn    string
		readerMockError error
		runnerReturn    string
		runnerMockError error
		expected        string
	}{
		{
			testName: "load openwrt config get timezone",
			readerReturn: `
config system
        option ttylogin '0'
        option urandom_seed '0'
        option hostname 'iStoreOS'
        option compat_version '1.0'
        option zonename 'Asia/Shanghai'
        option timezone 'CST-8'
        option log_proto 'udp'
        option conloglevel '8'
        option cronloglevel '5'
        option log_size '16384'
`,
			readerMockError: nil,
			runnerReturn:    "",
			runnerMockError: errors.New("test error"),
			expected:        "Asia/Shanghai",
		},
		{
			testName:        "timedatectl get timezone",
			readerReturn:    "",
			readerMockError: nil,
			runnerReturn:    "Asia/Shanghai",
			runnerMockError: nil,
			expected:        "Asia/Shanghai",
		},
		{
			testName:        "get timezone failed",
			readerReturn:    "",
			readerMockError: errors.New("test error"),
			runnerReturn:    "",
			runnerMockError: errors.New("test error"),
			expected:        model.StringDefault,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			reader := NewMockReader(testCase.readerReturn,
				testCase.readerMockError,
				testProcPaths.SystemConfig(),
			)
			runner := NewMockRunner(testCase.runnerReturn,
				testCase.runnerMockError,
				[]string{"timedatectl", "show", "-p", "Timezone", "--value"},
			)
			result := readLocalTimeZone(reader, runner)
			assert.Equal(t, testCase.expected, result)
		})
	}
}
