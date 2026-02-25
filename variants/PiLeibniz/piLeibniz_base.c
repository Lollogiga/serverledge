/*
 * Baseline C variant: n prefissato (1 000 000 iterazioni).
 * Ignora gli argomenti da riga di comando.
 *
 * IMPORTANT: must be compiled as a statically-linked binary so it runs
 * inside the Alpine-based native runtime container (no glibc available).
 * Compile: gcc -O2 -static -o piLeibniz_base piLeibniz_base.c
 */
#include <stdio.h>

#define FIXED_N 1000000

int main(int argc, char **argv) {
    double s     = 0.0;
    double sign  = 1.0;
    double denom = 1.0;

    for (int i = 0; i < FIXED_N; i++) {
        s    += sign / denom;
        sign  = -sign;
        denom += 2.0;
    }

    printf("%.15f\n", 4.0 * s);
    return 0;
}
