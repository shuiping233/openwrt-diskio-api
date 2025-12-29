package utils

import (
	"strings"
)

var (
	UnitList = []string{"B/S", "KB/S", "MB/S", "GB/S", "TB/S", "PB/S"}
)

// If "unit" is unknown unit , return unchanged.
func ConvertBytes(bytes float64, unit string) (float64, string) {
	unit = TrimBytesUnit(unit)
	unitListIndex := findIndex(UnitList, unit)
	if unitListIndex < 0 {
		return bytes, unit
	}
	newBytes := bytes / 1000
	if newBytes < 1 {
		return bytes, unit
	}
	newUnitListIndex := unitListIndex + 1
	if newUnitListIndex >= len(UnitList) {
		return bytes, unit
	}
	return ConvertBytes(newBytes, UnitList[newUnitListIndex])

}

func TrimBytesUnit(unit string) string {
	return strings.ToUpper(strings.TrimSpace(unit))
}

// if not found , return -1 , python list index be like
func findIndex(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}
