package media

import (
	"context"
	"fmt"
	neturl "net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"github.com/sakid00/massmaker-be/internal/domain"
)

const (
	presignTTL           = 2 * time.Minute
	MaxObjectBytes       = 2 * 1024 * 1024
	MaxInquiryImageBytes = 5 * 1024 * 1024
	MaxInquiryPDFBytes   = 10 * 1024 * 1024
)

var allowedTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

var inquiryTypes = map[string]string{
	"image/jpeg":      "jpg",
	"image/png":       "png",
	"image/webp":      "webp",
	"application/pdf": "pdf",
}

type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicBaseURL   string
}

func (c Config) Complete() bool {
	return strings.TrimSpace(c.AccountID) != "" &&
		strings.TrimSpace(c.AccessKeyID) != "" &&
		strings.TrimSpace(c.SecretAccessKey) != "" &&
		strings.TrimSpace(c.Bucket) != "" &&
		strings.TrimSpace(c.PublicBaseURL) != ""
}

type Result struct {
	PutURL    string            `json:"putUrl"`
	PublicURL string            `json:"publicUrl"`
	Headers   map[string]string `json:"headers"`
}

type Presigner struct {
	cfg     Config
	presign *s3.PresignClient
}

func New(ctx context.Context, cfg Config) (*Presigner, error) {
	p := &Presigner{cfg: cfg}
	if !cfg.Complete() {
		return p, nil
	}
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			strings.TrimSpace(cfg.AccessKeyID),
			strings.TrimSpace(cfg.SecretAccessKey),
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("r2 config: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", strings.TrimSpace(cfg.AccountID)))
		o.UsePathStyle = true
	})
	p.presign = s3.NewPresignClient(client)
	return p, nil
}

func (p *Presigner) Configured() bool {
	return p != nil && p.cfg.Complete() && p.presign != nil
}

func ValidateContentLength(n int64) error {
	if n <= 0 {
		return domain.NewAppError(400, "validation", "invalid content length")
	}
	if n > MaxObjectBytes {
		return domain.NewAppError(400, "validation", "file too large")
	}
	return nil
}

func ExtForContentType(contentType string) (string, error) {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	ext, ok := allowedTypes[ct]
	if !ok {
		return "", domain.NewAppError(400, "validation", "unsupported content type")
	}
	return ext, nil
}

func ObjectKey(userID, ext string) string {
	return fmt.Sprintf("profiles/%s/%s.%s", strings.TrimSpace(userID), uuid.NewString(), ext)
}

func InquiryExtForContentType(contentType string) (string, error) {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	ext, ok := inquiryTypes[ct]
	if !ok {
		return "", domain.NewAppError(400, "validation", "unsupported content type")
	}
	return ext, nil
}

func InquiryMaxBytes(contentType string) int64 {
	if strings.ToLower(strings.TrimSpace(contentType)) == "application/pdf" {
		return MaxInquiryPDFBytes
	}
	return MaxInquiryImageBytes
}

func ValidateInquiryContentLength(contentType string, n int64) error {
	if n <= 0 {
		return domain.NewAppError(400, "validation", "invalid content length")
	}
	if n > InquiryMaxBytes(contentType) {
		return domain.NewAppError(400, "validation", "file too large")
	}
	return nil
}

func InquiryObjectKey(userID, ext string) string {
	return fmt.Sprintf("inquiries/%s/%s.%s", strings.TrimSpace(userID), uuid.NewString(), ext)
}

func InquiryKeyPrefix(userID string) string {
	return "inquiries/" + strings.TrimSpace(userID) + "/"
}

func InquiryURLOwned(publicURL, publicBase, userID string) bool {
	url := strings.TrimSpace(publicURL)
	userID = strings.TrimSpace(userID)
	if url == "" || userID == "" {
		return false
	}
	prefix := InquiryKeyPrefix(userID)
	parsed, err := neturl.Parse(url)
	if err == nil && parsed.Path != "" {
		path := strings.TrimPrefix(parsed.Path, "/")
		return strings.HasPrefix(path, prefix)
	}
	base := strings.TrimRight(strings.TrimSpace(publicBase), "/") + "/"
	if publicBase == "" || !strings.HasPrefix(url, base) {
		return false
	}
	return strings.Contains(url, "/"+prefix)
}

func PublicURL(base, key string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/") + "/" + key
}

func ForPublicBase(publicBase string) *Presigner {
	return &Presigner{cfg: Config{PublicBaseURL: strings.TrimSpace(publicBase)}}
}

func (p *Presigner) Presign(ctx context.Context, userID, contentType string, contentLength int64) (*Result, error) {
	if !p.Configured() {
		return nil, domain.ErrMediaUnavailable
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, domain.ErrUnauthorized
	}
	ext, err := ExtForContentType(contentType)
	if err != nil {
		return nil, err
	}
	if err := ValidateContentLength(contentLength); err != nil {
		return nil, err
	}
	ct := strings.ToLower(strings.TrimSpace(contentType))
	key := ObjectKey(userID, ext)
	req, err := p.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(p.cfg.Bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(ct),
		ContentLength: aws.Int64(contentLength),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignTTL
	})
	if err != nil {
		return nil, fmt.Errorf("presign put: %w", err)
	}
	return &Result{
		PutURL:    req.URL,
		PublicURL: PublicURL(p.cfg.PublicBaseURL, key),
		Headers:   map[string]string{"Content-Type": ct},
	}, nil
}

func (p *Presigner) PresignInquiry(ctx context.Context, userID, contentType string, contentLength int64) (*Result, error) {
	if !p.Configured() {
		return nil, domain.ErrMediaUnavailable
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, domain.ErrUnauthorized
	}
	ext, err := InquiryExtForContentType(contentType)
	if err != nil {
		return nil, err
	}
	if err := ValidateInquiryContentLength(contentType, contentLength); err != nil {
		return nil, err
	}
	ct := strings.ToLower(strings.TrimSpace(contentType))
	key := InquiryObjectKey(userID, ext)
	req, err := p.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(p.cfg.Bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(ct),
		ContentLength: aws.Int64(contentLength),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignTTL
	})
	if err != nil {
		return nil, fmt.Errorf("presign inquiry put: %w", err)
	}
	return &Result{
		PutURL:    req.URL,
		PublicURL: PublicURL(p.cfg.PublicBaseURL, key),
		Headers:   map[string]string{"Content-Type": ct},
	}, nil
}

func (p *Presigner) PublicBaseURL() string {
	if p == nil {
		return ""
	}
	return p.cfg.PublicBaseURL
}
