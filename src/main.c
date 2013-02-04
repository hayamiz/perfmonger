
#include "perfmonger.h"

int nr_thargs = 0;

static void
sigint_handler(int signum)
{
    running = false;
}

int
main(int argc, char **argv)
{
    option_t opt;
    if (parse_args(argc, argv, &opt) != 0){
        fprintf(stderr, "Argument error. Exit.\n");
        exit(EXIT_FAILURE);
    }

    running = true;

    signal(SIGINT, sigint_handler);

    init_subsystem(&opt);
    collector_loop(&opt);
    destroy_subsystem(&opt);

    return 0;
}
