package localai

import "testing"

func TestCleanModelAnswerRemovesThinking(t *testing.T) {
	input := "<think>Preciso analisar genes e DNA em detalhes.</think> Boa hipótese! Que estrutura do DNA carrega uma característica?"
	want := "Boa hipótese! Que estrutura do DNA carrega uma característica?"
	if got := CleanModelAnswer(input); got != want {
		t.Fatalf("CleanModelAnswer() = %q, want %q", got, want)
	}
}

func TestCleanModelAnswerKeepsOnlyFinalAnswerMarker(t *testing.T) {
	input := "Raciocínio que não deve aparecer.\nResposta final: Você está perto. Qual característica veio de cada responsável?"
	want := "Você está perto. Qual característica veio de cada responsável?"
	if got := CleanModelAnswer(input); got != want {
		t.Fatalf("CleanModelAnswer() = %q, want %q", got, want)
	}
}
