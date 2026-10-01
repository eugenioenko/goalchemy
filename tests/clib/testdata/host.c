/* A C host calling a Goalchemy library. */
#include <stdio.h>
#include <string.h>

#include "goalchemy.h"

int main(void) {
    if (goalchemy_init() != 0) {
        printf("init failed: %s\n", goalchemy_panic_message());
        return 1;
    }
    int64_t r;
    calc_Add(40, 2, &r);
    printf("add %lld\n", (long long)r);
    if (calc_Divide(1, 0, &r) != 0) printf("divide: %s\n", goalchemy_panic_message());
    calc_Divide(84, 2, &r);
    printf("divide %lld\n", (long long)r);
    const char *s;
    size_t n;
    const char *name = "Ada";
    calc_Greet("fr", 2, name, strlen(name), &s, &n);
    printf("greet %.*s\n", (int)n, s);
    calc_Greet("xx", 2, name, strlen(name), &s, &n);
    printf("greet %.*s\n", (int)n, s);
    int64_t bytes, words;
    const char *text = "the quick  brown fox";
    calc_Stats(text, strlen(text), &bytes, &words);
    printf("stats %lld %lld\n", (long long)bytes, (long long)words);
    calc_Fib(25, &r);
    printf("fib %lld\n", (long long)r);
    uint64_t u;
    calc_MaxUint(&u);
    printf("max %llu\n", (unsigned long long)u);
    bool even;
    calc_IsEven(7, &even);
    printf("even %s\n", even ? "true" : "false");
    if (calc_Check(false, &r) != 0) printf("check: %s\n", goalchemy_panic_message());
    calc_Check(true, &r);
    calc_Calls(&r);
    printf("calls %lld\n", (long long)r);
    return 0;
}
