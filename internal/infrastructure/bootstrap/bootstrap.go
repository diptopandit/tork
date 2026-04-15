package bootstrap

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/infrastructure/db"
	"github.com/diptopandit/tork/internal/infrastructure/repository"
	"github.com/diptopandit/tork/internal/infrastructure/search"
)

// Services bundles the application services and DB connection for cleanup.
type Services struct {
	TaskSvc *application.TaskService
	ListSvc *application.ListService
	DB      *sql.DB
}

// Init opens the database (SQLite or MySQL based on remoteName), runs
// migrations, creates repos/services, and returns them bundled.
// remoteName should be "local" (or empty) for SQLite, or a key in cfg.Remotes.
// password is required for remote connections (ignored for local).
func Init(cfg *config.Config, remoteName string, password string) (*Services, error) {
	rc := cfg.ActiveRemote(remoteName)
	if rc != nil {
		return initMySQL(cfg, remoteName, rc, password)
	}
	return initSQLite(cfg)
}

func initSQLite(cfg *config.Config) (*Services, error) {
	dataDir, err := cfg.ResolveDataDir()
	if err != nil {
		return nil, fmt.Errorf("data dir: %w", err)
	}

	conn, err := db.Open(dataDir)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	if err := db.Migrate(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	taskRepo := repository.NewTaskRepo(conn)
	listRepo := repository.NewListRepo(conn)
	updateRepo := repository.NewUpdateRepo(conn)
	ftsSvc := search.NewFTSSearch(conn)

	return &Services{
		TaskSvc: application.NewTaskService(taskRepo, updateRepo, ftsSvc),
		ListSvc: application.NewListService(listRepo),
		DB:      conn,
	}, nil
}

func initMySQL(cfg *config.Config, remoteName string, rc *config.RemoteConfig, password string) (*Services, error) {
	dsn := rc.BuildDSN(password)
	conn, err := db.OpenMySQL(dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql: %w", err)
	}
	if err := db.MigrateMySQL(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("mysql migrate: %w", err)
	}

	// Username is the unique user identity on a shared server.
	userRepo := repository.NewUserRepoMySQL(conn)
	if err := userRepo.EnsureUser(&domain.User{
		ID:        rc.Username,
		Username:  rc.Username,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ensure user: %w", err)
	}

	taskRepo := repository.NewTaskRepoMySQL(conn, rc.Username)
	listRepo := repository.NewListRepoMySQL(conn, rc.Username)
	updateRepo := repository.NewUpdateRepoMySQL(conn)
	searchSvc := search.NewMySQLSearch(conn)

	return &Services{
		TaskSvc: application.NewTaskService(taskRepo, updateRepo, searchSvc),
		ListSvc: application.NewListService(listRepo),
		DB:      conn,
	}, nil
}
