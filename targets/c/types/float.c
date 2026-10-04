/* IEEE rounding and selected Go1.27 linux/amd64 integer conversions. */
#include "gx.h"
#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <locale.h>

double gx_round_float(double x, int bits) { volatile double result = bits == 32 ? (double)(float)x : x; return result; }
double gx_integer_float(int64_t value, bool uns, int bits) {
    if (bits == 32) { volatile float result = uns ? (float)(uint64_t)value : (float)value; return (double)result; }
    return uns ? (double)(uint64_t)value : (double)value;
}
int64_t gx_float_integer(double x, int bits, bool sign) {
    int64_t n;
    if (!sign && bits == 64) {
        if (!isfinite(x) || x < -0x1p63 || x >= 0x1p64) return INT64_MIN;
        if (x < 0x1p63) return (int64_t)x;
        uint64_t raw = (uint64_t)(x - 0x1p63) | (UINT64_C(1) << 63);
        memcpy(&n, &raw, sizeof(n)); return n;
    }
    int width = bits <= 16 || (bits == 32 && sign) ? 32 : 64;
    double limit = width == 32 ? 0x1p31 : 0x1p63;
    n = isfinite(x) && x >= -limit && x < limit ? (int64_t)x : width == 32 ? INT32_MIN : INT64_MIN;
    if (bits == 64) return n;
    uint64_t raw = (uint64_t)n & ((UINT64_C(1) << bits) - 1);
    if (sign && (raw & (UINT64_C(1) << (bits-1)))) raw |= UINT64_MAX << bits;
    memcpy(&n, &raw, sizeof(n)); return n;
}
double gx_float_min(double a, double b) { if (isnan(a) || isnan(b)) return NAN; if (a == 0 && b == 0) return signbit(a) || signbit(b) ? -0.0 : 0.0; return a < b ? a : b; }
double gx_float_max(double a, double b) { if (isnan(a) || isnan(b)) return NAN; if (a == 0 && b == 0) return !signbit(a) || !signbit(b) ? 0.0 : -0.0; return a > b ? a : b; }
gx_V gx_zero_float(void) { return gx_float(0.0); }

/* Candidate generation uses libc's correctly rounded scientific conversion
 * in a scoped numeric C locale, preserving the embedding host's locale. */
gx_V gx_float_print(double x, int bits) {
    x = gx_round_float(x, bits);
    if (isnan(x)) return gx_str("NaN", 3);
    if (isinf(x)) return gx_str(x < 0 ? "-Inf" : "+Inf", 4);
    if (x == 0) return gx_str(signbit(x) ? "-0" : "0", signbit(x) ? 2 : 1);
    locale_t numeric = newlocale(LC_NUMERIC_MASK, "C", (locale_t)0);
    if (!numeric) abort();
    locale_t previous = uselocale(numeric);
    if (!previous) { freelocale(numeric); abort(); }
    bool neg = x < 0; x = fabs(x);
    char text[64], digits[20], result[64];
    for (int n=1; n <= (bits == 32 ? 9 : 17); n++) {
        snprintf(text, sizeof(text), "%.*e", n-1, x);
        if (gx_round_float(strtod(text, NULL), bits) == x) break;
    }
    char *ep = strchr(text, 'e'); int exp = atoi(ep+1), nd = 0;
    for (char *p=text; p<ep; p++) if (*p != '.') digits[nd++] = *p;
    while (nd > 1 && digits[nd-1] == '0') nd--;
    digits[nd] = 0;
    int pos = 0; if (neg) result[pos++] = '-';
    if (exp < -4 || exp >= 6) {
        result[pos++] = digits[0];
        if (nd > 1) { result[pos++]='.'; memcpy(result+pos, digits+1, nd-1); pos+=nd-1; }
        pos += snprintf(result+pos, sizeof(result)-pos, "e%c%02d", exp<0?'-':'+', abs(exp));
    } else {
        int point=exp+1;
        if (point <= 0) { result[pos++]='0'; result[pos++]='.'; for(int k=0;k < -point;k++) result[pos++]='0'; memcpy(result+pos,digits,nd); pos+=nd; }
        else { for(int k=0;k < point || k < nd;k++) { if(k == point) result[pos++]='.'; result[pos++]=k<nd?digits[k]:'0'; } }
    }
    uselocale(previous);
    freelocale(numeric);
    return gx_str(result, (size_t)pos);
}
