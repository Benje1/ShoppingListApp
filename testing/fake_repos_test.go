package authntication_test

// This file collects all fake/in-memory repository implementations used across
// the integration-style tests in this package. Each fake stores state in maps
// so tests can run without a real database.

import (
	"context"
	"errors"
	"sync"
	"time"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// ── FakeShoppingListRepo ──────────────────────────────────────────────────────

// FakeShoppingListRepo simulates the shopping list storage layer.
// Entries are keyed by their ID; the fake auto-increments IDs.
type FakeShoppingListRepo struct {
	mu      sync.Mutex
	entries map[int32]sqlc.ShoppingList
	nextID  int32
	haveIt  map[int32]bool // item_id -> have-it flag
}

func NewFakeShoppingListRepo() *FakeShoppingListRepo {
	return &FakeShoppingListRepo{
		entries: make(map[int32]sqlc.ShoppingList),
		nextID:  1,
		haveIt:  make(map[int32]bool),
	}
}

// Add upserts an item onto the list, accumulating quantity on conflict (mirrors
// the real SQL ON CONFLICT DO UPDATE behaviour).
func (f *FakeShoppingListRepo) Add(_ context.Context, itemID, quantity int32, householdID, userID pgtype.Int4) (sqlc.ShoppingList, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check for an existing entry with the same item + scope
	for id, e := range f.entries {
		if e.ShoppingItemID == itemID && e.HouseholdID == householdID && e.UserID == userID {
			e.Quantity += quantity
			f.entries[id] = e
			return e, nil
		}
	}

	id := f.nextID
	f.nextID++
	entry := sqlc.ShoppingList{
		ID:             id,
		ShoppingItemID: itemID,
		Quantity:       quantity,
		HouseholdID:    householdID,
		UserID:         userID,
	}
	f.entries[id] = entry
	return entry, nil
}

// Remove deletes a list entry by its ID.
func (f *FakeShoppingListRepo) Remove(_ context.Context, id int32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.entries[id]; !ok {
		return errors.New("entry not found")
	}
	delete(f.entries, id)
	return nil
}

// List returns all entries matching either the household or user scope.
func (f *FakeShoppingListRepo) List(_ context.Context, householdID, userID pgtype.Int4) []sqlc.ShoppingList {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []sqlc.ShoppingList
	for _, e := range f.entries {
		if (householdID.Valid && e.HouseholdID == householdID) ||
			(userID.Valid && e.UserID == userID) {
			out = append(out, e)
		}
	}
	return out
}

// MarkHaveIt records that the user already has an item.
func (f *FakeShoppingListRepo) MarkHaveIt(_ context.Context, itemID int32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.haveIt[itemID] = true
	return nil
}

// UnmarkHaveIt removes the have-it flag for an item.
func (f *FakeShoppingListRepo) UnmarkHaveIt(_ context.Context, itemID int32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.haveIt, itemID)
	return nil
}

// HasItem returns whether the given itemID is in the have-it set.
func (f *FakeShoppingListRepo) HasItem(itemID int32) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.haveIt[itemID]
}

// ── FakePantryRepo ────────────────────────────────────────────────────────────

// FakePantryItem mirrors the fields used in pantry business logic tests.
type FakePantryItem struct {
	ID                int32
	ShoppingItemID    int32
	HouseholdID       pgtype.Int4
	UserID            pgtype.Int4
	PortionsRemaining float64
	ExpiresOn         pgtype.Date
	Status            string
}

// FakePantryRepo simulates pantry storage.
type FakePantryRepo struct {
	mu     sync.Mutex
	items  map[int32]FakePantryItem
	nextID int32
}

func NewFakePantryRepo() *FakePantryRepo {
	return &FakePantryRepo{
		items:  make(map[int32]FakePantryItem),
		nextID: 1,
	}
}

// Upsert adds portions to an existing pantry entry or creates a new one.
func (f *FakePantryRepo) Upsert(_ context.Context, itemID int32, portions float64, householdID, userID pgtype.Int4, expiresOn pgtype.Date) FakePantryItem {
	f.mu.Lock()
	defer f.mu.Unlock()

	for id, p := range f.items {
		if p.ShoppingItemID == itemID && p.HouseholdID == householdID && p.UserID == userID {
			p.PortionsRemaining += portions
			p.ExpiresOn = expiresOn
			p.Status = "fresh"
			f.items[id] = p
			return p
		}
	}

	id := f.nextID
	f.nextID++
	item := FakePantryItem{
		ID:                id,
		ShoppingItemID:    itemID,
		HouseholdID:       householdID,
		UserID:            userID,
		PortionsRemaining: portions,
		ExpiresOn:         expiresOn,
		Status:            "fresh",
	}
	f.items[id] = item
	return item
}

// Remove deletes a pantry entry by its ID.
func (f *FakePantryRepo) Remove(_ context.Context, id int32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.items[id]; !ok {
		return errors.New("pantry entry not found")
	}
	delete(f.items, id)
	return nil
}

// Decrement reduces portions for an item; does nothing if item is not present.
func (f *FakePantryRepo) Decrement(_ context.Context, itemID int32, qty float64, householdID, userID pgtype.Int4) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, p := range f.items {
		if p.ShoppingItemID == itemID && p.HouseholdID == householdID && p.UserID == userID {
			p.PortionsRemaining -= qty
			if p.PortionsRemaining < 0 {
				p.PortionsRemaining = 0
			}
			f.items[id] = p
			return
		}
	}
}

// Get returns the pantry item for the given shopping item, if present.
func (f *FakePantryRepo) Get(_ context.Context, itemID int32, householdID, userID pgtype.Int4) (FakePantryItem, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.items {
		if p.ShoppingItemID == itemID && p.HouseholdID == householdID && p.UserID == userID {
			return p, true
		}
	}
	return FakePantryItem{}, false
}

// Count returns the number of entries in the pantry.
func (f *FakePantryRepo) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.items)
}

// ── FakeHouseholdRepo ─────────────────────────────────────────────────────────

// FakeHouseholdRepo simulates household and invite storage.
type FakeHouseholdRepo struct {
	mu         sync.Mutex
	households map[int32]*sqlc.Household
	invites    map[int32]*sqlc.HouseholdInvite
	members    map[int32][]int32 // householdID -> []userID
	nextHID    int32
	nextIID    int32
}

func NewFakeHouseholdRepo() *FakeHouseholdRepo {
	return &FakeHouseholdRepo{
		households: make(map[int32]*sqlc.Household),
		invites:    make(map[int32]*sqlc.HouseholdInvite),
		members:    make(map[int32][]int32),
		nextHID:    1,
		nextIID:    1,
	}
}

func (f *FakeHouseholdRepo) InsertHousehold(_ context.Context, numPeople int32, name string) (*sqlc.Household, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextHID
	f.nextHID++
	h := &sqlc.Household{
		HouseholdID: id,
		NumPeople:   numPeople,
		Name:        pgtype.Text{String: name, Valid: name != ""},
	}
	f.households[id] = h
	return h, nil
}

func (f *FakeHouseholdRepo) GetHousehold(_ context.Context, id int32) (*sqlc.Household, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	h, ok := f.households[id]
	if !ok {
		return nil, errors.New("household not found")
	}
	return h, nil
}

func (f *FakeHouseholdRepo) RenameHousehold(_ context.Context, id int32, name string) (*sqlc.Household, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	h, ok := f.households[id]
	if !ok {
		return nil, errors.New("household not found")
	}
	h.Name = pgtype.Text{String: name, Valid: true}
	return h, nil
}

func (f *FakeHouseholdRepo) DeleteHousehold(_ context.Context, id int32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.households[id]; !ok {
		return errors.New("household not found")
	}
	delete(f.households, id)
	delete(f.members, id)
	return nil
}

func (f *FakeHouseholdRepo) AddMember(_ context.Context, householdID, userID int32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.members[householdID] = append(f.members[householdID], userID)
}

func (f *FakeHouseholdRepo) GetMembers(_ context.Context, householdID int32) []int32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.members[householdID]
}

func (f *FakeHouseholdRepo) CreateInvite(_ context.Context, householdID int32, code string, requestedBy int32) (*sqlc.HouseholdInvite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextIID
	f.nextIID++
	inv := &sqlc.HouseholdInvite{
		ID:                id,
		HouseholdID:       householdID,
		InviteCode:        code,
		RequestedByUserID: requestedBy,
		Status:            "pending",
		CreatedAt:         pgtype.Timestamp{Time: time.Now(), Valid: true},
	}
	f.invites[id] = inv
	return inv, nil
}

func (f *FakeHouseholdRepo) GetInviteByCode(_ context.Context, code string) (*sqlc.HouseholdInvite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, inv := range f.invites {
		if inv.InviteCode == code {
			return inv, nil
		}
	}
	return nil, errors.New("invite not found")
}

func (f *FakeHouseholdRepo) GetInviteByID(_ context.Context, id int32) (*sqlc.HouseholdInvite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	inv, ok := f.invites[id]
	if !ok {
		return nil, errors.New("invite not found")
	}
	return inv, nil
}

func (f *FakeHouseholdRepo) RespondToInvite(_ context.Context, id int32, status string) (*sqlc.HouseholdInvite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	inv, ok := f.invites[id]
	if !ok {
		return nil, errors.New("invite not found")
	}
	inv.Status = status
	return inv, nil
}

// InviteCount returns the total number of stored invites (useful in assertions).
func (f *FakeHouseholdRepo) InviteCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.invites)
}
