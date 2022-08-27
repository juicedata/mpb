package cwriter

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"
)

var (
	out   = ioutil.Discard
	lines = 99
)

func BenchmarkWithFprintf(b *testing.B) {
	verb := fmt.Sprintf("%s%%d%s", escOpen, cuuAndEd)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fmt.Fprintf(out, verb, lines)
	}
}

func BenchmarkWithJoin(b *testing.B) {
	bCuuAndEd := [][]byte{[]byte(escOpen), []byte(cuuAndEd)}
	for i := 0; i < b.N; i++ {
		_, _ = out.Write(bytes.Join(bCuuAndEd, []byte(strconv.Itoa(lines))))
	}
}

func BenchmarkWithAppend(b *testing.B) {
	escOpen := []byte(escOpen)
	cuuAndEd := []byte(cuuAndEd)
	for i := 0; i < b.N; i++ {
		_, _ = out.Write(append(strconv.AppendInt(escOpen, int64(lines), 10), cuuAndEd...))
	}
}

func BenchmarkWithCurrentImpl(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ansiCuuAndEd(out, lines)
	}
}
