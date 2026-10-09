/* Host log sink for std/log/slog; see gxc_set_log_handler in library.h. */
#include "gx.h"
#include "library.h"
#include <pthread.h>
#include <stdlib.h>

static pthread_mutex_t log_lock = PTHREAD_MUTEX_INITIALIZER;
static gxc_log_handler log_handler;
static void *log_state;
static int64_t log_level = 4;

void gxc_set_log_handler(gxc_log_handler handler, void *state, int64_t level) {
    pthread_mutex_lock(&log_lock);
    log_handler = handler; log_state = state; log_level = level;
    pthread_mutex_unlock(&log_lock);
}

bool gx_log_enabled(int64_t level) {
    pthread_mutex_lock(&log_lock);
    bool enabled = level >= log_level;
    pthread_mutex_unlock(&log_lock);
    return enabled;
}

static char *log_copy(gx_V s) {
    char *c = malloc(gx_slen(s) + 1);
    if (!c) gx_fault("out of memory");
    if (gx_slen(s)) memcpy(c, gx_sbytes(s), gx_slen(s));
    c[gx_slen(s)] = 0;
    return c;
}

void gx_log_emit(int64_t level, int64_t unix_nano, gx_V message, gx_V attrs, gx_V text) {
    pthread_mutex_lock(&log_lock);
    gxc_log_handler handler = log_handler;
    void *state = log_state;
    pthread_mutex_unlock(&log_lock);
    if (!handler) {
        gx_stderr(gx_sbytes(text), gx_slen(text));
        gx_stderr("\n", 1);
        return;
    }
    size_t n = attrs.l / 2 * 2;
    char **items = calloc(n ? n : 1, sizeof *items);
    size_t *lengths = calloc(n ? n : 1, sizeof *lengths);
    if (!items || !lengths) gx_fault("out of memory");
    gx_V *values = n ? gx_vals(attrs) : NULL;
    for (size_t i = 0; i < n; i++) { items[i] = log_copy(values[i]); lengths[i] = gx_slen(values[i]); }
    char *msg = log_copy(message), *line = log_copy(text);
    gxc_log_record record = {level, unix_nano, msg, gx_slen(message), (const char *const *)items, lengths, n / 2, line, gx_slen(text)};
    handler(state, &record);
    for (size_t i = 0; i < n; i++) free(items[i]);
    free(items); free(lengths); free(msg); free(line);
}
