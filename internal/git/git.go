package git

import (
	"os/exec"
	"strings"
)

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
