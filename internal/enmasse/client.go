package enmasse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sakid00/massmaker-be/internal/auth"
	"github.com/sakid00/massmaker-be/internal/domain"
)

type Client struct {
	base     string
	secret   string
	http     *http.Client
	slowHTTP *http.Client
}

type ProfileStatus struct {
	Complete bool     `json:"complete"`
	Missing  []string `json:"missing"`
}

type MeProfile struct {
	Role            string          `json:"role"`
	ProfileComplete bool            `json:"profileComplete"`
	Missing         []string        `json:"missing"`
	Vendor          *MeVendor       `json:"vendor,omitempty"`
	Artist          *MeArtist       `json:"artist,omitempty"`
	Raw             json.RawMessage `json:"-"`
}

type MeVendor struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	PhotoURL     string `json:"photoUrl"`
	City         string `json:"city"`
	Province     string `json:"province"`
	ProvinceCode string `json:"provinceCode"`
	CityCode     string `json:"cityCode"`
	District     string `json:"district"`
	DistrictCode string `json:"districtCode"`
	PICName      string `json:"picName"`
	WhatsApp     string `json:"whatsapp"`
	Address      string `json:"address"`
	Bio          string `json:"bio"`
}

type MeArtist struct {
	ID              string `json:"id"`
	ArtistName      string `json:"artistName"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	PhotoURL        string `json:"photoUrl"`
	Address         string `json:"address"`
	Province        string `json:"province"`
	ProvinceCode    string `json:"provinceCode"`
	City            string `json:"city"`
	CityCode        string `json:"cityCode"`
	District        string `json:"district"`
	DistrictCode    string `json:"districtCode"`
	WebsiteURL      string `json:"websiteUrl"`
	InstagramHandle string `json:"instagramHandle"`
	TwitterHandle   string `json:"twitterHandle"`
	WhatsApp        string `json:"whatsapp"`
	Bio             string `json:"bio"`
}

type InternalVendor struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	PhotoURL     string `json:"photoUrl"`
	City         string `json:"city"`
	Province     string `json:"province"`
	ProvinceCode string `json:"provinceCode"`
	CityCode     string `json:"cityCode"`
	District     string `json:"district"`
	DistrictCode string `json:"districtCode"`
	PICName      string `json:"picName"`
	WhatsApp     string `json:"whatsapp"`
	Address      string `json:"address"`
	Bio          string `json:"bio"`
}

func New(baseURL, serviceSecret string) *Client {
	return &Client{
		base:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		secret:   serviceSecret,
		http:     &http.Client{Timeout: 2 * time.Second, CheckRedirect: noRedirect},
		slowHTTP: &http.Client{Timeout: 15 * time.Second, CheckRedirect: noRedirect},
	}
}

type ForwardResult struct {
	Status      int
	Body        []byte
	ContentType string
}

func (c *Client) Forward(ctx context.Context, method, path string, body []byte, authorization string) (*ForwardResult, error) {
	if c.base == "" {
		return nil, domain.ErrUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("enmasse: forward: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	res, err := c.slowHTTP.Do(req)
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	ct := res.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}
	return &ForwardResult{Status: res.StatusCode, Body: raw, ContentType: ct}, nil
}

func noRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func (c *Client) ProfileStatus(ctx context.Context, userID string) (ProfileStatus, error) {
	if c.base == "" {
		return ProfileStatus{}, domain.ErrUnavailable
	}
	tok, err := auth.MintService(c.secret, 5*time.Minute)
	if err != nil {
		return ProfileStatus{}, fmt.Errorf("enmasse: mint service jwt: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/v1/internal/profile-status/"+userID, nil)
	if err != nil {
		return ProfileStatus{}, fmt.Errorf("enmasse: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return ProfileStatus{}, domain.ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return ProfileStatus{}, domain.ErrUnavailable
	}
	var body ProfileStatus
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return ProfileStatus{}, domain.ErrUnavailable
	}
	if body.Missing == nil {
		body.Missing = []string{}
	}
	return body, nil
}

func (c *Client) MeProfile(ctx context.Context, authorization string) (*MeProfile, error) {
	res, err := c.Forward(ctx, http.MethodGet, "/v1/me/profile", nil, authorization)
	if err != nil {
		return nil, err
	}
	if res.Status == http.StatusUnauthorized {
		return nil, domain.ErrUnauthorized
	}
	if res.Status != http.StatusOK {
		return nil, domain.ErrUnavailable
	}
	var body MeProfile
	if err := json.Unmarshal(res.Body, &body); err != nil {
		return nil, domain.ErrUnavailable
	}
	if body.Missing == nil {
		body.Missing = []string{}
	}
	body.Raw = append(json.RawMessage(nil), res.Body...)
	return &body, nil
}

func (c *Client) InternalVendor(ctx context.Context, vendorID string) (*InternalVendor, error) {
	if c.base == "" {
		return nil, domain.ErrUnavailable
	}
	tok, err := auth.MintService(c.secret, 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("enmasse: mint service jwt: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/v1/internal/vendors/"+vendorID, nil)
	if err != nil {
		return nil, fmt.Errorf("enmasse: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, domain.ErrUnavailable
	}
	var body InternalVendor
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, domain.ErrUnavailable
	}
	return &body, nil
}
