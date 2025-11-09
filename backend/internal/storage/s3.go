package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type S3Config struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	ForcePathStyle  bool
	Root            string
}

type s3Storage struct {
	client *s3.Client
	bucket string
	prefix string
}

func NewS3Storage(ctx context.Context, cfg S3Config) (FileStorage, error) {
	creds := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")
	awsCfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(cfg.Region), awscfg.WithCredentialsProvider(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.ForcePathStyle
	})

	return &s3Storage{
		client: client,
		bucket: cfg.Bucket,
		prefix: strings.Trim(cfg.Root, "/"),
	}, nil
}

func (s *s3Storage) Type() string {
	return TypeS3
}

func (s *s3Storage) Save(ctx context.Context, path string, data io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.buildObjectKey(path)),
		Body:   data,
	})
	return err
}

func (s *s3Storage) Open(ctx context.Context, path string) (io.ReadCloser, int64, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.buildObjectKey(path)),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, 0, fs.ErrNotExist
		}
		return nil, 0, err
	}
	return resp.Body, aws.ToInt64(resp.ContentLength), nil
}

func (s *s3Storage) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.buildObjectKey(path)),
	})
	return err
}

func (s *s3Storage) DeleteAll(ctx context.Context, path string) error {

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.buildObjectKey(path)),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}
		if len(page.Contents) == 0 {
			continue
		}
		objects := make([]s3types.ObjectIdentifier, 0, len(page.Contents))
		for _, obj := range page.Contents {
			objects = append(objects, s3types.ObjectIdentifier{Key: obj.Key})
		}
		_, err = s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &s3types.Delete{Objects: objects, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *s3Storage) List(ctx context.Context, path string) ([]ObjectInfo, error) {
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.buildObjectKey(path)),
	})
	var objects []ObjectInfo
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			objects = append(objects, ObjectInfo{
				Path:    aws.ToString(obj.Key),
				Size:    aws.ToInt64(obj.Size),
				ModTime: aws.ToTime(obj.LastModified),
			})
		}
	}
	return objects, nil
}

func (s *s3Storage) Walk(ctx context.Context, root string, fn func(ObjectInfo) error) error {
	objects, err := s.List(ctx, root)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		if err := fn(obj); err != nil {
			return err
		}
	}

	return nil
}

func (s *s3Storage) buildObjectKey(p string) string {
	p = filepath.Clean(p)
	p = filepath.ToSlash(p)
	p = strings.Trim(p, "/")

	if p == "" || p == "." {
		return s.prefix
	}

	if s.prefix == "" {
		return p
	}

	return s.prefix + "/" + p
}

func isS3NotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey" {
			return true
		}
	}
	var missingKey *s3types.NoSuchKey
	return errors.As(err, &missingKey)
}
