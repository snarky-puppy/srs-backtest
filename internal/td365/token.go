package td365

import "sync"

type token struct {
	mux sync.Mutex
	s   string
}

// GetToken atomically gets the token
func (t *token) GetToken() string {
	t.mux.Lock()
	defer t.mux.Unlock()
	return t.s
}

// SetToken atomically sets the token
func (t *token) SetToken(s string) {
	t.mux.Lock()
	defer t.mux.Unlock()
	t.s = s
}
