package main

import (
    "github.com/jelius-sama/OpenMediaCloud/internal/router"
    "github.com/jelius-sama/OpenMediaCloud/internal/util"
    "net/http"
    "os"
    "path/filepath"

    "github.com/jelius-sama/logger"
    "github.com/joho/godotenv"
)

const VERSION = "v2.0.0"

var (
    // Set at compile time (use makefile)
    IS_PROD       string
    PORT          string
    CustomEnvPath *string
)

func init() {
    logger.Configure(logger.Cnf{
        IsDev: logger.IsDev{
            EnvironmentVariable: nil,
            ExpectedValue:       nil,
            DirectValue:         logger.BoolPtr(IS_PROD == "FALSE"),
        },
        UseSyslog: false,
    })

    if shouldExit := handleFlags(); shouldExit == true {
        os.Exit(0)
    }

    if CustomEnvPath != nil && len(*CustomEnvPath) != 0 {
        if err := godotenv.Load(*CustomEnvPath); err != nil {
            logger.Fatal(err)
        }
    } else {
        loadFromEtc := func() error {
            return godotenv.Load(filepath.Join("/etc", "OpenMediaCloud", ".env"))
        }

        userHome, err := os.UserHomeDir()

        if err != nil {
            logger.Error("Couldn't get user's home directory, loading from `/etc/OpenMediaCloud`.")
            err = loadFromEtc()
            if err != nil {
                logger.Fatal("Error loading environment variables.")
            }
        } else {
            err = godotenv.Load(filepath.Join(userHome, ".config", "OpenMediaCloud", ".env"))
            if err != nil {
                err = loadFromEtc()
                if err != nil {
                    logger.Fatal("Error loading environment variables.")
                }
            }
        }
    }
}

func main() {
    err := util.EnsureENV()
    if err != nil {
        logger.Fatal(err)
    }

    if keyPair, privKeyPath := os.Getenv("CLOUDFRONT_KEY_PAIR_ID"), os.Getenv("CLOUDFRONT_PRIVATE_KEY_PATH"); len(keyPair) != 0 && len(privKeyPath) != 0 {
        // NOTE: os.Stat doesn't necessarily mean we have read permission.
        file, err := os.OpenFile(privKeyPath, os.O_RDONLY, 0)
        if err != nil {
            logger.Fatal("Failed to read cloudfront private key:", err)
        }
        defer file.Close()
    }

    logger.Info("Starting server on port:", PORT)
    if err := http.ListenAndServe(PORT, router.Router()); err != nil {
        logger.Error("Failed to start server:", err)
    }
}

