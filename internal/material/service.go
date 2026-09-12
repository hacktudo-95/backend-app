package material

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lucas/maiutica/internal/rag"
)

type Service struct { Index *rag.Meili }

func Extract(path, filename string) (string,error) {
	ext:=strings.ToLower(filepath.Ext(filename))
	if ext==".txt" || ext==".md" { b,e:=os.ReadFile(path); return string(b),e }
	if ext!=".pdf" { return "",fmt.Errorf("supported formats: PDF, TXT and MD") }
	out,err:=exec.Command("pdftotext","-layout",path,"-").Output()
	if err!=nil { return "",fmt.Errorf("extract PDF (install poppler-utils): %w",err) }
	return string(out),nil
}

func (s Service) Ingest(ctx context.Context, text, documentID, institutionID, classID, subject, title string) (int,error) {
	parts:=rag.ChunkText(text,1200,180); chunks:=make([]rag.Chunk,0,len(parts))
	for i,p:=range parts { chunks=append(chunks,rag.Chunk{ID:fmt.Sprintf("%s-%04d",documentID,i),DocumentID:documentID,InstitutionID:institutionID,ClassID:classID,Subject:subject,Title:title,Position:i,Content:p}) }
	if len(chunks)==0{return 0,fmt.Errorf("document has no extractable text")}
	return len(chunks),s.Index.Add(ctx,chunks)
}
