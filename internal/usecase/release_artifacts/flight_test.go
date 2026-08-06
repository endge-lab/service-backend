package release_artifacts

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestReaderDeduplicatesConcurrentMisses(t *testing.T) {
	release := testRelease("workspace", "release", "checksum")
	repository := &artifactRepositoryStub{artifact: testArtifact(release, `{}`), delay: 30 * time.Millisecond}
	reader := newReader(t, repository, enabledCache(1024, 1024))

	var wait sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 10)
	for range 10 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := reader.Read(context.Background(), "export", release.WorkspaceID, release)
			errs <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if repository.callCount() != 1 {
		t.Fatalf("repository calls = %d, want 1", repository.callCount())
	}
}

func TestReaderKeepsSharedLoadAfterLeaderRequestCanceled(t *testing.T) {
	release := testRelease("workspace", "release", "checksum")
	block := make(chan struct{})
	repository := &artifactRepositoryStub{
		artifact: testArtifact(release, `{}`),
		started:  make(chan struct{}),
		block:    block,
	}
	reader := newReader(t, repository, enabledCache(1024, 1024))

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderResult := make(chan error, 1)
	go func() {
		_, err := reader.Read(leaderCtx, "export", release.WorkspaceID, release)
		leaderResult <- err
	}()
	<-repository.started

	waiterResult := make(chan error, 1)
	go func() {
		_, err := reader.Read(context.Background(), "export", release.WorkspaceID, release)
		waiterResult <- err
	}()

	cancelLeader()
	if err := <-leaderResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("leader error = %v, want context canceled", err)
	}
	close(block)
	if err := <-waiterResult; err != nil {
		t.Fatalf("waiter error = %v", err)
	}
	if repository.callCount() != 1 {
		t.Fatalf("repository calls = %d, want 1", repository.callCount())
	}
}

func TestReaderSharedLoadTimeoutCleansFlightAndAllowsRetry(t *testing.T) {
	release := testRelease("workspace", "release", "checksum")
	repository := &artifactRepositoryStub{artifact: testArtifact(release, `{}`), block: make(chan struct{})}
	reader := newReader(t, repository, enabledCache(1024, 1024))
	reader.loadTimeout = 10 * time.Millisecond

	_, err := reader.Read(context.Background(), "export", release.WorkspaceID, release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout error = %v, want deadline exceeded", err)
	}
	reader.mu.Lock()
	flights := len(reader.inflight)
	reader.mu.Unlock()
	if flights != 0 {
		t.Fatalf("inflight entries = %d, want 0", flights)
	}

	repository.mu.Lock()
	repository.block = nil
	repository.mu.Unlock()
	reader.loadTimeout = time.Second
	if _, err = reader.Read(context.Background(), "export", release.WorkspaceID, release); err != nil {
		t.Fatalf("retry error = %v", err)
	}
	if repository.callCount() != 2 {
		t.Fatalf("repository calls = %d, want 2", repository.callCount())
	}
}
