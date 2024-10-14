package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
)

type TileFetcher struct {
	apikey string
	cache  *SingleFlightForeverCache[[]byte]
}

func NewTileFetcher(apikey string) *TileFetcher {
	return &TileFetcher{
		apikey: apikey,
		cache:  NewSingleFlightForeverCache[[]byte](),
	}
}

func (f *TileFetcher) GetRaw(ctx context.Context, date Date, z, x, y int) ([]byte, error) {
	key := fmt.Sprintf("%s/%d/%d/%d", date, z, x, y)
	return f.cache.Get(key, func() ([]byte, error) {
		return f.fetchRaw(ctx, date, z, x, y)
	})
}

func (f *TileFetcher) fetchRaw(ctx context.Context, date Date, z, x, y int) ([]byte, error) {
	url := fmt.Sprintf("https://api.nearmap.com/tiles/v3/Vert/%d/%d/%d.img?until=%s", z, x, y, date)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Apikey "+f.apikey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return notFoundTile, nil
	case http.StatusOK:
		return io.ReadAll(resp.Body)
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unexpected %d response from Tiles API", resp.StatusCode)
		}
		return nil, BadStatusError{body, resp.StatusCode}
	}
}

func (f *TileFetcher) GetCooked(ctx context.Context, date Date, z, x, y int) (image.Image, error) {
	raw, err := f.GetRaw(ctx, date, z, x, y)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	return img, err
}

func makeNotFoundTile() []byte {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

var notFoundTile = makeNotFoundTile()
