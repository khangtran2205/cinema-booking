package booking

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

var errSeatTaken = errors.New("seat already booked")

// testStore is a mutex-backed BookingStore for concurrent booking tests.
type testStore struct {
	mu     sync.Mutex
	booked map[string]struct{}
}

func newTestStore() *testStore {
	return &testStore{booked: make(map[string]struct{})}
}

func (s *testStore) Book(b Booking) (Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := b.MovieID + "|" + b.SeatID
	if _, ok := s.booked[key]; ok {
		return Booking{}, errSeatTaken
	}
	s.booked[key] = struct{}{}
	b.ID = uuid.New().String()
	b.Status = "confirmed"
	return b, nil
}

func (s *testStore) ListBookings(movieId string) []Booking {
	return nil
}

func TestConcurrentBooking_ExactlyOneWins(t *testing.T) {
	store := newTestStore()
	svc := NewService(store)

	const numGoroutines = 100_000 // 100k users trying to book a seat at the same time

	var (
		successes atomic.Int64
		failures  atomic.Int64
		wg        sync.WaitGroup
	)

	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(userNum int) {
			defer wg.Done()
			_, err := svc.store.Book(Booking{
				MovieID: "screen-1",
				SeatID:  "A1",
				UserID:  uuid.New().String(),
			})
			if err == nil {
				successes.Add(1)
			} else {
				failures.Add(1)
			}
		}(i)
	}
	wg.Wait()

	if got := successes.Load(); got != 1 {
		t.Errorf("expected exactly 1 success, got %d", got)
	}
	if got := failures.Load(); got != int64(numGoroutines-1) {
		t.Errorf("expected %d failures, got %d", numGoroutines-1, got)
	}
}
