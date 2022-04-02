package storage

import (
	"encoding/json"
	"io"
	"os"
	"sync"
)

type FileStorage struct {
	FileAccessMutex *sync.Mutex
	memMap          *MemoryMap
	encoder         *json.Encoder
}

type Record struct {
	ShortURL URL
	LongURL  URL
}

func NewFileStorage(filename string) (*FileStorage, error) {
	db := &FileStorage{
		memMap:          NewMemoryMap(),
		FileAccessMutex: &sync.Mutex{},
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
		record := &Record{}
		if err := decoder.Decode(&record); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		d.memMap.Urls[record.ShortURL] = record.LongURL
	}

}

func (d *FileStorage) SaveLongURL(long URL) (URL, error) {
	d.FileAccessMutex.Lock()
	defer d.FileAccessMutex.Unlock()
	short, err := d.memMap.SaveLongURL(long)
	if err != nil {
		return "", err
	}

	if err = d.encoder.Encode(Record{ShortURL: short, LongURL: long}); err != nil {
		return "", err
	}

	return short, nil
}

func (d *FileStorage) GetLongURL(short URL) (URL, error) {
	d.FileAccessMutex.Lock()
	defer d.FileAccessMutex.Unlock()
	return d.memMap.GetLongURL(short)
}