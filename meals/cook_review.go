package meals

import (
	"context"
	"time"

	sqlc "weekly-shopping-app/database/sqlc"
	"weekly-shopping-app/internal/api/httpx"
	"weekly-shopping-app/internal/logger"
	"weekly-shopping-app/pantry"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// cook_review.go
//
// The login "did you make these?" flow. On login the app fetches meals that
// were planned since the user's previous visit (capped to under 7 days) and
// asks whether each was actually cooked. Confirmed meals decrement the pantry;
// every answer is logged so the same day/meal is never asked about twice.

const dateLayout = "2006-01-02"

// maxLookbackDays caps the review window at strictly under one week.
const maxLookbackDays = 6

// ── Response types ────────────────────────────────────────────────────────────

type CookReviewMeal struct {
	CookDate        string `json:"cook_date"` // YYYY-MM-DD
	DayName         string `json:"day_name"`
	MealID          int32  `json:"meal_id"`
	MealName        string `json:"meal_name"`
	DefaultPortions int32  `json:"default_portions"`
	Scope           string `json:"scope"` // "household" | "personal"
	HouseholdID     int32  `json:"household_id"`
}

type CookReviewResponse struct {
	Since string           `json:"since"` // window start, YYYY-MM-DD
	Meals []CookReviewMeal `json:"meals"`
}

// ── Input types ───────────────────────────────────────────────────────────────

type CookReviewConfirmInput struct {
	Confirmations []CookConfirmation `json:"confirmations"`
}

type CookConfirmation struct {
	CookDate    string  `json:"cook_date"` // YYYY-MM-DD
	MealID      int32   `json:"meal_id"`
	Made        bool    `json:"made"`
	Portions    float64 `json:"portions"`
	Scope       string  `json:"scope"` // "household" | "personal"
	HouseholdID int32   `json:"household_id"`
}

type CookReviewConfirmResult struct {
	Status string `json:"status"`
	Cooked int    `json:"cooked"` // how many meals decremented the pantry
}

// ── Business logic ────────────────────────────────────────────────────────────

// getCookReview returns planned meals in the window
// [max(previous_seen_at, today-6d), yesterday] that the user has not answered.
func getCookReview(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) (*CookReviewResponse, error) {
	q := sqlc.New(db)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	until := today.AddDate(0, 0, -1)            // yesterday, inclusive
	since := today.AddDate(0, 0, -maxLookbackDays)

	if prev, err := q.GetPreviousSeen(ctx, userID); err == nil && prev.Valid {
		pd := time.Date(prev.Time.Year(), prev.Time.Month(), prev.Time.Day(), 0, 0, 0, 0, time.UTC)
		if pd.After(since) {
			since = pd
		}
	}

	resp := &CookReviewResponse{Since: since.Format(dateLayout), Meals: []CookReviewMeal{}}
	if since.After(until) {
		return resp, nil // window is empty (e.g. a second login the same day)
	}

	rows, err := q.ListCookReviewCandidates(ctx, sqlc.ListCookReviewCandidatesParams{
		HouseholdID: pgtype.Int4{Int32: householdID, Valid: householdID != 0},
		UserID:      pgtype.Int4{Int32: userID, Valid: true},
		WeekStart:   pgtype.Date{Time: since, Valid: true, InfinityModifier: pgtype.Finite},
		WeekStart_2: pgtype.Date{Time: until, Valid: true, InfinityModifier: pgtype.Finite},
	})
	if err != nil {
		return nil, logger.WithStack(err)
	}

	for _, r := range rows {
		m := CookReviewMeal{
			DayName:         r.DayName,
			DefaultPortions: r.DefaultPortions,
			Scope:           "personal",
		}
		if r.CookDate.Valid {
			m.CookDate = r.CookDate.Time.Format(dateLayout)
		}
		if r.MealID.Valid {
			m.MealID = r.MealID.Int32
		}
		m.MealName = r.MealName
		if r.HouseholdID.Valid {
			m.Scope = "household"
			m.HouseholdID = r.HouseholdID.Int32
		}
		resp.Meals = append(resp.Meals, m)
	}
	return resp, nil
}

// confirmCookReview records each answer and decrements the pantry for meals the
// user confirms they made. Answers are logged so they are not asked again.
func confirmCookReview(ctx context.Context, db *pgxpool.Pool, userID int32, input CookReviewConfirmInput) (*CookReviewConfirmResult, error) {
	q := sqlc.New(db)
	cooked := 0

	for _, c := range input.Confirmations {
		day, err := time.Parse(dateLayout, c.CookDate)
		if err != nil {
			return nil, httpx.NewClientError(err)
		}
		cookDate := pgtype.Date{Time: day, Valid: true, InfinityModifier: pgtype.Finite}
		hid, uid := planScope(userID, c.HouseholdID, c.Scope)

		if c.Made {
			if err := pantry.CookMeal(ctx, db, userID, pantry.CookMealInput{
				MealID:      c.MealID,
				Portions:    c.Portions,
				Scope:       c.Scope,
				HouseholdID: c.HouseholdID,
			}); err != nil {
				return nil, logger.WithStack(err)
			}
			cooked++
		}

		if err := q.UpsertMealCookLog(ctx, sqlc.UpsertMealCookLogParams{
			MealID:      c.MealID,
			CookDate:    cookDate,
			Made:        c.Made,
			HouseholdID: hid,
			UserID:      uid,
			AnsweredBy:  pgtype.Int4{Int32: userID, Valid: true},
		}); err != nil {
			return nil, logger.WithStack(err)
		}
	}

	return &CookReviewConfirmResult{Status: "recorded", Cooked: cooked}, nil
}
