# Maiêutica - backend Go 100% local

Pipeline para Ubuntu WSL2: áudio do aluno -> faster-whisper -> RAG híbrido no
Meilisearch -> Qwen no Ollama -> Kokoro -> áudio do professor.

## Componentes

- Go: orquestra o turno e aplica isolamento por escola/turma.
- Ollama `qwen3:4b`: professor socrático.
- Ollama `nomic-embed-text`: embeddings locais (768 dimensões).
- Meilisearch: busca lexical + vetorial.
- faster-whisper-server: endpoint OpenAI-compatible de transcrição.
- Kokoro-FastAPI: endpoint OpenAI-compatible de voz.

## Preparação

```bash
cp .env.example .env
docker compose up -d
ollama pull qwen3:4b
ollama pull nomic-embed-text
```

O Ollama continua rodando nativamente no WSL e usando a GPU; somente o
Meilisearch sobe pelo Compose. Suba também no WSL um servidor faster-whisper na porta 8000 e Kokoro-FastAPI na porta
8880. Os endereços e modelos podem ser trocados no `.env` sem recompilar.

```bash
set -a; source .env; set +a
go mod tidy
go test ./...
go run ./cmd/api
```

## Ingestão

```bash
curl -F institution_id=school-1 -F class_id=7a -F subject=historia \
  -F title='Brasil Colônia' -F file=@apostila.pdf \
  http://localhost:8080/v1/materials
```

## Turno de voz

O áudio deve ser WAV. O retorno contém transcrição, resposta, fontes e WAV em
Base64. Esse será o contrato usado pelo React Native.

```bash
curl -F institution_id=school-1 -F class_id=7a -F subject=historia \
  -F audio=@pergunta.wav http://localhost:8080/v1/voice/turn
```

O aplicativo mobile inicia a aula por este endpoint:

```bash
curl -X POST http://localhost:8080/v1/voice/start
```

Nos turnos seguintes, o campo multipart opcional `history` recebe um JSON com
as seis mensagens mais recentes. Isso permite interpretar respostas curtas e
continuar a condução maiêutica sem transformar o Meilisearch em banco de sessão.

As respostas faladas são limitadas a cerca de 55 palavras e 120 tokens de
geração. O backend usa no máximo os três chunks mais relevantes no prompt,
mantém os modelos do Ollama aquecidos por 30 minutos e remove blocos
`<think>...</think>` antes de enviar texto ao Kokoro ou ao aplicativo.

Para sua GTX 1660 Super de 6 GB, deixe o Qwen na GPU e configure inicialmente
Whisper/Kokoro para CPU. Depois teste mover o Whisper para CUDA; manter os três
modelos simultaneamente na GPU pode causar falta de VRAM.

## Limites da V1

- Conversa em turnos rápidos, não áudio full-duplex.
- O React Native deverá detectar fim da fala antes de enviar o WAV.
- PDF escaneado ainda exige OCR.
- Em produção, instituição e turma devem vir do JWT, não do formulário.
