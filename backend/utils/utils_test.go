package utils

import (
	"openwrt-diskio-api/backend/model"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertBytes(t *testing.T) {
	testCases := []struct {
		testName  string
		input     float64
		inputInit string
		expected1 float64
		expected2 string
	}{
		{"100 B/S -> 1 00B/S", float64(100), model.BSecond, float64(100), model.BSecond},
		{"1000 B/S -> 1 KB/S", float64(1000), model.BSecond, float64(1), model.KbSecond},
		{"-> 1 MB/S", float64(1000 * 1000), model.BSecond, float64(1), model.MbSecond},
		{"-> 1 GB/S", float64(1000 * 1000 * 1000), model.BSecond, float64(1), model.GbSecond},
		{"-> 1 TB/S", float64(1000 * 1000 * 1000 * 1000), model.BSecond, float64(1), model.TbSecond},
		{"-> 1 PB/S", float64(1000 * 1000 * 1000 * 1000 * 1000), model.BSecond, float64(1), model.PbSecond},
		{"-> 1000 PB/S", float64(1000 * 1000 * 1000 * 1000 * 1000 * 1000), model.BSecond, float64(1000), model.PbSecond},
		{"error input -1", float64(-1), model.BSecond, float64(-1), model.BSecond},
		{"error input 0", float64(0), model.BSecond, float64(0), model.BSecond},
		{"error input unit", float64(1000 * 1000), "K", float64(1000 * 1000), "K"},
	}

	for _, cases := range testCases {
		t.Run(cases.testName, func(t *testing.T) {
			bytes, unit := ConvertBytes(cases.input, cases.inputInit)
			assert.Equal(t, cases.expected1, bytes)
			assert.Equal(t, cases.expected2, unit)
		})
	}
}
func TestTrimBytesUnit(t *testing.T) {
	testCases := []struct {
		testName  string
		input     string
		expected1 string
	}{
		{model.BSecond, model.BSecond, model.BSecond},
		{" kb/s -> KB/S", " kb/s ", model.KbSecond},
	}

	for _, cases := range testCases {
		t.Run(cases.testName, func(t *testing.T) {
			unit := TrimBytesUnit(cases.input)
			assert.Equal(t, cases.expected1, unit)
		})
	}
}

func TestFindIndex(t *testing.T) {
	testCases := []struct {
		testName  string
		input1    []string
		input2    string
		expected1 int
	}{
		{"index success", []string{"1", "2", "3", "4"}, "3", 2},
		{"index failed", []string{"1", "2", "3", "4"}, "5", -1},
		{"nil list", nil, "5", -1},
		{"empty list", []string{}, "5", -1},
		{"empty string", []string{"1", "2", "3", "4"}, "5", -1},
	}
	for _, cases := range testCases {
		t.Run(cases.testName, func(t *testing.T) {
			index := FindIndex(cases.input1, cases.input2)
			assert.Equal(t, cases.expected1, index)
		})
	}
}
func TestTrimSubnetMask(t *testing.T) {
	testCases := []struct {
		testName  string
		input1    string
		expected1 string
	}{
		{"have slash", "192.168.0.0/24", "192.168.0.0"},
		{"no slash", "192.168.0.0", "192.168.0.0"},
	}
	for _, cases := range testCases {
		t.Run(cases.testName, func(t *testing.T) {
			actual := TrimSubnetMask(cases.input1)
			assert.Equal(t, cases.expected1, actual)
		})
	}
}
func TestCalculateRate(t *testing.T) {
	testCases := []struct {
		testName  string
		input1    float64
		input2    float64
		input3    uint
		expected1 float64
	}{
		{"positive rate +100%", float64(200), float64(100), uint(1), float64(100)},
		{"negative rate -100%", float64(100), float64(200), uint(1), float64(-100)},
		{"positive rate with interval +25%", float64(200), float64(100), uint(4), float64(25)},
		{"negative rate with interval -25%", float64(100), float64(200), uint(4), float64(-25)},
		{"zero interval", float64(200), float64(100), uint(0), float64(-1)},
	}
	for _, cases := range testCases {
		t.Run(cases.testName, func(t *testing.T) {
			rate := CalculateRate(cases.input1, cases.input2, cases.input3)
			assert.Equal(t, cases.expected1, rate)
		})
	}
}

func TestSumUint64(t *testing.T) {
	testCases := []struct {
		testName  string
		input1    []string
		expected1 uint64
		expected2 error
	}{
		{"sum success", []string{"1", "2", "3", "4"}, uint64(10), nil},
		{"sum negative result", []string{"1", "2", "-3", "-4"}, uint64(0), &strconv.NumError{}},
		{"empty list", []string{}, uint64(0), nil},
		{"nil list", nil, uint64(0), nil},
	}
	for _, cases := range testCases {
		t.Run(cases.testName, func(t *testing.T) {
			sum, err := SumUint64(cases.input1)
			assert.Equal(t, cases.expected1, sum)
			assert.IsType(t, cases.expected2, err)
		})
	}
}

func TestCalculateCpuUsage(t *testing.T) {
	testCases := []struct {
		testName      string
		nowCpuCycles  uint64
		lastCpuCycles uint64
		nowCpuIdle    uint64
		lastCpuIdle   uint64
		expected1     float64
	}{
		{"ok 1", uint64(296855682), uint64(296855042), uint64(292072657), uint64(292072026), float64(1.4062499999999978)},
		{"ok 2", uint64(296990965), uint64(296990260), uint64(292206508), uint64(292205811), float64(1.134751773049647)},
		{"totalDelta == 0", uint64(1000), uint64(1000), uint64(0), uint64(0), float64(0)},
		{"totalDelta < 0", uint64(1000), uint64(1001), uint64(0), uint64(0), float64(0)},
		{"idleDelta == 0", uint64(1000), uint64(999), uint64(1000), uint64(1000), float64(0)},
		{"idleDelta < 0", uint64(1000), uint64(999), uint64(1000), uint64(1001), float64(0)},
		{"idleDelta > totalDelta ", uint64(1000), uint64(999), uint64(10000), uint64(1000), float64(0)},
	}
	for _, cases := range testCases {
		t.Run(cases.testName, func(t *testing.T) {
			usage := CalculateCpuUsage(cases.nowCpuCycles, cases.lastCpuCycles, cases.nowCpuIdle, cases.lastCpuIdle)
			assert.Equal(t, cases.expected1, usage)
		})
	}
}

func TestRandHex(t *testing.T) {
	testCases := []struct {
		testName       string
		input          int
		expectedLength int
	}{
		{"odd number input", 32, 32},
		{"even number input", 15, 15},
		{"length == 0", 0, 0},
		{"length < ", -1, 0},
	}
	for _, cases := range testCases {
		t.Run(cases.testName, func(t *testing.T) {
			actual := RandHex(cases.input)
			assert.Equal(t, cases.expectedLength, len(actual), "output : %s", actual)
		})
	}
}
