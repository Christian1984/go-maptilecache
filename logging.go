package maptilecache

import (
	"fmt"
	"strconv"
)

// configure Log*Func with external callbacks to route log
// message to an existing logger of the embedding application
type LoggerConfig struct {
	LogPrefix    string
	LogDebugFunc func(string)
	LogInfoFunc  func(string)
	LogWarnFunc  func(string)
	LogErrorFunc func(string)
}

func (c *Cache) log(level string, message string, logFunc func(string)) {
	msg := c.Logger.LogPrefix + ": " + message
	if logFunc != nil {
		logFunc(msg)
	} else {
		fmt.Println("[" + level + "] " + msg)
	}

}

func (c *Cache) logDebug(message string) {
	c.log("DEBUG", message, c.Logger.LogDebugFunc)
}

func (c *Cache) logInfo(message string) {
	c.log("INFO", message, c.Logger.LogInfoFunc)
}

func (c *Cache) logWarn(message string) {
	c.log("WARN", message, c.Logger.LogWarnFunc)
}

func (c *Cache) logError(message string) {
	c.log("ERROR", message, c.Logger.LogErrorFunc)
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
