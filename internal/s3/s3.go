package s3

import (
    "context"
    "crypto"
    "errors"
    "fmt"
    "os"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/jelius-sama/logger"
)

type S3Client struct {
    Client *s3.Client
    Bucket string
}

func NewS3Client(bucket string) *S3Client {
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion(os.Getenv("AWS_REGION")),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(os.Getenv("ACCESS_KEY_ID"), os.Getenv("SECRET_ACCESS_KEY"), "")),
        config.WithUseDualStackEndpoint(aws.DualStackEndpointStateEnabled),
    )
    if err != nil {
        logger.Fatal("Failed to load default S3 config")
    }

    var client *s3.Client
    if baseURL := os.Getenv("BASE_URL"); len(baseURL) == 0 {
        client = s3.NewFromConfig(cfg)
    } else {
        client = s3.NewFromConfig(cfg, func(o *s3.Options) {
            // INFO: "https://" + os.Getenv("ACCOUNT_ID") + ".r2.cloudflarestorage.com"
            o.BaseEndpoint = aws.String(baseURL)
            o.UsePathStyle = true // This replaces S3ForcePathStyle
        })
    }

    return &S3Client{
        Client: client,
        Bucket: bucket,
    }
}

func loadPEMPrivKeyFile(name string) (key crypto.Signer, err error) {
    file, err := os.Open(name)
    if err != nil {
        return nil, err
    }

    defer func() {
        closeErr := file.Close()
        if err == nil {
            err = closeErr
        } else if closeErr != nil {
            err = fmt.Errorf("close error: %v, original error: %w", closeErr, err)
        }
    }()

    return sign.LoadPEMPrivKeyPKCS8AsSigner(file)
}

func (s3Client *S3Client) CreateSignedURL(ctx context.Context, objectKey string, fallbackContentType *string) (string, error) {
    presignClient := s3.NewPresignClient(s3Client.Client)

    headOutput, err := s3Client.Client.HeadObject(ctx, &s3.HeadObjectInput{
        Bucket: aws.String(s3Client.Bucket),
        Key:    aws.String(objectKey),
    })

    if err != nil {
        return "", err
    }

    contentType := headOutput.ContentType
    if contentType == nil && fallbackContentType == nil {
        return "", errors.New("couldn't get content type for the object")
    }
    if contentType == nil {
        contentType = fallbackContentType
    }

    disposition, ok := ctx.Value("disposition").(string)
    if !ok {
        disposition = "inline"
    }

    // Cloudfront mode
    if endpoint := os.Getenv("CLOUDFRONT_ENDPOINT"); len(endpoint) != 0 {
        cloudfrontResourceURL := "https://" + endpoint + "/" + objectKey + "?response-content-disposition=" + disposition + "&response-content-type=" + *contentType

        // Signed mode
        if keyPair, privKeyPath := os.Getenv("CLOUDFRONT_KEY_PAIR_ID"), os.Getenv("CLOUDFRONT_PRIVATE_KEY_PATH"); len(keyPair) != 0 && len(privKeyPath) != 0 {
            privateKey, err := loadPEMPrivKeyFile(privKeyPath)
            if err != nil {
                return "", errors.New("Failed to load private key: " + err.Error())
            }

            signer := sign.NewURLSigner(keyPair, privateKey)
            signedURL, err := signer.Sign(cloudfrontResourceURL, time.Now().UTC().Add(time.Hour))

            return signedURL, err
        }

        // Unsigned mode
        return cloudfrontResourceURL, nil
    }

    // Regular S3 mode
    presignedResult, err := presignClient.PresignGetObject(ctx,
        &s3.GetObjectInput{
            Bucket: aws.String(s3Client.Bucket),
            Key:    aws.String(objectKey),

            ResponseContentType:        aws.String(*contentType),
            ResponseContentDisposition: aws.String(disposition),
        },
        s3.WithPresignExpires(time.Hour),
    )

    if err != nil {
        return "", err
    }

    return presignedResult.URL, nil
}

