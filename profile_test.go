package bart

import (
	"bytes"
	"fmt"
	"testing"
)

var b1 = mpp("2001:db8::1/128").Addr().AsSlice()
var b2 = b1

// var b2 = mpp("::1/128").Addr().AsSlice()

var sinkBool bool

func BenchmarkBytesEqual(b *testing.B) {
	for i := range 16 {
		b.Run(fmt.Sprintf("%7d", i), func(b *testing.B) {
			for range b.N {
				sinkBool = bytes.Equal(b1[:i], b2[:i])
			}
		})
	}
}

func BenchmarkBytesHasPrefix(b *testing.B) {
	for i := range 16 {
		b.Run(fmt.Sprintf("%7d", i), func(b *testing.B) {
			for range b.N {
				sinkBool = bytes.HasPrefix(b1, b2[:i])
			}
		})
	}
}

func BenchmarkBytesCutPrefix(b *testing.B) {
	for i := range 16 {
		b.Run(fmt.Sprintf("%7d", i), func(b *testing.B) {
			for range b.N {
				_, sinkBool = bytes.CutPrefix(b1, b2[:i])
			}
		})
	}
}
