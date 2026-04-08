package main

import (
    "ClientToR2/internal/router"
    "ClientToR2/internal/util"
    "net/http"
    "os"
    "path/filepath"

    "github.com/jelius-sama/logger"
    "github.com/joho/godotenv"
)

const VERSION = "v1.0.1"

var (
    // Set at compile time (use makefile)
    IS_PROD string
    PORT    string
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

    loadFromEtc := func() error {
        return godotenv.Load(filepath.Join("/etc", "client-to-r2", ".env"))
    }

    userHome, err := os.UserHomeDir()

    if err != nil {
        logger.Error("Couldn't get user's home directory, loading from `/etc/client-to-r2`.")
        err = loadFromEtc()
        if err != nil {
            logger.Fatal("Error loading environment variables.")
        }
    } else {
        err = godotenv.Load(filepath.Join(userHome, ".config", "client-to-r2", ".env"))
        if err != nil {
            err = loadFromEtc()
            if err != nil {
                logger.Fatal("Error loading environment variables.")
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
        file, err := os.OpenFile(privKeyPath, os.O_RDWR, 0)
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

