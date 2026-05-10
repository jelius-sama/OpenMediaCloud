package immich

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"

    "github.com/jelius-sama/OpenMediaCloud/internal/s3"
    "github.com/jelius-sama/logger"
)

func getPersonThumbnail(w http.ResponseWriter, r *http.Request, ids []string, s3Client *s3.S3Client) error {
    if len(ids) > 1 || len(ids) == 0 {
        logger.Debug("Expected exactly 1 ID, got more than 1 or less than 0.")
        logger.Info("More than 1 ID parsed, using ID at index 0.")
    }

    req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/people/%s", os.Getenv("IMMICH_HOST"), ids[0]), nil)
    if err != nil {
        return err
    }
    req.Header.Set("x-api-key", os.Getenv("IMMICH_API_KEY"))
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to contact Immich server: %w", err)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("error code %d when making API request to Immich server", resp.StatusCode)
    }

    var personInfo map[string]string
    if err := json.Unmarshal(body, &personInfo); err != nil {
        return err
    }

    presignedURL, err := s3Client.CreateSignedURL(s3.CreateSignedURLT{
        Ctx:                 r.Context(),
        ObjectKey:           immichPathToS3Key(personInfo["thumbnailPath"]),
        FallbackContentType: nil,
        CFEndpoint:          os.Getenv("IMMICH_CLOUDFRONT_ENDPOINT"),
        CFKeyPairID:         os.Getenv("IMMICH_CLOUDFRONT_KEY_PAIR_ID"),
        CFPrivateKeyPath:    os.Getenv("IMMICH_CLOUDFRONT_PRIVATE_KEY_PATH"),
    })

    if err != nil {
        return fmt.Errorf("Failed to create presigned URL: %w", err)
    }
    logger.Debug("S3 URL:", presignedURL)

    // Redirect the client directly to S3.
    // From this point the client fetches the video bytes straight from S3,
    // our EC2 server is no longer in the data path.
    http.Redirect(w, r, presignedURL, http.StatusTemporaryRedirect)
    logger.Okay("Redirected client to S3 for object:" + "\n\t`" + immichPathToS3Key(personInfo["thumbnailPath"]) + "`")

    return nil
}

