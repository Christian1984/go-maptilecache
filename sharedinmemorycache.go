package maptilecache

import (
	"strconv"
	"sync"
)

type SharedMemoryCache struct {
	MemoryMap           map[string][]byte
	MemoryMapKeyHistory []string
	MemoryMapMutex      *sync.RWMutex
	MemoryMapSize       int
	MemoryMapMaxSize    int
	debugLogger func(string),
	infoLogger func(string),
	warnLogger func(string),
	errorLogger func(string),
}

func NewSharedMemoryCache (
	maxMemoryFootprint int,
	debugLogger func(string),
	infoLogger func(string),
	warnLogger func(string),
	errorLogger func(string),
) (*SharedMemoryCache) {
	m := SharedMemoryCache{
		MemoryMap: make(map[string][]byte),
		MemoryMapKeyHistory: []string{},
		// ...
	}
}

func (m *SharedMemoryCache) memoryMapRead(key string) ([]byte, bool) {
	m.MemoryMapMutex.RLock()
	data, exists := m.MemoryMap[key]
	m.MemoryMapMutex.RUnlock()

	return data, exists
}

func (m *SharedMemoryCache) memoryMapWrite(key string, data *[]byte) {
	m.MemoryMapMutex.Lock()

	for len(m.MemoryMapKeyHistory) > 0 && m.MemoryMapSize+len(*data) > m.MemoryMapMaxSize {
		deleteKey := m.MemoryMapKeyHistory[0]
		m.MemoryMapKeyHistory = m.MemoryMapKeyHistory[1:]

		deleteSize := len(m.MemoryMap[deleteKey])
		delete(m.MemoryMap, deleteKey)

		m.MemoryMapSize -= deleteSize

		m.logDebug("MemoryMapWrite would exceed maximum capacity. Deleted tile with key [" + deleteKey + "] from MemoryMap, recovered " + strconv.Itoa(deleteSize) + " Bytes.")
	}

	// check if existed, update size if so
	prevData, existed := m.MemoryMap[key]

	if existed {
		m.MemoryMapSize -= len(prevData)
	}

	m.MemoryMap[key] = *data
	m.MemoryMapKeyHistory = append(m.MemoryMapKeyHistory, key)

	m.MemoryMapSize += len(*data)

	m.MemoryMapMutex.Unlock()

}
