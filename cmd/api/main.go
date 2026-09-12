package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/lucas/maiutica/internal/config"
	"github.com/lucas/maiutica/internal/httpapi"
	"github.com/lucas/maiutica/internal/localai"
	"github.com/lucas/maiutica/internal/material"
	"github.com/lucas/maiutica/internal/rag"
	"github.com/lucas/maiutica/internal/voice"
)

func main(){
	cfg,err:=config.Load();if err!=nil{log.Fatal(err)}
	ai:=localai.New(cfg.OllamaURL,cfg.ChatModel,cfg.EmbedModel,cfg.STTURL,cfg.STTModel,cfg.TTSURL,cfg.TTSModel,cfg.TTSVoice)
	index:=rag.NewMeili(cfg.MeiliURL,cfg.MeiliKey,cfg.MeiliIndex,cfg.MeiliEmbedder,cfg.EmbeddingDimensions,cfg.SemanticRatio,cfg.RAGLimit,ai)
	ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second);defer cancel();if err=index.Setup(ctx);err!=nil{log.Fatal(err)}
	api:=httpapi.Server{Materials:material.Service{Index:index},RAG:index,Voice:voice.Service{AI:ai,RAG:index}}
	srv:=&http.Server{Addr:cfg.HTTPAddr,Handler:api.Handler(),ReadHeaderTimeout:5*time.Second}
	log.Printf("Maiêutica listening on %s",cfg.HTTPAddr);log.Fatal(srv.ListenAndServe())
}
