package job

import (
	"context"
	"fmt"

	"whatsnext/backend/internal/ai"
	"whatsnext/backend/internal/repository"
)

type LearningAssetProcessor struct {
	repo               *repository.LearningAssetRepository
	spaces             *repository.LearningSpaceRepository
	knowledgeGenerator ai.KnowledgeAssetGenerator
	planGenerator      ai.LearningPlanGenerator
}

func NewLearningAssetProcessor(repo *repository.LearningAssetRepository, spaces *repository.LearningSpaceRepository, knowledge ai.KnowledgeAssetGenerator, plan ai.LearningPlanGenerator) *LearningAssetProcessor {
	return &LearningAssetProcessor{repo: repo, spaces: spaces, knowledgeGenerator: knowledge, planGenerator: plan}
}

func (p *LearningAssetProcessor) Process(ctx context.Context, jobID string) error {
	job, claimed, err := p.repo.ClaimJob(ctx, jobID)
	if err != nil || !claimed {
		return err
	}
	space, err := p.spaces.Get(ctx, job.UserID, job.LearningSpaceID)
	if err == nil && job.JobType == "knowledge_assets" {
		var sources []ai.AssetSource
		sources, err = p.repo.IndexedSources(ctx, job.UserID, job.LearningSpaceID)
		var existingNodes []ai.ExistingKnowledgeNode
		if err == nil {
			existingNodes, err = p.repo.MergeableExamKnowledgeNodes(ctx, job.UserID, job.LearningSpaceID)
		}
		var assets ai.GeneratedAssets
		if err == nil {
			assets, err = p.knowledgeGenerator.GenerateKnowledge(ctx, space.Goal, sources, existingNodes)
		}
		if err == nil {
			err = p.repo.CompleteKnowledge(ctx, job, assets)
		}
	} else if err == nil && job.JobType == "learning_plan" {
		var sources []ai.PlanSource
		sources, err = p.repo.PlanSources(ctx, job.UserID, job.LearningSpaceID)
		var signals []ai.PlanSignal
		if err == nil {
			signals, err = p.repo.PendingPlanSignals(ctx, job.UserID, job.LearningSpaceID)
		}
		var plan ai.GeneratedPlan
		if err == nil {
			plan, err = p.planGenerator.GeneratePlan(ctx, ai.PlanContext{Goal: space.Goal, ExamDate: space.ExamDate, DailyMinutes: space.DailyMinutes}, sources, signals)
		}
		if err == nil {
			err = p.repo.CompletePlan(ctx, job, sources, plan, len(signals))
		}
	} else if err == nil {
		err = fmt.Errorf("unsupported learning asset job type %s", job.JobType)
	}
	if err != nil {
		if failErr := p.repo.Fail(ctx, job, "ASSET_GENERATION_FAILED", err.Error()); failErr != nil {
			return fmt.Errorf("generate assets: %v; persist failure: %w", err, failErr)
		}
	}
	return err
}
