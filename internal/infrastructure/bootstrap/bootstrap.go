package bootstrap

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

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

// Init opens the database (SQLite or MySQL based on config), runs migrations,
// creates repos/services, and returns them bundled.
func Init(cfg *config.Config) (*Services, error) {
	if cfg.RemoteDB != nil && cfg.RemoteDB.DSN != "" {
		return initMySQL(cfg)
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

func initMySQL(cfg *config.Config) (*Services, error) {
	conn, err := db.OpenMySQL(cfg.RemoteDB.DSN)
	if err != nil {
		return nil, fmt.Errorf("mysql: %w", err)
	}
	if err := db.MigrateMySQL(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("mysql migrate: %w", err)
	}

	// Ensure user exists.
	userID := cfg.UserID
	username := cfg.Username
	if userID == "" {
		userID = uuid.NewString()
		cfg.UserID = userID
		_ = config.Save(cfg) // persist generated user ID
	}
	if username == "" {
		username = "user-" + userID[:8]
		cfg.Username = username
		_ = config.Save(cfg)
	}

	userRepo := repository.NewUserRepoMySQL(conn)
	if err := userRepo.EnsureUser(&domain.User{
		ID:        userID,
		Username:  username,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ensure user: %w", err)
	}

	taskRepo := repository.NewTaskRepoMySQL(conn, userID)
	listRepo := repository.NewListRepoMySQL(conn, userID)
	updateRepo := repository.NewUpdateRepoMySQL(conn)
	searchSvc := search.NewMySQLSearch(conn)

	return &Services{
		TaskSvc: application.NewTaskService(taskRepo, updateRepo, searchSvc),
		ListSvc: application.NewListService(listRepo),
		DB:      conn,
	}, nil
}
