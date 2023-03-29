package service

import (
	"context"
	"fmt"
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"log"
	"sync"
	"time"
)

const (
	deleteChanSize  = 100
	deleteChunkSize = 10
	retryIn         = 5 * time.Second
	flushAfter      = 10 * time.Second
	deleteTimeout   = 30 * time.Second
)

type DeleteWorker interface {
	Delete(ids []internal.IDToDelete)
}

var _ DeleteWorker = (*DeleteURL)(nil)

type DeleteURL struct {
	inCh  chan internal.IDToDelete
	buf   []internal.IDToDelete
	mutex sync.RWMutex
	store storage.Storage
}

func NewDeleteWorker(store storage.Storage) *DeleteURL {
	return &DeleteURL{
		inCh:  make(chan internal.IDToDelete, deleteChanSize),
		buf:   make([]internal.IDToDelete, 0),
		store: store,
	}
}

type Signal struct {
	C chan struct{}
}

func NewSignal() Signal {
	return Signal{C: make(chan struct{}, 1)}
}

func (s Signal) Notify() {
	select {
	case s.C <- struct{}{}:
	default:
	}
}

func (w *DeleteURL) Run(ctx context.Context) {
	flushTick := time.NewTicker(flushAfter)
	flushSignal := NewSignal()

	go func() {
		for range flushSignal.C {
			err := w.flushBuf()
			if err != nil {
				log.Println(err)
				time.Sleep(retryIn)
				flushSignal.Notify()
			}
		}
	}()

	for {
		select {
		case v := <-w.inCh:
			func() {
				w.mutex.Lock()
				defer w.mutex.Unlock()
				w.buf = append(w.buf, v)
				if len(w.buf) >= deleteChunkSize {
					flushSignal.Notify()
					flushTick = time.NewTicker(flushAfter)
				}
			}()
		case <-flushTick.C:
			func() {
				w.mutex.RLock()
				defer w.mutex.RUnlock()
				if len(w.buf) > 0 {
					log.Printf("Flushing after %v\n", flushAfter)
					flushSignal.Notify()
				}
			}()
		case <-ctx.Done():
			log.Println("Stopping delete worker")
			close(w.inCh)
			err := w.flushAll()
			if err != nil {
				log.Println(err)
			}
			return
		}
	}
}

func (w *DeleteURL) flushAll() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	var ids []internal.IDToDelete
	ids = append(ids, w.buf...)
	for v := range w.inCh {
		ids = append(ids, v)
	}
	if len(ids) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
	defer cancel()
	err := w.store.DeleteBatch(ctx, ids)
	if err != nil {
		return fmt.Errorf(`URL ids to delete flushing error: %w`, err)
	}
	w.buf = w.buf[:0]
	return nil
}

func (w *DeleteURL) flushBuf() error {
	ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
	defer cancel()
	w.mutex.Lock()
	defer w.mutex.Unlock()
	err := w.store.DeleteBatch(ctx, w.buf)
	if err != nil {
		return fmt.Errorf(`URL ids to delete flushing error: %w`, err)
	}
	w.buf = w.buf[:0]
	return nil
}

func (w *DeleteURL) Delete(ids []internal.IDToDelete) {
	for _, v := range ids {
		w.inCh <- v
	}
}
