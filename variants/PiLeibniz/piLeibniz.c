#include <stdio.h>
#include <stdlib.h>
#include <string.h>


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

    if (n <= 0) {
        printf("0.0\n");
        return 0;
    }

    double s = 0.0;
    double sign = 1.0;
    double denom = 1.0;

    for (int i = 0; i < n; i++) {
        s += sign / denom;
        sign = -sign;
        denom += 2.0;
    }

    double pi = 4.0 * s;

    printf("%.15f\n", pi);

    return 0;
}
