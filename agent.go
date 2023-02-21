package gocbcorex

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type agentState struct {
	bucket             string
	tlsConfig          *tls.Config
	authenticator      Authenticator
	numPoolConnections uint

	lastClients  map[string]*KvClientConfig
	latestConfig *routeConfig
}

type Agent struct {
	logger *zap.Logger
	lock   sync.Mutex
	state  agentState

	poller      ConfigPoller
	configMgr   ConfigManager
	connMgr     KvClientManager
	collections CollectionResolver
	retries     RetryManager
	vbRouter    VbucketRouter
	httpMgr     HTTPClientManager

	cfgHandler *agentConfigHandler

	crud  *CrudComponent
	http  *HTTPComponent
	query *QueryComponent
}

func CreateAgent(ctx context.Context, opts AgentOptions) (*Agent, error) {
	var srcHTTPAddrs []string
	for _, hostPort := range opts.SeedConfig.HTTPAddrs {
		if opts.TLSConfig == nil {
			ep := fmt.Sprintf("http://%s", hostPort)
			srcHTTPAddrs = append(srcHTTPAddrs, ep)
		} else {
			ep := fmt.Sprintf("https://%s", hostPort)
			srcHTTPAddrs = append(srcHTTPAddrs, ep)
		}
	}

	// Default values.
	compressionMinSize := 32
	compressionMinRatio := 0.83
	httpIdleConnTimeout := 4500 * time.Millisecond
	httpConnectTimeout := 30 * time.Second
	confHTTPRetryDelay := 10 * time.Second
	confHTTPRedialPeriod := 10 * time.Second
	confHTTPMaxWait := 5 * time.Second

	disableDecompression := opts.CompressionConfig.DisableDecompression
	useCompression := opts.CompressionConfig.EnableCompression

	if opts.CompressionConfig.MinSize > 0 {
		compressionMinSize = opts.CompressionConfig.MinSize
	}
	if opts.CompressionConfig.MinRatio > 0 {
		compressionMinRatio = opts.CompressionConfig.MinRatio
		if compressionMinRatio >= 1.0 {
			compressionMinRatio = 1.0
		}
	}
	if opts.HTTPConfig.IdleConnectionTimeout > 0 {
		httpIdleConnTimeout = opts.HTTPConfig.IdleConnectionTimeout
	}
	if opts.HTTPConfig.ConnectTimeout > 0 {
		httpConnectTimeout = opts.HTTPConfig.ConnectTimeout
	}
	if opts.ConfigPollerConfig.HTTPRetryDelay > 0 {
		confHTTPRetryDelay = opts.ConfigPollerConfig.HTTPRetryDelay
	}
	if opts.ConfigPollerConfig.HTTPRedialPeriod > 0 {
		confHTTPRedialPeriod = opts.ConfigPollerConfig.HTTPRedialPeriod
	}
	if opts.ConfigPollerConfig.HTTPMaxWait > 0 {
		confHTTPMaxWait = opts.ConfigPollerConfig.HTTPMaxWait
	}

	agent := &Agent{
		logger: loggerOrNop(opts.Logger),

		state: agentState{
			bucket:             opts.BucketName,
			tlsConfig:          opts.TLSConfig,
			authenticator:      opts.Authenticator,
			numPoolConnections: 1,
		},

		configMgr: NewConfigManager(&RouteConfigManagerOptions{
			Logger: opts.Logger,
		}),
		retries: NewRetryManagerFastFail(),
	}
	agent.cfgHandler = &agentConfigHandler{agent: agent}

	clients := make(map[string]*KvClientConfig)
	for addrIdx, addr := range opts.SeedConfig.MemdAddrs {
		nodeId := fmt.Sprintf("bootstrap-%d", addrIdx)
		clients[nodeId] = &KvClientConfig{
			Logger:         agent.logger,
			Address:        addr,
			TlsConfig:      agent.state.tlsConfig,
			SelectedBucket: agent.state.bucket,
			Authenticator:  agent.state.authenticator,
		}
	}
	connMgr, err := NewKvClientManager(&KvClientManagerConfig{
		NumPoolConnections: agent.state.numPoolConnections,
		Clients:            clients,
	}, &KvClientManagerOptions{
		Logger: agent.logger,
	})
	if err != nil {
		return nil, err
	}
	agent.connMgr = connMgr

	coreCollections, err := NewCollectionResolverMemd(&CollectionResolverMemdOptions{
		Logger:  agent.logger,
		ConnMgr: agent.connMgr,
	})
	if err != nil {
		return nil, err
	}
	collections, err := NewCollectionResolverCached(&CollectionResolverCachedOptions{
		Logger:         agent.logger,
		Resolver:       coreCollections,
		ResolveTimeout: 10 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	agent.collections = collections

	httpMgr, err := NewHTTPClientManager(&HTTPClientManagerConfig{
		HTTPClientConfig: HTTPClientConfig{
			Authenticator: opts.Authenticator,
			MgmtEndpoints: srcHTTPAddrs,
		},
		TLSConfig: opts.TLSConfig,
	}, &HTTPClientManagerOptions{
		Logger:         agent.logger,
		ConnectTimeout: httpConnectTimeout,
		IdleTimeout:    httpIdleConnTimeout,
		// We default these values to whatever the Go HTTP library defaults are.
		MaxIdleConns:        opts.HTTPConfig.MaxIdleConns,
		MaxIdleConnsPerHost: opts.HTTPConfig.MaxIdleConnsPerHost,
	})
	agent.httpMgr = httpMgr

	agent.vbRouter = NewVbucketRouter(&VbucketRouterOptions{
		Logger: agent.logger,
	})

	agent.configMgr.RegisterCallback(agent.cfgHandler)

	agent.poller = newhttpConfigPoller(httpPollerProperties{
		Logger:               opts.Logger,
		ConfHTTPRetryDelay:   confHTTPRetryDelay,
		ConfHTTPRedialPeriod: confHTTPRedialPeriod,
		ConfHTTPMaxWait:      confHTTPMaxWait,
		BucketName:           opts.BucketName,
		HTTPClient:           agent.httpMgr,
	})

	err = agent.startConfigWatcher(ctx)
	if err != nil {
		return nil, err
	}

	agent.crud = &CrudComponent{
		logger:      agent.logger,
		collections: agent.collections,
		retries:     agent.retries,
		connManager: agent.connMgr,
		vbs:         agent.vbRouter,
		compression: &CompressionManagerDefault{
			disableCompression:   !useCompression,
			compressionMinSize:   compressionMinSize,
			compressionMinRatio:  compressionMinRatio,
			disableDecompression: disableDecompression,
		},
	}
	agent.http = &HTTPComponent{
		logger:  agent.logger,
		httpMgr: agent.httpMgr,
		retries: agent.retries,
	}
	agent.query = &QueryComponent{
		httpCmpt:   agent.http,
		logger:     agent.logger,
		queryCache: make(map[string]string),
	}

	return agent, nil
}

func (agent *Agent) Reconfigure(opts *AgentReconfigureOptions) error {
	agent.lock.Lock()
	defer agent.lock.Unlock()

	if agent.state.bucket != "" {
		if opts.BucketName != agent.state.bucket {
			return errors.New("cannot change an already-specified bucket name")
		}
	}

	agent.state.tlsConfig = opts.TLSConfig
	agent.state.authenticator = opts.Authenticator
	agent.state.bucket = opts.BucketName
	agent.updateStateLocked()

	return nil
}

func (agent *Agent) Close() error {
	if err := agent.poller.Close(); err != nil {
		agent.logger.Debug("failed to close poller", zap.Error(err))
	}

	return nil
}

func (agent *Agent) handleRouteConfig(rc *routeConfig) {
	agent.lock.Lock()
	agent.state.latestConfig = rc
	agent.updateStateLocked()
	agent.lock.Unlock()
}

func (agent *Agent) makeHTTPEndpoints(endpoints routeEndpoints, useTLS bool) []string {
	if useTLS {
		l := make([]string, len(endpoints.SSLEndpoints))
		for epIdx, ep := range endpoints.SSLEndpoints {
			l[epIdx] = "https://" + ep
		}

		return l
	}

	l := make([]string, len(endpoints.NonSSLEndpoints))
	for epIdx, ep := range endpoints.NonSSLEndpoints {
		l[epIdx] = "http://" + ep
	}

	return l
}

func (agent *Agent) updateStateLocked() {
	agent.logger.Debug("updating components",
		zap.Reflect("state", agent.state),
		zap.Reflect("config", *agent.state.latestConfig))

	routeCfg := agent.state.latestConfig

	var memdTlsConfig *tls.Config
	var nodeNames []string
	var memdList []string

	useTLS := agent.state.tlsConfig != nil
	mgmtList := agent.makeHTTPEndpoints(routeCfg.mgmtEpList, useTLS)
	queryList := agent.makeHTTPEndpoints(routeCfg.n1qlEpList, useTLS)
	searchList := agent.makeHTTPEndpoints(routeCfg.ftsEpList, useTLS)

	nodeNames = make([]string, len(routeCfg.kvServerList.NonSSLEndpoints))
	for nodeIdx, addr := range routeCfg.kvServerList.NonSSLEndpoints {
		nodeNames[nodeIdx] = fmt.Sprintf("node@%s", addr)
	}

	if useTLS {
		memdList = make([]string, len(routeCfg.kvServerList.SSLEndpoints))
		copy(memdList, routeCfg.kvServerList.SSLEndpoints)
		memdTlsConfig = agent.state.tlsConfig
	} else {
		memdList = make([]string, len(routeCfg.kvServerList.NonSSLEndpoints))
		copy(memdList, routeCfg.kvServerList.NonSSLEndpoints)
		memdTlsConfig = nil
	}

	clients := make(map[string]*KvClientConfig)
	for addrIdx, addr := range memdList {
		nodeName := nodeNames[addrIdx]
		clients[nodeName] = &KvClientConfig{
			Address:        addr,
			TlsConfig:      memdTlsConfig,
			SelectedBucket: agent.state.bucket,
			Authenticator:  agent.state.authenticator,
		}
	}

	// In order to avoid race conditions between operations selecting the
	// endpoint they need to send the request to, and fetching an actual
	// client which can send to that endpoint.  We must first ensure that
	// all the new endpoints are available in the manager.  Then update
	// the routing table.  Then go back and remove the old entries from
	// the connection manager list.

	oldClients := make(map[string]*KvClientConfig)
	if agent.state.lastClients != nil {
		for clientName, client := range agent.state.lastClients {
			oldClients[clientName] = client
		}
	}
	for clientName, client := range clients {
		if oldClients[clientName] == nil {
			oldClients[clientName] = client
		}
	}

	agent.connMgr.Reconfigure(&KvClientManagerConfig{
		NumPoolConnections: agent.state.numPoolConnections,
		Clients:            oldClients,
	})

	agent.vbRouter.UpdateRoutingInfo(&VbucketRoutingInfo{
		VbMap:      routeCfg.vbMap,
		ServerList: nodeNames,
	})

	agent.connMgr.Reconfigure(&KvClientManagerConfig{
		NumPoolConnections: agent.state.numPoolConnections,
		Clients:            clients,
	})

	agent.httpMgr.Reconfigure(&HTTPClientManagerConfig{
		HTTPClientConfig: HTTPClientConfig{
			Authenticator:   agent.state.authenticator,
			MgmtEndpoints:   mgmtList,
			QueryEndpoints:  queryList,
			SearchEndpoints: searchList,
		},
		TLSConfig: nil,
	})
}

func (agent *Agent) startConfigWatcher(ctx context.Context) error {
	configCh, err := agent.poller.Watch()
	if err != nil {
		return err
	}

	var firstConfig *TerseConfigJsonWithSource
	select {
	case config := <-configCh:
		firstConfig = config
	case <-ctx.Done():
		if err := agent.poller.Close(); err != nil {
			agent.logger.Debug("failed to close poller", zap.Error(err))
		}
		return ctx.Err()
	}

	agent.configMgr.ApplyConfig(firstConfig.SourceHostname, firstConfig.Config)

	go func() {
		for config := range configCh {
			agent.configMgr.ApplyConfig(config.SourceHostname, config.Config)
		}
	}()

	return nil
}

// agentConfigHandler exists for the purpose of satisfying the HandleRouteConfig interface for Agent, with having
// to publicly expose the function on Agent itself.
type agentConfigHandler struct {
	agent *Agent
}

func (ach *agentConfigHandler) HandleRouteConfig(config *routeConfig) {
	ach.agent.handleRouteConfig(config)
}
