package immich

import (
    "fmt"
    "net/http"
    "os"
    "path"
    "strings"

    "github.com/jelius-sama/OpenMediaCloud/internal/db"
    "github.com/jelius-sama/OpenMediaCloud/internal/s3"
    "github.com/jelius-sama/logger"
)

// TODO: Implement authentication
func viewAsset(w http.ResponseWriter, r *http.Request, ids []string, s3Client *s3.S3Client) error {
    assetKey := r.URL.Query().Get("key")
    assetSlug := r.URL.Query().Get("slug")
    size := r.URL.Query().Get("size")

    if len(ids) > 1 || len(ids) == 0 {
        logger.Debug("Expected exactly 1 ID, got more than 1 or less than 0.")
        logger.Info("More than 1 ID parsed, using ID at index 0.")
    }
    logger.Debug("Requesting Asset:\n\tAsset Key:", assetKey, "\n\tAsset Slug:", assetSlug)

    assetPaths, err := db.ImmichGetAssetPaths(ids[0])

    if err != nil {
        // propagate error upward
        // WARN: this WILL end up using immich server. If the error was related to configuration then
        // immich will happily serve the asset instead of returning a valid error to the client
        return err
    }

    var presignedURL string

    tempFuncToS3 := func(originalPath string) string {
        parts := strings.Split(path.Clean(originalPath), "/")

        // parts[0] seems to be empty and data is expected to be found at index 1
        if parts[1] != "data" {
            logger.Panic("Assertion failed this application requires the asset path to begin with `/data/`")
        }
        // hardcoded for now
        parts[1] = "immich-library"
        joined := strings.Join(parts, "/")
        return strings.TrimPrefix(joined, "/")
    }

    switch size {
    case asThumbnail.String():
        if assetPaths.Thumbnail != nil {
            presignedURL, err = s3Client.CreateSignedURL(s3.CreateSignedURLT{
                Ctx:                 r.Context(),
                ObjectKey:           tempFuncToS3(*assetPaths.Thumbnail),
                FallbackContentType: nil,
                CFEndpoint:          os.Getenv("IMMICH_CLOUDFRONT_ENDPOINT"),
                CFKeyPairID:         os.Getenv("IMMICH_CLOUDFRONT_KEY_PAIR_ID"),
                CFPrivateKeyPath:    os.Getenv("IMMICH_CLOUDFRONT_PRIVATE_KEY_PATH"),
            })
        }
    case asPreview.String():
        if assetPaths.Preview != nil {
            presignedURL, err = s3Client.CreateSignedURL(s3.CreateSignedURLT{
                Ctx:                 r.Context(),
                ObjectKey:           tempFuncToS3(*assetPaths.Preview),
                FallbackContentType: nil,
                CFEndpoint:          os.Getenv("IMMICH_CLOUDFRONT_ENDPOINT"),
                CFKeyPairID:         os.Getenv("IMMICH_CLOUDFRONT_KEY_PAIR_ID"),
                CFPrivateKeyPath:    os.Getenv("IMMICH_CLOUDFRONT_PRIVATE_KEY_PATH"),
            })
        }
    case asFullsize.String():
        if assetPaths.Fullsize != nil {
            presignedURL, err = s3Client.CreateSignedURL(s3.CreateSignedURLT{
                Ctx:                 r.Context(),
                ObjectKey:           tempFuncToS3(*assetPaths.Fullsize),
                FallbackContentType: nil,
                CFEndpoint:          os.Getenv("IMMICH_CLOUDFRONT_ENDPOINT"),
                CFKeyPairID:         os.Getenv("IMMICH_CLOUDFRONT_KEY_PAIR_ID"),
                CFPrivateKeyPath:    os.Getenv("IMMICH_CLOUDFRONT_PRIVATE_KEY_PATH"),
            })
        } else {
            // fallback to original if fullsize not generated
            presignedURL, err = s3Client.CreateSignedURL(s3.CreateSignedURLT{
                Ctx:                 r.Context(),
                ObjectKey:           tempFuncToS3(assetPaths.OriginalPath),
                FallbackContentType: nil,
                CFEndpoint:          os.Getenv("IMMICH_CLOUDFRONT_ENDPOINT"),
                CFKeyPairID:         os.Getenv("IMMICH_CLOUDFRONT_KEY_PAIR_ID"),
                CFPrivateKeyPath:    os.Getenv("IMMICH_CLOUDFRONT_PRIVATE_KEY_PATH"),
            })
        }
    case asOriginal.String():
        presignedURL, err = s3Client.CreateSignedURL(s3.CreateSignedURLT{
            Ctx:                 r.Context(),
            ObjectKey:           tempFuncToS3(assetPaths.OriginalPath),
            FallbackContentType: nil,
            CFEndpoint:          os.Getenv("IMMICH_CLOUDFRONT_ENDPOINT"),
            CFKeyPairID:         os.Getenv("IMMICH_CLOUDFRONT_KEY_PAIR_ID"),
            CFPrivateKeyPath:    os.Getenv("IMMICH_CLOUDFRONT_PRIVATE_KEY_PATH"),
        })
    }

    if err != nil {
        return fmt.Errorf("Failed to create presigned URL: %s", err)
    }
    logger.Debug("S3 URL:", presignedURL)

    // Redirect the client directly to S3.
    // From this point the client fetches the video bytes straight from S3,
    // our EC2 server is no longer in the data path.
    http.Redirect(w, r, presignedURL, http.StatusTemporaryRedirect)
    logger.Okay("Redirected client to S3 for object:"+"\n\t`"+tempFuncToS3(assetPaths.OriginalPath)+"`\n"+"with size", "`"+size+"`")

    return nil
}

