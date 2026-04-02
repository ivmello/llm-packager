package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/disiqueira/gotree"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/js"
)

var (
	maxSize    int64 = 10 * 1024 * 1024 // 10MB por arquivo MD
	ignoreFile       = ".llmignore"
	minifier   *minify.M
)

func init() {
	minifier = minify.New()
	minifier.AddFunc("text/javascript", js.Minify)
}

func main() {
	var rootCmd = &cobra.Command{Use: "llm-pack"}

	// --- COMANDO: PREPARE ---
	var prepareCmd = &cobra.Command{
		Use:   "prepare",
		Short: "Inicializa o .llmignore baseado no .gitignore",
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := os.Stat(ignoreFile); err == nil {
				fmt.Println("ℹ️  .llmignore já existe.")
				return
			}
			content := "# LLM Ignore\n.git/\nnode_modules/\ndist/\nvendor/\nbin/\n*.log\n*.map\n"
			if _, err := os.Stat(".gitignore"); err == nil {
				gitData, _ := os.ReadFile(".gitignore")
				content += "\n# Do .gitignore:\n" + string(gitData)
			}
			os.WriteFile(ignoreFile, []byte(content), 0644)
			fmt.Println("🚀 .llmignore criado!")
		},
	}

	// --- COMANDO: MAP ---
	var mapCmd = &cobra.Command{
		Use:   "map",
		Short: "Gera a árvore visual do projeto de forma hierárquica",
		Run: func(cmd *cobra.Command, args []string) {
			ignorer, _ := ignore.CompileIgnoreFile(ignoreFile)
			wd, _ := os.Getwd()
			
			// Iniciamos a raiz da árvore
			root := gotree.New(filepath.Base(wd))
			
			// O segredo é um mapa que guarda a referência de cada nó de diretório criado
			// A chave é o path relativo, o valor é o objeto gotree.Tree
			treeMap := make(map[string]gotree.Tree)
			treeMap["."] = root

			err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
				if err != nil || path == "." {
					return nil
				}

				// Verifica se deve ignorar
				if ignorer != nil && ignorer.MatchesPath(path) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				// Pegamos o diretório pai do item atual
				parentPath := filepath.Dir(path)
				
				// Buscamos o nó do pai no nosso mapa. 
				// Se por algum motivo o pai não existir (raro no WalkDir), usamos a raiz.
				parentNode, ok := treeMap[parentPath]
				if !ok {
					parentNode = root
				}

				if d.IsDir() {
					// Se for diretório, cria um novo nó e guarda no mapa para os filhos usarem
					newNode := parentNode.Add(d.Name())
					treeMap[path] = newNode
				} else {
					// Se for arquivo, apenas adiciona como folha do nó pai
					parentNode.Add(d.Name())
				}
				return nil
			})

			if err != nil {
				fmt.Printf("❌ Erro ao ler diretórios: %v\n", err)
				return
			}

			os.WriteFile("project_map.md", []byte("# Project Structure\n```\n"+root.Print()+"```"), 0644)
			fmt.Println("🗺️  Mapa gerado com sucesso em project_map.md!")
		},
	}

	// --- COMANDO: GENERATE ---
	var generateCmd = &cobra.Command{
		Use:   "generate",
		Short: "Processa, minifica e empacota o código",
		Run: func(cmd *cobra.Command, args []string) {
			ignorer, _ := ignore.CompileIgnoreFile(ignoreFile)
			var currentBlock strings.Builder
			blockID := 1

			filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
				if d.IsDir() || (ignorer != nil && ignorer.MatchesPath(path)) { return nil }
				if isBinary(path) { return nil }

				content, err := os.ReadFile(path)
				if err != nil { return nil }

				ext := filepath.Ext(path)
				minified := smartMinify(string(content), ext)

				header := fmt.Sprintf("\n\n---\n### LOCATION: %s\n---\n```%s\n%s\n```\n", path, strings.TrimPrefix(ext, "."), minified)

				if int64(currentBlock.Len()+len(header)) > maxSize {
					saveFile(currentBlock.String(), blockID)
					currentBlock.Reset()
					blockID++
				}
				currentBlock.WriteString(header)
				return nil
			})

			if currentBlock.Len() > 0 {
				saveFile(currentBlock.String(), blockID)
			}
			fmt.Println("✅ Documentação pronta para o NotebookLM!")
		},
	}

	rootCmd.AddCommand(prepareCmd, mapCmd, generateCmd)
	rootCmd.Execute()
}

func smartMinify(code string, ext string) string {
	switch ext {
	case ".ts", ".js", ".cjs", ".mjs":
		m, err := minifier.String("text/javascript", code)
		if err == nil { return m }
	case ".php":
		return basicMinify(code)
	case ".go":
		return basicMinify(code)
	}
	return basicMinify(code)
}

func basicMinify(code string) string {
	var out strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(code))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" { continue }
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") { continue }
		out.WriteString(line + "\n")
	}
	return out.String()
}

func saveFile(content string, id int) {
	name := fmt.Sprintf("llm_bundle_%03d.md", id)
	os.WriteFile(name, []byte(content), 0644)
	fmt.Printf("📦 Criado: %s\n", name)
}

func isBinary(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	skip := []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".zip", ".exe", ".sum", ".mod", ".lock"}
	return slices.Contains(skip, ext)
}