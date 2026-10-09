#include "goalchemy.h"
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

typedef struct { int count; int64_t levels[8]; char texts[8][128]; char attrs[8][4][64]; size_t attr_counts[8]; int64_t nanos[8]; } capture;

static void sink(void *state, const gxc_log_record *r) {
    capture *c = state;
    if (c->count >= 8) return;
    int i = c->count++;
    c->levels[i] = r->level; c->nanos[i] = r->unix_nano; c->attr_counts[i] = r->attr_count;
    assert(strlen(r->text) == r->text_length && r->text_length < 128);
    memcpy(c->texts[i], r->text, r->text_length + 1);
    for (size_t j = 0; j < 2 * r->attr_count && j < 4; j++) { assert(r->attr_lengths[j] < 64); memcpy(c->attrs[i][j], r->attrs[j], r->attr_lengths[j] + 1); }
}

static int64_t work(int64_t n) {
    gxc_value input = {.kind = GXC_INT, .integer = n}, result = {0};
    gxc_error error = {0};
    int status = goalchemy_invoke("Work", &input, 1, NULL, &result, &error);
    assert(status == 0 && result.kind == GXC_INT);
    gxc_error_free(&error);
    int64_t value = result.integer;
    gxc_value_free(&result);
    return value;
}

int main(void) {
    capture c = {0};
    gxc_set_log_handler(sink, &c, -4);
    assert(work(3) == 6);
    assert(c.count == 4);
    assert(c.levels[0] == -4 && !strcmp(c.texts[0], "level=DEBUG msg=start sdk=probe n=3"));
    assert(!strcmp(c.texts[1], "level=INFO msg=info sdk=probe unicode=\"h\xc3\xa9llo w\xc3\xb6rld\"") && !strcmp(c.attrs[1][3], "h\xc3\xa9llo w\xc3\xb6rld"));
    assert(c.levels[2] == 4 && c.attr_counts[2] == 3 && !strcmp(c.attrs[2][2], "kas.url") && !strcmp(c.attrs[2][3], "https://kas"));
    assert(!strcmp(c.texts[3], "level=ERROR msg=failed err=boom") && llabs(c.nanos[3] / 1000000000 - (int64_t)time(NULL)) < 60);
    capture warn = {0};
    gxc_set_log_handler(sink, &warn, 4);
    work(1);
    assert(warn.count == 2 && warn.levels[0] == 4 && warn.levels[1] == 8);
    gxc_set_log_handler(NULL, NULL, 4);
    printf("PASS log library\n");
    return 0;
}
