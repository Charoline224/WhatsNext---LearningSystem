package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"whatsnext/backend/internal/ai"
	appauth "whatsnext/backend/internal/auth"
	"whatsnext/backend/internal/config"
	"whatsnext/backend/internal/handler"
	appmiddleware "whatsnext/backend/internal/middleware"
	"whatsnext/backend/internal/queue"
	"whatsnext/backend/internal/rag"
	"whatsnext/backend/internal/repository"
	"whatsnext/backend/internal/response"
	"whatsnext/backend/internal/service"
	"whatsnext/backend/internal/storage"
)

func NewRouter(cfg config.Config, logger *slog.Logger, db *sqlx.DB) (http.Handler, error) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery(), appmiddleware.RequestID(), appmiddleware.Logger(logger), appmiddleware.CORS(cfg.FrontendOrigin))

	health := handler.NewHealthHandler(db.DB)
	router.GET("/health/live", health.Live)
	router.GET("/health/ready", health.Ready)

	tokens := appauth.NewTokenManager(cfg.JWTSigningKey, cfg.JWTIssuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authService := service.NewAuthService(repository.NewAuthRepository(db), tokens)
	authHandler := handler.NewAuthHandler(authService, cfg.AppEnv == "production", cfg.RefreshTokenTTL)
	spaceRepo := repository.NewLearningSpaceRepository(db)
	materialRepo := repository.NewMaterialRepository(db)
	embeddingRepo := repository.NewChunkEmbeddingRepository(db)
	assetRepo := repository.NewLearningAssetRepository(db)
	spaceHandler := handler.NewLearningSpaceHandler(service.NewLearningSpaceService(spaceRepo))
	objectStorage, err := storage.NewMinIO(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, cfg.MinIOBucket, cfg.MinIOSecure)
	if err != nil {
		return nil, err
	}
	jobQueue := queue.NewRedisQueue(cfg.RedisAddress, cfg.RedisPassword, cfg.JobQueueName)
	assetService := service.NewLearningAssetService(spaceRepo, assetRepo, jobQueue)
	vectorStore, err := rag.NewQdrant(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.QdrantCollection, cfg.EmbeddingDimension, cfg.QdrantTimeout)
	if err != nil {
		return nil, err
	}
	materialService := service.NewMaterialService(spaceRepo, materialRepo, objectStorage, jobQueue, cfg.MaxUploadBytes)
	materialService.SetVectorStore(vectorStore)
	materialService.SetEmbeddingModel(cfg.EmbeddingModel)
	materialHandler := handler.NewMaterialHandler(materialService, cfg.MaxUploadBytes)
	retrievalService := service.NewRetrievalService(spaceRepo, materialRepo, nil, nil)
	if cfg.MockEmbedding || (cfg.BailianBaseURL != "" && cfg.BailianAPIKey != "") {
		var retrievalEmbedder ai.Embedder
		if cfg.MockEmbedding {
			retrievalEmbedder, err = ai.NewMockEmbedding(cfg.EmbeddingDimension)
		} else {
			retrievalEmbedder, err = ai.NewBailianEmbedding(cfg.BailianBaseURL, cfg.BailianAPIKey, cfg.EmbeddingModel, cfg.EmbeddingDimension, cfg.EmbeddingTimeout)
		}
		if err != nil {
			return nil, err
		}
		retrievalService = service.NewRetrievalService(spaceRepo, materialRepo, retrievalEmbedder, vectorStore)
	}
	retrievalHandler := handler.NewRetrievalHandler(retrievalService)
	var chatGenerator ai.ChatGenerator = ai.UnavailableChatGenerator{}
	if cfg.MockGeneration {
		chatGenerator = ai.MockChatGenerator{}
	} else if cfg.BailianBaseURL != "" && cfg.BailianAPIKey != "" {
		chatGenerator, err = ai.NewBailianChat(cfg.BailianBaseURL, cfg.BailianAPIKey, cfg.GenerationModel, cfg.CheapModel, cfg.GenerationTimeout)
		if err != nil {
			return nil, err
		}
	}
	chatHandler := handler.NewChatHandler(service.NewChatService(retrievalService, chatGenerator, repository.NewChatRepository(db), assetService))
	indexStatusHandler := handler.NewIndexStatusHandler(service.NewIndexStatusService(spaceRepo, materialRepo, embeddingRepo, cfg.EmbeddingModel))
	assetHandler := handler.NewLearningAssetHandler(assetService)
	examHandler := handler.NewExamHandler(service.NewExamService(spaceRepo, repository.NewExamRepository(db), assetService))

	api := router.Group("/api/v1")
	authRoutes := api.Group("/auth")
	authRoutes.POST("/register", authHandler.Register)
	authRoutes.POST("/login", authHandler.Login)
	authRoutes.POST("/refresh", authHandler.Refresh)
	authRoutes.POST("/logout", authHandler.Logout)
	authorized := api.Group("")
	authorized.Use(appmiddleware.RequireAuth(tokens))
	authorized.GET("/users/me", authHandler.Me)
	authorized.GET("/spaces", spaceHandler.List)
	authorized.POST("/spaces", spaceHandler.Create)
	authorized.GET("/spaces/:spaceId", spaceHandler.Get)
	authorized.PATCH("/spaces/:spaceId", spaceHandler.Update)
	authorized.DELETE("/spaces/:spaceId", spaceHandler.Delete)
	authorized.GET("/spaces/:spaceId/materials", materialHandler.List)
	authorized.POST("/spaces/:spaceId/materials", materialHandler.Upload)
	authorized.GET("/spaces/:spaceId/jobs/:jobId", materialHandler.GetJob)
	authorized.GET("/spaces/:spaceId/materials/:materialId/chunks", materialHandler.ListChunks)
	authorized.GET("/spaces/:spaceId/materials/:materialId", materialHandler.Get)
	authorized.GET("/spaces/:spaceId/materials/:materialId/download-url", materialHandler.Download)
	authorized.POST("/spaces/:spaceId/materials/:materialId/retry", materialHandler.Retry)
	authorized.DELETE("/spaces/:spaceId/materials/:materialId", materialHandler.Delete)
	authorized.GET("/spaces/:spaceId/materials/:materialId/index-status", indexStatusHandler.Get)
	authorized.POST("/spaces/:spaceId/retrieval/search", retrievalHandler.Search)
	authorized.POST("/spaces/:spaceId/chat", chatHandler.Chat)
	authorized.POST("/spaces/:spaceId/learning-assets/generate", assetHandler.Generate)
	authorized.POST("/spaces/:spaceId/learning-plan/generate", assetHandler.GeneratePlan)
	authorized.GET("/spaces/:spaceId/learning-assets", assetHandler.Get)
	authorized.GET("/spaces/:spaceId/learning-assets/jobs/:jobId", assetHandler.GetJob)
	authorized.POST("/spaces/:spaceId/knowledge-map/nodes", assetHandler.CreateNode)
	authorized.PATCH("/spaces/:spaceId/knowledge-map/nodes/:nodeId", assetHandler.UpdateNode)
	authorized.PATCH("/spaces/:spaceId/knowledge-map/nodes/:nodeId/position", assetHandler.UpdateNodePosition)
	authorized.DELETE("/spaces/:spaceId/knowledge-map/nodes/:nodeId", assetHandler.DeleteNode)
	authorized.POST("/spaces/:spaceId/knowledge-map/edges", assetHandler.CreateEdge)
	authorized.DELETE("/spaces/:spaceId/knowledge-map/edges/:edgeId", assetHandler.DeleteEdge)
	authorized.PATCH("/spaces/:spaceId/knowledge-handbook/articles/:articleId", assetHandler.UpdateArticle)
	authorized.GET("/spaces/:spaceId/exam-analysis", examHandler.Get)
	authorized.PUT("/spaces/:spaceId/exam-questions/:questionId/feedback", examHandler.Feedback)

	router.NoRoute(func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "route not found", nil)
	})

	return router, nil
}
