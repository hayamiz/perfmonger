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
#include "sysstat/libiostat.h"


typedef struct {
    // device nr and list
    int nr_dev;
    char **dev_list;

    // data collection interval
    double interval;
} option_t;

extern option_t option;
extern volatile bool running;

int parse_args(int argc, char **argv);
void init_iostat_subsystem(void);
void destroy_iostat_subsystem(void);

void io_collector_loop(void);
void output_diskstats_stat(int curr);

#endif
