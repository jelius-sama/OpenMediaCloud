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

#include <stdarg.h>

typedef struct {
    const char *data;
    int64_t len;
} String;

typedef struct {
    String *cfg_path;
    String body_template;
    String title;
    String to;
    StrArr *cc;
    StrArr *bcc;
} DefaultMailAction;

void free_string(String);
String string(const char *, ...);
void Debug(const String msg);
void Info(const String msg);
void Okay(const String msg);
void Warn(const String msg);
void Error(const String msg);
void Fatal(const String msg);
void Panic(const String msg);
String MTTempl(const char *, ...);

typedef union {
    void (*callback)(void);
    DefaultMailAction send_mail;
} ActionChoice;

typedef struct {
    Choice choice;
    ActionChoice action;
} ActionItem;

typedef struct {
    ActionItem *on_debug;
    ActionItem *on_okay;
    ActionItem *on_info;
    ActionItem *on_warn;
    ActionItem *on_error;
    ActionItem *on_panic;
    ActionItem *on_fatal;
} Action;

void Configure(LogLevel level, LogStyle style, Action *action);

#endif // LOGGER_H
