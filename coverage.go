package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/peterstace/simplefeatures/geom"
)

type CoverageFetcher struct {
	apikey string
}

func NewCoverageFetcher(apikey string) *CoverageFetcher {
	return &CoverageFetcher{apikey: apikey}
}

type Date string

func (f *CoverageFetcher) GetCoverage(ctx context.Context, env geom.Envelope) ([]Date, error) {
	min, max, ok := env.MinMaxXYs()
	if !ok {
		return nil, fmt.Errorf("unexpected empty envelope")
	}
	var poly strings.Builder
	for i, f := range []float64{
		min.X, min.Y,
		max.X, min.Y,
		max.X, max.Y,
		min.X, max.Y,
		min.X, min.Y,
	} {
		if i > 0 {
			poly.WriteByte(',')
		}
		poly.WriteString(strconv.FormatFloat(f, 'f', -1, 64))
	}

	url := fmt.Sprintf("https://api.nearmap.com/coverage/v2/poly/%s?resources=tiles:Vert&limit=9999", poly.String())
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, BadStatusError{body, resp.StatusCode}
	}

	var bodyJSON struct {
		Surveys []struct {
			CaptureDate Date `json:"captureDate"`
		} `json:"surveys"`
	}
	if err := json.Unmarshal(body, &bodyJSON); err != nil {
		return nil, err
	}

	dates := []Date{} // Because JSON.
	for _, survey := range bodyJSON.Surveys {
		dates = append(dates, survey.CaptureDate)
	}
	return dates, nil
}
