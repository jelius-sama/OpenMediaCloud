package jellyfin

import (
    "errors"
    "net/http"
    "strings"

    "github.com/jelius-sama/OpenMediaCloud/internal/s3"
    "github.com/jelius-sama/OpenMediaCloud/internal/util"

    "github.com/jelius-sama/logger"
)

func ApplyPatch(w http.ResponseWriter, r *http.Request, s3Client *s3.S3Client) error {
    logger.Debug("Applying patch, original path:", r.URL.Path)

    itemId, err := util.ExtractItemId("/Videos/{itemId}/stream", r.URL.Path)
    if err != nil {
        return errors.New("Failed to extract itemId: " + err.Error())
    }
    logger.Debug("Extracted itemId:", itemId)

    filePath, err := util.GetItemPath(itemId)
    if err != nil {
        return errors.New("Failed to get item path from Jellyfin: " + err.Error())
    }
    filePath = strings.TrimPrefix(filePath, "/")
    filePath = strings.TrimSuffix(filePath, "/")
    logger.Debug("Jellyfin returned file path:", filePath)

    presignedURL, err := s3Client.CreateSignedURL(r.Context(), filePath, nil)
    if err != nil {
        return errors.New("Failed to create presigned URL: " + err.Error())
    }
    logger.Debug("S3 URL:", presignedURL)

    // Redirect the client directly to S3.
    // From this point the client fetches the video bytes straight from S3,
    // our EC2 server is no longer in the data path.
    http.Redirect(w, r, presignedURL, http.StatusTemporaryRedirect)
    logger.Okay("Redirected client to S3 for object:", filePath)
    return nil
}

