package maptilecache

import (
	"fmt"
	"strconv"
	"time"
)

// configure Log*Func with external callbacks to route log
// message to an existing logger of the embedding application
type LoggerConfig struct {
	LogPrefix     string
	LogDebugFunc  func(string)
	LogInfoFunc   func(string)
	LogWarnFunc   func(string)
	LogErrorFunc  func(string)
	StatsLogDelay time.Duration
}

func (c *Cache) log(message string, logFunc func(string)) {
	if logFunc != nil {
		logFunc(c.Logger.LogPrefix + ": " + message)
	}

}

// Default Logger
func println(level string, message string) {
	fmt.Println("[" + level + "] " + message)
}

func PrintlnDebugLogger(message string) {
	println("DEBUG", message)
}

func PrintlnInfoLogger(message string) {
	println("INFO", message)
}

func PrintlnWarnLogger(message string) {
	println("WARN", message)
}

func PrintlnErrorLogger(message string) {
	println("ERROR", message)
}

// Log Functions
func (c *Cache) logDebug(message string) {
	c.log(message, c.Logger.LogDebugFunc)
}

func (c *Cache) logInfo(message string) {
	c.log(message, c.Logger.LogInfoFunc)
}

func (c *Cache) logWarn(message string) {
	c.log(message, c.Logger.LogWarnFunc)
}

func (c *Cache) logError(message string) {
	c.log(message, c.Logger.LogErrorFunc)
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

func (c *Cache) InitLogStatsRunner() {
	if c.Logger.StatsLogDelay == 0 {
		c.logDebug("Will not log stats periodically, reason: StatsLogDelay not set")
		return
	}

	go func() {
		for {
			c.LogStats()
			time.Sleep(c.Logger.StatsLogDelay)
		}
	}()
}
