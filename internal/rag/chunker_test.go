package rag

import "testing"

func TestChunkText(t *testing.T){
	got:=ChunkText("primeiro parágrafo\n\nsegundo parágrafo com mais texto",24,5)
	if len(got)<2{t.Fatalf("expected multiple chunks, got %d",len(got))}
	for _,c:=range got{if len([]rune(c))>26{t.Fatalf("chunk unexpectedly large: %q",c)}}
}
