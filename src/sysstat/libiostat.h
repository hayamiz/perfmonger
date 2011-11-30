#ifndef LIBIOSTAT_H
#define LIBIOSTAT_H

#include "iostat.h"

void salloc_dev_list(int list_len);
void get_HZ(void);
void io_sys_init(void);
void io_sys_free(void);
void sfree_dev_list(void);
int update_dev_list(int *dlist_idx, char *device_name);
void read_diskstats_stat(int curr);

#endif
