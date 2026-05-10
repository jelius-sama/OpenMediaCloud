package immich

import (
    "fmt"
    "github.com/jelius-sama/OpenMediaCloud/internal/db"
    "github.com/jelius-sama/OpenMediaCloud/internal/s3"
    "github.com/jelius-sama/logger"
    "net/http"
    "os"
)

type AssetType uint8

const (
    ATVideo AssetType = iota
    ATImage
)

func (at AssetType) String() string {
    switch at {
    case ATImage:
        return "IMAGE"
    case ATVideo:
        return "VIDEO"
    default:
        logger.Panic("unreachable")
        return ""
    }
}

func viewAsset(w http.ResponseWriter, r *http.Request, ids []string, s3Client *s3.S3Client) error {
    size := r.URL.Query().Get("size")

    if len(ids) > 1 || len(ids) == 0 {
        logger.Debug("Expected exactly 1 ID, got more than 1 or less than 0.")
        logger.Info("More than 1 ID parsed, using ID at index 0.")
    }
    logger.Debug("Requesting Asset with ID:", ids[0])

    assetPaths, err := db.ImmichGetAssetPaths(ids[0])

    if err != nil {
        // propagate error upward
        // WARN: this WILL end up using immich server. If the error was related to configuration then
        // immich will happily serve the asset instead of returning a valid error to the client
        return err
    }

    var presignedURL string
    var targetPath *string

    disposition, dOK := r.Context().Value("disposition").(string)
    assetType, atOK := r.Context().Value("type").(string)

    if dOK && disposition == "attachment" {
        targetPath = &assetPaths.OriginalPath
    } else if atOK && assetType == ATVideo.String() {
        if assetPaths.EncodedVideoPath != nil {
            targetPath = assetPaths.EncodedVideoPath
        } else {
            targetPath = &assetPaths.OriginalPath
        }
    } else {
        switch size {
        case asThumbnail.String():
            targetPath = assetPaths.Thumbnail
        case asPreview.String():
            targetPath = assetPaths.Preview
        case asFullsize.String():
            targetPath = assetPaths.Fullsize
            if targetPath == nil {
                targetPath = &assetPaths.OriginalPath
            }
        case asOriginal.String():
            targetPath = &assetPaths.OriginalPath
        default:
            targetPath = &assetPaths.OriginalPath
        }
    }

    if targetPath != nil {
        presignedURL, err = s3Client.CreateSignedURL(s3.CreateSignedURLT{
            Ctx:                 r.Context(),
            ObjectKey:           immichPathToS3Key(*targetPath),
            FallbackContentType: nil,
            CFEndpoint:          os.Getenv("IMMICH_CLOUDFRONT_ENDPOINT"),
            CFKeyPairID:         os.Getenv("IMMICH_CLOUDFRONT_KEY_PAIR_ID"),
            CFPrivateKeyPath:    os.Getenv("IMMICH_CLOUDFRONT_PRIVATE_KEY_PATH"),
        })
    } else {
        return fmt.Errorf("failed to query asset path from immich db")
    }

    if err != nil {
        return fmt.Errorf("Failed to create presigned URL: %w", err)
    }
    logger.Debug("S3 URL:", presignedURL)

    // Redirect the client directly to S3.
    // From this point the client fetches the video bytes straight from S3,
    // our EC2 server is no longer in the data path.
    http.Redirect(w, r, presignedURL, http.StatusTemporaryRedirect)
    logger.Okay("Redirected client to S3 for object:" + "\n\t`" + immichPathToS3Key(*targetPath))

    return nil
}

