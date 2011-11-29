
#include "perfmonger.h"

int nr_thargs = 0;
volatile bool running = true;

static void
sigint_handler(int signum)
{
    running = false;
}

int
main(int argc, char **argv)
{
    if (parse_args(argc, argv) != 0){
        fprintf(stderr, "Argument error. Exit.\n");
        exit(EXIT_FAILURE);
    }

    signal(SIGINT, sigint_handler);

    init_iostat_subsystem();
    io_collector_loop();
    destroy_iostat_subsystem();

    return 0;
}
