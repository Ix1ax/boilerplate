package files

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type fakeStore struct {
	inserted  []File
	owner     string
	insertErr error
	deleteErr error
	getErr    error
	deleted   bool
}

func (f *fakeStore) List(context.Context, string, int, int) (Page, error) { return Page{}, nil }
func (f *fakeStore) Insert(_ context.Context, owner, _ string, file File) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.owner = owner
	f.inserted = append(f.inserted, file)
	return nil
}
func (f *fakeStore) Get(context.Context, string, string) (string, string, string, error) {
	if f.getErr != nil {
		return "", "", "", f.getErr
	}
	if f.deleted {
		return "", "", "", ErrNotFound
	}
	return "key", "name.txt", "text/plain; charset=utf-8", nil
}
func (f *fakeStore) Delete(context.Context, string, string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = true
	return nil
}

type fakeBlobs struct {
	putErr    error
	getErr    error
	deleted   []string
	deleteErr error
}

func (b *fakeBlobs) Put(context.Context, string, string, []byte) error { return b.putErr }
func (b *fakeBlobs) Get(context.Context, string) (io.ReadCloser, int64, error) {
	if b.getErr != nil {
		return nil, 0, b.getErr
	}
	return io.NopCloser(bytes.NewReader([]byte("data"))), 4, nil
}
func (b *fakeBlobs) Delete(_ context.Context, key string) error {
	b.deleted = append(b.deleted, key)
	return b.deleteErr
}

func TestUploadValidatesBeforeTouchingStorage(t *testing.T) {
	store := &fakeStore{}
	service := New(store, &fakeBlobs{})

	if _, err := service.Upload(context.Background(), "owner", "a.txt", nil); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("пустой файл: %v", err)
	}
	if _, err := service.Upload(context.Background(), "owner", "a.txt", make([]byte, MaxSize+1)); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("файл больше лимита: %v", err)
	}
	if _, err := service.Upload(context.Background(), "owner", "a.exe", []byte("MZ\x90\x00binary")); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("недопустимый тип: %v", err)
	}
	if _, err := service.Upload(context.Background(), "owner", strings.Repeat("a", 256)+".txt", []byte("text")); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("слишком длинное имя: %v", err)
	}
	if len(store.inserted) != 0 {
		t.Fatal("невалидный файл попал в хранилище метаданных")
	}

	f, err := service.Upload(context.Background(), "owner", "note.txt", []byte("text"))
	if err != nil || f.ID == "" || f.Name != "note.txt" || store.owner != "owner" || len(store.inserted) != 1 {
		t.Fatalf("валидный файл не сохранён: %+v %v", f, err)
	}
}

func TestUploadRollsBackBlobOnStoreFailure(t *testing.T) {
	store := &fakeStore{insertErr: errors.New("db unavailable")}
	blobs := &fakeBlobs{}
	service := New(store, blobs)

	if _, err := service.Upload(context.Background(), "owner", "note.txt", []byte("text")); err == nil {
		t.Fatal("ожидалась ошибка вставки метаданных")
	}
	if len(blobs.deleted) != 1 {
		t.Fatal("загруженный объект не удалён после неудачной записи метаданных")
	}
}

func TestUploadReportsStorageUnavailable(t *testing.T) {
	service := New(&fakeStore{}, &fakeBlobs{putErr: errors.New("connection refused")})

	_, err := service.Upload(context.Background(), "owner", "note.txt", []byte("text"))
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("ошибка blob-слоя должна мапиться в ErrStorageUnavailable: %v", err)
	}
}

func TestDownloadReportsStorageUnavailable(t *testing.T) {
	service := New(&fakeStore{}, &fakeBlobs{getErr: errors.New("connection refused")})

	_, _, _, _, err := service.Download(context.Background(), "owner", "id")
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("ошибка blob-слоя должна мапиться в ErrStorageUnavailable: %v", err)
	}
}

func TestDeletePropagatesNotFound(t *testing.T) {
	service := New(&fakeStore{getErr: ErrNotFound}, &fakeBlobs{})

	if err := service.Delete(context.Background(), "owner", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ожидался ErrNotFound: %v", err)
	}
}

func TestDeleteRetainsMetadataOnStorageFailure(t *testing.T) {
	store := &fakeStore{}
	blobs := &fakeBlobs{deleteErr: errors.New("S3 недоступен")}
	service := New(store, blobs)
	if err := service.Delete(context.Background(), "owner", "id"); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("ожидалась ошибка хранилища: %v", err)
	}
	if store.deleted {
		t.Fatal("метаданные удалены до успешного удаления объекта")
	}
	blobs.deleteErr = nil
	if err := service.Delete(context.Background(), "owner", "id"); err != nil || !store.deleted {
		t.Fatalf("повтор удаления не завершился: %v", err)
	}
}

func TestDeleteRetriesAfterMetadataFailure(t *testing.T) {
	failure := errors.New("БД недоступна")
	store := &fakeStore{deleteErr: failure}
	blobs := &fakeBlobs{}
	service := New(store, blobs)
	if err := service.Delete(context.Background(), "owner", "id"); !errors.Is(err, failure) {
		t.Fatalf("ожидалась ошибка БД: %v", err)
	}
	if store.deleted || len(blobs.deleted) != 1 {
		t.Fatal("неверный порядок удаления")
	}
	store.deleteErr = nil
	if err := service.Delete(context.Background(), "owner", "id"); err != nil || !store.deleted || len(blobs.deleted) != 2 {
		t.Fatalf("повтор удаления не завершился: %v", err)
	}
}

func TestDeleteDoesNotTouchStorageWithoutOwnedMetadata(t *testing.T) {
	for _, failure := range []error{ErrNotFound, errors.New("БД недоступна")} {
		store := &fakeStore{getErr: failure}
		blobs := &fakeBlobs{}
		if err := New(store, blobs).Delete(context.Background(), "other", "id"); !errors.Is(err, failure) {
			t.Fatalf("ожидалась ошибка чтения метаданных: %v", err)
		}
		if store.deleted || len(blobs.deleted) != 0 {
			t.Fatal("хранилища изменены без проверки владельца")
		}
	}
}
