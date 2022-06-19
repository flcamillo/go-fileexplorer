package main

import (
	"encoding/json"
	"log"
	"os"
)

// Define um caminho
type Path struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// define a configuração
type Config struct {
	Address         string  `json:"address"`
	Port            int     `json:"port"`
	DownloadEnabled bool    `json:"download_enabled"`
	AvoidPaths      []*Path `json:"avoid_paths"`
	AllowPaths      []*Path `json:"allow_paths"`
}

// retorna uma nova configuração
func NewConfig() *Config {
	return &Config{
		Address:         "0.0.0.0",
		Port:            80,
		DownloadEnabled: true,
	}
}

// carrega as configurações do arquivo
func (p *Config) Load(file string) {
	f, err := os.OpenFile(file, os.O_RDWR, 0755)
	if err != nil {
		log.Printf("Não foi possível abrir o arquivo de configuração, %s", err)
		return
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(p)
	if err != nil {
		log.Printf("Não foi possível ler a configuração, %s", err)
		return
	}
}

// salva as configurações no arquivo
func (p *Config) Save(file string) {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		log.Printf("Não foi possível abrir o arquivo de configuração, %s", err)
		return
	}
	defer f.Close()
	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	err = e.Encode(p)
	if err != nil {
		log.Printf("Não foi possível gravar a configuração, %s", err)
		return
	}
}
