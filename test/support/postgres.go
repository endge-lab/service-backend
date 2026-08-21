//go:build integration || e2e

// Package support содержит только инфраструктуру автоматических тестов сервиса.
package support

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	testDatabasePrefix = "endge_test_"
	guardSchema        = "endge_test_guard"
	guardTable         = "database_marker"
)

// PostgresSuite владеет единственным контейнером PostgreSQL тестового package.
// Она принципиально не принимает DSN извне.
type PostgresSuite struct {
	container *tcpostgres.PostgresContainer
	adminPool *pgxpool.Pool
	adminDSN  string
}

// TestDatabase описывает уникальную БД одного интеграционного сценария.
type TestDatabase struct {
	Pool       *pgxpool.Pool
	Name       string
	DSN        string
	guardToken string
	suite      *PostgresSuite
}

// StartPostgresSuite запускает PostgreSQL 17 на случайном порту.
// Недоступный Docker считается ошибкой критического тестового набора.
func StartPostgresSuite(ctx context.Context) (*PostgresSuite, error) {
	password, err := randomHex(24)
	if err != nil {
		return nil, fmt.Errorf("создать пароль тестовой БД: %w", err)
	}
	container, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("endge_test_admin"),
		tcpostgres.WithUsername("endge_test"),
		tcpostgres.WithPassword(password),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("запустить изолированный PostgreSQL через Docker: %w", err)
	}
	adminDSN, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("получить DSN тестового PostgreSQL: %w", err)
	}
	adminPool, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("создать admin pool тестового PostgreSQL: %w", err)
	}
	if err = adminPool.Ping(ctx); err != nil {
		adminPool.Close()
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("проверить тестовый PostgreSQL: %w", err)
	}
	return &PostgresSuite{container: container, adminPool: adminPool, adminDSN: adminDSN}, nil
}

// Close закрывает admin pool и удаляет контейнер независимо от результата suite.
func (s *PostgresSuite) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.adminPool != nil {
		s.adminPool.Close()
	}
	if s.container == nil {
		return nil
	}
	return testcontainers.TerminateContainer(s.container)
}

// NewDatabase создаёт чистую БД, устанавливает marker и применяет все миграции.
// Cleanup всегда проверяет marker до DROP DATABASE.
func (s *PostgresSuite) NewDatabase(t testing.TB) *TestDatabase {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	name := testDatabasePrefix + suffix
	guardToken, err := randomHex(32)
	if err != nil {
		t.Fatalf("создать marker тестовой БД: %v", err)
	}
	if _, err = s.adminPool.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("создать изолированную БД %s: %v", name, err)
	}

	dsn, err := databaseDSN(s.adminDSN, name)
	if err != nil {
		t.Fatalf("сформировать DSN изолированной БД: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("открыть изолированную БД: %v", err)
	}
	database := &TestDatabase{Pool: pool, Name: name, DSN: dsn, guardToken: guardToken, suite: s}
	if err = database.installGuard(ctx); err != nil {
		pool.Close()
		t.Fatalf("установить safety marker: %v", err)
	}
	if err = database.MigrateUp(ctx); err != nil {
		pool.Close()
		t.Fatalf("применить миграции к чистой БД: %v", err)
	}
	t.Cleanup(func() { database.cleanup(t) })
	return database
}

// MigrateUp применяет все embedded-миграции к проверенной тестовой БД.
func (d *TestDatabase) MigrateUp(ctx context.Context) error {
	if err := d.AssertSafe(ctx); err != nil {
		return err
	}
	return withMigrationProvider(d.Pool, func(provider *goose.Provider) error {
		_, err := provider.Up(ctx)
		return err
	})
}

// MigrateDownToZero откатывает все миграции, не удаляя safety marker.
func (d *TestDatabase) MigrateDownToZero(ctx context.Context) error {
	return d.MigrateDownTo(ctx, 0)
}

// MigrateDownTo откатывает миграции до указанной версии в изолированной тестовой БД.
func (d *TestDatabase) MigrateDownTo(ctx context.Context, version int64) error {
	if err := d.AssertSafe(ctx); err != nil {
		return err
	}
	return withMigrationProvider(d.Pool, func(provider *goose.Provider) error {
		_, err := provider.DownTo(ctx, version)
		return err
	})
}

// MigrationStatus возвращает фактическое состояние всех embedded-миграций.
func (d *TestDatabase) MigrationStatus(ctx context.Context) ([]*goose.MigrationStatus, error) {
	if err := d.AssertSafe(ctx); err != nil {
		return nil, err
	}
	var result []*goose.MigrationStatus
	err := withMigrationProvider(d.Pool, func(provider *goose.Provider) error {
		var err error
		result, err = provider.Status(ctx)
		return err
	})
	return result, err
}

// AssertSafe запрещает destructive-операции при несовпадении имени БД или marker.
func (d *TestDatabase) AssertSafe(ctx context.Context) error {
	if d == nil || d.Pool == nil {
		return fmt.Errorf("тестовая БД не инициализирована")
	}
	if !strings.HasPrefix(d.Name, testDatabasePrefix) || len(d.Name) <= len(testDatabasePrefix) {
		return fmt.Errorf("небезопасное имя БД %q", d.Name)
	}
	var currentDatabase string
	if err := d.Pool.QueryRow(ctx, `SELECT current_database()`).Scan(&currentDatabase); err != nil {
		return fmt.Errorf("прочитать имя текущей БД: %w", err)
	}
	if currentDatabase != d.Name {
		return fmt.Errorf("ожидалась БД %q, подключение ведёт в %q", d.Name, currentDatabase)
	}
	var marker string
	query := `SELECT token FROM ` + pgx.Identifier{guardSchema, guardTable}.Sanitize() + ` WHERE singleton=TRUE`
	if err := d.Pool.QueryRow(ctx, query).Scan(&marker); err != nil {
		return fmt.Errorf("safety marker отсутствует: %w", err)
	}
	if marker != d.guardToken {
		return fmt.Errorf("safety marker не совпадает")
	}
	return nil
}

func (d *TestDatabase) installGuard(ctx context.Context) error {
	if !strings.HasPrefix(d.Name, testDatabasePrefix) {
		return fmt.Errorf("отказ устанавливать marker в БД %q", d.Name)
	}
	statements := []string{
		"CREATE SCHEMA " + pgx.Identifier{guardSchema}.Sanitize(),
		"CREATE TABLE " + pgx.Identifier{guardSchema, guardTable}.Sanitize() + " (singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton), token TEXT NOT NULL)",
		"INSERT INTO " + pgx.Identifier{guardSchema, guardTable}.Sanitize() + " (singleton, token) VALUES (TRUE, $1)",
	}
	for index, statement := range statements {
		var err error
		if index == len(statements)-1 {
			_, err = d.Pool.Exec(ctx, statement, d.guardToken)
		} else {
			_, err = d.Pool.Exec(ctx, statement)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *TestDatabase) cleanup(t testing.TB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := d.AssertSafe(ctx); err != nil {
		t.Errorf("отказ удалить БД без safety-проверки: %v", err)
		return
	}
	d.Pool.Close()
	_, _ = d.suite.adminPool.Exec(ctx, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname=$1 AND pid<>pg_backend_pid()`, d.Name)
	if _, err := d.suite.adminPool.Exec(ctx, "DROP DATABASE "+pgx.Identifier{d.Name}.Sanitize()); err != nil {
		t.Errorf("удалить изолированную БД %s: %v", d.Name, err)
	}
}

func withMigrationProvider(pool *pgxpool.Pool, run func(*goose.Provider) error) error {
	standardDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = standardDB.Close() }()
	provider, err := goose.NewProvider(goose.DialectPostgres, standardDB, migrations.FS)
	if err != nil {
		return fmt.Errorf("создать migration provider: %w", err)
	}
	if err = run(provider); err != nil {
		return fmt.Errorf("выполнить миграции: %w", err)
	}
	return nil
}

func databaseDSN(adminDSN, database string) (string, error) {
	parsed, err := url.Parse(adminDSN)
	if err != nil {
		return "", err
	}
	parsed.Path = "/" + database
	return parsed.String(), nil
}

func randomHex(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

// SQLDB открывает стандартный database/sql facade над pool для редких проверок.
func (d *TestDatabase) SQLDB() *sql.DB { return stdlib.OpenDBFromPool(d.Pool) }
