package engine

import "context"

// contextMutex serializes manager ownership while allowing callers that have
// not acquired it yet to stop waiting when their context is canceled.
type contextMutex struct {
	token chan struct{}
}

func newContextMutex() contextMutex {
	token := make(chan struct{}, 1)
	token <- struct{}{}
	return contextMutex{token: token}
}

func (m *contextMutex) LockContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.token:
		if err := ctx.Err(); err != nil {
			m.token <- struct{}{}
			return err
		}
		return nil
	}
}

func (m *contextMutex) Lock() {
	<-m.token
}

func (m *contextMutex) Unlock() {
	m.token <- struct{}{}
}
