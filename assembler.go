package main

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"sync"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/sync/errgroup"
)

const tileSize = 256

type CookedImageSource interface {
	GetCooked(_ context.Context, _ Date, zoom, x, y int) (image.Image, error)
}

type Assembler struct {
	cooked CookedImageSource
}

func NewAssembler(cooked CookedImageSource) *Assembler {
	return &Assembler{cooked: cooked}
}

func (a *Assembler) Assemble(ctx context.Context, date Date, zoom int, xy, resolution image.Point, downsample int) (image.Image, error) {
	tiles, err := a.getTiles(ctx, date, zoom, xy, resolution)
	if err != nil {
		return nil, err
	}
	img, err := stitchAndCrop(tiles, xy, resolution), nil
	if err != nil {
		return nil, err
	}
	return downsampleImage(img, downsample)
}

func (a *Assembler) getTiles(ctx context.Context, date Date, zoom int, topLeft, resolution image.Point) (map[image.Point]image.Image, error) {
	group, ctx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	tiles := make(map[image.Point]image.Image)

	for pxX := topLeft.X / tileSize * tileSize; pxX < topLeft.X+resolution.X; pxX += tileSize {
		for pxY := topLeft.Y / tileSize * tileSize; pxY < topLeft.Y+resolution.Y; pxY += tileSize {
			pxX, pxY := pxX, pxY
			group.Go(func() error {
				img, err := a.cooked.GetCooked(ctx, date, zoom, pxX/tileSize, pxY/tileSize)
				mu.Lock()
				tiles[image.Pt(pxX, pxY)] = img
				mu.Unlock()
				return err
			})
		}
	}
	return tiles, group.Wait()
}

func stitchAndCrop(tiles map[image.Point]image.Image, topLeft, resolution image.Point) image.Image {
	rec := image.Rectangle{Min: topLeft, Max: topLeft.Add(resolution)}
	dest := image.NewRGBA(rec)
	for pt, tile := range tiles {
		destRec := image.Rectangle{Min: pt, Max: pt.Add(image.Pt(tileSize, tileSize))}
		draw.Draw(dest, destRec, tile, image.Point{}, draw.Src)
	}
	return dest
}

func downsampleImage(src image.Image, downsample int) (image.Image, error) {
	if downsample < 1 {
		return nil, fmt.Errorf("downsample must be at least 1, got %d", downsample)
	}
	if downsample == 1 {
		return src, nil
	}

	srcBounds := src.Bounds()
	dstBounds := image.Rectangle{
		Min: srcBounds.Min.Div(downsample),
		Max: srcBounds.Max.Div(downsample),
	}
	dst := image.NewRGBA(dstBounds)

	xdraw.CatmullRom.Scale(dst, dstBounds, src, srcBounds, xdraw.Src, nil)
	return dst, nil
}
