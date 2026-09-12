package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Scope struct { InstitutionID, ClassID, Subject string }
type Chunk struct {
	ID string `json:"id"`; DocumentID string `json:"document_id"`; InstitutionID string `json:"institution_id"`
	ClassID string `json:"class_id"`; Subject string `json:"subject"`; Title string `json:"title"`
	Page int `json:"page"`; Position int `json:"position"`; Content string `json:"content"`
	Vectors map[string][]float32 `json:"_vectors,omitempty"`
}
type SearchResult struct { Chunks []Chunk `json:"chunks"` }

type Vectorizer interface { Embed(context.Context,string)([]float32,error) }
type Meili struct { base,key,index,embedder string; dimensions int; ratio float64; limit int; vectorizer Vectorizer; http *http.Client }
func NewMeili(base,key,index,embedder string,dimensions int,ratio float64,limit int,vectorizer Vectorizer)*Meili {
	return &Meili{strings.TrimRight(base,"/"),key,index,embedder,dimensions,ratio,limit,vectorizer,&http.Client{Timeout:30*time.Second}}
}

func (m *Meili) Setup(ctx context.Context) error {
	_ = m.do(ctx, http.MethodPost, "/indexes", map[string]any{"uid":m.index,"primaryKey":"id"}, nil)
	settings := map[string]any{
		"searchableAttributes": []string{"title","subject","content"},
		"filterableAttributes": []string{"institution_id","class_id","subject","document_id"},
		"embedders": map[string]any{m.embedder: map[string]any{"source":"userProvided","dimensions":m.dimensions}},
	}
	return m.do(ctx, http.MethodPatch, "/indexes/"+url.PathEscape(m.index)+"/settings", settings, nil)
}
func (m *Meili) Add(ctx context.Context,chunks []Chunk)error{
	for i:=range chunks{v,err:=m.vectorizer.Embed(ctx,chunks[i].Title+"\n"+chunks[i].Subject+"\n"+chunks[i].Content);if err!=nil{return err};chunks[i].Vectors=map[string][]float32{m.embedder:v}}
	return m.do(ctx,http.MethodPost,"/indexes/"+url.PathEscape(m.index)+"/documents",chunks,nil)
}
func (m *Meili) Search(ctx context.Context, q string, s Scope) (SearchResult, error) {
	filters := []string{"institution_id = "+quote(s.InstitutionID), "class_id = "+quote(s.ClassID)}
	if s.Subject != "" { filters = append(filters, "subject = "+quote(s.Subject)) }
	vector,err:=m.vectorizer.Embed(ctx,q);if err!=nil{return SearchResult{},err}
	body := map[string]any{"q":q,"vector":vector,"filter":strings.Join(filters," AND "),"limit":m.limit,"rankingScoreThreshold":0.5,"hybrid":map[string]any{"embedder":m.embedder,"semanticRatio":m.ratio}}
	var raw struct { Hits []Chunk `json:"hits"` }
	err = m.do(ctx,http.MethodPost,"/indexes/"+url.PathEscape(m.index)+"/search",body,&raw)
	return SearchResult{Chunks:raw.Hits}, err
}
func quote(v string) string { return `"`+strings.ReplaceAll(v,`"`,`\\"`)+`"` }
func (m *Meili) do(ctx context.Context, method,path string, body,out any) error {
	b,err:=json.Marshal(body); if err!=nil{return err}
	req,err:=http.NewRequestWithContext(ctx,method,m.base+path,bytes.NewReader(b)); if err!=nil{return err}
	req.Header.Set("Authorization","Bearer "+m.key); req.Header.Set("Content-Type","application/json")
	resp,err:=m.http.Do(req); if err!=nil{return err}; defer resp.Body.Close()
	data,_:=io.ReadAll(io.LimitReader(resp.Body,4<<20))
	if resp.StatusCode<200||resp.StatusCode>=300 { if resp.StatusCode==409 && path=="/indexes" { return nil }; return fmt.Errorf("meilisearch %s: %s",resp.Status,string(data)) }
	if out!=nil && len(data)>0 { return json.Unmarshal(data,out) }; return nil
}
