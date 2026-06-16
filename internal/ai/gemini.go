package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"forge/internal/config" // Importação do gerenciador de configurações nativo do Forge
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GeminiResponse mapeia a resposta de sucesso e possíveis erros da API
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"` // Captura a mensagem de erro direto do Google
}

// PRData representa os dados estruturados retornados pela IA para criação de PR
type PRData struct {
	BranchName string `json:"branch_name"`
	Title      string `json:"title"`
	Body       string `json:"body"`
}

func GenerateCommitMessage(diff string) (string, error) {
	// 1. Tenta carregar a chave salva localmente pelo menu de configurações
	cfg, _ := config.Load()
	apiKey := cfg.GeminiAPIKey

	// 2. Fallback: Se não estiver no arquivo JSON, tenta buscar do terminal
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	// 3. Se ambas as tentativas falharem, orienta o usuário a configurar
	if apiKey == "" {
		return "", fmt.Errorf("API Key não encontrada. Vá a Configurações no menu do Forge para salvar")
	}

	url := "https://generativelanguage.googleapis.com/v1/models/gemini-2.5-flash:generateContent?key=" + apiKey

	// Criando o prompt técnico para o padrão Conventional Commits
	prompt := "Atue como um desenvolvedor sênior. Analise o diff abaixo e retorne APENAS uma mensagem de commit no padrão Conventional Commits. Não use Markdown, não dê explicações. Apenas o texto da mensagem.\n\nDIFF:\n" + diff

	// Montando o payload JSON manualmente
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	const maxRetries = 10
	const retryInterval = 4 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Fazendo a requisição HTTP direta para o Google
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		// Se o status HTTP não for 200 OK, expõe o corpo do erro retornado
		if resp.StatusCode != http.StatusOK {
			errorMsg := string(body)
			// Verifica se o erro indica que o modelo está sobrecarregado/sob demanda
			if isOverloadedError(errorMsg) {
				fmt.Printf("\n⚠️  Modelo do Gemini está sobrecarregado (tentativa %d/%d). Aguardando %d segundos para tentar novamente...\n", attempt, maxRetries, int(retryInterval.Seconds()))
				time.Sleep(retryInterval)
				continue
			}
			return "", fmt.Errorf("API respondeu com status %d: %s", resp.StatusCode, errorMsg)
		}

		var geminiResp GeminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			return "", fmt.Errorf("erro ao decodificar resposta da IA: %v", err)
		}

		// Se o JSON contiver uma estrutura de erro da API
		if geminiResp.Error != nil {
			errorMsg := geminiResp.Error.Message
			// Verifica se o erro indica que o modelo está sobrecarregado/sob demanda
			if isOverloadedError(errorMsg) {
				fmt.Printf("\n⚠️  Modelo do Gemini está sobrecarregado (tentativa %d/%d). Aguardando %d segundos para tentar novamente...\n", attempt, maxRetries, int(retryInterval.Seconds()))
				time.Sleep(retryInterval)
				continue
			}
			return "", fmt.Errorf("erro na API do Gemini: %s", errorMsg)
		}

		// Retorna a mensagem gerada com sucesso pela IA
		if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
			return geminiResp.Candidates[0].Content.Parts[0].Text, nil
		}

		return "", fmt.Errorf("IA respondeu com sucesso, mas não retornou nenhuma sugestão de texto válida")
	}

	return "", fmt.Errorf("modelo do Gemini permaneceu sobrecarregado após %d tentativas. Tente novamente mais tarde", maxRetries)
}

// isOverloadedError verifica se a mensagem de erro indica que o modelo está sobrecarregado
func isOverloadedError(errorMsg string) bool {
	lowerMsg := strings.ToLower(errorMsg)
	overloadedKeywords := []string{
		"overloaded",
		"sobrecarregado",
		"resource exhausted",
		"rate limit",
		"too many requests",
		"service unavailable",
		"503",
		"model is overloaded",
		"currently overloaded",
		"please try again later",
		"tente novamente mais tarde",
	}

	for _, keyword := range overloadedKeywords {
		if strings.Contains(lowerMsg, keyword) {
			return true
		}
	}
	return false
}

// extractJSON remove blocos de código markdown e retorna apenas o JSON puro
func extractJSON(text string) string {
	// Remove blocos de código markdown (```json ... ``` ou ``` ... ```)
	start := strings.Index(text, "```")
	if start != -1 {
		// Avança para depois do ```
		after := text[start+3:]
		// Remove "json" se presente
		after = strings.TrimPrefix(after, "json")
		// Encontra o primeiro { no resto
		bracePos := strings.Index(after, "{")
		if bracePos != -1 {
			text = after[bracePos:]
		}
	}

	// Remove marcadores de fechamento de código
	if idx := strings.Index(text, "```"); idx != -1 {
		text = text[:idx]
	}

	text = strings.TrimSpace(text)

	// Encontra o primeiro { e o último } para extrair apenas o JSON
	firstBrace := strings.Index(text, "{")
	lastBrace := strings.LastIndex(text, "}")
	if firstBrace != -1 && lastBrace != -1 && lastBrace > firstBrace {
		return text[firstBrace : lastBrace+1]
	}

	return text
}

// GeneratePRData envia o diff para a IA e retorna dados estruturados para criação de PR
func GeneratePRData(diff string) (*PRData, error) {
	cfg, _ := config.Load()
	apiKey := cfg.GeminiAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API Key não encontrada. Vá a Configurações no menu do Forge para salvar")
	}

	url := "https://generativelanguage.googleapis.com/v1/models/gemini-2.5-flash:generateContent?key=" + apiKey

	prompt := `Atue como um desenvolvedor sênior. Analise o diff abaixo e retorne APENAS um JSON válido (sem markdown, sem explicações) com a seguinte estrutura:
{
  "branch_name": "nome-da-branch-no-padrao-kebab-case",
  "title": "Título curto e descritivo do Pull Request",
  "body": "Descrição detalhada em Markdown do que foi alterado e porquê"
}

Regras:
- O branch_name deve ser curto, descritivo e seguir o padrão kebab-case (ex: feat/setup-config, fix/login-error).
- O title deve ser direto e seguir o padrão Conventional Commits quando possível.
- O body deve explicar as alterações de forma clara para revisão.
- NÃO adicione texto fora do JSON.

DIFF:
` + diff

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	const maxRetries = 10
	const retryInterval = 4 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			errorMsg := string(body)
			if isOverloadedError(errorMsg) {
				fmt.Printf("\n⚠️  Modelo do Gemini está sobrecarregado (tentativa %d/%d). Aguardando %d segundos para tentar novamente...\n", attempt, maxRetries, int(retryInterval.Seconds()))
				time.Sleep(retryInterval)
				continue
			}
			return nil, fmt.Errorf("API respondeu com status %d: %s", resp.StatusCode, errorMsg)
		}

		var geminiResp GeminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			return nil, fmt.Errorf("erro ao decodificar resposta da IA: %v", err)
		}

		if geminiResp.Error != nil {
			errorMsg := geminiResp.Error.Message
			if isOverloadedError(errorMsg) {
				fmt.Printf("\n⚠️  Modelo do Gemini está sobrecarregado (tentativa %d/%d). Aguardando %d segundos para tentar novamente...\n", attempt, maxRetries, int(retryInterval.Seconds()))
				time.Sleep(retryInterval)
				continue
			}
			return nil, fmt.Errorf("erro na API do Gemini: %s", errorMsg)
		}

		if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
			text := geminiResp.Candidates[0].Content.Parts[0].Text
			jsonText := extractJSON(text)
			var prData PRData
			if err := json.Unmarshal([]byte(jsonText), &prData); err != nil {
				return nil, fmt.Errorf("erro ao decodificar JSON da IA: %v\nResposta: %s", err, jsonText)
			}
			return &prData, nil
		}

		return nil, fmt.Errorf("IA respondeu com sucesso, mas não retornou dados válidos para PR")
	}

	return nil, fmt.Errorf("modelo do Gemini permaneceu sobrecarregado após %d tentativas. Tente novamente mais tarde", maxRetries)
}
