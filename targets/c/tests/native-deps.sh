# Native C test build dependencies. Optional isolated sysroot stays under SDK .local.
if ! pkg-config --exists libcurl 2>/dev/null; then
 gx_curl_prefix="$repo/../sdk/.local/root-c-development-prerequisites/prefix"
 if [ -d "$gx_curl_prefix/usr/lib/x86_64-linux-gnu/pkgconfig" ]; then
  export PKG_CONFIG_LIBDIR="$gx_curl_prefix/usr/lib/x86_64-linux-gnu/pkgconfig"
  export PKG_CONFIG_SYSROOT_DIR="$gx_curl_prefix" PKG_CONFIG_ALLOW_SYSTEM_LIBS=1
  export LD_LIBRARY_PATH="$gx_curl_prefix/usr/lib/x86_64-linux-gnu:${LD_LIBRARY_PATH:-}"
 fi
fi
GX_NATIVE_CFLAGS=$(pkg-config --cflags libcurl)
GX_NATIVE_LIBS="$(pkg-config --libs libcurl) -lz -lssl -lcrypto"
