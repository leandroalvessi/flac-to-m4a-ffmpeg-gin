package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Cria um roteador Gin
	router := gin.Default()
	workingDir, _ := os.Getwd()
	router.LoadHTMLGlob(workingDir + "/templates/*.html")

	// Define uma rota GET
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Year": time.Now().Year(),
		})
	})

	// Define uma rota POST
	router.POST("/convert", convertHandler)

	// Inicia o servidor na porta 8080
	go openBrowser("http://localhost:8080") // Abrir o navegador automaticamente
	router.Run(":8080")
}

func convertHandler(c *gin.Context) {
	// Obter o arquivo PDF enviado pelo formulário
	file, err := c.FormFile("inputFile")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Erro ao obter o arquivo: " + err.Error(),
		})
		return
	}

	// Obter a qualidade selecionada
	quality := c.PostForm("quality")

	// Criar um diretório para armazenar os arquivos enviados (se ainda não existir)
	uploadDir := "Arquivos Comprimidos"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Erro ao criar diretório de upload: " + err.Error(),
			})
			return
		}
	}

	// Definir o caminho completo para o arquivo enviado
	filePath := filepath.Join(uploadDir, file.Filename)

	// Salvar o arquivo enviado no diretório de upload
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Erro ao salvar o arquivo enviado: " + err.Error(),
		})
		return
	}

	// Definir o diretório de saída como o mesmo diretório onde o arquivo foi salvo
	outputDir := filepath.Dir(filePath)
	outputFile := filepath.Join(outputDir, "output.pdf")

	// Chamar a função de compressão de PDF (ajuste conforme sua lógica)
	err = comprimirPDF(filePath, outputFile, quality)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Erro ao comprimir o PDF: " + err.Error(),
		})
		return
	}

	// Verificar se o arquivo de saída foi criado
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Erro: O arquivo de saída não foi criado: " + outputFile,
		})
		return
	}

	// Retornar resposta de sucesso
	c.HTML(http.StatusOK, "message.html", gin.H{
		"Year":    time.Now().Year(),
		"message": "PDF comprimido com sucesso!",
	})
}

func comprimirPDF(input string, output string, screen string) error {
	println(output)
	cmd := exec.Command("gswin64",
		"-sDEVICE=pdfwrite",
		"-dCompatibilityLevel=1.4",
		"-dPDFSETTINGS=/"+screen,
		"-dNOPAUSE",
		"-dQUIET",
		"-dBATCH",
		"-sOutputFile="+output,
		input)

	// Executar o comando
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("erro ao comprimir o PDF: %v", err)
	}

	return nil
}

func openBrowser(url string) {
	var err error
	// Tenta abrir o navegador de forma multiplataforma
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("cmd", "/C", "start", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		fmt.Printf("Erro ao abrir o navegador: %v\n", err)
	}
}
