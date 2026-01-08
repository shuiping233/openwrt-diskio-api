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

func TestSelectPrivateAddress(t *testing.T) {

	privateCidr := []string{
		"192.168.0.0/24",
	}

	type result struct {
		srcAddr string
		srcPort int
		dstAddr string
		dstPort int
	}

	testCases := []struct {
		testName                string
		originConnectionSrcAddr string
		replyConnectionSrcAddr  string
		originConnectionSrcPort int
		replyConnectionSrcPort  int
		originConnectionDstAddr string
		replyConnectionDstAddr  string
		originConnectionDstPort int
		replyConnectionDstPort  int
		expected                result
	}{
		{
			// ipv4     2 udp      17 33 src=192.168.0.249 dst=175.153.161.77 sport=9993 dport=44315 packets=1 bytes=56 [UNREPLIED] src=175.153.161.77 dst=112.93.49.21 sport=44315 dport=9993 packets=0 bytes=0 mark=0 zone=0 use=2
			testName:                "nat outbound",
			originConnectionSrcAddr: "192.168.0.249",
			originConnectionDstAddr: "175.153.161.77",
			originConnectionSrcPort: 9993,
			originConnectionDstPort: 44315,
			replyConnectionSrcAddr:  "175.153.161.77",
			replyConnectionDstAddr:  "112.93.49.21",
			replyConnectionSrcPort:  44315,
			replyConnectionDstPort:  9993,
			expected: result{
				srcAddr: "192.168.0.249",
				srcPort: 9993,
				dstAddr: "175.153.161.77",
				dstPort: 44315,
			},
		},
		{
			// ipv4     2 tcp      6 7428 ESTABLISHED src=110.42.53.227 dst=112.93.49.21 sport=39451 dport=10091 packets=18086 bytes=1096158 src=192.168.0.173 dst=110.42.53.227 sport=10091 dport=39451 packets=18031 bytes=1159253 [ASSURED] mark=0 zone=0 use=2
			testName:                "nat inbound",
			originConnectionSrcAddr: "110.42.53.227",
			originConnectionDstAddr: "112.93.49.21",
			originConnectionSrcPort: 39451,
			originConnectionDstPort: 10091,
			replyConnectionSrcAddr:  "192.168.0.173",
			replyConnectionDstAddr:  "110.42.53.227",
			replyConnectionSrcPort:  10091,
			replyConnectionDstPort:  39451,
			expected: result{
				srcAddr: "110.42.53.227",
				srcPort: 39451,
				dstAddr: "192.168.0.173",
				dstPort: 10091,
			},
		},
		{
			// ipv4     2 udp      17 38 src=192.168.0.249 dst=192.168.0.1 sport=65268 dport=5351 packets=1 bytes=30 [UNREPLIED] src=192.168.0.1 dst=192.168.0.249 sport=5351 dport=65268 packets=0 bytes=0 mark=0 zone=0 use=2
			testName:                "all private address",
			originConnectionSrcAddr: "192.168.0.249",
			originConnectionDstAddr: "192.168.0.1",
			originConnectionSrcPort: 65268,
			originConnectionDstPort: 5351,
			replyConnectionSrcAddr:  "192.168.0.1",
			replyConnectionDstAddr:  "192.168.0.249",
			replyConnectionSrcPort:  5351,
			replyConnectionDstPort:  65268,
			expected: result{
				srcAddr: "192.168.0.249",
				srcPort: 65268,
				dstAddr: "192.168.0.1",
				dstPort: 5351,
			},
		},
		{
			// ipv4     2 udp      17 53 src=112.93.49.21 dst=116.116.116.116 sport=53441 dport=53 packets=1 bytes=74 src=116.116.116.116 dst=112.93.49.21 sport=53 dport=53441 packets=1 bytes=114 mark=0 zone=0 use=2
			testName:                "all public address",
			originConnectionSrcAddr: "112.93.49.21",
			originConnectionDstAddr: "116.116.116.116",
			originConnectionSrcPort: 53441,
			originConnectionDstPort: 53,
			replyConnectionSrcAddr:  "116.116.116.116",
			replyConnectionDstAddr:  "112.93.49.21",
			replyConnectionSrcPort:  53,
			replyConnectionDstPort:  53441,
			expected: result{
				srcAddr: "112.93.49.21",
				srcPort: 53441,
				dstAddr: "116.116.116.116",
				dstPort: 53,
			},
		},
		{
			// ipv4     2 unknown  2 500 src=192.168.0.249 dst=224.0.0.252 packets=11880 bytes=380160 [UNREPLIED] src=224.0.0.252 dst=192.168.0.249 packets=0 bytes=0 mark=0 zone=0 use=2
			testName:                "unknown protocol no port",
			originConnectionSrcAddr: "192.168.0.249",
			originConnectionDstAddr: "224.0.0.252",
			originConnectionSrcPort: -1,
			originConnectionDstPort: -1,
			replyConnectionSrcAddr:  "224.0.0.252",
			replyConnectionDstAddr:  "192.168.0.249",
			replyConnectionSrcPort:  -1,
			replyConnectionDstPort:  -1,
			expected: result{
				srcAddr: "192.168.0.249",
				srcPort: -1,
				dstAddr: "224.0.0.252",
				dstPort: -1,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			srcAddr, srcPort, dstAddr, dstPort := selectPrivateAddress(
				testCase.originConnectionSrcAddr,
				testCase.replyConnectionSrcAddr,
				testCase.originConnectionSrcPort,
				testCase.replyConnectionSrcPort,
				testCase.originConnectionDstAddr,
				testCase.replyConnectionDstAddr,
				testCase.originConnectionDstPort,
				testCase.replyConnectionDstPort,
				privateCidr,
			)
			assert.Equal(t, testCase.expected.srcAddr, srcAddr)
			assert.Equal(t, testCase.expected.srcPort, srcPort)
			assert.Equal(t, testCase.expected.dstAddr, dstAddr)
			assert.Equal(t, testCase.expected.dstPort, dstPort)
		})
	}
}
