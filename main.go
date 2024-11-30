package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var RenomearPorNumero = false

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
	router.Run(":8080")
}

func convertHandler(c *gin.Context) {
	// Obter os parâmetros do formulário
	inputDir := c.DefaultPostForm("inputDir", "C:\\Users\\leand\\Music")
	outputDir := c.DefaultPostForm("outputDir", "C:\\Users\\leand\\Music")
	quality := c.DefaultPostForm("quality", "10")
	renameByNumber := c.DefaultPostForm("renameByNumber", "false")

	// Converter a entrada do checkbox para booleano
	RenomearPorNumero = renameByNumber == "true"

	// Chamar a função de conversão
	err := converter(inputDir, outputDir, quality)
	if err != nil {
		c.JSON(500, gin.H{
			"message": fmt.Sprintf("Error: %v", err),
		})
		return
	}

	// Retornar resposta de sucesso
	c.HTML(http.StatusOK, "message.html", gin.H{
		"Year": time.Now().Year(),
		//"DateLicense": dateLicense.Format("02/01/2006 15:04:05"),
	})

}

func converter(inputDir, outputDir, Quality string) error {
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("error opening directory: %v", err)
	}

	var wg sync.WaitGroup

	// Obter os núcleos de CPU disponíveis
	numCPU := 1
	if runtime.NumCPU() > 1 {
		numCPU = runtime.NumCPU()
	}

	// Semáforo para limitar a execução concorrente
	sem := make(chan struct{}, numCPU)

	// Fila de arquivos para processar
	var fileQueue []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".flac") {
			fileQueue = append(fileQueue, filepath.Join(inputDir, file.Name()))
		}
	}

	// Iniciar o processamento dos arquivos
	startTime := time.Now()
	totalFiles := len(fileQueue)
	fmt.Printf("Starting conversion of %d FLAC files...\n", totalFiles)

	for idx, inputFile := range fileQueue {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, inputFile string) {
			defer wg.Done()

			var outputFile string

			if RenomearPorNumero {
				// Renomear arquivos com um prefixo numérico
				numeroMusica := fmt.Sprintf("%02d", idx+1)
				outputFile = filepath.Join(outputDir, numeroMusica+" - "+strings.TrimSuffix(filepath.Base(inputFile), ".flac")+".m4a")
			} else {
				outputFile = filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(inputFile), ".flac")+".m4a")
			}

			// Garantir que o nome do arquivo de saída seja único
			counter := 1
			for {
				if _, err := os.Stat(outputFile); err == nil {
					outputFile = filepath.Join(outputDir, fmt.Sprintf("%s (Copy %d).m4a", strings.TrimSuffix(filepath.Base(inputFile), ".flac"), counter))
					counter++
				} else {
					break
				}
			}

			// Executar o comando FFmpeg para converter o arquivo
			cmd := exec.Command(
				"ffmpeg",
				"-i", inputFile,
				"-c:a", "aac",
				"-q:a", Quality,
				"-map", "0",
				"-map_metadata", "0",
				"-c:v", "mjpeg",
				"-disposition:v", "attached_pic",
				"-avoid_negative_ts", "make_zero",
				outputFile,
			)

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Run()
			if err != nil {
				fmt.Printf("Error converting %s: %v\n", inputFile, err)
				<-sem
				return
			}

			fmt.Printf("File successfully converted: %s\n", outputFile)

			<-sem
		}(idx, inputFile)
	}

	wg.Wait()

	duration := time.Since(startTime)
	fmt.Printf("Conversion completed in %s\n", duration)

	return nil
}
