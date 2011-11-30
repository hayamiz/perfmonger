
#include "perfmonger.h"

/*
 * Variables in iostat/iostat.c
 */

extern struct stats_cpu *st_cpu[2];
extern unsigned long long uptime[2];
extern unsigned long long uptime0[2];
extern struct io_stats *st_iodev[2];
extern struct io_hdr_stats *st_hdr_iodev;
extern struct io_dlist *st_dev_list;

extern int iodev_nr;	/* Nb of devices and partitions found */
extern int cpu_nr;		/* Nb of processors on the machine */
extern int dlist_idx;	/* Nb of devices entered on the command line */
extern int flags;		/* Flag for common options and system state */
extern unsigned int dm_major;	/* Device-mapper major number */


option_t option;

int
parse_args(int argc, char **argv)
{
    int opt;
    GString *errmsg;

    errmsg = g_string_new("");
    optind = 1;

    option.nr_dev = 0;
    option.dev_list = NULL;
    option.interval = 1.0;

    while((opt = getopt(argc, argv, "d:i:p:D:")) != -1) {
        switch(opt) {
        case 'd': // device
            option.nr_dev ++;
            option.dev_list = realloc(option.dev_list, option.nr_dev * sizeof(char *));
            option.dev_list[option.nr_dev - 1] = strdup(optarg);
            break;
        case 'i':
            option.interval = strtod(optarg, NULL);
            break;
        }
    }

    if (option.nr_dev == 0) {
        g_string_append_printf(errmsg,
                               "No device specified.\n");
        goto error;
    }

    return 0;
error:
    fprintf(stderr, "%s", errmsg->str);
    return -1;
}


void
init_iostat_subsystem()
{
    int i;
    int idx;
    struct io_dlist *st_dev_list_i;

    get_HZ();
    salloc_dev_list(option.nr_dev);
    io_sys_init();

    for (i = 0; i < option.nr_dev; i++) {
        idx = update_dev_list(&dlist_idx, option.dev_list[i]);
        st_dev_list_i = st_dev_list + i;
        st_dev_list_i->disp_part = TRUE;
    }
}

void
destroy_iostat_subsystem(void)
{
    io_sys_free();
    sfree_dev_list();
}

void
io_collector_loop(void)
{
    int curr;
    struct timeval tv;
    long wait_until;
    long wait_interval;

    curr = 1;
    setbuf(stdout, NULL);

    gettimeofday(&tv, NULL);
    wait_until = tv.tv_sec * 1000000L + tv.tv_usec;

    while(running) {
        wait_until += option.interval * 1000000L;

        uptime0[curr] = 0;
        read_uptime(&(uptime0[curr]));

        read_diskstats_stat(curr);
        output_diskstats_stat(curr);

        curr ^= 1;
        gettimeofday(&tv, NULL);
        wait_interval = wait_until - (tv.tv_sec * 1000000L + tv.tv_usec);
        if (wait_interval < 0){
            g_print("panic!: %ld\n", wait_interval);
        } else {
            usleep(wait_interval);
        }
    }
}

void
output_diskstats_stat(int curr)
{
    unsigned long long interval;
    struct io_hdr_stats *shi;
    int dev_idx;
    int i;
    struct io_stats *ioi, *ioj;
    int nr_dev;
    struct stats_disk sdc, sdp;
    struct ext_disk_stats xds;
    struct timeval tv;

    double r_await, r_iops, w_iops, reqsz;

    interval = get_interval(uptime[!curr], uptime[curr]);
    gettimeofday(&tv, NULL);

    g_print("{\"time\": %.4lf, ",
            tv.tv_sec + ((double) tv.tv_usec) / 1000000.0);

    if (cpu_nr > 1) {
        /* On SMP machines, reduce itv to one processor (see note above) */
        interval = get_interval(uptime0[!curr], uptime0[curr]);

        r_await = 0;
        r_iops = 0;
        w_iops = 0;
        reqsz = 0;
        nr_dev = 0;

        for (i = 0, shi = st_hdr_iodev; i < iodev_nr; i++, shi++) {
            if (! shi->used) {
                continue;
            }

            for (dev_idx = 0; dev_idx < dlist_idx; dev_idx++) {
                if (! strcmp(shi->name, st_dev_list[dev_idx].dev_name)) {
                    break;
                }
            }
            if (dev_idx == dlist_idx) {
                continue;
            }

            ioi = st_iodev[curr] + i;
            ioj = st_iodev[!curr] + i;

            sdc.nr_ios    = ioi->rd_ios + ioi->wr_ios;
            sdp.nr_ios    = ioj->rd_ios + ioj->wr_ios;

            sdc.tot_ticks = ioi->tot_ticks;
            sdp.tot_ticks = ioj->tot_ticks;

            sdc.rd_ticks  = ioi->rd_ticks;
            sdp.rd_ticks  = ioj->rd_ticks;
            sdc.wr_ticks  = ioi->wr_ticks;
            sdp.wr_ticks  = ioj->wr_ticks;

            sdc.rd_sect   = ioi->rd_sectors;
            sdp.rd_sect   = ioj->rd_sectors;
            sdc.wr_sect   = ioi->wr_sectors;
            sdp.wr_sect   = ioj->wr_sectors;

            compute_ext_disk_stats(&sdc, &sdp, interval, &xds);

            r_await += (ioi->rd_ios - ioj->rd_ios) ?
                (ioi->rd_ticks - ioj->rd_ticks) /
                ((double) (ioi->rd_ios - ioj->rd_ios)) : 0.0;
            r_iops += S_VALUE(ioj->rd_ios, ioi->rd_ios, interval);
            w_iops += S_VALUE(ioj->rd_ios, ioi->rd_ios, interval);
            reqsz += xds.arqsz;
            nr_dev ++;

            g_print("\"%s\": {\"r/s\": %.4lf, \"w/s\": %.4lf}, ",
                    st_dev_list[dev_idx].dev_name,
                    S_VALUE(ioj->rd_ios, ioi->rd_ios, interval),
                    S_VALUE(ioj->rd_ios, ioi->rd_ios, interval)
                );
        }
        r_await /= nr_dev;
        reqsz /= nr_dev;

        g_print("\"total\": {\"r/s\": %.4lf, \"w/s\": %.4lf}}\n",
                r_iops,
                w_iops
            );

        // g_print("millitime\t%ld\n"
        //         "iops\t%lf\n"
        //         "await\t%lf\n"
        //         "avgreqsz\t%lf\n",
        //         tv.tv_sec * 1000 + tv.tv_usec / 1000,
        //         r_iops,
        //         r_await,
        //         reqsz);
    }
}
