package handler

import (
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "os"
    "path"
    "strings"

    "ClientToR2/internal/s3"

    "github.com/jelius-sama/logger"
)

// itemDetails holds only the fields we care about from /Items/{itemId} response.
type itemDetails struct {
    Path string `json:"Path"`
}

// getItemPath queries Jellyfin for the file path of a given itemId.
// It returns the raw filesystem path that Jellyfin has on record, e.g:
//
//	/AMVs/some_video.mp4
func getItemPath(itemId string) (string, error) {
    endpoint := fmt.Sprintf("%s/Items/%s?UserId=%s", os.Getenv("JELLYFIN_HOST"), itemId, os.Getenv("JELLYFIN_USER_ID"))

    req, err := http.NewRequest("GET", endpoint, nil)
    if err != nil {
        return "", fmt.Errorf("failed to build request: %w", err)
    }

    req.Header.Set("X-Emby-Token", os.Getenv("JELLYFIN_API_KEY"))

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("failed to contact Jellyfin: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("Jellyfin returned status %d for itemId %s", resp.StatusCode, itemId)
    }

    var details itemDetails
    if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
        return "", fmt.Errorf("failed to decode Jellyfin response: %w", err)
    }

    if details.Path == "" {
        return "", fmt.Errorf("Jellyfin returned empty path for itemId %s", itemId)
    }

    return details.Path, nil
}

// extractItemId pulls the itemId out of a path like /Videos/{itemId}/stream
func extractItemId(urlPath string) (string, error) {
    parts := strings.Split(path.Clean(urlPath), "/")

    // Expected: ["", "Videos", "{itemId}", "stream"]
    if len(parts) < 3 {
        return "", fmt.Errorf("unexpected path format: %s", urlPath)
    }

    return parts[2], nil
}

func ApplyVideosPatch(w http.ResponseWriter, r *http.Request, s3Client *s3.S3Client) error {
    logger.Debug("Applying videos patch, original path:", r.URL.Path)

    itemId, err := extractItemId(r.URL.Path)
    if err != nil {
        return errors.New("Failed to extract itemId: " + err.Error())
    }
    logger.Debug("Extracted itemId:", itemId)

    filePath, err := getItemPath(itemId)
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

