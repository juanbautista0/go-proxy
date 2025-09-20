package domain

import "time"

type Config struct {
	Proxy    ProxyConfig             `yaml:"proxy"`
	Backends []Backend               `yaml:"backends"`
	Triggers TriggerConfig           `yaml:"triggers"`
	Actions  map[string]ActionConfig `yaml:"actions"`
	Security SecurityConfig          `yaml:"security"`
}

type ProxyConfig struct {
	Port int `yaml:"port"`
}

type Backend struct {
	Name            string            `yaml:"name"`
	Servers         []Server          `yaml:"servers"`
	HealthCheck     string            `yaml:"health_check"`
	BalanceMode     string            `yaml:"balance_mode,omitempty"`
	StickySessions  bool              `yaml:"sticky_sessions,omitempty"`
	HealthInterval  time.Duration     `yaml:"health_interval,omitempty"`
	Timeout         time.Duration     `yaml:"timeout,omitempty"`
	Retries         int               `yaml:"retries,omitempty"`
	CircuitBreaker  CircuitBreakerCfg `yaml:"circuit_breaker,omitempty"`
	MinServers      int               `yaml:"min_servers,omitempty"`
	MaxServers      int               `yaml:"max_servers,omitempty"`
}

type Server struct {
	URL                 string        `yaml:"url"`
	Weight              int           `yaml:"weight"`
	Active              bool          `yaml:"active,omitempty"`
	MaxConnections      int           `yaml:"max_connections,omitempty"`
	HealthCheckEndpoint string        `yaml:"health_check_endpoint,omitempty"`
	CurrentConns        int64         `yaml:"-"`
	TotalRequests       int64         `yaml:"-"`
	FailedRequests      int64         `yaml:"-"`
	ResponseTime        time.Duration `yaml:"-"`
	LastHealthCheck     time.Time     `yaml:"-"`
	Healthy             bool          `yaml:"-"`
	CircuitOpen         bool          `yaml:"-"`
	CircuitOpenUntil    time.Time     `yaml:"-"`
}

type TriggerConfig struct {
	Smart    SmartTrigger      `yaml:"smart"`
	Traffic  TrafficTrigger    `yaml:"traffic"`
	Schedule []ScheduleTrigger `yaml:"schedule"`
}

type SmartTrigger struct {
	Enabled             bool          `yaml:"enabled"`
	EvaluationInterval  time.Duration `yaml:"evaluation_interval"`
	ShortWindow         time.Duration `yaml:"short_window"`
	LongWindow          time.Duration `yaml:"long_window"`
	Cooldown            time.Duration `yaml:"cooldown"`
	StabilityThreshold  float64       `yaml:"stability_threshold"`
	ScaleUpScore        float64       `yaml:"scale_up_score"`
	ScaleDownScore      float64       `yaml:"scale_down_score"`
	LongAvgScaleUpMin   float64       `yaml:"long_avg_scale_up_min"`
	LongAvgScaleDownMax float64       `yaml:"long_avg_scale_down_max"`
	TrendThreshold      float64       `yaml:"trend_threshold"`
}

type TrafficTrigger struct {
	HighThreshold int    `yaml:"high_threshold"`
	LowThreshold  int    `yaml:"low_threshold"`
	HighAction    string `yaml:"high_action"`
	LowAction     string `yaml:"low_action"`
}

type ScheduleTrigger struct {
	Time   string `yaml:"time"`
	Action string `yaml:"action"`
}

type ActionConfig struct {
	URL    string `yaml:"url"`
	Method string `yaml:"method"`
}

type TrafficMetrics struct {
	RequestsPerSecond   int
	TotalRequests       int64
	ActiveConnections   int64
	AverageResponseTime time.Duration
	ErrorRate           float64
	LastUpdated         time.Time
}

type SecurityConfig struct {
	APIKeys      []string `yaml:"api_keys"`
	AdminAPIKeys []string `yaml:"admin_api_keys"`
}

type CircuitBreakerCfg struct {
	FailureThreshold int           `yaml:"failure_threshold,omitempty"`
	RecoveryTimeout  time.Duration `yaml:"recovery_timeout,omitempty"`
	Enabled          bool          `yaml:"enabled,omitempty"`
}

type BalanceMode string

const (
	RoundRobin    BalanceMode = "roundrobin"
	LeastConn     BalanceMode = "leastconn"
	Weighted      BalanceMode = "weighted"
	IPHash        BalanceMode = "iphash"
	LeastResponse BalanceMode = "leastresponse"
)