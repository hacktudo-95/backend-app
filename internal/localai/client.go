package localai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	OllamaURL, ChatModel, EmbedModel, STTURL, STTModel, TTSURL, TTSModel, TTSVoice string
	HTTP                                                                           *http.Client
}

func New(ollamaURL, chatModel, embedModel, sttURL, sttModel, ttsURL, ttsModel, ttsVoice string) *Client {
	return &Client{strings.TrimRight(ollamaURL, "/"), chatModel, embedModel, sttURL, sttModel, ttsURL, ttsModel, ttsVoice, &http.Client{Timeout: 2 * time.Minute}}
}
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	var out struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := c.json(ctx, c.OllamaURL+"/api/embed", map[string]any{"model": c.EmbedModel, "input": text, "keep_alive": "30m", "think": false}, &out); err != nil {
		return nil, err
	}
	if len(out.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama returned no embedding")
	}
	return out.Embeddings[0], nil
}
func (c *Client) Chat(ctx context.Context, system, user string) (string, error) {
	var out struct {
		Message struct {
			Content  string `json:"content"`
			Thinking string `json:"thinking"`
		} `json:"message"`
	}
	body := map[string]any{"model": c.ChatModel, "stream": false, "think": false, "keep_alive": "30m", "messages": []map[string]string{{"role": "system", "content": system}, {"role": "user", "content": user}}, "options": map[string]any{"temperature": 0, "num_predict": -1, "repeat_penalty": 1.08}}
	if err := c.json(ctx, c.OllamaURL+"/api/chat", body, &out); err != nil {
		return "", err
	}
	answer := CleanModelAnswer(out.Message.Content)
	if answer == "" {
		return "", fmt.Errorf("o modelo não retornou uma resposta final")
	}
	return answer, nil
}
func (c *Client) Transcribe(ctx context.Context, audio []byte, filename string) (string, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(filename)))
	h.Set("Content-Type", "audio/wav")
	p, _ := mw.CreatePart(h)
	_, _ = p.Write(audio)
	_ = mw.WriteField("model", c.STTModel)
	_ = mw.WriteField("language", "pt")
	_ = mw.Close()
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, c.STTURL, &body)
	if e != nil {
		return "", e
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, e := c.HTTP.Do(req)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("STT %s: %s", resp.Status, string(data))
	}
	var out struct {
		Text string `json:"text"`
	}
	if e = json.Unmarshal(data, &out); e != nil {
		return "", e
	}
	return strings.TrimSpace(out.Text), nil
}
func (c *Client) Speak(ctx context.Context, text string) ([]byte, string, error) {
	b, _ := json.Marshal(map[string]any{"model": c.TTSModel, "voice": c.TTSVoice, "input": text, "response_format": "wav", "speed": 1})
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, c.TTSURL, bytes.NewReader(b))
	if e != nil {
		return nil, "", e
	}
	req.Header.Set("Content-Type", "application/json")
	resp, e := c.HTTP.Do(req)
	if e != nil {
		return nil, "", e
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode/100 != 2 {
		return nil, "", fmt.Errorf("TTS %s: %s", resp.Status, string(data))
	}
	return data, "audio/wav", nil
}
func (c *Client) json(ctx context.Context, url string, in, out any) error {
	b, _ := json.Marshal(in)
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	resp, e := c.HTTP.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("local AI %s: %s", resp.Status, string(data))
	}
	return json.Unmarshal(data, out)
}

var thinkBlock = regexp.MustCompile(`(?is)<think\b[^>]*>(.*?</think\s*>|.*)`)

// CleanModelAnswer is a final safety barrier: internal reasoning is never sent
// to the student or to text-to-speech, even if a model ignores think=false.
func CleanModelAnswer(raw string) string {
	answer := strings.TrimSpace(thinkBlock.ReplaceAllString(raw, ""))

	// Caso reste algum resíduo por conta de má formatação do modelo
	lower := strings.ToLower(answer)
	if end := strings.LastIndex(lower, "</think>"); end >= 0 {
		answer = strings.TrimSpace(answer[end+len("</think>"):])
		lower = strings.ToLower(answer)
	}
	if start := strings.Index(lower, "<think"); start >= 0 {
		answer = strings.TrimSpace(answer[:start])
	}

	for _, marker := range []string{"final answer:", "resposta final:"} {
		if index := strings.LastIndex(strings.ToLower(answer), marker); index >= 0 {
			answer = strings.TrimSpace(answer[index+len(marker):])
		}
	}
	return strings.TrimSpace(answer)
}
