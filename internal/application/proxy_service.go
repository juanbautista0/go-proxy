package application

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
	"github.com/juanbautista0/go-proxy/internal/infrastructure"
)

type ProxyServiceImpl struct {
	config        *domain.Config
	metrics       *domain.TrafficMetrics
	mu            sync.RWMutex
	requestCount  int64
	loadBalancer  domain.LoadBalancer
	healthChecker domain.HealthChecker
	sessions      map[string]string
}

func NewProxyService(lb domain.LoadBalancer, hc domain.HealthChecker) *ProxyServiceImpl {
	return &ProxyServiceImpl{
		metrics:       &domain.TrafficMetrics{},
		loadBalancer:  lb,
		healthChecker: hc,
		sessions:      make(map[string]string),
	}
}

func (p *ProxyServiceImpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	atomic.AddInt64(&p.requestCount, 1)
	atomic.AddInt64(&p.metrics.ActiveConnections, 1)
	defer atomic.AddInt64(&p.metrics.ActiveConnections, -1)

	p.mu.RLock()
	config := p.config
	p.mu.RUnlock()

	if config == nil || len(config.Backends) == 0 {
		http.Error(w, "No backends available", http.StatusServiceUnavailable)
		return
	}

	backend := &config.Backends[0]
	clientIP := p.getClientIP(r)
	server := p.selectServerWithRetry(backend, clientIP, r)

	if server == nil {
		http.Error(w, "No active servers", http.StatusServiceUnavailable)
		return
	}

	target, _ := url.Parse(server.URL)
	proxy := p.createIntelligentProxy(target, server, start)
	proxy.ServeHTTP(w, r)
}

func (p *ProxyServiceImpl) selectServerWithRetry(backend *domain.Backend, clientIP string, r *http.Request) *domain.Server {
	if backend.StickySessions {
		if sessionServer := p.getSessionServer(r, backend); sessionServer != nil {
			return sessionServer
		}
	}

	retries := backend.Retries
	if retries == 0 {
		retries = 3
	}

	for i := 0; i < retries; i++ {
		server := p.loadBalancer.SelectServer(backend, clientIP)
		if server != nil {
			if backend.StickySessions {
				p.setSessionServer(r, server)
			}
			return server
		}
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}

func (p *ProxyServiceImpl) UpdateConfig(config *domain.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
	
	// Actualizar servidores en el balanceador
	if len(config.Backends) > 0 {
		if eb, ok := p.loadBalancer.(*infrastructure.EnterpriseBalancer); ok {
			eb.UpdateServers(config.Backends[0].Servers)
		}
	}
	
	return nil
}

func (p *ProxyServiceImpl) GetMetrics() *domain.TrafficMetrics {
	count := atomic.LoadInt64(&p.requestCount)
	p.metrics.RequestsPerSecond = int(count)
	p.metrics.TotalRequests = atomic.LoadInt64(&p.requestCount)
	p.metrics.LastUpdated = time.Now()
	atomic.StoreInt64(&p.requestCount, 0)
	return p.metrics
}

func (p *ProxyServiceImpl) GetServerStats() map[string]*domain.Server {
	// Obtener métricas reales del load balancer
	return p.loadBalancer.GetServerMetrics()
}

func (p *ProxyServiceImpl) getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func (p *ProxyServiceImpl) createIntelligentProxy(target *url.URL, server *domain.Server, start time.Time) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ModifyResponse = func(resp *http.Response) error {
		duration := time.Since(start)
		success := resp.StatusCode < 500
		p.loadBalancer.UpdateStats(server, duration, success)
		
		// Actualizar métricas globales
		if success {
			p.updateGlobalMetrics(duration, true)
		} else {
			p.updateGlobalMetrics(duration, false)
		}
		
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		duration := time.Since(start)
		p.loadBalancer.UpdateStats(server, duration, false)
		p.updateGlobalMetrics(duration, false)
		
		// Retry logic para alta disponibilidad
		if p.shouldRetry(err) {
			p.mu.RLock()
			currentConfig := p.config
			p.mu.RUnlock()
			
			if currentConfig != nil && len(currentConfig.Backends) > 0 {
				if retryServer := p.loadBalancer.SelectServer(&currentConfig.Backends[0], p.getClientIP(r)); retryServer != nil && retryServer.URL != server.URL {
					retryTarget, _ := url.Parse(retryServer.URL)
					retryProxy := httputil.NewSingleHostReverseProxy(retryTarget)
					retryProxy.ServeHTTP(w, r)
					return
				}
			}
		}
		
		http.Error(w, "Service Temporarily Unavailable", http.StatusServiceUnavailable)
	}

	return proxy
}

func (p *ProxyServiceImpl) getSessionServer(r *http.Request, backend *domain.Backend) *domain.Server {
	sessionID := p.getSessionID(r)
	if sessionID == "" {
		return nil
	}

	p.mu.RLock()
	serverURL, exists := p.sessions[sessionID]
	p.mu.RUnlock()

	if !exists {
		return nil
	}

	for i := range backend.Servers {
		server := &backend.Servers[i]
		if server.URL == serverURL && server.Active && server.Healthy {
			return server
		}
	}
	return nil
}

func (p *ProxyServiceImpl) setSessionServer(r *http.Request, server *domain.Server) {
	sessionID := p.getSessionID(r)
	if sessionID == "" {
		return
	}

	p.mu.Lock()
	p.sessions[sessionID] = server.URL
	p.mu.Unlock()
}

func (p *ProxyServiceImpl) getSessionID(r *http.Request) string {
	if cookie, err := r.Cookie("JSESSIONID"); err == nil {
		return cookie.Value
	}
	return r.Header.Get("X-Session-ID")
}

func (p *ProxyServiceImpl) updateGlobalMetrics(duration time.Duration, success bool) {
	// Actualizar tiempo de respuesta promedio
	if p.metrics.AverageResponseTime == 0 {
		p.metrics.AverageResponseTime = duration
	} else {
		p.metrics.AverageResponseTime = (p.metrics.AverageResponseTime + duration) / 2
	}
	
	// Actualizar error rate
	if !success {
		totalReqs := atomic.LoadInt64(&p.metrics.TotalRequests)
		if totalReqs > 0 {
			errorCount := float64(totalReqs) * p.metrics.ErrorRate
			errorCount++
			p.metrics.ErrorRate = errorCount / float64(totalReqs+1)
		}
	} else {
		totalReqs := atomic.LoadInt64(&p.metrics.TotalRequests)
		if totalReqs > 0 {
			errorCount := float64(totalReqs) * p.metrics.ErrorRate
			p.metrics.ErrorRate = errorCount / float64(totalReqs+1)
		}
	}
}

func (p *ProxyServiceImpl) shouldRetry(err error) bool {
	// Retry en casos específicos de error de red
	return err != nil && (strings.Contains(err.Error(), "connection refused") || 
						 strings.Contains(err.Error(), "timeout") ||
						 strings.Contains(err.Error(), "no route to host"))
}
