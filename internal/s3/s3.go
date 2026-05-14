package s3

/*
#include "../../libs/logger/logger.h"
*/
import "C"
import (
    "context"
    "crypto"
    "errors"
    "fmt"
    "net/url"
    "os"
    "strings"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
    Client *s3.Client
    Bucket string
}

type NewS3ClientT struct {
    Bucket          string
    Region          string
    AccessId        string
    SecretAccessKey string
    BaseURL         string
}

func NewS3Client(nc NewS3ClientT) *S3Client {
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion(nc.Region),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(nc.AccessId, nc.SecretAccessKey, "")),
        config.WithUseDualStackEndpoint(aws.DualStackEndpointStateEnabled),
    )
    if err != nil {
        C.Fatal("Failed to load default S3 config")
    }

    var client *s3.Client
    if baseURL := nc.BaseURL; len(baseURL) == 0 {
        client = s3.NewFromConfig(cfg)
    } else {
        client = s3.NewFromConfig(cfg, func(o *s3.Options) {
            o.BaseEndpoint = aws.String(baseURL)
            o.UsePathStyle = true // This replaces S3ForcePathStyle
        })
    }

    return &S3Client{
        Client: client,
        Bucket: nc.Bucket,
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

type CreateSignedURLT struct {
    Ctx                 context.Context
    ObjectKey           string
    FallbackContentType *string
    CFEndpoint          string
    CFKeyPairID         string
    CFPrivateKeyPath    string
}

func (s3Client *S3Client) CreateSignedURL(csu CreateSignedURLT) (string, error) {
    presignClient := s3.NewPresignClient(s3Client.Client)

    headOutput, err := s3Client.Client.HeadObject(csu.Ctx, &s3.HeadObjectInput{
        Bucket: aws.String(s3Client.Bucket),
        Key:    aws.String(csu.ObjectKey),
    })

    if err != nil {
        return "", err
    }

    contentType := headOutput.ContentType
    if contentType == nil && csu.FallbackContentType == nil {
        return "", errors.New("couldn't get content type for the object")
    }
    if contentType == nil {
        contentType = csu.FallbackContentType
    }

    disposition, ok := csu.Ctx.Value("disposition").(string)
    if !ok {
        disposition = "inline"
    }

    // Cloudfront mode
    if endpoint := csu.CFEndpoint; len(endpoint) != 0 {
        encodedKey := url.PathEscape(csu.ObjectKey)
        encodedContentType := url.QueryEscape(*contentType)
        encodedDisposition := url.QueryEscape(disposition)

        cloudfrontResourceURL := "https://" + endpoint + "/" + encodedKey +
            "?response-content-disposition=" + encodedDisposition +
            "&response-content-type=" + encodedContentType

        // Signed mode
        if keyPair, privKeyPath := csu.CFKeyPairID, csu.CFPrivateKeyPath; len(keyPair) != 0 && len(privKeyPath) != 0 {
            var privateKey crypto.Signer
            // NOTE: Try PKCS#1 first
            privateKey, err := sign.LoadPEMPrivKeyFile(privKeyPath)

            // NOTE: On failure try PKCS#8
            if err != nil && strings.Contains(err.Error(), "x509: failed to parse private key (use ParsePKCS8PrivateKey instead for this key format)") {
                privateKey, err = loadPEMPrivKeyFile(privKeyPath)
            }

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
    presignedResult, err := presignClient.PresignGetObject(csu.Ctx,
        &s3.GetObjectInput{
            Bucket: aws.String(s3Client.Bucket),
            Key:    aws.String(csu.ObjectKey),

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

