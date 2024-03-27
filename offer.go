package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

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

var ErrOfferNotFound = errors.New("offer not found")

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
		id, err := uuid.Parse(r.PathValue("ID"))
		if err != nil {
			problem.New(
				problem.Title("invalid request"),
				problem.Wrap(err),
				problem.Status(http.StatusBadRequest),
			).WriteTo(w)
			return
		}

		offer, err := offerRepository.FindByID(r.Context(), id)

		// 2. Fehler prüfen
		if errors.Is(err, ErrOfferNotFound) {
			problem.New(
				problem.Title("offer not found"),
				problem.Status(http.StatusNotFound),
			).WriteTo(w)
			return
		}
		if err != nil {
			problem.New(
				problem.Title(err.Error()),
				problem.Status(http.StatusInternalServerError),
			).WriteTo(w)
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
	defer tx.Rollback(ctx)

	// 3. Offer per Insert speichern
	_, err := tx.Exec(
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
	if err != nil {
		return nil, err
	}

	// 4. Transaktion commiten
	tx.Commit(ctx)

	// 5. Gespeicherte Offer zurückgeben
	return offer, nil
}

func (r *OfferRepository) FindByID(ctx context.Context, offerID uuid.UUID) (*Offer, error) {
	row := r.connPool.QueryRow(ctx,
		`SELECT id, customer, age, breed, name 
         FROM offers 
	     WHERE id = $1`,
		offerID)

	var (
		id       string
		customer string
		age      int
		breed    string
		name     string
	)

	err := row.Scan(&id, &customer, &age, &breed, &name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOfferNotFound
		}

		return nil, err
	}

	offer := &Offer{
		ID:       id,
		Customer: customer,
		Breed:    breed,
		Age:      age,
		Name:     name,
	}

	return offer, nil
}
