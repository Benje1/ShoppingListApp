package integration_test

// main_test.go — entry point and shared helpers for the integration suite.
//
// If DATABASE_URL is not set the entire package is skipped, so the unit CI
// job continues to work without any infrastructure.
//
// When DATABASE_URL is set TestMain:
//   1. Connects to Postgres and verifies the connection with a ping.
//   2. Runs all pending migrations.
//   3. Seeds read-only catalogue items (SeedItems) and a reference user
//      (SeedUser) that read-only tests can borrow.
//   4. Closes the pool after all tests have run.
//
// Test independence: every test that writes data calls makeUser/makeHousehold/
// makeMeal to obtain its own isolated rows. SeedItems are never mutated.
// Because the Docker DB is ephemeral no cleanup is needed.

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Shared state ──────────────────────────────────────────────────────────────

var (
	testPool *pgxpool.Pool

	// SeedUser is a read-only reference user for tests that only need to look
	// something up. Tests that write should call makeUser(t) instead.
	SeedUser struct {
		ID       int32
		Username string
		Password string
	}

	// SeedItems are read-only catalogue entries referenced by ID in tests.
	// No test ever mutates these rows.
	SeedItems struct {
		Milk      int32
		Bread     int32
		Eggs      int32
		Butter    int32
		Cheese    int32
		Pasta     int32
		Rice      int32
		Oats      int32
		Flour     int32
		Chickpeas int32
		Lentils   int32
	}
)

// TestPool returns the shared pool. Tests must not close it.
func sharedPool() *pgxpool.Pool { return testPool }

// uniqueUsername returns a name that won't collide between concurrent tests.
func uniqueUsername(base string) string {
	return fmt.Sprintf("%s_%d", base, time.Now().UnixNano())
}

// makeUser creates an isolated user for a single test.
func makeUser(t *testing.T) (id int32, username, password string) {
	t.Helper()
	username = uniqueUsername("user")
	password = "test_password"
	hash, err := authentication.HashPassword(password)
	if err != nil {
		t.Fatalf("makeUser HashPassword: %v", err)
	}
	u, err := (&database.PostgresUserRepo{DB: testPool}).InsertUser(
		context.Background(), "Test User", username, hash,
	)
	if err != nil {
		t.Fatalf("makeUser InsertUser: %v", err)
	}
	return u.ID, username, password
}

// makeHousehold creates an isolated household and adds ownerID as its member.
func makeHousehold(t *testing.T, ownerID int32) int32 {
	t.Helper()
	ctx := context.Background()
	h, err := (&database.PostgresHouseholdRepo{DB: testPool}).InsertHousehold(ctx, 2, "Test Household")
	if err != nil {
		t.Fatalf("makeHousehold InsertHousehold: %v", err)
	}
	if err := (&database.PostgresUserRepo{DB: testPool}).AddUserToHousehold(ctx, ownerID, h.HouseholdID); err != nil {
		t.Fatalf("makeHousehold AddUserToHousehold: %v", err)
	}
	return h.HouseholdID
}

// makeMeal creates an isolated meal catalogue entry for a single test.
func makeMeal(t *testing.T, name string) int32 {
	t.Helper()
	meal, err := sqlc.New(testPool).CreateMeal(context.Background(), sqlc.CreateMealParams{
		Name:            fmt.Sprintf("%s_%d", name, time.Now().UnixNano()),
		DefaultPortions: 2,
	})
	if err != nil {
		t.Fatalf("makeMeal %q: %v", name, err)
	}
	return meal.ID
}

// ── TestMain ──────────────────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	if os.Getenv("DATABASE_URL") == "" {
		fmt.Println("DATABASE_URL not set — skipping integration tests")
		os.Exit(0)
	}

	ctx := context.Background()

	pool, err := database.Conn(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: failed to connect: %v\n", err)
		os.Exit(1)
	}
	testPool = pool
	defer testPool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "integration: ping failed — is the DB running?\n  %v\n", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "integration: migration failed: %v\n", err)
		os.Exit(1)
	}

	if err := seed(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "integration: seed failed: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func seed(ctx context.Context) error {
	urepo := &database.PostgresUserRepo{DB: testPool}
	q := sqlc.New(testPool)

	SeedUser.Username = "seed_user"
	SeedUser.Password = "seed_password"
	hash, err := authentication.HashPassword(SeedUser.Password)
	if err != nil {
		return fmt.Errorf("hashing seed password: %w", err)
	}
	user, err := urepo.InsertUser(ctx, "Seed User", SeedUser.Username, hash)
	if err != nil {
		return fmt.Errorf("inserting seed user: %w", err)
	}
	SeedUser.ID = user.ID

	type itemDef struct {
		name     string
		itemType sqlc.ShoppingItemType
		dest     *int32
	}
	for _, item := range []itemDef{
		{"Seed Milk", sqlc.ShoppingItemTypeDairy, &SeedItems.Milk},
		{"Seed Bread", sqlc.ShoppingItemTypeBakery, &SeedItems.Bread},
		{"Seed Eggs", sqlc.ShoppingItemTypeDairy, &SeedItems.Eggs},
		{"Seed Butter", sqlc.ShoppingItemTypeDairy, &SeedItems.Butter},
		{"Seed Cheese", sqlc.ShoppingItemTypeDairy, &SeedItems.Cheese},
		{"Seed Pasta", sqlc.ShoppingItemTypePantry, &SeedItems.Pasta},
		{"Seed Rice", sqlc.ShoppingItemTypePantry, &SeedItems.Rice},
		{"Seed Oats", sqlc.ShoppingItemTypePantry, &SeedItems.Oats},
		{"Seed Flour", sqlc.ShoppingItemTypePantry, &SeedItems.Flour},
		{"Seed Chickpeas", sqlc.ShoppingItemTypePantry, &SeedItems.Chickpeas},
		{"Seed Lentils", sqlc.ShoppingItemTypePantry, &SeedItems.Lentils},
	} {
		created, err := q.CreateShoppingItem(ctx, sqlc.CreateShoppingItemParams{
			Name: item.name, ItemType: item.itemType, PortionsPerUnit: 1,
		})
		if err != nil {
			return fmt.Errorf("inserting seed item %q: %w", item.name, err)
		}
		*item.dest = created.ID
	}
	return nil
}
