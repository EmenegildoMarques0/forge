package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckGHCLI verifica se o GitHub CLI (gh) está instalado
func CheckGHCLI() error {
	cmd := exec.Command("gh", "--version")
	return cmd.Run()
}

// GetCurrentBranch retorna o nome da branch Git atual
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CreateAndSwitchBranch cria uma nova branch e troca para ela
func CreateAndSwitchBranch(name string) error {
	cmd := exec.Command("git", "checkout", "-b", name)
	return cmd.Run()
}

// SwitchBranch troca para uma branch existente
func SwitchBranch(name string) error {
	cmd := exec.Command("git", "checkout", name)
	return cmd.Run()
}

// CreatePullRequest cria uma pull request no repositório remoto usando o GitHub CLI
func CreatePullRequest(title, body, head, base string) error {
	args := []string{"pr", "create", "--title", title, "--body", body, "--head", head, "--base", base}
	cmd := exec.Command("gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh pr create falhou: %v\nSaída: %s", err, string(out))
	}
	return nil
}

// AddAll executa o comando 'git add -A' para preparar todos os arquivos (inclui modificações, novos arquivos e deleções)
func AddAll() error {
	cmd := exec.Command("git", "add", "-A")
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

// GetUnstagedDiff extrai as mudanças que ainda não foram adicionadas ao stage
func GetUnstagedDiff() (string, error) {
	cmd := exec.Command("git", "diff")
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
	cmd := exec.Command("git", "push", "-u", "origin", "HEAD")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// GetRemoteURL retorna a URL do repositório remoto configurado
func GetRemoteURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
