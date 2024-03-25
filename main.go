package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/hellofresh/health-go/v5"
	healthPgx "github.com/hellofresh/health-go/v5/checks/pgx4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelseyhightower/envconfig"
	"schneider.vip/problem"
)

//go:embed migrations
var migrations embed.FS

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
		problem.New(problem.Wrap(err), problem.Status(http.StatusBadRequest)).WriteTo(w)
		return
	}

	tariff := Tariff{Name: "Dog OP _ Basic", Rate: 12.4}
	quote.Tariffs = []Tariff{tariff}

	err = json.NewEncoder(w).Encode(quote)
	if err != nil {
		problem.New(
			problem.Wrap(err),
			problem.Status(http.StatusInternalServerError),
		).WriteTo(w)
	}
}

func Connect(dbURL string) (*pgxpool.Pool, error) {
	err := Migrate(dbURL)
	if err != nil {
		return nil, err
	}

	connPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, err
	}

	return connPool, nil
}

func Migrate(dbURL string) error {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}

	dbMigrateURL := strings.Replace(dbURL, "postgres://", "pgx://", 1)
	m, err := migrate.NewWithSourceInstance("iofs", source, dbMigrateURL)
	if err != nil {
		return err
	}
	defer m.Close()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func main() {
	var config Config
	err := envconfig.Process("dogop", &config)
	if err != nil {
		log.Fatal(err.Error())
	}

	connPool, err := Connect(config.Db)
	if err != nil {
		log.Fatal(err.Error())
	}

	r := chi.NewRouter()
	r.Post("/api/quote", HandleQuote)

	// CRUD API für Angebote
	offerRepository := &OfferRepository{connPool: connPool}
	r.Route("/api/offer", func(sr chi.Router) {
		sr.Post("/", HandleCreateOffer(offerRepository))
		sr.Get("/{ID}", HandleReadOffer(offerRepository))
	})

	// Register Health Check
	h, _ := health.New(health.WithChecks(
		health.Config{
			Name:      "db",
			Timeout:   time.Second * 2,
			SkipOnErr: false,
			Check: healthPgx.New(healthPgx.Config{
				DSN: config.Db,
			}),
		},
	))

	// Register Health Check Handler Function
	r.Get("/health", h.HandlerFunc)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello DogOp!"))
	})

	log.Println(fmt.Sprintf("Listening on port %v", config.Port))
	http.ListenAndServe(fmt.Sprintf(":%v", config.Port), r)
}
