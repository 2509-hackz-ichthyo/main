package usecases

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
)

func TestExecuteCommandWhitespaceToDecimal(t *testing.T) {
	repo := newMemoryRepository()
	clock := fixedClock{now: time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)}
	idGen := &sequenceIDGenerator{values: []string{"cmd-1"}}

	executor := NewCommandExecutor(
		repo,
		domain.NewWhitespaceDecoder(),
		domain.NewWhitespaceEncoder(),
		WithClock(clock),
		WithIDGenerator(idGen),
	)

	input := ExecuteCommandInput{
		CommandType: string(domain.CommandTypeWhitespaceToDecimal),
		Payload:     " \t\n",
	}

	output, err := executor.ExecuteCommand(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ID != "cmd-1" {
		t.Errorf("unexpected id: got %s", output.ID)
	}

	if output.ResultKind != domain.ResultKindDecimalSequence {
		t.Fatalf("unexpected result kind: %s", output.ResultKind)
	}

	expected := []int{32, 9, 10}
	if diff := diffInts(expected, output.ResultDecimals); len(diff) != 0 {
		t.Errorf("unexpected decimals diff: %v", diff)
	}

	saved, err := repo.FindByID(context.Background(), "cmd-1")
	if err != nil {
		t.Fatalf("saved record not found: %v", err)
	}

	if saved.CreatedAt != clock.now {
		t.Errorf("unexpected createdAt: %v", saved.CreatedAt)
	}
}

func TestExecuteCommandDecimalToWhitespace(t *testing.T) {
	repo := newMemoryRepository()
	clock := fixedClock{now: time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)}
	idGen := &sequenceIDGenerator{values: []string{"cmd-2"}}

	executor := NewCommandExecutor(
		repo,
		domain.NewWhitespaceDecoder(),
		domain.NewWhitespaceEncoder(),
		WithClock(clock),
		WithIDGenerator(idGen),
	)

	input := ExecuteCommandInput{
		CommandType: string(domain.CommandTypeDecimalToWhitespace),
		Payload:     "32 9 10",
	}

	output, err := executor.ExecuteCommand(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ResultKind != domain.ResultKindWhitespace {
		t.Fatalf("unexpected result kind: %s", output.ResultKind)
	}

	if output.ResultText == nil || *output.ResultText != " \t\n" {
		t.Fatalf("unexpected result text: %v", output.ResultText)
	}
}

func TestExecuteCommandValidationFailure(t *testing.T) {
	executor := NewCommandExecutor(newMemoryRepository(), domain.NewWhitespaceDecoder(), domain.NewWhitespaceEncoder())

	_, err := executor.ExecuteCommand(context.Background(), ExecuteCommandInput{})
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

func TestGetExecutionValidation(t *testing.T) {
	executor := NewCommandExecutor(newMemoryRepository(), domain.NewWhitespaceDecoder(), domain.NewWhitespaceEncoder())

	_, err := executor.GetExecution(context.Background(), " ")
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

// --- テスト用のスタブ実装 ---

type memoryRepository struct {
	mu    sync.Mutex
	store map[string]CommandExecution
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{store: make(map[string]CommandExecution)}
}

func (m *memoryRepository) Save(_ context.Context, execution CommandExecution) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[execution.ID] = execution
	return nil
}

func (m *memoryRepository) FindByID(_ context.Context, id string) (CommandExecution, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	execution, ok := m.store[id]
	if !ok {
		return CommandExecution{}, ErrExecutionNotFound
	}
	return execution, nil
}

func (m *memoryRepository) ListRecent(_ context.Context, limit int) ([]CommandExecution, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	results := make([]CommandExecution, 0, len(m.store))
	for _, execution := range m.store {
		results = append(results, execution)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

type fixedClock struct {
	now time.Time
}

func (f fixedClock) Now() time.Time { return f.now }

type sequenceIDGenerator struct {
	mu     sync.Mutex
	values []string
	index  int
}

func (s *sequenceIDGenerator) NewID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index >= len(s.values) {
		return ""
	}
	value := s.values[s.index]
	s.index++
	return value
}

func diffInts(expected, actual []int) []int {
	if len(expected) != len(actual) {
		return []int{-1}
	}
	for i := range expected {
		if expected[i] != actual[i] {
			return []int{expected[i], actual[i]}
		}
	}
	return nil
}
