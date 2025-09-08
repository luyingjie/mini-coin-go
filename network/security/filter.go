package security

import (
	"context"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// FilterType 过滤器类型
type FilterType int

const (
	FilterTypeBlacklist FilterType = iota // 黑名单
	FilterTypeWhitelist                   // 白名单
	FilterTypeRateLimit                   // 速率限制
	FilterTypeDDoS                        // DDoS防护
)

// Filter 消息过滤器接口
type Filter interface {
	ShouldAllow(ctx context.Context, addr string, messageType string) bool
	GetFilterType() FilterType
	GetName() string
}

// BlacklistFilter 黑名单过滤器
type BlacklistFilter struct {
	name      string
	blacklist map[string]time.Time // IP -> 封禁时间
	mutex     sync.RWMutex
}

// NewBlacklistFilter 创建黑名单过滤器
func NewBlacklistFilter() *BlacklistFilter {
	return &BlacklistFilter{
		name:      "黑名单过滤器",
		blacklist: make(map[string]time.Time),
	}
}

// ShouldAllow 检查是否允许
func (bf *BlacklistFilter) ShouldAllow(ctx context.Context, addr string, messageType string) bool {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	ip := extractIP(addr)
	if banTime, exists := bf.blacklist[ip]; exists {
		// 检查封禁是否已过期
		if time.Now().Before(banTime) {
			log.Printf("阻止黑名单IP: %s", ip)
			return false
		}
		// 封禁已过期，移除
		delete(bf.blacklist, ip)
	}

	return true
}

// GetFilterType 获取过滤器类型
func (bf *BlacklistFilter) GetFilterType() FilterType {
	return FilterTypeBlacklist
}

// GetName 获取过滤器名称
func (bf *BlacklistFilter) GetName() string {
	return bf.name
}

// AddToBlacklist 添加到黑名单
func (bf *BlacklistFilter) AddToBlacklist(ip string, duration time.Duration) {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	bf.blacklist[ip] = time.Now().Add(duration)
	log.Printf("添加IP到黑名单: %s，持续时间: %v", ip, duration)
}

// RemoveFromBlacklist 从黑名单移除
func (bf *BlacklistFilter) RemoveFromBlacklist(ip string) {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	delete(bf.blacklist, ip)
	log.Printf("从黑名单移除IP: %s", ip)
}

// GetBlacklistIPs 获取黑名单IP列表
func (bf *BlacklistFilter) GetBlacklistIPs() []string {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	var ips []string
	for ip, banTime := range bf.blacklist {
		if time.Now().Before(banTime) {
			ips = append(ips, ip)
		}
	}
	return ips
}

// WhitelistFilter 白名单过滤器
type WhitelistFilter struct {
	name      string
	whitelist map[string]bool // IP -> 是否允许
	enabled   bool
	mutex     sync.RWMutex
}

// NewWhitelistFilter 创建白名单过滤器
func NewWhitelistFilter() *WhitelistFilter {
	return &WhitelistFilter{
		name:      "白名单过滤器",
		whitelist: make(map[string]bool),
		enabled:   false,
	}
}

// ShouldAllow 检查是否允许
func (wf *WhitelistFilter) ShouldAllow(ctx context.Context, addr string, messageType string) bool {
	if !wf.enabled {
		return true // 白名单未启用，允许所有
	}

	wf.mutex.RLock()
	defer wf.mutex.RUnlock()

	ip := extractIP(addr)
	if allowed, exists := wf.whitelist[ip]; exists && allowed {
		return true
	}

	log.Printf("阻止非白名单IP: %s", ip)
	return false
}

// GetFilterType 获取过滤器类型
func (wf *WhitelistFilter) GetFilterType() FilterType {
	return FilterTypeWhitelist
}

// GetName 获取过滤器名称
func (wf *WhitelistFilter) GetName() string {
	return wf.name
}

// Enable 启用白名单
func (wf *WhitelistFilter) Enable() {
	wf.mutex.Lock()
	defer wf.mutex.Unlock()
	wf.enabled = true
}

// Disable 禁用白名单
func (wf *WhitelistFilter) Disable() {
	wf.mutex.Lock()
	defer wf.mutex.Unlock()
	wf.enabled = false
}

// AddToWhitelist 添加到白名单
func (wf *WhitelistFilter) AddToWhitelist(ip string) {
	wf.mutex.Lock()
	defer wf.mutex.Unlock()

	wf.whitelist[ip] = true
	log.Printf("添加IP到白名单: %s", ip)
}

// RemoveFromWhitelist 从白名单移除
func (wf *WhitelistFilter) RemoveFromWhitelist(ip string) {
	wf.mutex.Lock()
	defer wf.mutex.Unlock()

	delete(wf.whitelist, ip)
	log.Printf("从白名单移除IP: %s", ip)
}

// RateLimitFilter 速率限制过滤器
type RateLimitFilter struct {
	name        string
	requests    map[string][]time.Time // IP -> 请求时间列表
	maxRequests int                    // 最大请求数
	timeWindow  time.Duration          // 时间窗口
	mutex       sync.RWMutex
}

// NewRateLimitFilter 创建速率限制过滤器
func NewRateLimitFilter(maxRequests int, timeWindow time.Duration) *RateLimitFilter {
	return &RateLimitFilter{
		name:        "速率限制过滤器",
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		timeWindow:  timeWindow,
	}
}

// ShouldAllow 检查是否允许
func (rlf *RateLimitFilter) ShouldAllow(ctx context.Context, addr string, messageType string) bool {
	rlf.mutex.Lock()
	defer rlf.mutex.Unlock()

	ip := extractIP(addr)
	now := time.Now()

	// 获取该IP的请求历史
	requests, exists := rlf.requests[ip]
	if !exists {
		requests = make([]time.Time, 0)
	}

	// 清理过期的请求记录
	var validRequests []time.Time
	cutoff := now.Add(-rlf.timeWindow)
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// 检查是否超过限制
	if len(validRequests) >= rlf.maxRequests {
		log.Printf("速率限制阻止IP: %s，请求数: %d/%d", ip, len(validRequests), rlf.maxRequests)
		return false
	}

	// 添加当前请求
	validRequests = append(validRequests, now)
	rlf.requests[ip] = validRequests

	return true
}

// GetFilterType 获取过滤器类型
func (rlf *RateLimitFilter) GetFilterType() FilterType {
	return FilterTypeRateLimit
}

// GetName 获取过滤器名称
func (rlf *RateLimitFilter) GetName() string {
	return rlf.name
}

// GetRequestStats 获取请求统计
func (rlf *RateLimitFilter) GetRequestStats() map[string]int {
	rlf.mutex.RLock()
	defer rlf.mutex.RUnlock()

	stats := make(map[string]int)
	cutoff := time.Now().Add(-rlf.timeWindow)

	for ip, requests := range rlf.requests {
		validCount := 0
		for _, reqTime := range requests {
			if reqTime.After(cutoff) {
				validCount++
			}
		}
		stats[ip] = validCount
	}

	return stats
}

// DDoSFilter DDoS防护过滤器
type DDoSFilter struct {
	name            string
	connectionCount map[string]int       // IP -> 连接数
	lastRequest     map[string]time.Time // IP -> 最后请求时间
	maxConnections  int                  // 最大连接数
	suspiciousIPs   map[string]time.Time // 可疑IP -> 检测时间
	mutex           sync.RWMutex
	blacklistFilter *BlacklistFilter // 黑名单过滤器引用
}

// NewDDoSFilter 创建DDoS防护过滤器
func NewDDoSFilter(maxConnections int, blacklistFilter *BlacklistFilter) *DDoSFilter {
	return &DDoSFilter{
		name:            "DDoS防护过滤器",
		connectionCount: make(map[string]int),
		lastRequest:     make(map[string]time.Time),
		maxConnections:  maxConnections,
		suspiciousIPs:   make(map[string]time.Time),
		blacklistFilter: blacklistFilter,
	}
}

// ShouldAllow 检查是否允许
func (df *DDoSFilter) ShouldAllow(ctx context.Context, addr string, messageType string) bool {
	df.mutex.Lock()
	defer df.mutex.Unlock()

	ip := extractIP(addr)
	now := time.Now()

	// 更新连接统计
	df.connectionCount[ip]++
	df.lastRequest[ip] = now

	// 检查连接数是否超过限制
	if df.connectionCount[ip] > df.maxConnections {
		log.Printf("DDoS检测到可疑IP: %s，连接数: %d", ip, df.connectionCount[ip])

		// 标记为可疑IP
		df.suspiciousIPs[ip] = now

		// 如果连接数严重超标，添加到黑名单
		if df.connectionCount[ip] > df.maxConnections*2 && df.blacklistFilter != nil {
			df.blacklistFilter.AddToBlacklist(ip, 10*time.Minute)
		}

		return false
	}

	return true
}

// GetFilterType 获取过滤器类型
func (df *DDoSFilter) GetFilterType() FilterType {
	return FilterTypeDDoS
}

// GetName 获取过滤器名称
func (df *DDoSFilter) GetName() string {
	return df.name
}

// CleanupConnections 清理过期连接统计
func (df *DDoSFilter) CleanupConnections() {
	df.mutex.Lock()
	defer df.mutex.Unlock()

	now := time.Now()
	timeout := 5 * time.Minute

	for ip, lastTime := range df.lastRequest {
		if now.Sub(lastTime) > timeout {
			delete(df.connectionCount, ip)
			delete(df.lastRequest, ip)
		}
	}

	// 清理过期的可疑IP
	for ip, detectTime := range df.suspiciousIPs {
		if now.Sub(detectTime) > time.Hour {
			delete(df.suspiciousIPs, ip)
		}
	}
}

// IsAllowed 检查是否允许请求（为了兼容性）
func (df *DDoSFilter) IsAllowed(addr string) bool {
	return df.ShouldAllow(context.Background(), addr, "")
}

// GetStats 获取统计信息
func (df *DDoSFilter) GetStats() map[string]interface{} {
	df.mutex.RLock()
	defer df.mutex.RUnlock()

	totalConnections := 0
	for _, count := range df.connectionCount {
		totalConnections += count
	}

	return map[string]interface{}{
		"total_connections": totalConnections,
		"unique_ips":        len(df.connectionCount),
		"suspicious_ips":    len(df.suspiciousIPs),
		"max_connections":   df.maxConnections,
	}
}

// GetSuspiciousIPs 获取可疑IP列表
func (df *DDoSFilter) GetSuspiciousIPs() []string {
	df.mutex.RLock()
	defer df.mutex.RUnlock()

	var ips []string
	for ip := range df.suspiciousIPs {
		ips = append(ips, ip)
	}
	return ips
}

// extractIP 从地址中提取IP
func extractIP(addr string) string {
	if strings.Contains(addr, ":") {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return addr
		}
		return host
	}
	return addr
}

// MessageFilterManager 消息过滤器管理器
type MessageFilterManager struct {
	filters []Filter
	mutex   sync.RWMutex
	stats   *FilterStats
}

// FilterStats 过滤器统计信息
type FilterStats struct {
	TotalRequests   int64 // 总请求数
	BlockedRequests int64 // 被阻止的请求数
	AllowedRequests int64 // 被允许的请求数
	mutex           sync.RWMutex
}

// NewMessageFilterManager 创建消息过滤器管理器
func NewMessageFilterManager() *MessageFilterManager {
	return &MessageFilterManager{
		filters: make([]Filter, 0),
		stats:   &FilterStats{},
	}
}

// AddFilter 添加过滤器
func (mfm *MessageFilterManager) AddFilter(filter Filter) {
	mfm.mutex.Lock()
	defer mfm.mutex.Unlock()

	mfm.filters = append(mfm.filters, filter)
	log.Printf("添加消息过滤器: %s", filter.GetName())
}

// RemoveFilter 移除过滤器
func (mfm *MessageFilterManager) RemoveFilter(filterType FilterType) {
	mfm.mutex.Lock()
	defer mfm.mutex.Unlock()

	for i, filter := range mfm.filters {
		if filter.GetFilterType() == filterType {
			mfm.filters = append(mfm.filters[:i], mfm.filters[i+1:]...)
			log.Printf("移除消息过滤器: %s", filter.GetName())
			break
		}
	}
}

// ShouldAllow 检查消息是否应被允许
func (mfm *MessageFilterManager) ShouldAllow(ctx context.Context, addr string, messageType string) bool {
	mfm.mutex.RLock()
	filters := make([]Filter, len(mfm.filters))
	copy(filters, mfm.filters)
	mfm.mutex.RUnlock()

	mfm.stats.mutex.Lock()
	mfm.stats.TotalRequests++
	mfm.stats.mutex.Unlock()

	// 逐个检查过滤器
	for _, filter := range filters {
		if !filter.ShouldAllow(ctx, addr, messageType) {
			mfm.stats.mutex.Lock()
			mfm.stats.BlockedRequests++
			mfm.stats.mutex.Unlock()

			log.Printf("消息被过滤器阻止: %s，来源: %s，类型: %s",
				filter.GetName(), addr, messageType)
			return false
		}
	}

	mfm.stats.mutex.Lock()
	mfm.stats.AllowedRequests++
	mfm.stats.mutex.Unlock()

	return true
}

// GetStats 获取过滤器统计信息
func (mfm *MessageFilterManager) GetStats() *FilterStats {
	mfm.stats.mutex.RLock()
	defer mfm.stats.mutex.RUnlock()

	return &FilterStats{
		TotalRequests:   mfm.stats.TotalRequests,
		BlockedRequests: mfm.stats.BlockedRequests,
		AllowedRequests: mfm.stats.AllowedRequests,
	}
}

// GetFilterInfo 获取过滤器信息
func (mfm *MessageFilterManager) GetFilterInfo() map[string]interface{} {
	mfm.mutex.RLock()
	defer mfm.mutex.RUnlock()

	var filterNames []string
	for _, filter := range mfm.filters {
		filterNames = append(filterNames, filter.GetName())
	}

	stats := mfm.GetStats()
	blockRate := float64(0)
	if stats.TotalRequests > 0 {
		blockRate = float64(stats.BlockedRequests) / float64(stats.TotalRequests) * 100
	}

	return map[string]interface{}{
		"filter_count":     len(mfm.filters),
		"filter_names":     filterNames,
		"total_requests":   stats.TotalRequests,
		"blocked_requests": stats.BlockedRequests,
		"allowed_requests": stats.AllowedRequests,
		"block_rate":       blockRate,
	}
}
