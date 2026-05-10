package util

import (
    "fmt"
    "path"
    "strings"
    "unicode"
)

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

func ValidateHost(host string) error {
    if len(host) > 253 {
        return fmt.Errorf("invalid hostname: %s, has more than 253 characters", host)
    }

    label := strings.Split(host, ".")

    validity := func(label string) bool {
        for i, r := range label {
            if label[i] == '-' || unicode.IsNumber(r) || unicode.IsLetter(r) {
                continue
            }
            return false
        }
        return true
    }

    for i := range label {
        if len(label[i]) > 63 || len(label[i]) < 1 {
            return fmt.Errorf("invlid hostname: %s, label exceeds length of 63 or is less than 1", host)
        }

        if valid := validity(label[i]); valid == false {
            return fmt.Errorf("invalid hostname: %s, includes invalid character", host)
        }

        if strings.HasPrefix(label[i], "-") || strings.HasSuffix(label[i], "-") {
            return fmt.Errorf("invalid hostname: %s, has leading or trailing dash character", host)
        }
    }

    return nil
}

