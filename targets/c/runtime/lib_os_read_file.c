#include "gx.h"
#include <errno.h>
#include <fcntl.h>
#include <stdlib.h>
#include <sys/stat.h>
#include <unistd.h>
#define GX_OS_MAX_FILE ((size_t)1 << 30)
int64_t gx_os_status(int e) {
    switch (e) {
    case ENOENT: return 1;
    case EEXIST: return 2;
    case EACCES: case EPERM: return 3;
    case EISDIR: return 4;
    case ENOTDIR: return 5;
    case EFBIG: return 6;
#if defined(ENOTSUP) && defined(EOPNOTSUPP) && ENOTSUP != EOPNOTSUPP
    case ENOTSUP: return 7;
#endif
    case ENOSYS: case EOPNOTSUPP: return 7;
    case EINVAL: return 8;
    default: return 9;
    }
}
char *gx_os_path(gx_V name, int64_t *status) {
    size_t n = gx_slen(name);
    const uint8_t *p = gx_sbytes(name);
    if (n == 0) { *status = 1; return NULL; }
    if (memchr(p, 0, n)) { *status = 8; return NULL; }
    char *path = malloc(n + 1);
    if (!path) gx_fault("out of memory");
    memcpy(path, p, n);
    path[n] = 0;
    return path;
}
gx_V gx_lib_os_read_file(gx_V name) {
    int64_t status = 0;
    uint8_t *data = NULL;
    size_t len = 0;
    char *path = gx_os_path(name, &status);
    if (path) {
        int fd = open(path, O_RDONLY | O_CLOEXEC);
        free(path);
        if (fd < 0) status = gx_os_status(errno);
        else {
            struct stat st;
            if (fstat(fd, &st) == 0 && (uint64_t)st.st_size > GX_OS_MAX_FILE) status = 6;
            else {
                size_t cap = fstat(fd, &st) == 0 && st.st_size > 0 ? (size_t)st.st_size + 1 : 4096;
                data = gx_alloc_bytes(cap);
                for (;;) {
                    if (len == cap) {
                        if (cap > GX_OS_MAX_FILE) { status = 6; break; }
                        size_t next = cap * 2 > GX_OS_MAX_FILE + 1 ? GX_OS_MAX_FILE + 1 : cap * 2;
                        uint8_t *grown = gx_alloc_bytes(next);
                        memcpy(grown, data, len);
                        data = grown; cap = next;
                    }
                    ssize_t r = read(fd, data + len, cap - len);
                    if (r < 0) { if (errno == EINTR) continue; status = gx_os_status(errno); break; }
                    if (r == 0) break;
                    len += (size_t)r;
                }
                if (status == 0 && len > GX_OS_MAX_FILE) status = 6;
            }
            close(fd);
        }
    }
    gx_V a[2] = {status ? gx_nil_byte_slice() : gx_byte_slice(data, (uint32_t)len, (uint32_t)len), gx_int(status)};
    return gx_tuple(2, a);
}
