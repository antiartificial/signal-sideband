package digest

import (
	"context"
	"log"
	"time"
)

type Scheduler struct {
	generator *Generator
	insights  *InsightsGenerator
	interval  time.Duration
}

func NewScheduler(g *Generator, insights *InsightsGenerator, interval time.Duration) *Scheduler {
	return &Scheduler{generator: g, insights: insights, interval: interval}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Printf("Digest scheduler started (interval: %s)", s.interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Digest scheduler stopped")
			return
		case <-ticker.C:
			s.generateDaily(ctx)
			s.generateInsights(ctx)
		}
	}
}

func (s *Scheduler) generateDaily(ctx context.Context) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day()-1, 23, 59, 59, 0, now.Location())

	log.Printf("Generating daily digest for %s", start.Format("2006-01-02"))
	digest, err := s.generator.Generate(ctx, start, end, nil)
	if err != nil {
		log.Printf("Daily digest generation failed: %v", err)
		return
	}
	log.Printf("Daily digest generated: %s (id: %s)", digest.Title, digest.ID)
}

func (s *Scheduler) generateInsights(ctx context.Context) {
	if s.insights == nil {
		return
	}
	if err := s.insights.GenerateDailyInsights(ctx); err != nil {
		log.Printf("Daily insights generation failed: %v", err)
	}
}
