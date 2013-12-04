
#include "perfmonger.h"

int nr_thargs = 0;

int
main(int argc, char **argv)
{
    option_t opt;
    if (parse_args(argc, argv, &opt) != 0){
        fprintf(stderr, "Argument error. Exit.\n");
        exit(EXIT_FAILURE);
    }

    init_subsystem(&opt);
    collector_loop(&opt);
    destroy_subsystem(&opt);

    return 0;
}
