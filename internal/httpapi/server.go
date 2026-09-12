package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/lucas/maiutica/internal/material"
	"github.com/lucas/maiutica/internal/rag"
	"github.com/lucas/maiutica/internal/voice"
)

type Server struct { Materials material.Service; RAG *rag.Meili; Voice voice.Service }
func (s Server) Handler() http.Handler {
	mux:=http.NewServeMux(); mux.HandleFunc("GET /health",func(w http.ResponseWriter,_ *http.Request){write(w,200,map[string]string{"status":"ok"})})
	mux.HandleFunc("POST /v1/materials",s.ingest); mux.HandleFunc("POST /v1/rag/search",s.search); mux.HandleFunc("POST /v1/voice/start",s.voiceStart); mux.HandleFunc("POST /v1/voice/turn",s.voiceTurn)
	return cors(logRequests(mux))
}
func (s Server) voiceStart(w http.ResponseWriter,r *http.Request){turn,err:=s.Voice.Start(r.Context());if err!=nil{http.Error(w,err.Error(),502);return};write(w,200,turn)}
func (s Server) voiceTurn(w http.ResponseWriter,r *http.Request){
	if err:=r.ParseMultipartForm(16<<20);err!=nil{http.Error(w,err.Error(),400);return}
	inst,class,subject:=r.FormValue("institution_id"),r.FormValue("class_id"),r.FormValue("subject");if inst==""||class==""{http.Error(w,"institution_id and class_id are required",400);return}
	f,h,err:=r.FormFile("audio");if err!=nil{http.Error(w,"audio is required",400);return};defer f.Close();audio,err:=io.ReadAll(io.LimitReader(f,16<<20));if err!=nil{http.Error(w,err.Error(),400);return}
	var history []voice.Message
	if raw:=r.FormValue("history");raw!=""{if err:=json.Unmarshal([]byte(raw),&history);err!=nil{http.Error(w,"history must be valid JSON",400);return}}
	turn,err:=s.Voice.Turn(r.Context(),audio,h.Filename,rag.Scope{InstitutionID:inst,ClassID:class,Subject:subject},history);if err!=nil{http.Error(w,err.Error(),502);return};write(w,200,turn)
}
func (s Server) ingest(w http.ResponseWriter,r *http.Request) {
	if err:=r.ParseMultipartForm(32<<20);err!=nil{http.Error(w,err.Error(),400);return}
	inst, class, subject, title:=r.FormValue("institution_id"),r.FormValue("class_id"),r.FormValue("subject"),r.FormValue("title")
	if inst==""||class==""||subject==""||title==""{http.Error(w,"institution_id, class_id, subject and title are required",400);return}
	f,h,err:=r.FormFile("file");if err!=nil{http.Error(w,"file is required",400);return};defer f.Close()
	tmp,err:=os.CreateTemp("","maiutica-*"+filepath.Ext(h.Filename));if err!=nil{http.Error(w,err.Error(),500);return};path:=tmp.Name();defer os.Remove(path)
	if _,err=io.Copy(tmp,io.LimitReader(f,32<<20));err!=nil{tmp.Close();http.Error(w,err.Error(),500);return};tmp.Close()
	text,err:=material.Extract(path,h.Filename);if err!=nil{http.Error(w,err.Error(),422);return}
	docID:=fmt.Sprintf("doc-%d",time.Now().UnixNano());count,err:=s.Materials.Ingest(r.Context(),text,docID,inst,class,subject,title);if err!=nil{http.Error(w,err.Error(),502);return}
	write(w,201,map[string]any{"document_id":docID,"chunks":count})
}
func (s Server) search(w http.ResponseWriter,r *http.Request) {
	var q struct{InstitutionID string `json:"institution_id"`;ClassID string `json:"class_id"`;Subject string `json:"subject"`;Query string `json:"query"`}
	if json.NewDecoder(io.LimitReader(r.Body,1<<20)).Decode(&q)!=nil||q.InstitutionID==""||q.ClassID==""||q.Query==""{http.Error(w,"invalid request",400);return}
	res,err:=s.RAG.Search(r.Context(),q.Query,rag.Scope{InstitutionID:q.InstitutionID,ClassID:q.ClassID,Subject:q.Subject});if err!=nil{http.Error(w,err.Error(),502);return};write(w,200,res)
}
func write(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
func logRequests(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){next.ServeHTTP(w,r)})}
func cors(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.Header().Set("Access-Control-Allow-Origin","*");w.Header().Set("Access-Control-Allow-Headers","Content-Type");w.Header().Set("Access-Control-Allow-Methods","GET,POST,OPTIONS");if r.Method==http.MethodOptions{w.WriteHeader(http.StatusNoContent);return};next.ServeHTTP(w,r)})}
