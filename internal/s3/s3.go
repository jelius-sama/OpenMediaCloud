package s3

import (
    "context"
    "errors"
    "os"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/jelius-sama/logger"
)

type S3Client struct {
    Client *s3.Client
    Bucket string
}

func NewS3Client(bucket string) *S3Client {
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion("ap-south-1"), // TODO: Later change to "auto" when deploying for R2
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(os.Getenv("ACCESS_KEY_ID"), os.Getenv("SECRET_ACCESS_KEY"), "")),
        config.WithUseDualStackEndpoint(aws.DualStackEndpointStateEnabled),
    )
    if err != nil {
        logger.Panic("Failed to load default S3 config")
    }

    // client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    //     o.BaseEndpoint = aws.String("https://" + os.Getenv("ACCOUNT_ID") + ".r2.cloudflarestorage.com")
    //     o.UsePathStyle = true // This replaces S3ForcePathStyle
    // })
    client := s3.NewFromConfig(cfg)

    return &S3Client{
        Client: client,
        Bucket: bucket,
    }
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

    presignedResult, err := presignClient.PresignGetObject(ctx,
        &s3.GetObjectInput{
            Bucket: aws.String(s3Client.Bucket),
            Key:    aws.String(objectKey),

            // This tells R2 to send back this header, forcing the browser to view it
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

