/* Fresh-process diagnostic: actual runtime payloads, no mandatory thresholds. */
#define _POSIX_C_SOURCE 200809L
#include "gx.h"
#include <gc/gc_version.h>
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <time.h>
#include <sys/resource.h>

static double seconds(void) {
    struct timespec t; assert(clock_gettime(CLOCK_MONOTONIC,&t) == 0);
    return (double)t.tv_sec + (double)t.tv_nsec/1e9;
}
static uint8_t pattern(size_t i) { return (uint8_t)(i*17+31); }
static void verify(gx_V x, size_t n) {
    for (size_t i=0;i<n;i++) {
        uint8_t actual = gx_byte_backing(x) ? gx_bytes(x)[i] : (uint8_t)gx_i(gx_vals(x)[i]);
        assert(actual == pattern(i));
    }
}
int main(int argc, char **argv) {
    assert(argc==3);
    bool native = !strcmp(argv[1],"native");
    assert(native || !strcmp(argv[1],"generic"));
    size_t n=(size_t)strtoull(argv[2],NULL,10);
    assert(n==2*1024*1024 || n==8*1024*1024);
    GC_INIT(); assert(GC_get_all_interior_pointers()); GC_gcollect();
    printf("bdwgc=%d.%d.%d\n",GC_VERSION_MAJOR,GC_VERSION_MINOR,GC_VERSION_MICRO);
    size_t initial_heap=GC_get_heap_size();
    gx_V src = native ? gx_make_byte_slice(gx_int(n),gx_int(n)) : gx_make_slice(gx_int(n),gx_int(n),gx_zero_int);
    gx_V dst = native ? gx_make_byte_slice(gx_int(n),gx_int(n)) : gx_make_slice(gx_int(n),gx_int(n),gx_zero_int);
    for (size_t i=0;i<n;i++) {
        if(native) gx_bytes(src)[i]=pattern(i); else gx_vals(src)[i]=gx_int(pattern(i));
    }
    double started=seconds();
    for (int i=0;i<16;i++) assert(gx_i(gx_copy(dst,src,NULL))==(int64_t)n);
    double copied=seconds()-started;
    started=seconds(); gx_V doubled=gx_append_slice(src,src,NULL); double appended=seconds()-started;
    assert(src.l==n && src.c==n && dst.l==n && dst.c==n && doubled.l==2*n && doubled.c==2*n);
    assert(src.u.p!=dst.u.p && doubled.u.p!=src.u.p && doubled.u.p!=dst.u.p);
    assert(gx_byte_backing(src)==native && gx_byte_backing(dst)==native && gx_byte_backing(doubled)==native);
    assert(GC_base(src.u.p) && GC_base(dst.u.p) && GC_base(doubled.u.p));
    verify(src,n); verify(dst,n); verify(doubled,2*n);
    /* Selected writes must not change input or the append output. */
    gx_sset(dst,gx_int(0),gx_int(9));
    assert(gx_i(gx_sget(src,gx_int(0)))==pattern(0));
    assert(gx_i(gx_sget(doubled,gx_int(0)))==pattern(0));
    gx_sset(dst,gx_int(0),gx_int(pattern(0)));
    GC_gcollect();
    size_t element=native ? sizeof(uint8_t) : sizeof(gx_V);
    size_t capacity_bytes=((size_t)src.c+dst.c+doubled.c)*element;
    size_t granules=GC_size(GC_base(src.u.p))+GC_size(GC_base(dst.u.p))+GC_size(GC_base(doubled.u.p));
    struct rusage usage; assert(getrusage(RUSAGE_SELF,&usage)==0);
    printf("mode=%s N=%zu gx_V=%zu element=%zu capacity=%u+%u+%u exact_element_bytes=%zu allocator_bytes=%zu initial_heap=%zu gc_heap=%zu gc_free=%zu rss_max_kib=%ld collections=%zu interior=%d copy_16_seconds=%.6f copy_MiB_s=%.2f append_seconds=%.6f\n",argv[1],n,sizeof(gx_V),element,src.c,dst.c,doubled.c,capacity_bytes,granules,initial_heap,(size_t)GC_get_heap_size(),(size_t)GC_get_free_bytes(),usage.ru_maxrss,(size_t)GC_get_gc_no(),GC_get_all_interior_pointers(),copied,16.0*n/(1024*1024*copied),appended);
    verify(src,n);verify(dst,n);verify(doubled,2*n);
    return 0;
}
