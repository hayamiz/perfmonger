/* -*- indent-tabs-mode: nil -*- */

#ifndef PERFMONGER_H
#define PERFMONGER_H

#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>
#include <math.h>
#include <ctype.h>
#include <limits.h>
#include <stdbool.h>
#include <pthread.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <sys/time.h>
#include <signal.h>

#include "sysstat/libsysstat.h"


typedef struct {
    // switches
    bool report_cpu;
    bool report_io;
    bool report_ctxsw;

    // device nr and list
    int nr_dev;
    char **dev_list;
	bool all_devices;

    // OUTPUT FILE
    FILE *output;

    // data collection interval
    double interval;
    bool interval_backoff;

    double start_delay;

    double timeout;

    bool verbose;
} option_t;

int  parse_args(int argc, char **argv, option_t *opt);
void print_help(void);
void init_subsystem(option_t *opt);
void destroy_subsystem(option_t *opt);

void collector_loop(option_t *opt);
void output_stat(option_t *opt, int curr);

#endif
