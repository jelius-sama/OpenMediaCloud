package jellyfin

/*
#include "../../libs/logger/logger.h"
*/
import "C"
import (
    "errors"
    "net/http"
    "os"
    "strings"

    "github.com/jelius-sama/OpenMediaCloud/internal/s3"
    "github.com/jelius-sama/OpenMediaCloud/internal/util"
)

func ApplyPatch(w http.ResponseWriter, r *http.Request, s3Client *s3.S3Client) error {
    C.Debug("Applying patch, original path: " + r.URL.Path)

    itemId, err := util.ExtractItemId("/Videos/{itemId}/stream", r.URL.Path)
    if err != nil {
        return errors.New("Failed to extract itemId: " + err.Error())
    }
    C.Debug("Extracted itemId: " + itemId)

    filePath, err := getItemPath(itemId)
    if err != nil {
        return errors.New("Failed to get item path from Jellyfin: " + err.Error())
    }
    filePath = strings.TrimPrefix(filePath, "/")
    filePath = strings.TrimSuffix(filePath, "/")
    C.Debug("Jellyfin returned file path: " + filePath)

    presignedURL, err := s3Client.CreateSignedURL(s3.CreateSignedURLT{
        Ctx:                 r.Context(),
        ObjectKey:           filePath,
        FallbackContentType: nil,
        CFEndpoint:          os.Getenv("JELLYFIN_CLOUDFRONT_ENDPOINT"),
        CFKeyPairID:         os.Getenv("JELLYFIN_CLOUDFRONT_KEY_PAIR_ID"),
        CFPrivateKeyPath:    os.Getenv("JELLYFIN_CLOUDFRONT_PRIVATE_KEY_PATH"),
    })
    if err != nil {
        return errors.New("Failed to create presigned URL: " + err.Error())
    }
    C.Debug("S3 URL: " + presignedURL)

    // Redirect the client directly to S3.
    // From this point the client fetches the video bytes straight from S3,
    // our EC2 server is no longer in the data path.
    http.Redirect(w, r, presignedURL, http.StatusTemporaryRedirect)
    C.Okay("Redirected client to S3 for object: " + filePath)
    return nil
}

