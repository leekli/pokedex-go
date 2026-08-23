package tui

import (
	"context"
	"image"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leekli/pokedex-go/internal/pokeapi"
	"github.com/leekli/pokedex-go/internal/pokemon"
)

// lookupTimeout bounds a single Search Screen submission: the PokeAPI
// lookup plus, on success, the sprite image fetch.
const lookupTimeout = 10 * time.Second

// lookupCmd resolves q against PokeAPI and reports the outcome as a
// lookupResultMsg. A failed sprite fetch is not treated as an overall
// failure — the Result Screen falls back to "No sprite available" instead
// of losing an otherwise-successful stat lookup over a broken image.
func lookupCmd(client *pokeapi.Client, q pokemon.Query) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
		defer cancel()

		p, err := client.Lookup(ctx, q)
		if err != nil {
			return lookupResultMsg{err: err}
		}

		stat := pokeapi.BuildStatBlock(p)
		var sprite image.Image
		if p.SpriteURL != "" {
			if img, spriteErr := client.FetchSprite(ctx, p.SpriteURL); spriteErr == nil {
				sprite = img
			}
		}
		return lookupResultMsg{stat: stat, sprite: sprite}
	}
}
