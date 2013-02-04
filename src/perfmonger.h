#ifndef PERFMONGER_H
#define PERFMONGER_H

#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <pthread.h>
#include <glib.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <sys/time.h>

#include "../config.h"
#include "sysstat/libsysstat.h"


typedef struct {
    // device nr and list
    int nr_dev;
    char **dev_list;

    // show cpu usage
    bool show_cpu;

    // show io info
    bool show_io;

    // data collection interval
    double interval;

    bool verbose;
} option_t;

extern volatile bool running;

int  parse_args(int argc, char **argv, option_t *opt);
void print_help(void);
void init_subsystem(option_t *opt);
void destroy_subsystem(option_t *opt);

void collector_loop(option_t *opt);
void output_diskstats_stat(int curr);

#endif
