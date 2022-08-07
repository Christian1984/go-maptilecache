package maptilecache

import (
	"strconv"
	"sync"
)

type MemoryMap struct {
	Tiles *map[string][]byte
}

type MemoryMapKeyHistoyItem struct {
	MemoryMapKey string
	TileKey      string
}

type SharedMemoryCache struct {
	MemoryMaps          map[string]*MemoryMap
	MemoryMapKeyHistory []MemoryMapKeyHistoyItem
	MemoryMapMutex      *sync.RWMutex
	MemoryMapSize       int
	MemoryMapMaxSize    int
	DebugLogger         func(string)
	InfoLogger          func(string)
	WarnLogger          func(string)
	ErrorLogger         func(string)
}

func NewSharedMemoryCache(
	maxMemoryFootprint int,
	debugLogger func(string),
	infoLogger func(string),
	warnLogger func(string),
	errorLogger func(string),
) *SharedMemoryCache {
	m := SharedMemoryCache{
		MemoryMaps:          make(map[string]*MemoryMap),
		MemoryMapKeyHistory: []MemoryMapKeyHistoyItem{},
		MemoryMapMutex:      &sync.RWMutex{},
		MemoryMapMaxSize:    maxMemoryFootprint,
		DebugLogger:         debugLogger,
		InfoLogger:          infoLogger,
		WarnLogger:          warnLogger,
		ErrorLogger:         errorLogger,
	}

	return &m
}

func (m *SharedMemoryCache) log(message string, logFunc func(string)) {
	if logFunc != nil {
		logFunc(message)
	}
}

func (m *SharedMemoryCache) logDebug(message string) {
	m.log(message, m.DebugLogger)
}

func (m *SharedMemoryCache) logInfo(message string) {
	m.log(message, m.InfoLogger)
}

func (m *SharedMemoryCache) logWarn(message string) {
	m.log(message, m.WarnLogger)
}

func (m *SharedMemoryCache) logError(message string) {
	m.log(message, m.ErrorLogger)
}

func (m *SharedMemoryCache) MemoryMapRead(mapKey string, tileKey string) ([]byte, bool) {
	m.MemoryMapMutex.RLock()
	defer m.MemoryMapMutex.RUnlock()

	memoryMap, mapExists := m.MemoryMaps[mapKey]

	if !mapExists {
		return nil, false
	}

	data, exists := (*memoryMap.Tiles)[tileKey]
	return data, exists
}

func (m *SharedMemoryCache) MemoryMapWrite(mapKey string, tileKey string, data *[]byte) {
	m.MemoryMapMutex.Lock()
	defer m.MemoryMapMutex.Unlock()

	i := 0

	for len(m.MemoryMapKeyHistory) > 0 && m.MemoryMapSize+len(*data) > m.MemoryMapMaxSize {
		i++
		m.DebugLogger("i" + strconv.Itoa(i))
		deleteKeys := m.MemoryMapKeyHistory[0]
		m.MemoryMapKeyHistory = m.MemoryMapKeyHistory[1:]

		deleteMemoryMap, deleteMapExisted := m.MemoryMaps[deleteKeys.MemoryMapKey]

		if deleteMapExisted {
			deleteSize := len((*deleteMemoryMap.Tiles)[deleteKeys.TileKey])
			delete(*deleteMemoryMap.Tiles, deleteKeys.TileKey)

			m.MemoryMapSize -= deleteSize

			m.logDebug("MemoryMapWrite would exceed maximum capacity. Deleted tile with key [" + deleteKeys.TileKey + "] from MemoryMap [" + deleteKeys.MemoryMapKey + "], recovered " + strconv.Itoa(deleteSize) + " Bytes.")
		} else {
			m.logDebug("MemoryMap with key [" + deleteKeys.MemoryMapKey + "] not found. Cannot delete tile to free up space...")
		}
	}

	// check if existed, update size if so
	memoryMap, mapExists := m.MemoryMaps[mapKey]

	if !mapExists {
		newMap := make(map[string][]byte)
		memoryMap = &MemoryMap{Tiles: &newMap}
		m.MemoryMaps[mapKey] = memoryMap
		m.logDebug("Memory Map with key [" + mapKey + "] did not exist. Created map!")
	}

	prevData, existed := (*memoryMap.Tiles)[tileKey]

	if existed {
		m.MemoryMapSize -= len(prevData)
	}

	(*memoryMap.Tiles)[tileKey] = *data
	m.MemoryMapKeyHistory = append(m.MemoryMapKeyHistory, MemoryMapKeyHistoyItem{MemoryMapKey: mapKey, TileKey: tileKey})

	m.MemoryMapSize += len(*data)
}
