package util

import (
    "errors"
    "net/http/httputil"
    "net/url"
    "os"
)

func MakeReverseProxy(target string) (*httputil.ReverseProxy, error) {
    parsed, err := url.Parse(target)
    if err != nil {
        return nil, err
    }
    return httputil.NewSingleHostReverseProxy(parsed), nil
}

// TODO: Cleanup required with the latest introduced UPSTREAM_{feature}_HOST variable.
// HINT: Grouping related variables into one (mutually inclusive/exclusive)
// returns uint8, error.
// uint8 is three digit, the first digit will indicate for jellyfin, second will
// indicate for immich and third for komga. 1 means the service is active, 0 otherwise.
func EnsureENV() (uint8, error) {
    var errs string = "The following environment variables are not set:\n"
    var errCount int = 0

    var jellyfinEnabled uint8 = 0
    var immichEnabled uint8 = 0
    var komgaEnabled uint8 = 0

    if upstreamJellyfinHost, upstreamImmichHost, upstreamKomgaHost := os.Getenv("UPSTREAM_JELLYFIN_HOST"), os.Getenv("UPSTREAM_IMMICH_HOST"), os.Getenv("UPSTREAM_KOMGA_HOST"); len(upstreamImmichHost) == 0 && len(upstreamJellyfinHost) == 0 && len(upstreamKomgaHost) == 0 {
        errCount++
        errs = errs + "\tNone of the following is set\n" + "\t\tUPSTREAM_JELLYFIN_HOST\n" + "\t\tUPSTREAM_IMMICH_HOST\n" + "\t\tUPSTREAM_KOMGA_HOST\n" + "\tPlease set at least one upstream hostname.\n"
    } else {
        if len(upstreamJellyfinHost) > 0 {
            jellyfinEnabled = 1
            if err := ValidateHost(upstreamJellyfinHost); err != nil {
                errCount++
                errs = errs + "\tUPSTREAM_JELLYFIN_HOST:\n\t\t" + err.Error() + "\n"
            }

            if val := os.Getenv("JELLYFIN_HOST"); len(val) == 0 {
                errCount++
                errs = errs + "\tJELLYFIN_HOST is not set\n"
            }

            if val := os.Getenv("JELLYFIN_API_KEY"); len(val) == 0 {
                errCount++
                errs = errs + "\tJELLYFIN_API_KEY is not set\n"
            }

            if val := os.Getenv("JELLYFIN_USER_ID"); len(val) == 0 {
                errCount++
                errs = errs + "\tJELLYFIN_USER_ID is not set\n"
            }

            if val := os.Getenv("JELLYFIN_AWS_REGION"); len(val) == 0 {
                errCount++
                errs = errs + "\tJELLYFIN_AWS_REGION is not set\n"
            }

            if val := os.Getenv("JELLYFIN_ACCESS_KEY_ID"); len(val) == 0 {
                errCount++
                errs = errs + "\tJELLYFIN_ACCESS_KEY_ID is not set\n"
            }

            if val := os.Getenv("JELLYFIN_SECRET_ACCESS_KEY"); len(val) == 0 {
                errCount++
                errs = errs + "\tJELLYFIN_SECRET_ACCESS_KEY is not set\n"
            }

            if val := os.Getenv("JELLYFIN_BUCKET_NAME"); len(val) == 0 {
                errCount++
                errs = errs + "\tJELLYFIN_BUCKET_NAME is not set\n"
            }
        }

        if len(upstreamImmichHost) > 0 {
            immichEnabled = 1
            if err := ValidateHost(upstreamImmichHost); err != nil {
                errCount++
                errs = errs + "\tUPSTREAM_KOMGA_HOST:\n\t\t" + err.Error() + "\n"
            }

            if val := os.Getenv("IMMICH_HOST"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_HOST is not set\n"
            }

            if val := os.Getenv("IMMICH_DB_PORT"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_DB_PORT is not set\n"
            }

            if val := os.Getenv("IMMICH_DB_HOST"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_DB_HOST is not set\n"
            }

            if val := os.Getenv("IMMICH_DB_NAME"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_DB_NAME is not set\n"
            }

            if val := os.Getenv("IMMICH_DB_USER"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_DB_USER is not set\n"
            }

            if val := os.Getenv("IMMICH_DB_PASSWORD"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_DB_PASSWORD is not set\n"
            }

            if val := os.Getenv("IMMICH_API_KEY"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_API_KEY is not set\n"
            }

            if val := os.Getenv("IMMICH_AWS_REGION"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_AWS_REGION is not set\n"
            }

            if val := os.Getenv("IMMICH_ACCESS_KEY_ID"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_ACCESS_KEY_ID is not set\n"
            }

            if val := os.Getenv("IMMICH_SECRET_ACCESS_KEY"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_SECRET_ACCESS_KEY is not set\n"
            }

            if val := os.Getenv("IMMICH_BUCKET_NAME"); len(val) == 0 {
                errCount++
                errs = errs + "\tIMMICH_BUCKET_NAME is not set\n"
            }
        }

        if len(upstreamKomgaHost) > 0 {
            komgaEnabled = 1
            if err := ValidateHost(upstreamKomgaHost); err != nil {
                errCount++
                errs = errs + "\tUPSTREAM_KOMGA_HOST:\n\t\t" + err.Error() + "\n"
            }
        }
    }

    if errCount > 0 {
        return (jellyfinEnabled << 0) | (immichEnabled << 1) | (komgaEnabled << 2), errors.New(errs)
    }

    return (jellyfinEnabled << 0) | (immichEnabled << 1) | (komgaEnabled << 2), nil
}

