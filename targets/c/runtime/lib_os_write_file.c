#include "gx.h"
#include <errno.h>
#include <fcntl.h>
#include <stdlib.h>
#include <unistd.h>
gx_V gx_lib_os_write_file(gx_V name, gx_V data, gx_V perm) {
    int64_t status = 0;
    if (data.l > ((size_t)1 << 30)) return gx_int(6);
    char *path = gx_os_path(name, &status);
    if (!path) return gx_int(status);
    int fd = open(path, O_WRONLY | O_CREAT | O_TRUNC | O_CLOEXEC, (mode_t)(gx_i(perm) & 0777));
    free(path);
    if (fd < 0) return gx_int(gx_os_status(errno));
    size_t done = 0;
    while (done < data.l) {
        ssize_t w;
        if (gx_byte_backing(data)) w = write(fd, gx_bytes(data) + done, data.l - done);
        else { uint8_t b = (uint8_t)gx_i(gx_vals(data)[done]); w = write(fd, &b, 1); }
        if (w < 0) { if (errno == EINTR) continue; status = gx_os_status(errno); break; }
        done += (size_t)w;
    }
    if (close(fd) != 0 && status == 0) status = gx_os_status(errno);
    return gx_int(status);
}
