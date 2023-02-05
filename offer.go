package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"schneider.vip/problem"
)

type Offer struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
	Age      int    `json:"age"`
	Breed    string `json:"breed"`
	Name     string `json:"name"`
}

var OfferNotFound = errors.New("offer not found")

func HandleCreateOffer(offerRepository *OfferRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. JSON Request lesen
		var offer Offer
		json.NewDecoder(r.Body).Decode(&offer)

		// 2. Offer speichern
		createdOffer, _ := offerRepository.Insert(r.Context(), &offer)

		// 3. JSON Response schreiben
		json.NewEncoder(w).Encode(createdOffer)
	}
}

func HandleReadOffer(offerRepository *OfferRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Offer lesen
		id := chi.URLParam(r, "ID")
		offer, err := offerRepository.FindByID(r.Context(), id)

		// 2. Fehler prüfen
		if errors.Is(err, OfferNotFound) {
			http.Error(w, problem.New(problem.Title("offer not found")).JSONString(), http.StatusNotFound)
			return
		}

		// 2. JSON Response schreiben
		json.NewEncoder(w).Encode(offer)
	}
}

type OfferRepository struct {
	connPool *pgxpool.Pool
}

func (r *OfferRepository) Insert(ctx context.Context, offer *Offer) (*Offer, error) {
	// 1. ID generieren
	offer.ID = uuid.New().String()

	// 2. Transaktion beginnen
	tx, _ := r.connPool.Begin(ctx)

	// 3. Offer per Insert speichern
	_, _ = tx.Exec(
		ctx,
		`INSERT INTO offers 
		   (id, customer, age, breed, name) 
		 VALUES 
		   ($1, $2, $3, $4, $5)`,
		offer.ID,
		offer.Customer,
		offer.Age,
		offer.Breed,
		offer.Name,
	)

	// 4. Transaktion commiten
	tx.Commit(ctx)

	// 5. Gespeicherte Offer zurückgeben
	return offer, nil
}

func (r *OfferRepository) FindByID(ctx context.Context, offerID string) (*Offer, error) {
	row := r.connPool.QueryRow(ctx,
		`SELECT activity_id as id, description, start_time, end_time, username, org_id, project_id 
         FROM activities 
	     WHERE activity_id = $1`,
		offerID)

	var (
		id string
	)

	err := row.Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, OfferNotFound
		}

		return nil, err
	}

	offer := &Offer{
		ID: id,
	}

	return offer, nil
}
