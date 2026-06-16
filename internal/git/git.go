package git

import (
	"os/exec"
	"strings"
)

// CheckGHCLI verifica se o GitHub CLI (gh) está instalado
func CheckGHCLI() error {
	cmd := exec.Command("gh", "--version")
	return cmd.Run()
}

// CreatePullRequest cria uma pull request no repositório remoto usando o GitHub CLI
func CreatePullRequest(title, body string) error {
	args := []string{"pr", "create", "--title", title, "--body", body}
	cmd := exec.Command("gh", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// AddAll executa o comando 'git add .' para preparar todos os arquivos
func AddAll() error {
	cmd := exec.Command("git", "add", ".")
	return cmd.Run()
}

// GetDiff extrai as mudanças que estão no 'stage' (git add)
func GetDiff() (string, error) {
	// Comando: git diff --cached (mostra o que foi adicionado para commit)
	cmd := exec.Command("git", "diff", "--cached")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// GetStagedFiles lista apenas os nomes dos arquivos alterados de forma limpa
func GetStagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return []string{}, nil
	}

	files := strings.Split(output, "\n")
	return files, nil
}

func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}

// Push dispara as alterações para o repositório remoto (ex: GitHub)
func Push() error {
	cmd := exec.Command("git", "push")
	return cmd.Run()
}
