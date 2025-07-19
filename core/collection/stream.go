package collection

import "context"

type DocumentLike interface {
	~map[string]any
}

// DocumentStream provides an iterator-like interface for streaming documents.
type DocumentStream[T DocumentLike] struct {
	results chan T
	errors  chan error
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewDocumentStream creates a new document stream with the specified buffer size.
func NewDocumentStream[T DocumentLike](bufferSize int) *DocumentStream[T] {
	ctx, cancel := context.WithCancel(context.Background())

	var results chan T
	if bufferSize > 0 {
		results = make(chan T, bufferSize)
	} else {
		results = make(chan T)
	}

	err := make(chan error, 1)

	return &DocumentStream[T]{
		ctx:     ctx,
		cancel:  cancel,
		results: results,
		errors:  err,
	}
}

// Next returns the next document from the stream.
func (ds *DocumentStream[T]) Next() (T, error) {
	var zero T
	select {
	case result, ok := <-ds.results:
		if !ok {
			return zero, ErrStreamClosed
		}
		return result, nil

	case err, ok := <-ds.errors:
		if !ok {
			// Error channel closed, check if there are remaining results
			select {
			case result, ok := <-ds.results:
				if ok {
					return result, nil
				}
			default:
			}
			return nil, ErrStreamClosed
		}
		return nil, err

	case <-ds.ctx.Done():
		return zero, ds.ctx.Err()
	}
}

// Close cancels the stream and releases resources.
func (ds *DocumentStream[T]) Close() {
	ds.cancel()
}

func (ds *DocumentStream[T]) closeStreamWithError(err error) {
	go func() {
		defer close(ds.results)
		defer close(ds.errors)
		ds.errors <- err
	}()
}

// streamDocuments runs the actual streaming logic in a goroutine.
func (ds *DocumentStream[T]) streamDocuments(docs []T) {
	defer close(ds.results)
	defer close(ds.errors)

	for _, entry := range docs {
		select {
		case ds.results <- entry:
		case <-ds.ctx.Done():
			return
		default:
			return
		}
	}
}

// ReadAll returns all visible documents for this transaction
func (t *Transaction) Stream(bufferSize int) (*DocumentStream[map[string]any], error) {
	if t.closed {
		return nil, ErrTransactionClosed
	}
	ds := NewDocumentStream[map[string]any](bufferSize)
	t.collection.mu.RLock()
	defer t.collection.mu.RUnlock()

	result := make([]map[string]any, 0)

	// Read all documents that are visible to this transaction
	for _, entry := range t.collection.documents {
		entry.mu.RLock()
		// Walk the version chain to find the visible version
		for version := entry.versions; version != nil; version = version.next {
			if t.collection.isVersionVisible(version, t.TxnID, t.TxnTime) {
				if !version.deleted {

					result = append(result, version.data)
				}
				break
			}
		}

		entry.mu.RUnlock()
	}
	go ds.streamDocuments(result)
	return ds, nil
}
