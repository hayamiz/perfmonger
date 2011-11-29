
#include "test.h"
#include <perfmonger.h>


/* global variables */

int argc;
char **argv;

/* utility function proto */

void setup_arguments(const char *arg, ...);

/* test function proto */
void test_parse_args(void);


/* cutter setup/teardown */

void
cut_setup(void)
{
    argc = 0;
    argv = NULL;
}

void
cut_teardown(void)
{
    
}

/* utility function bodies */

void
setup_arguments(const char *arg, ...)
{
    va_list ap;

    argc = 1;
    argv = malloc(sizeof(char *));
    argv[0] = (char *) arg;

    va_start(ap, arg);
    while((arg = va_arg(ap, char *)) != NULL) {
        argv = realloc(argv, sizeof(char *) * (argc + 1));
        argv[argc ++] = (char *) arg;
    }
    va_end(ap);

    cut_take_memory(argv);
}

/* test function bodies */

void
test_parse_args(void)
{
    setup_arguments("collector", NULL);
    cut_assert_equal_int(-1, parse_args(argc, argv));

    /* check -d option and default values */
    setup_arguments("collector", "-d", "/path/to/dev", NULL);
    cut_assert_equal_int(0, parse_args(argc, argv));
    cut_assert_equal_int(1, option.nr_dev);
    cut_assert_equal_string("/path/to/dev", option.dev_list[0]);
    cut_assert_equal_double(1.0, 0.0001, option.interval);

    setup_arguments("collector", "-d", "/path/to/dev0", "-d", "/path/to/dev1", NULL);
    cut_assert_equal_int(0, parse_args(argc, argv));
    cut_assert_equal_int(2, option.nr_dev);
    cut_assert_equal_string("/path/to/dev0", option.dev_list[0]);
    cut_assert_equal_string("/path/to/dev1", option.dev_list[1]);

    setup_arguments("collector", "-d", "/path/to/dev", "-i", "10", NULL);
    cut_assert_equal_int(0, parse_args(argc, argv));
    cut_assert_equal_double(10.0, 0.0001, option.interval);

    setup_arguments("collector", "-d", "/path/to/dev", "-i", "0.5", NULL);
    cut_assert_equal_int(0, parse_args(argc, argv));
    cut_assert_equal_double(0.5, 0.0001, option.interval);
}
