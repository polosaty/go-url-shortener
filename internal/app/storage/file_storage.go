package storage

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
)

type FileStorage struct {
	FileAccessMutex sync.RWMutex
	memMap          *MemoryMap
	encoder         *json.Encoder
}

type FileRecord struct {
	ShortURL URL
	LongURL  URL
	UserID   string
}

func NewFileStorage(filename string) (*FileStorage, error) {
	db := &FileStorage{
		memMap:          NewMemoryMap(),
		FileAccessMutex: sync.RWMutex{},
		//encoder: json.NewEncoder(file),
	}

	if err := db.LoadFromFile(filename); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	db.encoder = json.NewEncoder(file)

	return db, nil
}

func (d *FileStorage) LoadFromFile(filename string) error {
	d.FileAccessMutex.RLock()
	defer d.FileAccessMutex.RUnlock()
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	return d.LoadFromBuff(file)
}

func (d *FileStorage) LoadFromBuff(buf io.Reader) error {
	decoder := json.NewDecoder(buf)
	d.memMap.Mutex.Lock()
	defer d.memMap.Mutex.Unlock()

	for {
		record := &FileRecord{}
		if err := decoder.Decode(&record); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		d.memMap.SetLongURL(record.LongURL, record.ShortURL, record.UserID)
	}

}

func (d *FileStorage) SaveLongURL(long URL, userID string) (URL, error) {
	d.FileAccessMutex.Lock()
	defer d.FileAccessMutex.Unlock()
	short, err := d.memMap.SaveLongURL(long, userID)
	if err != nil {
		return "", err
	}

	if err = d.encoder.Encode(FileRecord{ShortURL: short, LongURL: long, UserID: userID}); err != nil {
		return "", err
	}

	return short, nil
}

func (d *FileStorage) GetLongURL(short URL) (URL, error) {
	return d.memMap.GetLongURL(short)
}

func (d *FileStorage) GetUsersURLs(userID string) []URLPair {
	return d.memMap.GetUsersURLs(userID)
}

func (d *FileStorage) Ping() bool {
	return d.memMap.Ping()
}

func (d *FileStorage) SaveLongBatchURL(longURLS []CorrelationLongPair, userID string) ([]CorrelationShortPair, error) {
	result := make([]CorrelationShortPair, 0, len(longURLS))
	for _, p := range longURLS {

		short, err := d.SaveLongURL(p.LongURL, userID)
		if err != nil {
			log.Printf("SaveLongBatchURL error(%v):  cant shor url %v", err, p.LongURL)
			continue
		}
		result = append(result, CorrelationShortPair{p.CorrelationID, short})
	}
	return result, nil
}

func (d *FileStorage) DeleteUsersURLs(userID string, shortUrls ...URL) error {
	//TODO implement me
	panic("implement me")
}

func (d *FileStorage) DelayedDeleteUsersURLs(userID string, shortUrls ...URL) error {
	//TODO implement me
	panic("implement me")
}
