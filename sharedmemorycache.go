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

type TileKeyHistoryItem struct {
	MemoryMapKey string
	TileKey      string
}

type SharedMemoryCache struct {
	MemoryMaps            map[string]*MemoryMap
	TileKeyHistory        []TileKeyHistoryItem
	MapMutes              *sync.RWMutex
	HistoryMutex          *sync.RWMutex
	SizeBytes             int
	MaxSizeBytes          int
	EnsureMaxSizeInterval time.Duration
	DebugLogger           func(string)
	InfoLogger            func(string)
	WarnLogger            func(string)
	ErrorLogger           func(string)
}

type SharedMemoryCacheConfig struct {
	MaxSizeBytes          int
	EnsureMaxSizeInterval time.Duration
	DebugLogger           func(string)
	InfoLogger            func(string)
	WarnLogger            func(string)
	ErrorLogger           func(string)
}

func NewSharedMemoryCache(config SharedMemoryCacheConfig) *SharedMemoryCache {
	m := SharedMemoryCache{
		MemoryMaps:            make(map[string]*MemoryMap),
		TileKeyHistory:        []TileKeyHistoryItem{},
		MapMutes:              &sync.RWMutex{},
		HistoryMutex:          &sync.RWMutex{},
		MaxSizeBytes:          config.MaxSizeBytes,
		EnsureMaxSizeInterval: config.EnsureMaxSizeInterval,
		DebugLogger:           config.DebugLogger,
		InfoLogger:            config.InfoLogger,
		WarnLogger:            config.WarnLogger,
		ErrorLogger:           config.ErrorLogger,
	}

	if m.EnsureMaxSizeInterval > 0 {
		ticker := time.NewTicker(5 * time.Second)
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
	} else if m.MaxSizeBytes > 0 {
		m.logWarn("Memory Cache Max Size set, but ensure-interval to enforce size limit not set")
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
	m.logDebug("EnsureMaxSize() called...")
	start := time.Now()

	m.HistoryMutex.Lock()
	defer m.HistoryMutex.Unlock()

	deleteCount := 0
	for len(m.TileKeyHistory) > 0 && m.SizeBytes > m.MaxSizeBytes {
		deleteKeys := m.TileKeyHistory[0]
		m.TileKeyHistory = m.TileKeyHistory[1:]

		m.MapMutes.RLock()
		deleteMemoryMap, deleteMapExisted := m.getMemoryMap(deleteKeys.MemoryMapKey)
		m.MapMutes.RUnlock()

		if deleteMapExisted {
			deleteMemoryMap.Mutex.Lock()
			deleteTile, _ := deleteMemoryMap.getTile(deleteKeys.TileKey)
			deleteSize := len(*deleteTile)
			m.SizeBytes -= deleteSize
			deleteMemoryMap.removeTile(deleteKeys.TileKey)
			deleteMemoryMap.Mutex.Unlock()

			deleteCount++

			m.logDebug("MemoryMapWrite would exceed maximum capacity. Deleted tile with key [" + deleteKeys.TileKey + "] from MemoryMap [" + deleteKeys.MemoryMapKey + "], recovered " + strconv.Itoa(deleteSize) + " Bytes.")
		} else {
			m.logDebug("MemoryMap with key [" + deleteKeys.MemoryMapKey + "] not found. Cannot delete tile to free up space...")
		}
	}

	duration := time.Since(start)
	m.logDebug("EnsureMaxSize() finished. Removed " + strconv.Itoa(deleteCount) + " tiles (took " + duration.String() + ").")
}

func (m *SharedMemoryCache) MemoryMapRead(mapKey string, tileKey string) (*[]byte, bool) {
	m.MapMutes.RLock()
	memoryMap, mapExists := m.getMemoryMap(mapKey)
	m.MapMutes.RUnlock()

	if !mapExists {
		return nil, false
	}

	memoryMap.Mutex.RLock()
	data, exists := memoryMap.getTile(tileKey)
	memoryMap.Mutex.RUnlock()

	return data, exists
}

func (m *SharedMemoryCache) MemoryMapWrite(mapKey string, tileKey string, data *[]byte) {
	m.MapMutes.Lock()
	memoryMap := m.addMemoryMapIfNotExists(mapKey)
	m.MapMutes.Unlock()

	memoryMap.Mutex.Lock()
	prevData, _ := memoryMap.getTile(tileKey)
	oldDataSize := len(*prevData)
	newDataSize := len(*data)
	memoryMap.addTile(tileKey, data)
	memoryMap.Mutex.Unlock()

	m.HistoryMutex.Lock()
	m.SizeBytes -= oldDataSize
	m.TileKeyHistory = append(m.TileKeyHistory, TileKeyHistoryItem{MemoryMapKey: mapKey, TileKey: tileKey})
	m.SizeBytes += newDataSize
	m.HistoryMutex.Unlock()
}
