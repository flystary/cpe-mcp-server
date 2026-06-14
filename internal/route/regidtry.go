package route

import "sync"

var (
	mu        sync.RWMutex
	factories = map[string]Factory{}
)

func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()

	factories[name] = f
}

func Factories() map[string]Factory {
	mu.RLock()
	defer mu.RUnlock()

	cp := make(map[string]Factory)
	for k, v := range factories {
		cp[k] = v
	}
	return cp
}
