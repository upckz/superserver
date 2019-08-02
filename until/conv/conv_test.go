package conv

import (
	"testing"
)

func Test_conv(t *testing.T) {
	str := Float64ToString(2.3)
	t.Log(str)
	str = Float32ToString(2.3)
	t.Log(str)
	fl64 := StringToFloat64(str)
	t.Log(fl64)
	fl32 := StringToFloat32(str)
	t.Log(fl32)
}
