package bootstrap

import (
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/infrastructure/db"
	"github.com/diptopandit/tork/internal/infrastructure/logger"
	"github.com/diptopandit/tork/internal/infrastructure/repository"
	"github.com/diptopandit/tork/internal/infrastructure/search"
)

// Services bundles the application services and DB connection for cleanup.
type Services struct {
	TaskSvc *application.TaskService
	ListSvc *application.ListService
	DB      *sql.DB
	Log     logger.Logger
}

// Init opens the database (SQLite or MySQL based on remoteName), runs
// migrations, creates repos/services, and returns them bundled.
// remoteName should be "local" (or empty) for SQLite, or a key in cfg.Remotes.
// password is required for remote connections (ignored for local).
// log may be nil; a nop logger will be used in that case.
func Init(cfg *config.Config, remoteName string, password string, log logger.Logger) (*Services, error) {
	if log == nil {
		log = logger.Nop()
	}
	rc := cfg.ActiveRemote(remoteName)
	if rc != nil {
		return initMySQL(remoteName, rc, password, log)
	}
	return initSQLite(cfg, log)
}

func initSQLite(cfg *config.Config, log logger.Logger) (*Services, error) {
	dataDir, err := cfg.ResolveDataDir()
	if err != nil {
		return nil, fmt.Errorf("data dir: %w", err)
	}

	log.Info("opening sqlite database", zap.String("dir", dataDir))
	conn, err := db.Open(dataDir)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	if err := db.Migrate(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	log.Info("sqlite ready")

	taskRepo := repository.NewTaskRepo(conn)
	listRepo := repository.NewListRepo(conn)
	updateRepo := repository.NewUpdateRepo(conn)
	ftsSvc := search.NewFTSSearch(conn)

	return &Services{
		TaskSvc: application.NewTaskService(taskRepo, updateRepo, ftsSvc, log),
		ListSvc: application.NewListService(listRepo, log),
		DB:      conn,
		Log:     log,
	}, nil
}

func initMySQL(remoteName string, rc *config.RemoteConfig, password string, log logger.Logger) (*Services, error) {
	log.Info("connecting to mysql",
		zap.String("remote", remoteName),
		zap.String("host", rc.Host),
		zap.Int("port", rc.EffectivePort()),
		zap.String("database", rc.EffectiveDatabase()),
		zap.String("user", rc.Username),
	)
	dsn := rc.BuildDSN(password)
	conn, err := db.OpenMySQL(dsn)
	if err != nil {
		log.Error("mysql connection failed", zap.String("remote", remoteName), zap.Error(err))
		return nil, fmt.Errorf("mysql: %w", err)
	}
	if err := db.MigrateMySQL(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("mysql migrate: %w", err)
	}
	log.Info("mysql ready", zap.String("remote", remoteName))

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
		TaskSvc: application.NewTaskService(taskRepo, updateRepo, searchSvc, log),
		ListSvc: application.NewListService(listRepo, log),
		DB:      conn,
		Log:     log,
	}, nil
}
