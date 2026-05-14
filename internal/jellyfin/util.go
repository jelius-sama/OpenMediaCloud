package jellyfin

/*
#include "../../libs/logger/logger.h"
*/
import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "regexp"
)

/*
 * EXAMPLE:
 * /Videos/877d0f740648605d91c17d147a9a9ff8/stream
 * /Videos/{id}/stream.{container}
 */
var mediaPaths = []*regexp.Regexp{
    regexp.MustCompile(`^/Videos/[^/]+/stream$`),
    regexp.MustCompile(`^/Videos/[^/]+/stream\.[a-zA-Z0-9]+$`),
    regexp.MustCompile(`^/Audio/[^/]+/stream$`),
    regexp.MustCompile(`^/Audio/[^/]+/universal$`),
    regexp.MustCompile(`^/Audio/[^/]+/stream\.[a-zA-Z0-9]+$`),
}

var mediaInfoPath = []*regexp.Regexp{
    regexp.MustCompile(`^/Items/[^/]+/PlaybackInfo$`),
}

/*
 * EXAMPLE:
 * /videos/877d0f74-0648-605d-91c1-7d147a9a9ff8/master.m3u8
 * /videos/877d0f74-0648-605d-91c1-7d147a9a9ff8/main.m3u8
 * /videos/877d0f74-0648-605d-91c1-7d147a9a9ff8/hls1/main/0.mp4
 */
var hlsPaths = []*regexp.Regexp{
    regexp.MustCompile(`^/audio/[^/]+/master\.m3u8$`),
    regexp.MustCompile(`^/audio/[^/]+/main\.m3u8$`),
    regexp.MustCompile(`^/audio/[^/]+/hls[^/]+/main/-?\d+\.[a-zA-Z0-9]+$`),
    regexp.MustCompile(`^/videos/[^/]+/master\.m3u8$`),
    regexp.MustCompile(`^/videos/[^/]+/main\.m3u8$`),
    regexp.MustCompile(`^/videos/[^/]+/hls[^/]+/main/-?\d+\.[a-zA-Z0-9]+$`),
}

// /Items/877d0f740648605d91c17d147a9a9ff8/Images/Primary
// /Items/{itemId}/Images/{imageType}
// INFO: Since image files are relatively small in size I doubt we really
//       need a solution for this part of the problem. This feature may
//       not be implemented or even if it were to be implemented it would be
//       very much at a later point as it is not a priority feature.
/* NOTE: Jellyfin does not really store images in a very accessible place
 * it is stored in it's cache directory, we could just make the cache
 * directory an rclone mount but that may slow things down, this is
 * basically gonna use the same principle as the media files.
 * Another approach is, when a request for an image comes in we serve it
 * and then we check if that specific image exists in R2 bucket with some
 * sort of KeyValue DB and if it doesn't exists we can store the image in
 * our own R2 "cache" folder and then update the KV DB, this was any
 * subsequent request with the same image request signature can be served
 * by redirecting to R2 instead of letting the VPS handle it.
 */
var imagePaths = []*regexp.Regexp{}

// /Items/{itemId}/Download
var downloadPaths = []*regexp.Regexp{
    regexp.MustCompile(`^/Items/[^/]+/Download$`),
}

type pathKindT = uint8

const (
    pathKindMedia pathKindT = iota
    pathKindMediaInfo
    pathKindDownloads
    pathKindImage
    pathKindHLS
    pathKindDefault
)

func forwardTo(path string) pathKindT {
    for _, pattern := range mediaPaths {
        if pattern.MatchString(path) {
            return pathKindMedia
        }
    }

    for _, pattern := range mediaInfoPath {
        if pattern.MatchString(path) {
            return pathKindMediaInfo
        }
    }

    for _, pattern := range downloadPaths {
        if pattern.MatchString(path) {
            return pathKindDownloads
        }
    }

    for _, pattern := range hlsPaths {
        if pattern.MatchString(path) {
            return pathKindHLS
        }
    }

    for _, pattern := range imagePaths {
        if pattern.MatchString(path) {
            return pathKindImage
        }
    }

    return pathKindDefault
}

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

