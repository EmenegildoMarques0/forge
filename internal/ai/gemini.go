package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"forge/internal/config" // Importação do gerenciador de configurações nativo do Forge
	"io"
	"net/http"
	"os"
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
		return "", fmt.Errorf("API respondeu com status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta da IA: %v", err)
	}

	// Se o JSON contiver uma estrutura de erro da API
	if geminiResp.Error != nil {
		return "", fmt.Errorf("erro na API do Gemini: %s", geminiResp.Error.Message)
	}

	// Retorna a mensagem gerada com sucesso pela IA
	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("IA respondeu com sucesso, mas não retornou nenhuma sugestão de texto válida")
}
