
#include "perfmonger.h"


volatile bool running = true;


/*
 * Variables in sysstat/iostat.c
 */

extern struct io_stats *st_iodev[2];
extern struct io_hdr_stats *st_hdr_iodev;
extern struct io_dlist *st_dev_list;

extern int iodev_nr;	/* Nb of devices and partitions found */
extern int cpu_nr;		/* Nb of processors on the machine */
extern int dlist_idx;	/* Nb of devices entered on the command line */
extern unsigned int dm_major;	/* Device-mapper major number */

/* Variables in sysstat/mpstat.c */
extern unsigned long long uptime[3];
extern unsigned long long uptime0[3];

extern unsigned char *cpu_bitmap;	/* Bit 0: Global; Bit 1: 1st proc; etc. */

extern struct stats_cpu *st_cpu[3];
extern struct stats_irq *st_irq[3];
extern struct tm mp_tstamp[3];
extern struct stats_irqcpu *st_irqcpu[3];
extern struct stats_irqcpu *st_softirqcpu[3];
extern struct tm mp_tstamp[3];
/* Nb of interrupts per processor */
extern int irqcpu_nr;
/* Nb of soft interrupts per processor */
extern int softirqcpu_nr;
extern unsigned int actflags;


/* long options */
int
parse_args(int argc, char **argv, option_t *opt)
{
    int optval;
    GString *errmsg;

    errmsg = g_string_new("");
    optind = 1;

    opt->nr_dev = 0;
    opt->dev_list = NULL;
    opt->interval = 1.0;
    opt->verbose = false;
    opt->show_cpu = false;

    while((optval = getopt(argc, argv, "d:i:vhC")) != -1) {
        switch(optval) {
        case 'd': // device
            opt->nr_dev ++;
            opt->dev_list = realloc(opt->dev_list, opt->nr_dev * sizeof(char *));
            opt->dev_list[opt->nr_dev - 1] = strdup(optarg);
            break;
        case 'i': // interval
            opt->interval = strtod(optarg, NULL);
            break;
        case 'v': // verbose
            opt->verbose = true;
            break;
        case 'h': // help
            print_help();
            goto error;
            break;
        case 'C': // show CPU
            opt->show_cpu = true;
            break;
        default:
            print_help();
            goto error;
        }
    }

    if (opt->nr_dev == 0)
    {
        opt->show_io = false;
        opt->show_cpu = true;
    }
    else
    {
        opt->show_io = true;
    }

    return 0;
error:
    fprintf(stderr, "%s", errmsg->str);
    return -1;
}

/*
 * print_help:
 *
 * Prints usage of PerfMonger in the standard output and call exit(2)
 * with @exit_status. If @exit_status < 0, do not invoke exit(2).
 */
void
print_help(void)
{
    printf("Usage: pgr [options]\n");
}

void
init_subsystem(option_t *opt)
{
    int i;
    struct io_dlist *st_dev_list_i;

    get_HZ();
    salloc_dev_list(opt->nr_dev);
    io_sys_init();

    for (i = 0; i < opt->nr_dev; i++) {
        update_dev_list(&dlist_idx, opt->dev_list[i]);
        st_dev_list_i = st_dev_list + i;
        st_dev_list_i->disp_part = TRUE;
    }

    
    /* ----------------------------------------- */
    /* initialization for mpstat functionalities */
    /* ----------------------------------------- */

    cpu_nr = get_cpu_nr(~0);
    irqcpu_nr = get_irqcpu_nr(INTERRUPTS, NR_IRQS, cpu_nr) +
        NR_IRQCPU_PREALLOC;
    softirqcpu_nr = get_irqcpu_nr(SOFTIRQS, NR_IRQS, cpu_nr) +
        NR_IRQCPU_PREALLOC;

    salloc_mp_struct(cpu_nr + 1);

    /* Enable all activity flags */
    actflags |= M_D_CPU;
    actflags |= M_D_IRQ_SUM;
    actflags |= M_D_IRQ_CPU;
    actflags |= M_D_SOFTIRQS;
    actflags |= M_D_IRQ_SUM + M_D_IRQ_CPU + M_D_SOFTIRQS;

    /* set bit for every processor */
    memset(cpu_bitmap, 0xff, ((cpu_nr + 1) >> 3) + 1);


    // init uptime
    if (cpu_nr > 1) {
        uptime0[0] = 0;
        read_uptime(&(uptime0[0]));
    }
    // init st_cpu
    read_stat_cpu(st_cpu[0], cpu_nr + 1, &(uptime[0]), &(uptime0[0]));
    // init st_irq
    read_stat_irq(st_irq[0], 1);
    // init st_interrupts_stat
    read_interrupts_stat(SOFTIRQS, st_softirqcpu, softirqcpu_nr, 0);

    /* Save the first stats collected. Will be used to compute the average */
    mp_tstamp[2] = mp_tstamp[0];
    uptime[2] = uptime[0];
    uptime0[2] = uptime0[0];
    memcpy(st_cpu[2], st_cpu[0], STATS_CPU_SIZE * (cpu_nr + 1));
    memcpy(st_irq[2], st_irq[0], STATS_IRQ_SIZE * (cpu_nr + 1));
    memcpy(st_irqcpu[2], st_irqcpu[0], STATS_IRQCPU_SIZE * (cpu_nr + 1) * irqcpu_nr);
    if (DISPLAY_SOFTIRQS(actflags)) {
        memcpy(st_softirqcpu[2], st_softirqcpu[0],
               STATS_IRQCPU_SIZE * (cpu_nr + 1) * softirqcpu_nr);
    }
}

void
destroy_subsystem(option_t *opt)
{
    io_sys_free();
    sfree_dev_list();
}

void output_mpstat(int curr);

void
collector_loop(option_t *opt)
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
        wait_until += opt->interval * 1000000L;

        uptime0[curr] = 0;
        read_uptime(&(uptime0[curr]));

        read_stat_cpu(st_cpu[curr], cpu_nr + 1, &(uptime[0]), &(uptime0[0]));
        read_diskstats_stat(curr);

        output_diskstats_stat(curr);
        output_mpstat(curr);

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
output_mpstat(curr)
{
    struct stats_cpu *scc, *scp;
    unsigned long long itv, pc_itv, g_itv;
    int cpu;

    g_itv = get_interval(uptime[!curr], uptime[curr]);

    for (cpu = 1; cpu <= cpu_nr; cpu++) {
        scc = st_cpu[curr] + cpu;
        scp = st_cpu[!curr] + cpu;

        /* Recalculate itv for current proc */
        pc_itv = get_per_cpu_interval(scc, scp);
        if (!pc_itv) {
            /*
             * If the CPU is tickless then there is no change in CPU values
             * but the sum of values is not zero.
             */
            printf("  %6.2f  %6.2f  %6.2f  %6.2f  %6.2f  %6.2f"
                   "  %6.2f  %6.2f  %6.2f\n",
                   0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 100.0);
        }
        else
        {
            printf("  %6.2f  %6.2f  %6.2f  %6.2f  %6.2f  %6.2f"
                   "  %6.2f  %6.2f  %6.2f\n",
                   (scc->cpu_user - scc->cpu_guest) < (scp->cpu_user - scp->cpu_guest) ?
                   0.0 :
                   ll_sp_value(scp->cpu_user - scp->cpu_guest,
                               scc->cpu_user - scc->cpu_guest,
                               pc_itv),
                   ll_sp_value(scp->cpu_nice,
                               scc->cpu_nice,
                               pc_itv),
                   ll_sp_value(scp->cpu_sys,
                               scc->cpu_sys,
                               pc_itv),
                   ll_sp_value(scp->cpu_iowait,
                               scc->cpu_iowait,
                               pc_itv),
                   ll_sp_value(scp->cpu_hardirq,
                               scc->cpu_hardirq,
                               pc_itv),
                   ll_sp_value(scp->cpu_softirq,
                               scc->cpu_softirq,
                               pc_itv),
                   ll_sp_value(scp->cpu_steal,
                               scc->cpu_steal,
                               pc_itv),
                   ll_sp_value(scp->cpu_guest,
                               scc->cpu_guest,
                               pc_itv),
                   (scc->cpu_idle < scp->cpu_idle) ?
                   0.0 :
                   ll_sp_value(scp->cpu_idle,
                               scc->cpu_idle,
                               pc_itv));
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

    double r_iops, w_iops;
    double r_await, w_await;
    double reqsz;

    interval = get_interval(uptime[!curr], uptime[curr]);
    gettimeofday(&tv, NULL);

    g_print("{\"time\": %.4lf, \"ioinfo\": {\"devices\": [",
            tv.tv_sec + ((double) tv.tv_usec) / 1000000.0);

    int nr_dev_printed = 0;
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

        if (nr_dev_printed > 0) {
            g_print(", ");
        }
        g_print("\"%s\"", shi->name);

        nr_dev_printed++;
    }
    g_print("], ");


    interval = get_interval(uptime0[!curr], uptime0[curr]);

    r_await = 0;
    w_await = 0;
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
        w_await += (ioi->wr_ios - ioj->wr_ios) ?
            (ioi->wr_ticks - ioj->wr_ticks) /
            ((double) (ioi->wr_ios - ioj->wr_ios)) : 0.0;
        r_iops += S_VALUE(ioj->rd_ios, ioi->rd_ios, interval);
        w_iops += S_VALUE(ioj->wr_ios, ioi->wr_ios, interval);
        reqsz += xds.arqsz;
        nr_dev ++;

        g_print("\"%s\": {\"r/s\": %.4lf, \"w/s\": %.4lf, \"r_await\": %.4lf, \"w_await\": %.4lf}, ",
                shi->name,
                S_VALUE(ioj->rd_ios, ioi->rd_ios, interval),
                S_VALUE(ioj->wr_ios, ioi->wr_ios, interval),
                (ioi->rd_ios - ioj->rd_ios) ?
                (ioi->rd_ticks - ioj->rd_ticks) /
                ((double) (ioi->rd_ios - ioj->rd_ios)) : 0.0,
                (ioi->wr_ios - ioj->wr_ios) ?
                (ioi->wr_ticks - ioj->wr_ticks) /
                ((double) (ioi->wr_ios - ioj->wr_ios)) : 0.0
            );
    }
    r_await /= nr_dev;
    w_await /= nr_dev;
    reqsz /= nr_dev;

    g_print("\"total\": {\"r/s\": %.4lf, \"w/s\": %.4lf, \"r_await\": %.4lf, \"w_await\": %.4lf}}}\n",
            r_iops,
            w_iops,
            r_await,
            w_await
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
