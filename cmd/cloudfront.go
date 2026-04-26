package main

import (
    "path"
    "strings"
    "sync"
)

type decodeTask struct {
    currentPath string
    remaining   []string
    isWildcard  bool // true if any segment in the original input contained a wildcard
}

// TODO: Integrate with S3 compatible bucket storage
func listBucketItems(prefix string) []string {
    switch prefix {
    case "/":
        return []string{"Anime/"}
    case "/Anime/":
        return []string{"Naruto/", "One Piece/"}
    case "/Anime/Naruto/":
        return []string{"Season 1/", "Season 2/"}
    case "/Anime/One Piece/":
        return []string{"Season 1/"}
    case "/Anime/Naruto/Season 1/":
        return []string{"Ep1.mkv", "Ep2.mkv"}
    case "/Anime/Naruto/Season 2/":
        return []string{"Ep1.mkv"}
    case "/Anime/One Piece/Season 1/":
        return []string{"Ep1.mkv"}
    default:
        return []string{}
    }
}

func hasWildcard(s string) bool {
    return strings.ContainsAny(s, "*?[")
}

func inputHasWildcard(parts []string) bool {
    for _, p := range parts {
        if hasWildcard(p) {
            return true
        }
    }
    return false
}

// emitResult decides what to emit for a resolved path.
//
// Wildcard mode (e.g. ls Anime/\*):
//   - Mimics s5cmd: emits every matched entry as-is, whether file or directory.
//
// Non-wildcard mode (e.g. ls Anime/ or ls Anime/Naruto):
//   - Mimics aws s3 ls: if the resolved path is a directory, list its contents.
//   - If it is a file, emit the file directly.
func emitResult(resolvedPath string, isDir bool, isWildcard bool, resultsChan chan<- string) {
    if isWildcard {
        // Wildcard: emit the match itself, like s5cmd does
        resultsChan <- resolvedPath
        return
    }

    if isDir {
        // Non-wildcard directory: emit its contents, like aws s3 ls
        children := listBucketItems(resolvedPath)
        for _, child := range children {
            resultsChan <- resolvedPath + child
        }
    } else {
        // Non-wildcard file: emit directly
        resultsChan <- resolvedPath
    }
}

func decodePath(inputPath string, workerCount int) []string {
    // Normalize: strip s3:/ prefix, clean to a canonical slash-prefixed path
    inputPath = strings.TrimPrefix(inputPath, "s3:/")
    inputPath = "/" + strings.Trim(inputPath, "/")

    // Root listing: directly emit root contents
    if inputPath == "/" {
        entries := listBucketItems("/")
        var result []string
        for _, e := range entries {
            result = append(result, "/"+e)
        }
        return result
    }

    // Split into non-empty parts, preserving wildcard segments
    parts := strings.Split(strings.Trim(inputPath, "/"), "/")
    wildcardMode := inputHasWildcard(parts)

    taskQueue := make(chan decodeTask, 2000000)
    resultsChan := make(chan string, 2000000)

    var workerWg sync.WaitGroup
    var activeTasks sync.WaitGroup

    activeTasks.Add(1)
    taskQueue <- decodeTask{
        currentPath: "/",
        remaining:   parts,
        isWildcard:  wildcardMode,
    }

    for range workerCount {
        workerWg.Add(1)
        workerWg.Go(func() {
            defer workerWg.Done()
            for t := range taskQueue {
                segment := t.remaining[0]
                nextRemaining := t.remaining[1:]

                entries := listBucketItems(t.currentPath)

                if len(entries) == 0 {
                    activeTasks.Done()
                    continue
                }

                for _, entry := range entries {
                    cleanEntry := strings.TrimSuffix(entry, "/")
                    isDir := strings.HasSuffix(entry, "/")

                    matched, err := path.Match(segment, cleanEntry)
                    if err != nil || !matched {
                        continue
                    }

                    newPath := t.currentPath + entry

                    if len(nextRemaining) == 0 {
                        emitResult(newPath, isDir, t.isWildcard, resultsChan)
                    } else if isDir {
                        // More segments to resolve — recurse into directory
                        activeTasks.Add(1)
                        taskQueue <- decodeTask{
                            currentPath: newPath,
                            remaining:   nextRemaining,
                            isWildcard:  t.isWildcard,
                        }
                    }
                    // nextRemaining non-empty but entry is a file — cannot descend, skip
                }

                activeTasks.Done()
            }
        })
    }

    go func() {
        activeTasks.Wait()
        close(taskQueue)
        close(resultsChan)
    }()

    var final []string
    for r := range resultsChan {
        final = append(final, r)
    }
    workerWg.Wait()
    return final
}

