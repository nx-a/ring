package engine

type Closable interface {
	Close() error
}
