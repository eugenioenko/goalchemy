/* Go maps: an open-addressing index from encoded keys to insertion-ordered
 * entries. Deleted entries stay in the entry list, marked dead, until the
 * list is compacted; iteration walks a snapshot of the list. */
#include "gx.h"

#define TOMB ((size_t)-1)

static uint64_t hash(const uint8_t *k, size_t n) {
    uint64_t h = 1469598103934665603ULL;
    for (size_t i = 0; i < n; i++) {
        h ^= k[i];
        h *= 1099511628211ULL;
    }
    return h;
}

static void rebuild(gx_Map *m, size_t nslots) {
    m->slots = GC_MALLOC_ATOMIC(nslots * sizeof(size_t));
    memset(m->slots, 0, nslots * sizeof(size_t));
    m->nslots = nslots;
    for (size_t i = 0; i < m->n; i++) {
        gx_Entry *e = m->entries[i];
        if (!e->live) continue;
        size_t s = hash(e->key, e->keyn) & (nslots - 1);
        while (m->slots[s]) s = (s + 1) & (nslots - 1);
        m->slots[s] = i + 1;
    }
}

gx_Entry *gx_map_find(gx_Map *m, const uint8_t *key, size_t n) {
    if (!m->nslots) return NULL;
    size_t s = hash(key, n) & (m->nslots - 1);
    for (;;) {
        size_t x = m->slots[s];
        if (x == 0) return NULL;
        if (x != TOMB) {
            gx_Entry *e = m->entries[x - 1];
            if (e->keyn == n && memcmp(e->key, key, n) == 0) return e;
        }
        s = (s + 1) & (m->nslots - 1);
    }
}

static void compact(gx_Map *m) {
    size_t j = 0;
    for (size_t i = 0; i < m->n; i++)
        if (m->entries[i]->live) m->entries[j++] = m->entries[i];
    for (size_t i = j; i < m->n; i++) m->entries[i] = NULL;
    m->n = j;
}

void gx_map_insert(gx_Map *m, gx_V k, gx_V v, uint8_t *key, size_t n) {
    if (m->n > 32 && m->n > 2 * m->live) compact(m);
    if (m->n == m->cap) {
        size_t c = m->cap ? m->cap * 2 : 8;
        gx_Entry **ne = GC_MALLOC(c * sizeof(gx_Entry *));
        if (m->n) memcpy(ne, m->entries, m->n * sizeof(gx_Entry *));
        m->entries = ne;
        m->cap = c;
    }
    gx_Entry *e = GC_MALLOC(sizeof(gx_Entry));
    e->k = k;
    e->v = v;
    e->live = true;
    e->key = key;
    e->keyn = n;
    m->entries[m->n++] = e;
    m->live++;
    if ((m->n + 1) * 2 > m->nslots) {
        size_t ns = 16;
        while (ns < (m->n + 1) * 4) ns *= 2;
        rebuild(m, ns);
        return;
    }
    size_t s = hash(key, n) & (m->nslots - 1);
    while (m->slots[s] && m->slots[s] != TOMB) s = (s + 1) & (m->nslots - 1);
    m->slots[s] = m->n;
}

bool gx_map_remove(gx_Map *m, const uint8_t *key, size_t n) {
    if (!m->nslots) return false;
    size_t s = hash(key, n) & (m->nslots - 1);
    for (;;) {
        size_t x = m->slots[s];
        if (x == 0) return false;
        if (x != TOMB) {
            gx_Entry *e = m->entries[x - 1];
            if (e->keyn == n && memcmp(e->key, key, n) == 0) {
                e->live = false;
                m->slots[s] = TOMB;
                m->live--;
                return true;
            }
        }
        s = (s + 1) & (m->nslots - 1);
    }
}

void gx_map_reset(gx_Map *m) {
    for (size_t i = 0; i < m->n; i++) m->entries[i]->live = false;
    m->entries = NULL;
    m->n = m->cap = 0;
    m->slots = NULL;
    m->nslots = 0;
    m->live = 0;
}
