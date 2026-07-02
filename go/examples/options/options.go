package options

import "time"

const defaultTimeout = 30
const defaultCaching = false

type Connection struct {
	addr    string
	cache   bool
	timeout time.Duration
}

func NewConnection(addr string) *Connection {
	return &Connection{
		addr:    addr,
		cache:   defaultCaching,
		timeout: defaultTimeout * time.Second,
	}
}

func NewConnectionWithOptions(addr string, cache bool, timeout time.Duration) *Connection {
	conn := NewConnection(addr)
	conn.cache = cache
	conn.timeout = timeout
	return conn
}

type ConnectionOption struct {
	Caching bool
	Timeout time.Duration
}

func NewDefaultOptions() *ConnectionOption {
	return &ConnectionOption{
		Caching: defaultCaching,
		Timeout: defaultTimeout * time.Second,
	}
}

func NewConnect(addr string, option *ConnectionOption) (*Connection, error) {
	return &Connection{
		addr:    addr,
		cache:   option.Caching,
		timeout: option.Timeout,
	}, nil
}

type options struct {
	caching bool
	timeout time.Duration
}

type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(opts *options) {
	f(opts)
}

func WithCaching(caching bool) Option {
	return optionFunc(func(opts *options) {
		opts.caching = caching
	})
}

func WithTimeout(timeout time.Duration) Option {
	return optionFunc(func(opts *options) {
		opts.timeout = timeout
	})
}

func NewConnections(addr string, opts ...Option) *Connection {
	option := options{
		caching: defaultCaching,
		timeout: defaultTimeout * time.Second,
	}

	for _, opt := range opts {
		opt.apply(&option)
	}

	return &Connection{
		addr:    addr,
		cache:   option.caching,
		timeout: option.timeout,
	}
}
