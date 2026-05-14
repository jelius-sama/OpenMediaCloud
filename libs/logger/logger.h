#ifndef LOGGER_H
#define LOGGER_H

#include <stddef.h>
#include <stdint.h>

typedef enum {
    LDebug = 0,
    LOkay = 1,
    LInfo = 2,
    LWarn = 3,
    LError = 4,
    LFatal = 5,
    LPanic = 6,
} LogLevel;

typedef enum {
    SBrackets = 0,
    SColon = 1,
    SNone = 2,
} LogStyle;

typedef enum {
    ChoiceMail = 0,
    ChoiceCallback = 1,
} Choice;

typedef enum {
    MTMessage = 0,
    MTLLevel = 1,
} DefaultMailBodyTemplate;

typedef struct {
    char *str;
    size_t len;
    size_t count;
} StrArr;

#ifdef __clangd__
typedef struct {
} _GoString_;
#endif

typedef struct {
    _GoString_ *cfg_path;
    _GoString_ body_template;
    _GoString_ title;
    _GoString_ to;
    StrArr *cc;
    StrArr *bcc;
} DefaultMailAction;

void Info(const _GoString_ msg);
void Debug(const _GoString_ msg);
void Okay(const _GoString_ msg);
void Warn(const _GoString_ msg);
void Error(const _GoString_ msg);
void Fatal(const _GoString_ msg);
void Panic(const _GoString_ msg);
// _GoString_ MTTempl(const char *, ...);
_GoString_ MTTempl(const char *, DefaultMailBodyTemplate,
                   DefaultMailBodyTemplate, DefaultMailBodyTemplate, int);

typedef union {
    void (*callback)(void);
    DefaultMailAction send_mail;
} ActionChoice;

typedef struct {
    Choice choice;
    ActionChoice action;
} ActionItem;

// typedef struct {
//     ActionItem *on_debug;
//     ActionItem *on_okay;
//     ActionItem *on_info;
//     ActionItem *on_warn;
//     ActionItem *on_error;
//     ActionItem *on_panic;
//     ActionItem *on_fatal;
// } Action;

typedef struct {
    void *on_debug;
    void *on_okay;
    void *on_info;
    void *on_warn;
    void *on_error;
    void *on_panic;
    void *on_fatal;
} Action;

void Configure(LogLevel level, LogStyle style, Action *action);

#endif // LOGGER_H
