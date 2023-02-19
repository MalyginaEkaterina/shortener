package service

import (
	"context"
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"log"
	"time"
)

const (
	deleteChanSize  = 100
	deleteChunkSize = 10
	retryIn         = 5 * time.Second
	flushAfter      = time.Second * 10
	deleteTimeout   = 30 * time.Second
)

type DeleteWorker struct {
	inCh  chan internal.IDToDelete
	buf   []internal.IDToDelete
	store storage.Storage
}

func NewDeleteWorker(store storage.Storage) *DeleteWorker {
	return &DeleteWorker{
		inCh:  make(chan internal.IDToDelete, deleteChanSize),
		buf:   make([]internal.IDToDelete, 0, deleteChunkSize),
		store: store,
	}
}

func (w *DeleteWorker) Run(ctx context.Context) {
	for {
		select {
		case v := <-w.inCh:
			w.buf = append(w.buf, v)
			if len(w.buf) == cap(w.buf) {
				err := w.flushBuf()
				for err != nil {
					log.Printf("Retrying flush in %v\n", retryIn)
					time.Sleep(retryIn)
					err = w.flushBuf()
				}
			}
		case <-time.After(flushAfter):
			if len(w.buf) > 0 {
				log.Printf("Flushing after %v\n", flushAfter)
				_ = w.flushBuf()
			}
		case <-ctx.Done():
			log.Println("Stopping delete worker")
			close(w.inCh)
			w.flushAll()
			return
		}
	}
}

func (w *DeleteWorker) flushAll() {
	var ids []internal.IDToDelete
	ids = append(ids, w.buf...)
	for v := range w.inCh {
		ids = append(ids, v)
	}
	if len(ids) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
	defer cancel()
	err := w.store.DeleteBatch(ctx, ids)
	if err != nil {
		log.Println("URL ids to delete flushing error", err)
	}
	w.buf = w.buf[:0]
}

func (w *DeleteWorker) flushBuf() error {
	ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
	defer cancel()
	err := w.store.DeleteBatch(ctx, w.buf)
	if err != nil {
		log.Println("URL ids to delete flushing error", err)
		return err
	}
	w.buf = w.buf[:0]
	return nil
}

func (w *DeleteWorker) Delete(ids []internal.IDToDelete) {
	for _, v := range ids {
		w.inCh <- v
	}
}
