#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/*
 * Estrae il parametro x passato come:
 *   --x=10
 * Ritorna 0 se non presente.
 */
static int parse_x(int argc, char **argv) {
    for (int i = 1; i < argc; i++) {
        if (strncmp(argv[i], "--x=", 4) == 0) {
            return atoi(argv[i] + 4);
        }
    }
    return 0;
}

int main(int argc, char **argv) {
    int n = parse_x(argc, argv);

    // Caso n <= 0 (coerente con Python)
    if (n <= 0) {
        printf("0\n");
        return 0;
    }

    long long a = 0;
    long long b = 1;

    // Stampa iniziale
    printf("0");

    if (n >= 1) {
        printf(", 1");
    }

    for (int i = 2; i <= n; i++) {
        long long c = a + b;
        printf(", %lld", c);
        a = b;
        b = c;
    }

    printf("\n");
    return 0;
}
