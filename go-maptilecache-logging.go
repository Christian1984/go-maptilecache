package maptilecache

import (
	"fmt"
	"strconv"
)

type LoggerConfig struct {
	LogPrefix    string
	LogDebugFunc func(string, bool)
	LogInfoFunc  func(string, bool)
	LogWarnFunc  func(string, bool)
	LogErrorFunc func(string, bool)
}

func (c *Cache) logDebug(message string) {
	msg := c.Logger.LogPrefix + ": " + message
	if c.Logger.LogDebugFunc != nil {
		c.Logger.LogDebugFunc(msg, false)
	} else {
		fmt.Println("[DEBUG] " + msg)
	}
}

func (c *Cache) logInfo(message string) {
	msg := c.Logger.LogPrefix + ": " + message
	if c.Logger.LogDebugFunc != nil {
		c.Logger.LogDebugFunc(msg, false)
	} else {
		fmt.Println("[INFO] " + msg)
	}
}

func (c *Cache) logWarn(message string) {
	msg := c.Logger.LogPrefix + ": " + message
	if c.Logger.LogDebugFunc != nil {
		c.Logger.LogDebugFunc(msg, false)
	} else {
		fmt.Println("[WARN] " + msg)
	}
}

func (c *Cache) logError(message string) {
	msg := c.Logger.LogPrefix + ": " + message
	if c.Logger.LogDebugFunc != nil {
		c.Logger.LogDebugFunc(msg, false)
	} else {
		fmt.Println("[ERROR] " + msg)
	}
}

func (c *Cache) LogStats() {
	cachePercentage := "0"
	originPercentage := "0"

	if c.Stats.BytesServedFromCache+c.Stats.BytesServedFromOrigin > 0 {
		cachePercentage = fmt.Sprintf("%.2f", 100*float64(c.Stats.BytesServedFromCache)/float64(c.Stats.BytesServedFromCache+c.Stats.BytesServedFromOrigin))
		originPercentage = fmt.Sprintf("%.2f", 100*float64(c.Stats.BytesServedFromOrigin)/float64(c.Stats.BytesServedFromCache+c.Stats.BytesServedFromOrigin))
	}

	c.logDebug("Served from Cache: " + strconv.Itoa(c.Stats.BytesServedFromCache) + " Bytes (" + cachePercentage + "%), Served from Origin: " + strconv.Itoa(c.Stats.BytesServedFromOrigin) + " Bytes (" + originPercentage + "%)")
}
