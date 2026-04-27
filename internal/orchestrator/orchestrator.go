package orchestrator

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/fakeapate/pry/config"
	"github.com/fakeapate/pry/internal/classify"
	"github.com/fakeapate/pry/internal/scanner"
	"github.com/fakeapate/pry/internal/store"
	"github.com/fakeapate/pry/internal/store/db"
	"github.com/fakeapate/pry/model"
)

type scanJob struct {
	scanID string
	url    string
	cancel context.CancelFunc
}

var lock = &sync.Mutex{}

type Orchestrator struct {
	db         *sql.DB
	queries    *db.Queries
	cfg        config.AppConfig
	ctx        context.Context
	ctxCancel context.CancelFunc
	wg         sync.WaitGroup
	program    *tea.Program
	programMu  sync.RWMutex
	workerSem  chan struct{} // semaphore enforcing cfg.Workers concurrency
}

var instance *Orchestrator

func GetInstance(ctx *context.Context) *Orchestrator {
	if instance == nil {
		lock.Lock()
		defer lock.Unlock()
		if instance == nil {
			var err error
			instance, err = newOrchestrator(ctx)
			if err != nil {
				panic(err)
			}
		}
	}
	return instance
}

func newOrchestrator(ctx *context.Context) (*Orchestrator, error) {
	o := &Orchestrator{}
	cfg := config.GetConfig()
	dbPath := cfg.Database.DBPath

	var (
		oCtx   context.Context
		cancel context.CancelFunc
	)
	if ctx != nil {
		oCtx, cancel = context.WithCancel(*ctx)
	} else {
		oCtx, cancel = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	}

	database, err := store.OpenDB(dbPath)
	if err != nil {
		cancel()
		return nil, err
	}

	if err := store.MigrateUp(database); err != nil {
		cancel()
		database.Close()
		return nil, err
	}

	workers := cfg.Workers
	if workers < 1 {
		workers = 1
	}

	o.cfg = *cfg
	o.ctx = oCtx
	o.ctxCancel = cancel
	o.db = database
	o.queries = db.New(database)
	o.workerSem = make(chan struct{}, workers)
	return o, nil
}

// GetDB returns the underlying database connection for read-only TUI use.
func (o *Orchestrator) GetDB() *sql.DB {
	return o.db
}

// SetProgram registers the bubbletea program so the orchestrator can push events to the TUI.
func (o *Orchestrator) SetProgram(p *tea.Program) {
	o.programMu.Lock()
	defer o.programMu.Unlock()
	o.program = p
}

func (o *Orchestrator) send(msg any) {
	o.programMu.RLock()
	p := o.program
	o.programMu.RUnlock()
	if p != nil {
		p.Send(msg)
	}
}

// Dispatch creates a scan record and starts a scanner goroutine. Non-blocking.
// The scan starts as PENDING and transitions to RUNNING once it acquires a
// worker slot (cfg.Workers caps concurrent scans).
func (o *Orchestrator) Dispatch(u string) (string, error) {
	scanID := uuid.New().String()
	if _, err := o.queries.CreateScan(o.ctx, db.CreateScanParams{ScanID: scanID, Url: u}); err != nil {
		return "", fmt.Errorf("orchestrator: create scan: %w", err)
	}

	ctx, cancel := context.WithCancel(o.ctx)
	job := &scanJob{scanID: scanID, url: u, cancel: cancel}

	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		defer cancel()

		// Block until a worker slot is available or the context is cancelled.
		select {
		case o.workerSem <- struct{}{}:
			defer func() { <-o.workerSem }()
		case <-ctx.Done():
			return
		}

		if err := o.queries.UpdateScanStatus(ctx, db.UpdateScanStatusParams{ScanID: scanID, Status: "RUNNING"}); err != nil {
			o.failScan(ctx, job, fmt.Errorf("mark running: %w", err))
			return
		}
		o.runScan(ctx, job)
	}()

	return scanID, nil
}

// Wait blocks until all dispatched scans have finished.
func (o *Orchestrator) Wait() {
	o.wg.Wait()
}

// Close cancels all in-flight scans, waits for them to finish, then closes the DB.
func (o *Orchestrator) Close() {
	o.ctxCancel()
	o.wg.Wait()
	o.db.Close()
}

func (o *Orchestrator) runScan(ctx context.Context, job *scanJob) {
	progress := func(ev scanner.ProgressEvent) {
		o.send(model.ScanProgressEvent{
			ScanID:   job.scanID,
			Folders:  int64(ev.FolderCount),
			Findings: int64(ev.FindingCount),
			Skipped:  int64(ev.SkippedCount),
			Errors:   int64(ev.ErrorCount),
			Warnings: int64(ev.WarningCount),
		})
	}

	s, err := scanner.NewScanner(ctx, o.cfg.Scanner, o.cfg.Mullvad, progress)
	if err != nil {
		o.failScan(ctx, job, fmt.Errorf("scanner init: %w", err))
		return
	}

	result, err := s.RunScan(ctx, job.scanID, job.url)
	if err != nil {
		// Scanner-level error means the run could not start at all (e.g. context cancel).
		o.failScan(ctx, job, err)
		return
	}

	// Partial failure: scan finished but found nothing useful. Treat as failed
	// so the user sees it didn't work, rather than a quietly-empty DONE row.
	if result.Stats.FindingCount == 0 && result.Stats.FolderCount == 0 && result.Stats.ErrorCount > 0 {
		o.failScan(ctx, job, fmt.Errorf("scan produced no findings (%d errors)", result.Stats.ErrorCount))
		return
	}

	if err := o.persistFindings(ctx, job.scanID, result); err != nil {
		o.failScan(ctx, job, fmt.Errorf("persist findings: %w", err))
		return
	}

	log.Info("Scan complete",
		"url", job.url,
		"findings", result.Stats.FindingCount,
		"folders", result.Stats.FolderCount,
		"errors", result.Stats.ErrorCount,
		"warnings", result.Stats.WarningCount,
		"duration", fmt.Sprintf("%dms", result.Stats.DurationMs),
	)
	o.send(model.ScanDoneEvent{ScanID: job.scanID, Result: result})
}

func (o *Orchestrator) failScan(ctx context.Context, job *scanJob, reason error) {
	log.Error("Scan failed", "url", job.url, "err", reason)
	if err := o.queries.FailScan(ctx, db.FailScanParams{
		ScanID:        job.scanID,
		FailureReason: sql.NullString{String: reason.Error(), Valid: true},
	}); err != nil {
		log.Error("Failed to record scan failure", "scan_id", job.scanID, "err", err)
	}
	o.send(model.ScanFailedEvent{ScanID: job.scanID, Reason: reason.Error()})
}

func (o *Orchestrator) persistFindings(ctx context.Context, scanID string, result model.ScanResult) (err error) {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	qtx := o.queries.WithTx(tx)
	for _, f := range result.Findings {
		lastMod := sql.NullString{}
		if !f.LastModified.IsZero() {
			lastMod = sql.NullString{String: f.LastModified.Format(time.RFC3339), Valid: true}
		}
		cl := classify.Classify(f.Url, f.ContentType, f.ContentLength)
		if err = qtx.InsertScanFinding(ctx, db.InsertScanFindingParams{
			ScanID:        scanID,
			Url:           f.Url,
			ScanTime:      f.ScanTime.Format(time.RFC3339),
			ContentType:   f.ContentType,
			ContentLength: f.ContentLength,
			LastModified:  lastMod,
			Category:      cl.Category,
			InterestScore: int64(cl.InterestScore),
			Tags:          strings.Join(cl.Tags, ","),
		}); err != nil {
			return fmt.Errorf("insert finding %s: %w", f.Url, err)
		}
	}

	statsJSON, err := json.Marshal(result.Stats)
	if err != nil {
		return fmt.Errorf("marshal stats: %w", err)
	}
	if err = qtx.CompleteScan(ctx, db.CompleteScanParams{
		ScanID: scanID,
		Result: sql.NullString{String: string(statsJSON), Valid: true},
	}); err != nil {
		return fmt.Errorf("complete scan: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
