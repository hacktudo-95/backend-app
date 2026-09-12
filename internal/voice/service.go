package voice

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/lucas/maiutica/internal/localai"
	"github.com/lucas/maiutica/internal/rag"
)

type Service struct {
	AI  *localai.Client
	RAG *rag.Meili
}
type Turn struct {
	Transcript  string   `json:"transcript"`
	Answer      string   `json:"answer"`
	AudioBase64 string   `json:"audio_base64"`
	AudioMIME   string   `json:"audio_mime"`
	Sources     []Source `json:"sources"`
}
type Source struct {
	DocumentID string `json:"document_id"`
	Title      string `json:"title"`
	Subject    string `json:"subject"`
	Position   int    `json:"position"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s Service) Start(ctx context.Context) (Turn, error) {
	const greeting = "Olá! Eu sou a Dora, sua professora. Na turma de biologia de hoje vamos conversar sobre genética e evolução. Para começar: o que você acha que faz filhos se parecerem com seus pais, mas não serem exatamente iguais a eles?"
	return s.speak(ctx, "", greeting, nil)
}

func (s Service) Turn(ctx context.Context, audio []byte, filename string, scope rag.Scope, history []Message) (Turn, error) {
	transcript, err := s.AI.Transcribe(ctx, audio, filename)
	if err != nil {
		return Turn{}, err
	}
	if transcript == "" {
		return Turn{}, fmt.Errorf("não foi possível identificar a fala")
	}
	history = sanitizeHistory(history, 6)
	searchQuery := transcript
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			searchQuery = history[i].Content + "\n" + transcript
			break
		}
	}
	found, err := s.RAG.Search(ctx, searchQuery, scope)
	if err != nil {
		return Turn{}, err
	}
	if len(found.Chunks) == 0 {
		return s.speak(ctx, transcript, "Isso não consta no material disponibilizado pela escola. Você pode perguntar ao seu professor ou tentar outra questão.", nil)
	}
	chunks := found.Chunks
	if len(chunks) > 3 {
		chunks = chunks[:3]
	}
	var contextBuilder strings.Builder
	sources := make([]Source, 0, len(chunks))
	for i, c := range chunks {
		fmt.Fprintf(&contextBuilder, "[%d] %s - %s\n%s\n\n", i+1, c.Title, c.Subject, c.Content)
		sources = append(sources, Source{c.DocumentID, c.Title, c.Subject, c.Position})
	}
	system := `Você é Dora, professora de Biologia para estudantes de 12 a 14 anos.

Responda somente com base no CONTEXTO.

Sua resposta será falada em uma chamada. Sem títulos, listas, markdown ou explicações sobre seu raciocínio.

Use a maiêutica: considere o que o aluno disse, ofereça uma pista curta e termine com exatamente uma pergunta que o ajude a avançar.

Comece diretamente pelo conteúdo. Não comece com interjeições, elogios ou confirmações genéricas como "Ah", "Entendi", "Muito bem", "Ótimo", "Boa pergunta", "Você está certo" ou expressões semelhantes.

Varie a construção das respostas e não repita a abertura utilizada anteriormente. Corrija erros sem constranger. Não entregue toda a explicação imediatamente.

Se o contexto não sustentar a resposta, diga isso brevemente. Não invente fatos nem utilize conhecimento externo.`
	var dialogue strings.Builder
	for _, m := range history {
		fmt.Fprintf(&dialogue, "%s: %s\n", m.Role, m.Content)
	}
	answer, err := s.AI.Chat(ctx, system, "CONTEXTO:\n"+contextBuilder.String()+"\nCONVERSA RECENTE:\n"+dialogue.String()+"\nFALA ATUAL DO ALUNO:\n"+transcript)
	if err != nil {
		return Turn{}, err
	}
	return s.speak(ctx, transcript, answer, sources)
}
func (s Service) speak(ctx context.Context, transcript, answer string, sources []Source) (Turn, error) {
	audio, mime, err := s.AI.Speak(ctx, answer)
	if err != nil {
		return Turn{}, err
	}
	return Turn{transcript, answer, base64.StdEncoding.EncodeToString(audio), mime, sources}, nil
}

func sanitizeHistory(history []Message, limit int) []Message {
	if len(history) > limit {
		history = history[len(history)-limit:]
	}
	out := make([]Message, 0, len(history))
	for _, m := range history {
		m.Role = strings.TrimSpace(m.Role)
		m.Content = strings.TrimSpace(m.Content)
		if (m.Role == "user" || m.Role == "assistant") && m.Content != "" {
			if len([]rune(m.Content)) > 1200 {
				m.Content = string([]rune(m.Content)[:1200])
			}
			out = append(out, m)
		}
	}
	return out
}
