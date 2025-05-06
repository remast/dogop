package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port string `default:"8080"`
	Db   string `default:"postgres://postgres:postgres@localhost:5432/dogop"`
}

type Quote struct {
	Age     int      `json:"age"`
	Breed   string   `json:"breed"`
	Tariffs []Tariff `json:"tariffs"`
}

type Tariff struct {
	Name string  `json:"name"`
	Rate float64 `json:"rate"`
}

func HandleQuote(w http.ResponseWriter, r *http.Request) {
	var quote Quote
	err := json.NewDecoder(r.Body).Decode(&quote)
	if err != nil {
		http.Error(w, "Could not decode quote.😔", http.StatusBadRequest)
		return
	}

	tariff := Tariff{Name: "Dog OP _ Basic", Rate: 12.4}
	quote.Tariffs = []Tariff{tariff}

	err = json.NewEncoder(w).Encode(quote)
	if err != nil {
		http.Error(w, "Could not encode quote.😔", http.StatusBadRequest)
	}
}

func main() {
	var config Config
	err := envconfig.Process("dogop", &config)
	if err != nil {
		log.Fatal(err.Error())
	}

	r := http.NewServeMux()
	r.HandleFunc("POST /api/quote", HandleQuote)

	r.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello DogOp!"))
	})

	http.ListenAndServe(fmt.Sprintf(":%v", config.Port), r)
}
