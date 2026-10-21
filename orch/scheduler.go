package orch

import (
	"context"
	"sync"
	"time"

	"git.commsnet.org/commstech/repository-detective/limiter"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// ScanRunner executes a scheduled full-repo scan.
type ScanRunner func(ctx context.Context, repo store.ScheduledRepository) error

// Config controls scheduler behavior.
type Config struct {
	Enabled       bool
	PollInterval  time.Duration
	MaxConcurrent int
}

// Scheduler polls repo schedules and triggers full-repo scans.
type Scheduler struct {
	store           store.QueryStore
	runScan         ScanRunner
	analysisLimiter *limiter.ConcurrencyLimiter
	logger          *logrus.Logger
	cfg             Config

	mu       sync.Mutex
	states   map[int64]*repoScheduleState
	inFlight sync.WaitGroup

	stopCh chan struct{}
	doneCh chan struct{}
}

type repoScheduleState struct {
	repo     store.ScheduledRepository
	schedule cron.Schedule
	cronExpr string
	lastRun  time.Time
}

// NewScheduler creates a scheduler. Caller must invoke Start.
func NewScheduler(s store.QueryStore, runScan ScanRunner, analysisLimiter *limiter.ConcurrencyLimiter, cfg Config, logger *logrus.Logger) *Scheduler {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 60 * time.Second
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 1
	}
	if logger == nil {
		logger = logrus.New()
	}
	return &Scheduler{
		store:           s,
		runScan:         runScan,
		analysisLimiter: analysisLimiter,
		logger:          logger,
		cfg:             cfg,
		states:          make(map[int64]*repoScheduleState),
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}
}

// Start begins the polling loop.
func (s *Scheduler) Start(ctx context.Context) {
	if !s.cfg.Enabled || s.store == nil || s.runScan == nil {
		s.logger.Info("Scheduler disabled — not starting")
		close(s.doneCh)
		return
	}
	s.logger.WithFields(logrus.Fields{
		"poll_interval":  s.cfg.PollInterval.String(),
		"max_concurrent": s.cfg.MaxConcurrent,
	}).Info("Scheduler starting")
	go s.loop(ctx)
}

// Stop waits for the polling loop and in-flight scheduled scans to exit.
func (s *Scheduler) Stop() {
	select {
	case <-s.doneCh:
		s.inFlight.Wait()
		return
	default:
	}
	close(s.stopCh)
	<-s.doneCh
	s.inFlight.Wait()
	s.logger.Info("Scheduler stopped")
}

func (s *Scheduler) loop(ctx context.Context) {
	defer close(s.doneCh)

	s.refreshSchedules(ctx)

	ticker := time.NewTicker(s.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) refreshSchedules(ctx context.Context) {
	repos, err := s.store.ListScheduledRepositories(ctx)
	if err != nil {
		s.logger.Errorf("Scheduler failed to load schedules: %v", err)
		return
	}

	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[int64]struct{}, len(repos))
	for _, repo := range repos {
		seen[repo.ID] = struct{}{}
		schedule, err := store.ParseCronSchedule(repo.ScheduleCron)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"repo":          repo.FullName,
				"schedule_cron": repo.ScheduleCron,
			}).Warnf("Invalid cron — skipping repo schedule: %v", err)
			delete(s.states, repo.ID)
			continue
		}

		state, exists := s.states[repo.ID]
		if !exists || state.cronExpr != repo.ScheduleCron {
			lastRun := s.baselineLastRun(ctx, repo.ID, now)
			s.states[repo.ID] = &repoScheduleState{
				repo:     repo,
				schedule: schedule,
				cronExpr: repo.ScheduleCron,
				lastRun:  lastRun,
			}
			s.logger.WithFields(logrus.Fields{
				"repo":          repo.FullName,
				"schedule_cron": repo.ScheduleCron,
				"next_run":      schedule.Next(lastRun).UTC().Format(time.RFC3339),
			}).Info("Repo schedule loaded")
			continue
		}
		state.repo = repo
	}

	for id := range s.states {
		if _, ok := seen[id]; !ok {
			delete(s.states, id)
		}
	}
}

func (s *Scheduler) baselineLastRun(ctx context.Context, repositoryID int64, now time.Time) time.Time {
	finished, err := s.store.GetLastScheduledScanFinishedAt(ctx, repositoryID)
	if err != nil {
		s.logger.Warnf("Scheduler baseline last run for repo %d: %v", repositoryID, err)
		return now
	}
	if finished != nil && !finished.IsZero() {
		return finished.UTC()
	}
	return now
}

func (s *Scheduler) tick(ctx context.Context) {
	s.refreshSchedules(ctx)

	now := time.Now().UTC()
	s.mu.Lock()
	due := s.collectDue(now)
	s.mu.Unlock()

	if len(due) == 0 {
		return
	}

	sem := make(chan struct{}, s.cfg.MaxConcurrent)

	for _, item := range due {
		item := item
		select {
		case sem <- struct{}{}:
		default:
			s.logSkip(item.repo, "max concurrent scheduled scans reached")
			continue
		}

		s.inFlight.Add(1)
		go func() {
			defer s.inFlight.Done()
			defer func() { <-sem }()
			s.tryRunScheduled(ctx, item, now)
		}()
	}
}

func (s *Scheduler) collectDue(now time.Time) []repoScheduleState {
	var due []repoScheduleState
	for _, state := range s.states {
		next := state.schedule.Next(state.lastRun)
		if !next.After(now) {
			due = append(due, *state)
		}
	}
	return due
}

func (s *Scheduler) tryRunScheduled(ctx context.Context, state repoScheduleState, now time.Time) {
	repo := state.repo
	fields := logrus.Fields{
		"repo":          repo.FullName,
		"schedule_cron": repo.ScheduleCron,
		"trigger_type":  store.TriggerScheduled,
	}

	running, err := s.store.HasRunningScanForRepository(ctx, repo.ID)
	if err != nil {
		s.logger.WithFields(fields).Warnf("Scheduled scan skipped — running check failed: %v", err)
		return
	}
	if running {
		s.logSkip(repo, "repository already has a running scan")
		return
	}

	if s.analysisLimiter != nil && !s.analysisLimiter.HasCapacity() {
		s.logSkip(repo, "global analysis limiter full")
		return
	}

	dueTime := state.schedule.Next(state.lastRun)

	s.mu.Lock()
	if current, ok := s.states[repo.ID]; ok {
		current.lastRun = dueTime
	}
	s.mu.Unlock()

	s.logger.WithFields(fields).Info("Scheduled scan starting")

	runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	runErr := s.runScan(runCtx, repo)

	if runErr != nil {
		s.logger.WithFields(fields).Warnf("Scheduled scan finished with error: %v", runErr)
	} else {
		s.logger.WithFields(fields).Info("Scheduled scan finished")
	}
}

func (s *Scheduler) logSkip(repo store.ScheduledRepository, reason string) {
	s.logger.WithFields(logrus.Fields{
		"repo":          repo.FullName,
		"schedule_cron": repo.ScheduleCron,
		"trigger_type":  store.TriggerScheduled,
		"reason":        reason,
	}).Info("Scheduled scan skipped")
}

// NextRunForRepo returns the next scheduled run for a repository (for UI/API).
func (s *Scheduler) NextRunForRepo(repositoryID int64) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[repositoryID]
	if !ok || state.schedule == nil {
		return time.Time{}, false
	}
	next := state.schedule.Next(state.lastRun)
	return next, true
}

// DescribeNextRun computes next run from cron and optional last finished time.
func DescribeNextRun(cronExpr string, lastFinished *time.Time) store.CronDescription {
	baseline := time.Now().UTC()
	if lastFinished != nil && !lastFinished.IsZero() {
		baseline = lastFinished.UTC()
	}
	return store.DescribeCron(cronExpr, baseline)
}
