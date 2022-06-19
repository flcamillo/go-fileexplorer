package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var (
	// caminho de onde estão os arquivos html, jss, css, etc
	documentRoot = "./root"
	// templates compilados do servidor
	templates *template.Template
	// define o path da aplicação
	applicationPath = ""
	// define o nome do arquivo de configuração
	configFile = "config.json"
	// define as configurações
	config = NewConfig()
	// define as expressões regulares para os caminhos proibidos
	avoidedPaths []*regexp.Regexp
	// define as expressões regulares para os caminhos permitidos
	allowedPaths []*regexp.Regexp
)

// função principal
func main() {
	// identifica o caminho da aplicação
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatalf("Não foi possível identificar o diretório da aplicação, %s", err)
	}
	applicationPath = dir
	// inicializa o arquivo de logs
	logFile, err := os.OpenFile("./log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0775)
	if err != nil {
		log.Printf("Não foi possível abrir o arquivo de log. Erro: %s", err)
	} else {
		log.SetOutput(logFile)
	}
	defer logFile.Close()
	// carrega a configuração
	config.Load(filepath.Join(applicationPath, configFile))
	config.Save(filepath.Join(applicationPath, configFile))
	// compila os regexp para os caminhos proibidos
	for k, v := range config.AvoidPaths {
		mask := regexp.QuoteMeta(v.Path)
		mask = strings.ReplaceAll(mask, `\*`, ".*")
		flag := "(?i)"
		pattern, err := regexp.Compile(fmt.Sprintf("%s%s", flag, mask))
		if err != nil {
			log.Printf("Expressão {%d} {%s} de caminho proibido ignorada pois esta inválida, %s", k, v.Path, err)
		}
		avoidedPaths = append(avoidedPaths, pattern)
	}
	// compila os regexp para os caminhos permitidos
	for k, v := range config.AllowPaths {
		mask := regexp.QuoteMeta(v.Path)
		mask = strings.ReplaceAll(mask, `\*`, ".*")
		flag := "(?i)"
		pattern, err := regexp.Compile(fmt.Sprintf("%s%s", flag, mask))
		if err != nil {
			log.Printf("Expressão {%d} {%s} de caminho permitido ignorada pois esta inválida, %s", k, v.Path, err)
		}
		allowedPaths = append(allowedPaths, pattern)
	}
	// define as funções para serem usadas nos templates
	templatesFunc := template.FuncMap{
		"Add":       Add,
		"FileSize":  FileSize,
		"HostName":  HostName,
		"Plataform": Plataform,
		"Hostname":  HostName,
	}
	// carrega os templates para memória
	tmpl, err := template.New("main").Funcs(templatesFunc).ParseGlob(filepath.Join(applicationPath, documentRoot, "/html/*"))
	if err != nil {
		log.Printf("Não foi possível ler o diretório de templates. Erro: %s", err)
	} else {
		templates = tmpl
	}
	// cria o roteador do servidor http
	mux := http.NewServeMux()
	// adiciona as rotas de arquivos estaticos
	mux.HandleFunc("/js/", handlerJavaScript)
	mux.HandleFunc("/css/", handlerCSS)
	mux.HandleFunc("/img/", handlerImage)
	mux.HandleFunc("/favicon.ico", handlerFavicon)
	// adiciona as rotas de navegação
	mux.HandleFunc("/", handleIndex)
	// configura o servidor http
	srv := &http.Server{
		Handler:           mux,
		Addr:              fmt.Sprintf("%s:%d", config.Address, config.Port),
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10,
	}
	log.Fatal(srv.ListenAndServe())
}

// processa o envio de arquivos javascript para o browser
func handlerJavaScript(w http.ResponseWriter, r *http.Request) {
	dir, file := filepath.Split(r.URL.Path)
	if file == "" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, filepath.Join(documentRoot, dir, file))
}

// processa o envio de arquivos css para o browser
func handlerCSS(w http.ResponseWriter, r *http.Request) {
	dir, file := filepath.Split(r.URL.Path)
	if file == "" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, filepath.Join(documentRoot, dir, file))
}

// processa o envio de arquivos de imagem para o browser
func handlerImage(w http.ResponseWriter, r *http.Request) {
	dir, file := filepath.Split(r.URL.Path)
	if file == "" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, filepath.Join(documentRoot, dir, file))
}

// processa o envio do icone da pagina para o browser
func handlerFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(documentRoot, "img", "favicon.png"))
}

// processa a pagina inicial
func handleIndex(w http.ResponseWriter, r *http.Request) {
	// cria uma mapa para retornar os dados para pagina
	records := make(map[string]interface{})
	// define os erros
	messages := make(map[string][]string)
	// identifica o caminho a ser consultado
	path := r.FormValue("dir")
	if path == "" {
		path = applicationPath
	}
	// identifica a operação
	op := r.FormValue("op")
	// acessa a pasta no diretório atual caso a operação seja esta
	if op == "in" {
		path = filepath.Join(path, r.FormValue("folder"))
	}
	// acessa o diretório anterior
	if op == "up" {
		path = filepath.Join(path, "..")
	}
	// define se o caminho esta proibido
	pathAvoided := false
	// identifica se o caminho esta na relação de proibidos
	for _, pattern := range avoidedPaths {
		if pattern.Match([]byte(path)) {
			pathAvoided = true
			break
		}
	}
	// define se o caminho esta permitido
	// caso não exista caminhos permitidos então entende-se que todos
	// são permitidos
	pathAllowed := len(allowedPaths) == 0
	// identifica se o caminho esta na relação de permitidos
	for _, pattern := range allowedPaths {
		if pattern.Match([]byte(path)) {
			pathAllowed = true
			break
		}
	}
	// processa o download caso seja esta a opção
	if op == "download" && !pathAvoided && pathAllowed {
		if !config.DownloadEnabled {
			messages["Alerts"] = append(messages["Alerts"], "O download de arquivos não esta habilitado")
		} else {
			id := r.FormValue("id")
			if id != "" {
				// abre o arquivo para leitura
				filePath := filepath.Join(path, id)
				f, err := os.OpenFile(filePath, os.O_RDONLY, 0755)
				if err != nil {
					messages["Errors"] = append(messages["Erros"], fmt.Sprintf("Não foi possível abrir o arquivo {%s}, %s", filePath, err))
				} else {
					defer f.Close()
					// le os atributos do arquivo
					stat, err := f.Stat()
					if err != nil {
						messages["Errors"] = append(messages["Erros"], fmt.Sprintf("Não foi possível abrir o arquivo {%s}, %s", filePath, err))
					} else {
						// ajusta os headers para forçar o donwload do arquivo
						w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, id))
						w.Header().Set("Content-Type", "application/octet-stream")
						w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
						// envia os dados
						_, err := io.Copy(w, f)
						if err != nil {
							messages["Errors"] = append(messages["Erros"], fmt.Sprintf("Falha no download do arquivo {%s}, %s", filePath, err))
						}
						return
					}
				}
			}
		}
	}
	// processa o template ao terminar
	defer func() {
		err := templates.ExecuteTemplate(w, "Home", map[string]interface{}{"Title": "Home", "Records": records, "Messages": messages})
		if err != nil {
			log.Printf("Erro no processamento do template, %s", err)
		}
	}()
	// caso o caminho não seja permitido retorna
	if pathAvoided {
		messages["Alerts"] = append(messages["Alerts"], "O caminho solicitado está na relação de proibidos")
		return
	}
	if !pathAllowed {
		messages["Alerts"] = append(messages["Alerts"], "O caminho solicitado não esta na relação de permitidos")
		return
	}
	// define se pode ou não realizar download para a pagina tratar
	records["Download"] = config.DownloadEnabled
	// processa o formulario
	err := r.ParseForm()
	if err != nil {
		log.Printf("Não foi possível processar o formulário da pagina, %s", err)
	}
	// identifica a coluna que esta sendo ordenada
	order := r.FormValue("o")
	if order != "name" && order != "date" && order != "size" {
		order = "name"
	}
	records["Order"] = order
	// identifica a classificação a ser aplicação na coluna
	orderType := r.FormValue("ot")
	if orderType == "z" {
		orderType = "a"
	} else {
		orderType = "z"
	}
	records["OrderType"] = orderType
	// define o diretório atual para a pagina tratar
	records["Dir"] = path
	// identifica todos os objetos do diretório
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		messages["Errors"] = append(messages["Erros"], fmt.Sprintf("Não foi possível ler o diretório {%s}, %s", path, err))
		return
	}
	// ordena
	sort.Slice(entries, func(p, q int) bool {
		switch order {
		case "date":
			if orderType == "z" {
				return entries[p].ModTime().Before(entries[q].ModTime())
			}
			return entries[p].ModTime().After(entries[q].ModTime())
		case "size":
			if orderType == "z" {
				return entries[p].Size() < entries[q].Size()
			}
			return entries[p].Size() > entries[q].Size()
		default:
			if orderType == "z" {
				return entries[p].Name() < entries[q].Name()
			}
			return entries[p].Name() > entries[q].Name()
		}
	})
	records["Files"] = entries
}

// Add retorna a soma dos dois valores
func Add(value1 int, value2 int) int {
	return value1 + value2
}

// FileSize retorna o tamanho do arquivo formatado
func FileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dbytes", size)
	}
	nsize := float64(size) / 1024
	if nsize < 1024 {
		return fmt.Sprintf("%.2fKB", nsize)
	}
	nsize = float64(nsize / 1024)
	if nsize < 1024 {
		return fmt.Sprintf("%.2fMB", nsize)
	}
	nsize = float64(nsize / 1024)
	if nsize < 1024 {
		return fmt.Sprintf("%.2fGB", nsize)
	}
	nsize = float64(nsize / 1024)
	if nsize < 1024 {
		return fmt.Sprintf("%.2fTB", nsize)
	}
	nsize = float64(nsize / 1024)
	return fmt.Sprintf("%.2fPB", nsize)
}

// Retorna a plataforma de execução da aplicação
func Plataform() string {
	return fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH)
}

// Retorna o nome da máquina
func HostName() string {
	host, err := os.Hostname()
	if err != nil {
		return ""
	}
	return host
}
