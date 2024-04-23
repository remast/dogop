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

	"github.com/exaring/otelpgx"
	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/hellofresh/health-go/v5"
	healthPgx "github.com/hellofresh/health-go/v5/checks/pgx4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelseyhightower/envconfig"
	"github.com/riandyrn/otelchi"
	"schneider.vip/problem"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
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
	_, span := otel.Tracer("mux-server").Start(r.Context(), "quote")
	defer span.End()

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

	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	cfg.ConnConfig.Tracer = otelpgx.NewTracer()

	connPool, err := pgxpool.NewWithConfig(context.Background(), cfg)
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

	// Set up OpenTelemetry.
	setupOTelSDK(context.Background())

	r := chi.NewRouter()
	r.Use(otelchi.Middleware("my-server", otelchi.WithChiRoutes(r)))
	r.HandleFunc("POST /api/quote", HandleQuote)

	// CRUD API für Angebote
	offerRepository := &OfferRepository{connPool: connPool}
	r.HandleFunc("POST /api/offer", HandleCreateOffer(offerRepository))
	r.HandleFunc("GET /api/offer/{ID}", HandleReadOffer(offerRepository))

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
	r.HandleFunc("GET /health", h.HandlerFunc)

	r.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello DogOp!"))
	})

	log.Printf("Listening on port %v", config.Port)
	http.ListenAndServe(fmt.Sprintf(":%v", config.Port), r)
}

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func setupOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider()
	if err != nil {
		handleErr(err)
		return
	}
	// initialize tracer
	otel.Tracer("mux-server")
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	meterProvider, err := newMeterProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider() (*trace.TracerProvider, error) {

	var opts []otlptracehttp.Option
	opts = []otlptracehttp.Option{otlptracehttp.WithInsecure(), otlptracehttp.WithEndpointURL("http://localhost:4318")}

	traceExporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(opts...),
	)
	if err != nil {
		return nil, err
	}

	/*
		traceProvider := trace.NewTracerProvider(
			trace.WithBatcher(traceExporter,
				// Default is 5s. Set to 1s for demonstrative purposes.
				trace.WithBatchTimeout(time.Second)),
		)
	*/
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("mux-server"),
		),
	)
	if err != nil {
		log.Fatalf("unable to initialize resource due: %v", err)
	}
	return trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	), nil
}

// See also https://uptrace.dev/opentelemetry/prometheus-metrics.html#sending-metrics-from-go-to-prometheus
func newMeterProvider() (*metric.MeterProvider, error) {

	metricExporter, err := otlpmetrichttp.New(context.Background(), otlpmetrichttp.WithInsecure(), otlpmetrichttp.WithEndpointURL("http://localhost:4318"))
	//metricExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			// Default is 1m. Set to 3s for demonstrative purposes.
			metric.WithInterval(3*time.Second))),
	)
	return meterProvider, nil
}
