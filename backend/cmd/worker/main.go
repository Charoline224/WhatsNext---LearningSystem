package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"whatsnext/backend/internal/ai"
	"whatsnext/backend/internal/config"
	"whatsnext/backend/internal/database"
	"whatsnext/backend/internal/job"
	"whatsnext/backend/internal/materialparser"
	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/queue"
	"whatsnext/backend/internal/rag"
	"whatsnext/backend/internal/repository"
	"whatsnext/backend/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel()}))
	db, err := database.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		logger.Error("connect mysql", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	objectStorage, err := storage.NewMinIO(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, cfg.MinIOBucket, cfg.MinIOSecure)
	if err != nil {
		logger.Error("create storage", "error", err)
		os.Exit(1)
	}
	jobQueue := queue.NewRedisQueue(cfg.RedisAddress, cfg.RedisPassword, cfg.JobQueueName)
	defer jobQueue.Close()
	// 创建根context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err = objectStorage.EnsureBucket(ctx); err != nil {
		logger.Error("ensure bucket", "error", err)
		os.Exit(1)
	}
	if err = jobQueue.Ping(ctx); err != nil {
		logger.Error("connect redis", "error", err)
		os.Exit(1)
	}
	repo := repository.NewMaterialRepository(db)
	embeddingRepo := repository.NewChunkEmbeddingRepository(db)
	assetRepo := repository.NewLearningAssetRepository(db)
	spaceRepo := repository.NewLearningSpaceRepository(db)
	recoveredJobs, err := repo.RequeueInterruptedJobs(ctx)
	if err != nil {
		logger.Error("recover interrupted material jobs", "error", err)
		os.Exit(1)
	}
	if recoveredJobs > 0 {
		logger.Info("requeued interrupted material jobs", "count", recoveredJobs)
	}
	// 创建AI组件
	var assetGenerator ai.KnowledgeAssetGenerator = ai.UnavailableAssetGenerator{}
	var planGenerator ai.LearningPlanGenerator = ai.UnavailableAssetGenerator{}
	if cfg.MockGeneration {
		assetGenerator = ai.MockAssetGenerator{}
		planGenerator = ai.MockAssetGenerator{}
	} else if cfg.BailianBaseURL != "" && cfg.BailianAPIKey != "" {
		generator, createErr := ai.NewBailianChat(cfg.BailianBaseURL, cfg.BailianAPIKey, cfg.GenerationModel, cfg.CheapModel, cfg.GenerationTimeout)
		if createErr != nil {
			logger.Error("create generation provider", "error", createErr)
			os.Exit(1)
		}
		assetGenerator = generator
		planGenerator = generator
	}
	assetProcessor := job.NewLearningAssetProcessor(assetRepo, spaceRepo, assetGenerator, planGenerator)
	var examAnalyzer ai.ExamAnalyzer = ai.UnavailableExamAnalyzer{}
	if cfg.MockGeneration {
		examAnalyzer = ai.MockExamAnalyzer{}
	} else if cfg.BailianBaseURL != "" && cfg.BailianAPIKey != "" {
		generator, createErr := ai.NewBailianChat(cfg.BailianBaseURL, cfg.BailianAPIKey, cfg.GenerationModel, cfg.CheapModel, cfg.GenerationTimeout)
		if createErr != nil {
			logger.Error("create exam analysis provider", "error", createErr)
			os.Exit(1)
		}
		examAnalyzer = generator
	}
	parser := materialparser.New(cfg.MaxPDFPages)
	if cfg.PDFOCREnabled {
		ocr, createErr := materialparser.NewTesseractOCR(cfg.PDFOCRLanguages, cfg.PDFOCRDPI, cfg.PDFOCRWorkers, cfg.PDFOCRPageTimeout)
		if createErr != nil {
			logger.Error("create PDF OCR provider", "error", createErr)
			os.Exit(1)
		}
		parser = materialparser.New(cfg.MaxPDFPages, ocr)
	}
	materialProcessor := job.NewMaterialProcessor(repo, objectStorage, parser, materialparser.NewChunker(cfg.ChunkTargetRunes, cfg.ChunkOverlapRunes), cfg.EmbeddingModel, cfg.EmbeddingDimension, examAnalyzer)
	var embeddingProcessor *job.EmbeddingProcessor
	if cfg.MockEmbedding || (cfg.BailianBaseURL != "" && cfg.BailianAPIKey != "") {
		var embedder ai.Embedder
		var createErr error
		if cfg.MockEmbedding {
			embedder, createErr = ai.NewMockEmbedding(cfg.EmbeddingDimension)
		} else {
			embedder, createErr = ai.NewBailianEmbedding(cfg.BailianBaseURL, cfg.BailianAPIKey, cfg.EmbeddingModel, cfg.EmbeddingDimension, cfg.EmbeddingTimeout)
		}
		if createErr != nil {
			logger.Error("create embedding provider", "error", createErr)
			os.Exit(1)
		}
		vectorStore, createErr := rag.NewQdrant(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.QdrantCollection, cfg.EmbeddingDimension, cfg.QdrantTimeout)
		if createErr != nil {
			logger.Error("create qdrant client", "error", createErr)
			os.Exit(1)
		}
		if createErr = vectorStore.EnsureCollection(ctx); createErr != nil {
			logger.Error("ensure qdrant collection", "error", createErr)
			os.Exit(1)
		}
		embeddingProcessor = job.NewEmbeddingProcessor(repo, embeddingRepo, embedder, vectorStore, cfg.EmbeddingModel, cfg.EmbeddingBatchSize)
	} else {
		logger.Warn("embedding indexing disabled until AI_MOCK_EMBEDDING or Bailian credentials are configured")
	}
	logger.Info("worker started", "environment", cfg.AppEnv, "queue", cfg.JobQueueName)
	// 循环处理队列里的任务
	for ctx.Err() == nil {
		jobID, dequeueErr := jobQueue.Dequeue(ctx, 2*time.Second)
		if dequeueErr != nil && ctx.Err() == nil {
			logger.Error("dequeue job", "error", dequeueErr)
			time.Sleep(cfg.WorkerPollDelay)
			continue
		}
		if jobID == "" {
			jobID, err = repo.NextQueuedJobID(ctx)
			if err != nil {
				logger.Error("recover queued job", "error", err)
				time.Sleep(cfg.WorkerPollDelay)
				continue
			}
		}
		if jobID == "" {
			jobID, err = assetRepo.NextQueuedJobID(ctx)
			if err != nil {
				logger.Error("recover queued asset job", "error", err)
				time.Sleep(cfg.WorkerPollDelay)
				continue
			}
		}
		if jobID == "" {
			continue
		}

		queuedJob, getErr := repo.GetJobByID(ctx, jobID)
		if getErr == repository.ErrNotFound {
			if _, assetErr := assetRepo.GetJobByID(ctx, jobID); assetErr != nil {
				logger.Error("load dequeued job", "job_id", jobID, "error", assetErr)
				continue
			}
			err = assetProcessor.Process(ctx, jobID)
			if err != nil {
				logger.Error("process learning asset job", "job_id", jobID, "error", err)
			} else {
				logger.Info("learning asset job processed", "job_id", jobID)
			}
			continue
		}
		if getErr != nil {
			logger.Error("load dequeued job", "job_id", jobID, "error", getErr)
			continue
		}
		switch queuedJob.JobType {
		case "process_material":
			var followUpID string
			followUpID, err = materialProcessor.Process(ctx, jobID)
			if err == nil && followUpID != "" {
				if enqueueErr := jobQueue.Enqueue(ctx, followUpID); enqueueErr != nil {
					logger.Warn("embedding job persisted but queue notification failed", "job_id", followUpID, "error", enqueueErr)
				}
			}
		case "embed_material":
			if embeddingProcessor == nil {
				err = failUnavailableEmbedding(ctx, repo, embeddingRepo, queuedJob, cfg.EmbeddingModel)
			} else {
				err = embeddingProcessor.Process(ctx, jobID)
			}
		default:
			err = fmt.Errorf("unsupported job type %s", queuedJob.JobType)
		}
		if err != nil {
			logger.Error("process material job", "job_id", jobID, "error", err)
		} else {
			logger.Info("material job processed", "job_id", jobID)
		}
	}
	logger.Info("worker stopped")
}

func failUnavailableEmbedding(ctx context.Context, jobs *repository.MaterialRepository, embeddings *repository.ChunkEmbeddingRepository, queued model.GenerationJob, embeddingModel string) error {
	claimed, ok, err := jobs.ClaimJob(ctx, queued.ID)
	if err != nil || !ok {
		return err
	}
	message := "embedding provider is not configured"
	if err = jobs.FailJob(ctx, claimed, "EMBEDDING_NOT_CONFIGURED", message); err != nil {
		return err
	}
	if claimed.Attempts >= claimed.MaxAttempts {
		if markErr := embeddings.MarkMaterialFailed(ctx, claimed.MaterialID, embeddingModel, message); markErr != nil {
			return markErr
		}
	}
	return fmt.Errorf("%s", message)
}
