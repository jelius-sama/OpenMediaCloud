#include "logger.h"
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/select.h>
#include <threads.h>

void ConfigureLogging() {
    DefaultMailAction mail = {
        .cfg_path = NULL,
        .body_template =
            MTTempl("An error occurred.\n\tLevel: %s\n\t%s logs: %s\n",
                    MTLLevel, MTLLevel, MTMessage, -1),
        .title = string("Error Alert"),
        .to = string("contact@jelius.dev"),
        .cc = NULL,
        .bcc = NULL,
    };

    Action action = {.on_fatal = &(ActionItem){.choice = ChoiceMail,
                                               .action = {.send_mail = mail}}};
    Configure(LDebug, SBrackets, &action);

    while (1) {
        select(0, NULL, NULL, NULL, NULL);
    }
}
