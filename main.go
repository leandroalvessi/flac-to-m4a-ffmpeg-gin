package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*.html
var templatesFS embed.FS

func main() {
	// Cria um roteador Gin
	router := gin.Default()

	// Carrega os templates HTML incorporados
	templ := template.Must(template.New("").ParseFS(templatesFS, "templates/*.html"))
	router.SetHTMLTemplate(templ)

	// Define uma rota GET
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Year": time.Now().Year(),
		})
	})

	// Define uma rota POST
	router.POST("/compress", comprimirHandler)

	// Inicia o servidor na porta 8080
	go openBrowser("http://localhost:8080") // Abrir o navegador automaticamente
	router.Run(":8080")
}

func comprimirHandler(c *gin.Context) {
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

	// Gerar o nome de arquivo baseado na data e hora atuais
	currentTime := time.Now().Format("20060102150405") // Inclui segundos
	fileNameWithoutExt := filepath.Base(file.Filename)
	ext := filepath.Ext(fileNameWithoutExt)
	newFileName := fileNameWithoutExt[:len(fileNameWithoutExt)-len(ext)] + "_" + currentTime + "_Comprimido" + ext

	// Definir o caminho completo para o arquivo enviado
	filePath := filepath.Join(uploadDir, file.Filename)

	// Salvar o arquivo enviado no diretório de upload
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Erro ao salvar o arquivo enviado: " + err.Error(),
		})
		return
	}

	// Definir o diretório de saída e o nome do arquivo comprimido
	outputDir := filepath.Dir(filePath)
	outputFile := filepath.Join(outputDir, newFileName)

	// Chamar a função de compressão de PDF
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

	// Excluir o arquivo original enviado
	if err := os.Remove(filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Erro ao excluir o arquivo original: " + err.Error(),
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
