package util

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "path"
    "strings"
)

// itemDetails holds only the fields we care about from /Items/{itemId} response.
type itemDetails struct {
    Path string `json:"Path"`
}

// getItemPath queries Jellyfin for the file path of a given itemId.
// It returns the raw filesystem path that Jellyfin has on record, e.g:
//
//	/AMVs/some_video.mp4
func GetItemPath(itemId string) (string, error) {
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
func ExtractItemId(pattern, urlPath string) (string, error) {
    parts := strings.Split(path.Clean(urlPath), "/")
    parsedPattern := strings.Split(path.Clean(pattern), "/")
    var idIndex int = -1

    for i := range parsedPattern {
        if strings.HasPrefix(parsedPattern[i], "{") && strings.HasSuffix(parsedPattern[i], "}") {
            idIndex = i
            break
        }
    }
    if idIndex == -1 {
        return "", fmt.Errorf("unexpected path format: %s", urlPath)
    }

    if len(parts) < len(parsedPattern) {
        return "", fmt.Errorf("unexpected path format: %s", urlPath)
    }

    return parts[idIndex], nil
}

