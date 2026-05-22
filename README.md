```markdown
# 🌑 Forge

<p align="center">
  <img src="https://img.shields.io/badge/Eclipse-Dynamics-5f5fff?style=for-the-badge" alt="Eclipse Dynamics">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/AI-Powered-brightgreen?style=for-the-badge" alt="AI Powered">
</p>

**Forge** é um assistente de terminal inteligente (CLI) projetado para desenvolvedores que buscam elegância e automação. Ele analisa as suas alterações locais com IA para forjar mensagens de commit perfeitas no padrão corporativo e disparar o push em segundos.

---

## ✨ Funcionalidades

- 🤖 **AI Commit:** Gera mensagens precisas seguindo o rigoroso padrão *Conventional Commits*.
- ⚙️ **Configuração Nativa:** Menu integrado para salvar a sua API Key localmente, sem poluir o terminal.
- 🎨 **Modern TUI:** Interface interativa rica no terminal desenvolvida com a tecnologia Charmbracelet (Bubble Tea & LipGloss).
- 🚀 **Turbo Workflow:** Executa o `git add`, gera a mensagem com IA, realiza o `commit` e faz o `push` em uma única interação.

---

## 🛠️ Instalação e Compilação Global

Como o Forge foi projetado para rodar globalmente em qualquer diretório do seu sistema operacional, a instalação compila o código em um binário independente e o move para as rotas nativas de execução do Linux.

```bash
# 1. Clone o repositório
git clone [https://github.com/EmenegildoMarques0/forge.git](https://github.com/EmenegildoMarques0/forge.git)

# 2. Entre na pasta do projeto
cd forge

# 3. Baixe as dependências do ecossistema TUI
go mod tidy

# 4. Compile e instale o binário no PATH global do sistema
go build -o forge cmd/forge/main.go && sudo mv forge /usr/local/bin/

```

Após a instalação, você pode fechar o repositório do Forge. Para utilizá-lo, basta entrar na pasta de **qualquer projeto seu que utilize Git** e digitar:

```bash
forge

```

---

## ⚙️ Configuração Inicial

Na primeira execução, você não precisa configurar variáveis de ambiente no arquivo `.bashrc`.

1. Execute o comando `forge`.
2. Use as setas do teclado para navegar até a opção **`⚙️ Configurações`**.
3. Insira a sua `GEMINI_API_KEY` e pressione `Enter`.
4. O Forge salvará a chave de forma segura no arquivo oculto `~/.forge_config.json` e estará pronto para uso definitivo.

---

## 🔄 Como Atualizar o Forge

Se você fizer modificações no código-fonte do Forge ou puxar atualizações do repositório remoto, o comando global precisa ser reconstruído para refletir as mudanças. Para atualizar, execute dentro da pasta do projeto:

```bash
git pull origin main
go build -o forge cmd/forge/main.go && sudo mv forge /usr/local/bin/

```

---

## ❌ Resolução de Problemas (Troubleshooting)

### 1. O terminal exibe: `Comando 'forge' não encontrado`

* **Causa:** O binário não foi movido corretamente para uma pasta monitorada pela variável `$PATH` do Linux, ou você pulou a etapa do `sudo mv`.
* **Como resolver:** Certifique-se de que o arquivo está em `/usr/local/bin/` rodando:

```bash
  ls /usr/local/bin/forge

```

Se não listar o arquivo, refaça o comando de compilação e movimentação contido no guia de instalação.

### 2. O comando `forge` abre uma versão antiga do programa

* **Causa:** O Ubuntu pode ter armazenado o binário antigo em outra rota de cache, como `~/go/bin/forge`.
* **Como resolver:** Remova o binário antigo do Go para evitar conflitos de escopo:

```bash
  rm -f ~/go/bin/forge

```

### 3. Erro: `API respondeu com status 404: NOT_FOUND`

* **Causa:** A URL interna do motor de IA está tentando acessar uma versão depreciada ou inexistente dos modelos do Gemini.
* **Como resolver:** Verifique se o arquivo `internal/ai/gemini.go` está utilizando o endpoint estável atualizado:

```go
  url := "[https://generativelanguage.googleapis.com/v1/models/gemini-2.5-flash:generateContent?key=](https://generativelanguage.googleapis.com/v1/models/gemini-2.5-flash:generateContent?key=)" + apiKey

```

### 4. Erro: `nenhuma alteração detectada no Git`

* **Causa:** Você executou o comando em uma pasta que não possui um repositório Git inicializado (`.git`), ou não modificou nenhuma linha de código antes de chamar o assistente.
* **Como resolver:** Execute `git status` para certificar-se de que existem arquivos modificados (na área de *Staged* ou *Unstaged*). O Forge precisa de um `diff` de código real para conseguir estruturar a mensagem.

### 5. Erro ao compilar: `no required module provides package github.com/charmbracelet...`

* **Causa:** O cache de dependências local do Go está desatualizado em relação às novas implementações da TUI.
* **Como resolver:** Force a limpeza e sincronização dos pacotes rodando:

```bash
  go mod tidy

```

```


```
