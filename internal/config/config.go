package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr string
	MeiliURL, MeiliKey, MeiliIndex, MeiliEmbedder string
	OllamaURL, ChatModel, EmbedModel string
	STTURL, STTModel, TTSURL, TTSModel, TTSVoice string
	EmbeddingDimensions int
	SemanticRatio float64
	RAGLimit int
}

func Load() (Config,error) {
	c:=Config{
		HTTPAddr:env("HTTP_ADDR",":8080"),MeiliURL:env("MEILI_URL","http://localhost:7700"),MeiliKey:os.Getenv("MEILI_MASTER_KEY"),MeiliIndex:env("MEILI_INDEX","school_chunks"),MeiliEmbedder:env("MEILI_EMBEDDER","local"),
		OllamaURL:env("OLLAMA_URL","http://localhost:11434"),ChatModel:env("OLLAMA_CHAT_MODEL","qwen3:4b"),EmbedModel:env("OLLAMA_EMBED_MODEL","nomic-embed-text"),
		STTURL:env("STT_URL","http://localhost:8000/v1/audio/transcriptions"),STTModel:env("STT_MODEL","Systran/faster-whisper-small"),TTSURL:env("TTS_URL","http://localhost:8880/v1/audio/speech"),TTSModel:env("TTS_MODEL","kokoro"),TTSVoice:env("TTS_VOICE","pf_dora"),
		EmbeddingDimensions:envInt("EMBEDDING_DIMENSIONS",768),SemanticRatio:envFloat("RAG_SEMANTIC_RATIO",.65),RAGLimit:envInt("RAG_LIMIT",6),
	}
	if c.MeiliKey=="" { return Config{},fmt.Errorf("MEILI_MASTER_KEY is required") }
	return c,nil
}
func env(k,f string)string{if v:=os.Getenv(k);v!=""{return v};return f}
func envFloat(k string,f float64)float64{v,e:=strconv.ParseFloat(os.Getenv(k),64);if e!=nil{return f};return v}
func envInt(k string,f int)int{v,e:=strconv.Atoi(os.Getenv(k));if e!=nil{return f};return v}
