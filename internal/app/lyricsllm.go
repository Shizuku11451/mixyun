package app

import (
	"errors"
	"log"
	"sync"

	"github.com/egoist/mygo"
)

// llmSecret is the key of the API key of the language model that
// translates lyrics, in the credential store.
const llmSecret = "lyrics-llm-key"

// llmKeyStore keeps the API key of the language model that translates
// lyrics, read from the credential store once a run.
type llmKeyStore struct {
	key jsonSecret[string]

	mu     sync.Mutex
	loaded bool
	value  string
}

// load returns the API key, "" for none: a server on this computer, as
// Ollama's, takes none.
func (k *llmKeyStore) load() string {
	k.mu.Lock()
	defer k.mu.Unlock()
	if !k.loaded {
		v, err := k.key.Load()
		if err != nil && !errors.Is(err, mygo.ErrSecretNotFound) {
			log.Printf("mixmusic: the API key of the language model of lyrics: %v", err)
		}
		k.value, k.loaded = v, err == nil || errors.Is(err, mygo.ErrSecretNotFound)
	}
	return k.value
}

// HasLyricsLLMKey tells whether the user gave the API key of the language
// model that translates lyrics.
func (s *System) HasLyricsLLMKey() bool { return s.a.llm.load() != "" }

// SetLyricsLLMKey keeps the API key of the language model that translates
// lyrics, or forgets it, for "".
func (s *System) SetLyricsLLMKey(key string) error {
	k := &s.a.llm
	k.mu.Lock()
	defer k.mu.Unlock()
	var err error
	if key == "" {
		err = k.key.Clear()
	} else {
		err = k.key.Save(key)
	}
	if err != nil {
		return err
	}
	k.value, k.loaded = key, true
	return nil
}
