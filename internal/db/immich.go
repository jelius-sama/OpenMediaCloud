package db

import (
    "database/sql"
    "fmt"

    _ "github.com/lib/pq"
)

type AssetPaths struct {
    ID           string
    OriginalPath string
    // nil if not generated
    EncodedVideoPath *string
    Thumbnail        *string
    Preview          *string
    Fullsize         *string
}

var ImmichConn *sql.DB

func ImmichConnect(cfg Config) error {
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password,
    )

    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return fmt.Errorf("db: open: %w", err)
    }

    if err := db.Ping(); err != nil {
        return fmt.Errorf("db: ping: %w", err)
    }

    ImmichConn = db
    return nil
}

func ImmichClose() error {
    if ImmichConn != nil {
        return ImmichConn.Close()
    }
    return nil
}

func ImmichGetAssetPaths(assetID string) (*AssetPaths, error) {
    const query = `
SELECT
    a.id,
    a."originalPath",
    a."encodedVideoPath",
    MAX(CASE WHEN af.type = 'thumbnail' THEN af.path END) AS thumbnail_path,
    MAX(CASE WHEN af.type = 'preview'   THEN af.path END) AS preview_path,
    MAX(CASE WHEN af.type = 'fullsize'  THEN af.path END) AS fullsize_path
FROM asset a
LEFT JOIN asset_file af ON a.id = af."assetId"
WHERE a.id = $1
GROUP BY a.id, a."originalPath", a."encodedVideoPath";
    `

    row := ImmichConn.QueryRow(query, assetID)

    var p AssetPaths
    err := row.Scan(&p.ID, &p.OriginalPath, &p.EncodedVideoPath, &p.Thumbnail, &p.Preview, &p.Fullsize)
    if err != nil {
        return nil, fmt.Errorf("db: query asset paths: %w", err)
    }

    return &p, nil
}

