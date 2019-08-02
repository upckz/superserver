package conv

import (
	"fmt"
	"strconv"
)

func IntToString(a int) string {
	return strconv.Itoa(a)
}
func Int64ToInt32(a int64) int32 {
	return int32(a)
}

func Int32ToInt64(a int32) int64 {
	return int64(a)
}

func Int64ToString(a int64) string {
	return strconv.FormatInt(a, 10)
}

func Int32ToString(a int32) string {
	return strconv.Itoa(int(a))
}

func Uint64ToString(a uint64) string {
	return strconv.FormatUint(a, 10)
}

func Uint32ToString(a uint32) string {
	return strconv.FormatUint(uint64(a), 10)
}

func StringToInt(a string) int {
	if s, err := strconv.Atoi(a); err == nil {
		return s
	}
	return 0
}

func StringToInt64(a string) int64 {
	if s, err := strconv.Atoi(a); err == nil {
		return int64(s)
	}
	return 0
}

func StringToInt32(a string) int32 {
	if s, err := strconv.Atoi(a); err == nil {
		return int32(s)
	}
	return 0
}

func StringToUInt64(a string) uint64 {
	if s, err := strconv.ParseUint(a, 10, 64); err == nil {
		return s
	}
	return 0
}

func StringToUInt32(a string) uint32 {
	if s, err := strconv.ParseUint(a, 10, 32); err == nil {
		return uint32(s)
	}
	return 0
}

func Uint32ToHexString(a uint32) string {
	return fmt.Sprintf("0x%04x", a)
}

func Float64ToString(a float64, bitSize int) string {
	return strconv.FormatFloat(a, 'f', bitSize, 64)
}

func Float32ToString(a float32, bitSize int) string {
	return strconv.FormatFloat(float64(a), 'f', bitSize, 32)
}

func StringToFloat64(s string) float64 {
	if s, err := strconv.ParseFloat(s, 64); err == nil {
		return s
	}
	return 0
}

func StringToFloat32(s string) float32 {
	if s, err := strconv.ParseFloat(s, 32); err == nil {
		return float32(s)
	}
	return 0
}
