package s3

import (
	"context"
	"fmt"
	urlpkg "net/url"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	// Amazon Accelerated Transfer endpoint
	transferAccelEndpoint = "s3-accelerate.amazonaws.com"

	// Google Cloud Storage endpoint
	gcsEndpoint = "storage.googleapis.com"
)

var sentinelURL = urlpkg.URL{}

type Options struct {
	MaxRetries     int
	NoSignRequest  bool
	Endpoint       string
	Profile        string
	CredentialFile string
	Bucket         string
	Region         string
}

func (o *Options) SetRegion(region string) {
	o.Region = region
}

func (o *Options) SetEndpoint(endpoint string) {
	o.Endpoint = endpoint
}

func (o *Options) SetBucket(bucket string) {
	o.Bucket = bucket
}
func (o *Options) SetNoSignRequest(noSignRequest bool) {
	o.NoSignRequest = noSignRequest
}

type S3Client struct {
	s3 *s3.S3
}

func NewS3Client(ctx context.Context, opts Options) (*S3Client, error) {
	awsCfg := aws.NewConfig()
	endpointURL, err := parseEndpoint(opts.Endpoint)
	if err != nil {
		return nil, err
	}

	// use virtual-host-style if the endpoint is known to support it,
	// otherwise use the path-style approach.
	isVirtualHostStyle := isVirtualHostStyle(endpointURL)

	useAccelerate := supportsTransferAcceleration(endpointURL)
	// AWS SDK handles transfer acceleration automatically. Setting the
	// Endpoint to a transfer acceleration endpoint would cause bucket
	// operations fail.
	if useAccelerate {
		endpointURL = sentinelURL
	}

	if opts.NoSignRequest {
		// do not sign requests when making service API calls
		awsCfg = awsCfg.WithCredentials(credentials.AnonymousCredentials)
	} else {
		awsCfg = awsCfg.WithCredentials(
			credentials.NewSharedCredentials(opts.CredentialFile, opts.Profile),
		)
	}
	awsCfg = awsCfg.
		WithEndpoint(endpointURL.String()).
		WithS3ForcePathStyle(!isVirtualHostStyle).
		WithS3UseAccelerate(useAccelerate).
		// TODO WithLowerCaseHeaderMaps and WithDisableRestProtocolURICleaning options
		// are going to be unnecessary and unsupported in AWS-SDK version 2.
		// They should be removed during migration.
		WithLowerCaseHeaderMaps(true).
		// Disable URI cleaning to allow adjacent slashes to be used in S3 object keys.
		WithDisableRestProtocolURICleaning(true)

	awsCfg.Retryer = newCustomRetryer(opts.MaxRetries)

	useSharedConfig := session.SharedConfigEnable
	{
		// Reverse of what the SDK does: if AWS_SDK_LOAD_CONFIG is 0 (or a
		// falsy value) disable shared configs
		loadCfg := os.Getenv("AWS_SDK_LOAD_CONFIG")
		if loadCfg != "" {
			if enable, _ := strconv.ParseBool(loadCfg); !enable {
				useSharedConfig = session.SharedConfigDisable
			}
		}
	}

	sess, err := session.NewSessionWithOptions(
		session.Options{
			Config:            *awsCfg,
			SharedConfigState: useSharedConfig,
		},
	)
	if err != nil {
		return nil, err
	}

	// get region of the bucket and create session accordingly. if the region
	// is not provided, it means we want region-independent session
	// for operations such as listing buckets, making a new bucket etc.
	// only get bucket region when it is not specified.
	if opts.Region != "" {
		sess.Config.Region = aws.String(opts.Region)
	} else {
		if err := setSessionRegion(ctx, sess, opts.Bucket); err != nil {
			return nil, err
		}
	}

	return &S3Client{
		s3: s3.New(sess),
	}, nil
}

func (c *S3Client) GeneratePresignedURL(bucket, key string, expiration time.Duration) (string, error) {
	req, _ := c.s3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	url, err := req.Presign(expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url, nil
}

func (c *S3Client) GetObjectSize(bucket, key string) (int64, error) {
	resp, err := c.s3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get object size: %w", err)
	}
	return *resp.ContentLength, nil
}

func parseEndpoint(endpoint string) (urlpkg.URL, error) {
	if endpoint == "" {
		return sentinelURL, nil
	}

	u, err := urlpkg.Parse(endpoint)
	if err != nil {
		return sentinelURL, fmt.Errorf("parse endpoint %q: %v", endpoint, err)
	}

	return *u, nil
}

// isVirtualHostStyle reports whether the given endpoint supports S3 virtual
// host style bucket name resolving. If a custom S3 API compatible endpoint is
// given, resolve the bucketname from the URL path.
func isVirtualHostStyle(endpoint urlpkg.URL) bool {
	return endpoint == sentinelURL || supportsTransferAcceleration(endpoint) || IsGoogleEndpoint(endpoint)
}

func supportsTransferAcceleration(endpoint urlpkg.URL) bool {
	return endpoint.Hostname() == transferAccelEndpoint
}

func IsGoogleEndpoint(endpoint urlpkg.URL) bool {
	return endpoint.Hostname() == gcsEndpoint
}

type customRetryer struct {
	client.DefaultRetryer
}

func newCustomRetryer(maxRetries int) *customRetryer {
	return &customRetryer{
		DefaultRetryer: client.DefaultRetryer{
			NumMaxRetries: maxRetries,
		},
	}
}

func setSessionRegion(ctx context.Context, sess *session.Session, bucket string) error {
	region := aws.StringValue(sess.Config.Region)

	if region != "" {
		return nil
	}

	// set default region
	sess.Config.Region = aws.String(endpoints.UsWest1RegionID)

	if bucket == "" {
		return nil
	}

	// auto-detection
	region, err := s3manager.GetBucketRegion(ctx, sess, bucket, "", func(r *request.Request) {
		// s3manager.GetBucketRegion uses Path style addressing and
		// AnonymousCredentials by default, updating Request's Config to match
		// the session config.
		r.Config.S3ForcePathStyle = sess.Config.S3ForcePathStyle
		r.Config.Credentials = sess.Config.Credentials
	})
	if err != nil {
		// if errHasCode(err, "NotFound") {
		// 	return err
		// }
		// don't deny any request to the service if region auto-fetching
		// receives an error. Delegate error handling to command execution.
		err = fmt.Errorf("session: fetching region failed: %v", err)
		return err
	} else {
		sess.Config.Region = aws.String(region)
	}

	return nil
}
