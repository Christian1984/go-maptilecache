package maptilecache

import (
	"strconv"
	"sync"
	"time"
)

type MemoryMap struct {
	Tiles *map[string][]byte
	Mutex *sync.RWMutex
}

type MemoryMapKeyHistoyItem struct {
	MemoryMapKey string
	TileKey      string
}

type SharedMemoryCache struct {
	MemoryMaps            map[string]*MemoryMap
	MemoryMapKeyHistory   []MemoryMapKeyHistoyItem
	SharedMutex           *sync.RWMutex
	MemoryMapHistoryMutex *sync.RWMutex
	MemoryMapSize         int
	MemoryMapMaxSize      int
	DebugLogger           func(string)
	InfoLogger            func(string)
	WarnLogger            func(string)
	ErrorLogger           func(string)
}

type SharedMemoryCacheConfig struct {
	MaxMemoryFootprint int
	DebugLogger        func(string)
	InfoLogger         func(string)
	WarnLogger         func(string)
	ErrorLogger        func(string)
}

func NewSharedMemoryCache(config SharedMemoryCacheConfig) *SharedMemoryCache {
	m := SharedMemoryCache{
		MemoryMaps:            make(map[string]*MemoryMap),
		MemoryMapKeyHistory:   []MemoryMapKeyHistoyItem{},
		SharedMutex:           &sync.RWMutex{},
		MemoryMapHistoryMutex: &sync.RWMutex{},
		MemoryMapMaxSize:      config.MaxMemoryFootprint,
		DebugLogger:           config.DebugLogger,
		InfoLogger:            config.InfoLogger,
		WarnLogger:            config.WarnLogger,
		ErrorLogger:           config.ErrorLogger,
	}

	ticker := time.NewTicker(30 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				m.EnsureMaxSize()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

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

func (m *SharedMemoryCache) getMemoryMap(mapKey string) (*MemoryMap, bool) {
	memoryMap, mapExists := m.MemoryMaps[mapKey]

	return memoryMap, mapExists
}

func (m *SharedMemoryCache) addMemoryMapIfNotExists(mapKey string) *MemoryMap {
	memoryMap := m.MemoryMaps[mapKey]

	if memoryMap == nil {
		newMap := make(map[string][]byte)
		memoryMap = &MemoryMap{Tiles: &newMap, Mutex: &sync.RWMutex{}}
		m.MemoryMaps[mapKey] = memoryMap
		m.logDebug("Memory Map with key [" + mapKey + "] did not exist. Created map!")
	}

	return memoryMap
}

func (mm *MemoryMap) getTile(tileKey string) (*[]byte, bool) {
	data, exists := (*mm.Tiles)[tileKey]
	return &data, exists
}

func (mm *MemoryMap) addTile(tileKey string, data *[]byte) {
	(*mm.Tiles)[tileKey] = *data
}

func (mm *MemoryMap) removeTile(tileKey string) {
	delete(*mm.Tiles, tileKey)
}

func (m *SharedMemoryCache) EnsureMaxSize() {
	m.logDebug("EnsureMaxSize()  called...")

	m.MemoryMapHistoryMutex.Lock()
	defer m.MemoryMapHistoryMutex.Unlock()

	for len(m.MemoryMapKeyHistory) > 0 && m.MemoryMapSize > m.MemoryMapMaxSize {
		deleteKeys := m.MemoryMapKeyHistory[0]
		m.MemoryMapKeyHistory = m.MemoryMapKeyHistory[1:]

		m.SharedMutex.RLock()
		deleteMemoryMap, deleteMapExisted := m.getMemoryMap(deleteKeys.MemoryMapKey)
		m.SharedMutex.RUnlock()

		if deleteMapExisted {
			deleteMemoryMap.Mutex.Lock()
			deleteTile, _ := deleteMemoryMap.getTile(deleteKeys.TileKey)
			deleteSize := len(*deleteTile)
			deleteMemoryMap.removeTile(deleteKeys.TileKey)
			deleteMemoryMap.Mutex.Unlock()

			m.MemoryMapSize -= deleteSize

			m.logDebug("MemoryMapWrite would exceed maximum capacity. Deleted tile with key [" + deleteKeys.TileKey + "] from MemoryMap [" + deleteKeys.MemoryMapKey + "], recovered " + strconv.Itoa(deleteSize) + " Bytes.")
		} else {
			m.logDebug("MemoryMap with key [" + deleteKeys.MemoryMapKey + "] not found. Cannot delete tile to free up space...")
		}
	}
}

func (m *SharedMemoryCache) MemoryMapRead(mapKey string, tileKey string) (*[]byte, bool) {
	m.SharedMutex.RLock()
	memoryMap, mapExists := m.getMemoryMap(mapKey)
	m.SharedMutex.RUnlock()

	if !mapExists {
		return nil, false
	}

	memoryMap.Mutex.RLock()
	data, exists := memoryMap.getTile(tileKey)
	memoryMap.Mutex.RUnlock()

	return data, exists
}

func (m *SharedMemoryCache) MemoryMapWrite(mapKey string, tileKey string, data *[]byte) {
	m.SharedMutex.Lock()
	memoryMap := m.addMemoryMapIfNotExists(mapKey)
	m.SharedMutex.Unlock()

	memoryMap.Mutex.Lock()
	prevData, _ := memoryMap.getTile(tileKey)
	oldDataSize := len(*prevData)
	newDataSize := len(*data)
	memoryMap.addTile(tileKey, data)
	memoryMap.Mutex.Unlock()

	m.MemoryMapHistoryMutex.Lock()
	m.MemoryMapSize -= oldDataSize
	m.MemoryMapKeyHistory = append(m.MemoryMapKeyHistory, MemoryMapKeyHistoyItem{MemoryMapKey: mapKey, TileKey: tileKey})
	m.MemoryMapSize += newDataSize
	m.MemoryMapHistoryMutex.Unlock()
}
