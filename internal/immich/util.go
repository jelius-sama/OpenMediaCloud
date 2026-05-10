package immich

import (
    "net/http"
    "os"
    "path"
    "strings"

    "github.com/jelius-sama/logger"
)

type pathKindT uint8

// Format: 1...n0|1
// 0 indicates GET 1 indicates POST
// 1st value will be the specific route kind and the second value is the method.
// Only two method are supported as those are the only method we need to get this running.
const (
    pathKindPersonThumbnail pathKindT = (iota+1)<<4 | 0
    pathKindProfileImage
    pathKindDownloadAsset
    pathKindViewAsset
    pathKindAssetVideo
    pathKindDownloadArchive pathKindT = (iota+1)<<4 | 1
    pathKindFallback        pathKindT = (iota+1)<<4 | 2 // special case: 2 here means any possible meethod
)

func (p pathKindT) ID() uint8 {
    return uint8(p >> 4)
}

// Method extracts the RHS (the method bit)
func (p pathKindT) Method() string {
    m := uint8(p & 0x0F)
    switch m {
    case 0:
        return http.MethodGet
    case 1:
        return http.MethodPost
    case 2:
        return "ANY"
    default:
        return "UNKNOWN"
    }
}

func matchPattern(pattern string, str string) (bool, []string) {
    str, _ = strings.CutSuffix(str, "?")
    logger.Debug("matchPattern()\n\tstr:", str, "\n\tpattern:", pattern)
    patternParts := strings.Split(path.Clean(pattern), "/")
    strParts := strings.Split(path.Clean(str), "/")

    var dynamicParts []string

    if len(patternParts) != len(strParts) {
        return false, dynamicParts
    }

    for i := range patternParts {
        if strings.HasPrefix(patternParts[i], "{") && strings.HasSuffix(patternParts[i], "}") {
            dynamicParts = append(dynamicParts, strParts[i])
            continue
        } else {
            if patternParts[i] != strParts[i] {
                return false, dynamicParts
            }
        }
    }

    return true, dynamicParts
}

func forwardTo(path string, method string) (pathKindT, []string) {
    if matched, dynamicParts := matchPattern("/api/people/{id}/thumbnail", path); matched && pathKindPersonThumbnail.Method() == method {
        return pathKindPersonThumbnail, dynamicParts
    }

    if matched, dynamicParts := matchPattern("/api/users/{id}/profile-image", path); matched && pathKindProfileImage.Method() == method {
        return pathKindProfileImage, dynamicParts
    }

    if matched, dynamicParts := matchPattern("/api/assets/{id}/original", path); matched && pathKindDownloadAsset.Method() == method {
        return pathKindDownloadAsset, dynamicParts
    }

    if matched, dynamicParts := matchPattern("/api/assets/{id}/thumbnail", path); matched && pathKindViewAsset.Method() == method {
        return pathKindViewAsset, dynamicParts
    }

    if matched, dynamicParts := matchPattern("/api/assets/{id}/video/playback", path); matched && pathKindAssetVideo.Method() == method {
        return pathKindAssetVideo, dynamicParts
    }

    if matched, _ := matchPattern("/api/download/archive", path); matched && pathKindDownloadArchive.Method() == method {
        return pathKindDownloadArchive, nil
    }

    return pathKindFallback, nil
}

// NOTE: List of required permissions:
// asset.view
// asset.download
// person.read
// userProfileImage.read
// user.read
// Use only the permission listed above to follow the principle of least priviledge.

type assetSize uint8

const (
    asOriginal assetSize = iota
    asFullsize
    asPreview
    asThumbnail
)

func (as assetSize) String() string {
    switch as {
    case asOriginal:
        return "original"
    case asFullsize:
        return "fullsize"
    case asPreview:
        return "preview"
    case asThumbnail:
        return "thumbnail"
    default:
        logger.Panic("unreachable")
        return ""
    }
}

func immichPathToS3Key(originalPath string) string {
    cleaned := path.Clean(originalPath)
    // strip the mount prefix
    trimmed := strings.TrimPrefix(cleaned, os.Getenv("IMMICH_PATH_PREFIX"))
    // join with the s3 prefix and clean up leading slash
    joined := path.Join(os.Getenv("IMMICH_S3_PREFIX"), trimmed)
    return strings.TrimPrefix(joined, "/")
}

