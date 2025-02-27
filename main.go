package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Hash esperado para comparação (substitua pelo hash desejado)
const expectedHash = "8c6f1ec103bc95a87afdc44086eb138fa98cd55dc2ffd6ab67de7115bd20c8a8" //Dell I5 Leandro

func main() {
	gin.SetMode(gin.ReleaseMode) // Define o modo de execução como "release"

	generatedHash, isValid := compareHash()

	// Cria um roteador Gin
	//router := gin.Default()

	router := gin.New()                      // Cria um Engine sem middlewares padrão
	router.Use(gin.Logger(), gin.Recovery()) // Adiciona os middlewares manualmente

	// Configura proxies confiáveis
	// Substitua []string{"127.0.0.1"} pelos IPs dos seus proxies confiáveis
	// Exemplo: []string{"192.168.1.0/24", "10.0.0.0/8"}
	router.SetTrustedProxies([]string{"127.0.0.1"}) // Confia apenas no localhost

	// Carrega os templates HTML incorporados
	templ := template.Must(template.New("").ParseFS(templatesFS, "templates/*.html"))
	router.SetHTMLTemplate(templ)

	if isValid {
		// Define uma rota GET para a página inicial
		router.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", gin.H{
				"Year": time.Now().Year(),
			})
		})

		// Define uma rota POST para compressão de PDF
		router.POST("/compress", comprimirHandler)
	} else {
		// Redireciona todas as requisições para a página de mensagem
		router.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "message.html", gin.H{
				"Year":         time.Now().Year(),
				"Message":      "Acesso negado! O hardware não corresponde ao esperado.",
				"ExpectedHash": generatedHash,
			})
		})
	}

	// Inicia o servidor na porta 8080
	go openBrowser("http://localhost:8080") // Abrir o navegador automaticamente
	router.Run(":8080")
}

// Handler para compressão de PDF
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
		"Year":         time.Now().Year(),
		"Message":      "PDF comprimido com sucesso!",
		"ExpectedHash": "",
	})
}

// Função para comprimir PDF usando Ghostscript
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

// Função para abrir o navegador automaticamente
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

// getCommandOutput executa um comando e retorna sua saída como string
func getCommandOutput(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getProcessorSerial obtém o número de série do processador
func getProcessorSerial() string {
	var serial string
	var err error

	switch {
	case isLinux():
		serial, err = getCommandOutput("cat", "/proc/cpuinfo")
		if err == nil {
			lines := strings.Split(serial, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "Serial") {
					serial = strings.TrimSpace(strings.Split(line, ":")[1])
					break
				}
			}
		}
	case isWindows():
		serial, err = getCommandOutput("wmic", "cpu", "get", "ProcessorId")
		if err == nil {
			lines := strings.Split(serial, "\n")
			if len(lines) > 1 {
				serial = strings.TrimSpace(lines[1])
			}
		}
	case isMac():
		serial, err = getCommandOutput("sysctl", "-n", "machdep.cpu.brand_string")
	}

	if err != nil {
		return "unknown"
	}
	return serial
}

// getMotherboardSerial obtém o número de série da placa-mãe
func getMotherboardSerial() string {
	var serial string
	var err error

	switch {
	case isLinux():
		serial, err = getCommandOutput("sudo", "dmidecode", "-s", "baseboard-serial-number")
	case isWindows():
		serial, err = getCommandOutput("wmic", "baseboard", "get", "SerialNumber")
	case isMac():
		serial, err = getCommandOutput("ioreg", "-l")
	}

	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(serial)
}

// isLinux verifica se o sistema operacional é Linux
func isLinux() bool {
	return strings.Contains(strings.ToLower(getOS()), "linux")
}

// isWindows verifica se o sistema operacional é Windows
func isWindows() bool {
	return strings.Contains(strings.ToLower(getOS()), "windows")
}

// isMac verifica se o sistema operacional é MacOS
func isMac() bool {
	return strings.Contains(strings.ToLower(getOS()), "darwin")
}

// getOS retorna o nome do sistema operacional
func getOS() string {
	cmd := exec.Command("uname")
	output, err := cmd.Output()
	if err != nil {
		return "windows" // Assume Windows como padrão se falhar
	}
	return strings.TrimSpace(string(output))
}

// generateHardwareHash gera um hash SHA-256 concatenando os números de série do processador e da placa-mãe
func generateHardwareHash() string {
	processorSerial := getProcessorSerial()
	motherboardSerial := getMotherboardSerial()
	concatenated := processorSerial + motherboardSerial

	hash := sha256.Sum256([]byte(concatenated))
	return hex.EncodeToString(hash[:])
}

// compareHash verifica se o hash gerado corresponde ao hash esperado
func compareHash() (string, bool) {
	generatedHash := generateHardwareHash()
	return generatedHash, generatedHash == expectedHash
}
