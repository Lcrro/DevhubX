package logfmt

import "testing"

func TestCleanANSIAndCLIXML(t *testing.T) {
	got := Clean("\x1b[31m失败\x1b[0m\r\n")
	if got != "失败\n" {
		t.Fatalf("ansi cleanup: %q", got)
	}
	clixml := "#< CLIXML\r\n<Objs><Obj><S N=\"Message\">中文错误</S></Obj></Objs>"
	if got := Clean(clixml); got != "中文错误\n" {
		t.Fatalf("clixml cleanup: %q", got)
	}
}

func TestBytesUTF16(t *testing.T) {
	got := Bytes([]byte{0xff, 0xfe, 0x2d, 0x4e, 0x87, 0x65, 0x0d, 0x00, 0x0a, 0x00})
	if got != "中文\n" {
		t.Fatalf("utf16 decode: %q", got)
	}
}
