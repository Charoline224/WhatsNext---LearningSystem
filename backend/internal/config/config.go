package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppEnv             string        `env:"APP_ENV" envDefault:"development"`
	LogLevelName       string        `env:"LOG_LEVEL" envDefault:"info"`
	HTTPHost           string        `env:"HTTP_HOST" envDefault:"0.0.0.0"`
	HTTPPort           int           `env:"HTTP_PORT" envDefault:"8080"`
	HTTPReadTimeout    time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"2m"`
	HTTPWriteTimeout   time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"2m"`
	FrontendOrigin     string        `env:"FRONTEND_ORIGIN" envDefault:"http://localhost:5173"`
	MySQLDSN           string        `env:"MYSQL_DSN,required"`
	RedisAddress       string        `env:"REDIS_ADDRESS" envDefault:"localhost:6379"`
	RedisPassword      string        `env:"REDIS_PASSWORD"`
	MinIOEndpoint      string        `env:"MINIO_ENDPOINT" envDefault:"localhost:9000"`
	MinIOAccessKey     string        `env:"MINIO_ACCESS_KEY" envDefault:"whatsnext"`
	MinIOSecretKey     string        `env:"MINIO_SECRET_KEY,required"`
	MinIOBucket        string        `env:"MINIO_BUCKET" envDefault:"whatsnext-materials"`
	MinIOSecure        bool          `env:"MINIO_SECURE" envDefault:"false"`
	MaxUploadBytes     int64         `env:"MAX_UPLOAD_BYTES" envDefault:"104857600"`
	JobQueueName       string        `env:"JOB_QUEUE_NAME" envDefault:"whatsnext:jobs:material"`
	WorkerPollDelay    time.Duration `env:"WORKER_POLL_DELAY" envDefault:"5s"`
	MaxPDFPages        int           `env:"MAX_PDF_PAGES" envDefault:"500"`
	PDFOCREnabled      bool          `env:"PDF_OCR_ENABLED" envDefault:"true"`
	PDFOCRLanguages    string        `env:"PDF_OCR_LANGUAGES" envDefault:"chi_sim+eng"`
	PDFOCRDPI          int           `env:"PDF_OCR_DPI" envDefault:"200"`
	PDFOCRWorkers      int           `env:"PDF_OCR_WORKERS" envDefault:"4"`
	PDFOCRPageTimeout  time.Duration `env:"PDF_OCR_PAGE_TIMEOUT" envDefault:"45s"`
	ChunkTargetRunes   int           `env:"CHUNK_TARGET_RUNES" envDefault:"1200"`
	ChunkOverlapRunes  int           `env:"CHUNK_OVERLAP_RUNES" envDefault:"120"`
	QdrantURL          string        `env:"QDRANT_URL" envDefault:"http://localhost:6333"`
	QdrantAPIKey       string        `env:"QDRANT_API_KEY"`
	QdrantCollection   string        `env:"QDRANT_COLLECTION" envDefault:"material_chunks"`
	QdrantTimeout      time.Duration `env:"QDRANT_TIMEOUT" envDefault:"10s"`
	JWTSigningKey      string        `env:"JWT_SIGNING_KEY,required"`
	JWTIssuer          string        `env:"JWT_ISSUER" envDefault:"whatsnext"`
	AccessTokenTTL     time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL    time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"720h"`
	BailianBaseURL     string        `env:"BAILIAN_BASE_URL"`
	BailianAPIKey      string        `env:"BAILIAN_API_KEY"`
	GenerationModel    string        `env:"AI_GENERATION_MODEL" envDefault:"qwen3.7-plus"`
	CheapModel         string        `env:"AI_CHEAP_MODEL" envDefault:"qwen3.6-flash"`
	GenerationTimeout  time.Duration `env:"AI_GENERATION_TIMEOUT" envDefault:"5m"`
	EmbeddingModel     string        `env:"AI_EMBEDDING_MODEL" envDefault:"text-embedding-v4"`
	EmbeddingDimension int           `env:"AI_EMBEDDING_DIMENSIONS" envDefault:"1024"`
	EmbeddingBatchSize int           `env:"AI_EMBEDDING_BATCH_SIZE" envDefault:"10"`
	EmbeddingTimeout   time.Duration `env:"AI_EMBEDDING_TIMEOUT" envDefault:"30s"`
	MockEmbedding      bool          `env:"AI_MOCK_EMBEDDING" envDefault:"false"`
	MockGeneration     bool          `env:"AI_MOCK_GENERATION" envDefault:"false"`
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}
	if len(cfg.JWTSigningKey) < 32 {
		return Config{}, fmt.Errorf("JWT_SIGNING_KEY must contain at least 32 characters")
	}
	if cfg.AccessTokenTTL <= 0 || cfg.RefreshTokenTTL <= 0 {
		return Config{}, fmt.Errorf("token TTL values must be positive")
	}
	if cfg.MaxUploadBytes <= 0 || cfg.WorkerPollDelay <= 0 {
		return Config{}, fmt.Errorf("upload limit and worker poll delay must be positive")
	}
	if cfg.MaxPDFPages <= 0 || cfg.ChunkTargetRunes <= 0 || cfg.ChunkOverlapRunes < 0 || cfg.ChunkOverlapRunes >= cfg.ChunkTargetRunes {
		return Config{}, fmt.Errorf("invalid parser or chunk configuration")
	}
	if cfg.PDFOCREnabled && (strings.TrimSpace(cfg.PDFOCRLanguages) == "" || cfg.PDFOCRDPI <= 0 || cfg.PDFOCRWorkers <= 0 || cfg.PDFOCRPageTimeout <= 0) {
		return Config{}, fmt.Errorf("invalid PDF OCR configuration")
	}
	if strings.TrimSpace(cfg.QdrantCollection) == "" || cfg.QdrantTimeout <= 0 {
		return Config{}, fmt.Errorf("invalid qdrant configuration")
	}
	if strings.TrimSpace(cfg.EmbeddingModel) == "" || cfg.EmbeddingDimension <= 0 || cfg.EmbeddingBatchSize <= 0 || cfg.EmbeddingTimeout <= 0 {
		return Config{}, fmt.Errorf("invalid embedding configuration")
	}
	if !cfg.MockGeneration && (strings.TrimSpace(cfg.GenerationModel) == "" || strings.TrimSpace(cfg.CheapModel) == "" || cfg.GenerationTimeout <= 0) {
		return Config{}, fmt.Errorf("invalid generation configuration")
	}
	if cfg.AppEnv == "production" && (cfg.MockEmbedding || cfg.MockGeneration) {
		return Config{}, fmt.Errorf("mock AI providers must not be enabled in production")
	}
	return cfg, nil
}

func (c Config) HTTPAddress() string {
	return fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort)
}

func (c Config) LogLevel() slog.Level {
	switch strings.ToLower(c.LogLevelName) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
