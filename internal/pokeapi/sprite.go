package pokeapi

import (
	"context"
	"fmt"
	"image"
	_ "image/png" // registers the PNG decoder used by image.Decode
	"net/http"
)

// FetchSprite downloads and decodes the image at url (as returned in a
// Pokemon's SpriteURL). PokeAPI serves sprite images from a different host
// than the API itself, so this issues its own request rather than going
// through Client.get, but failures are still classified as ServiceError for
// consistency — a missing or broken sprite is never the user's mistake. The
// decoded image is cached by url for the rest of the Client's lifetime (see
// cache's doc comment), so revisiting the same Pokémon never re-downloads or
// re-decodes its sprite.
func (c *Client) FetchSprite(ctx context.Context, url string) (image.Image, error) {
	c.cache.mu.Lock()
	img, ok := c.cache.sprites[url]
	c.cache.mu.Unlock()
	if ok {
		return img, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &ServiceError{Err: err}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &ServiceError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &ServiceError{Err: fmt.Errorf("unexpected status %d fetching sprite", resp.StatusCode)}
	}

	img, _, err = image.Decode(resp.Body)
	if err != nil {
		return nil, &ServiceError{Err: fmt.Errorf("decoding sprite image: %w", err)}
	}

	c.cache.mu.Lock()
	c.cache.sprites[url] = img
	c.cache.mu.Unlock()
	return img, nil
}
