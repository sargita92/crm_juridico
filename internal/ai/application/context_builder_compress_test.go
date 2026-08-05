package application

import "testing"

func TestCompressPrompt(t *testing.T) {
	in := "Você  é\ta   Dra.\n\n\n\nColete:   nome,   idade  .   \n   próxima linha  "
	want := "Você é a Dra.\n\nColete: nome, idade .\npróxima linha"
	if got := compressPrompt(in); got != want {
		t.Fatalf("compressPrompt()\n got: %q\nwant: %q", got, want)
	}
}
