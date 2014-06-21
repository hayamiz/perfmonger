/* -*- indent-tabs-mode: nil -*- */

#include "perfmonger.h"

typedef struct {
    char *buffer;
    char *cursor;
    size_t size;
    size_t len;
} strbuf_t;

/*
 * Private functions
 */
static void sigint_handler(int signum, siginfo_t *info, void *handler);
static void sigterm_handler(int signum, siginfo_t *info, void *handler);

static strbuf_t *strbuf_new(void);
static void      strbuf_free(strbuf_t *strbuf);
static int       strbuf_append(strbuf_t *strbuf, const char *format, ...);

static void output_io_stat   (strbuf_t *output, int curr);
static void output_cpu_stat  (strbuf_t *output, int curr);
static void output_ctxsw_stat(strbuf_t *output, int curr);


/*
 * Global variables
 */

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

/* Variables for pcsw */
struct stats_pcsw st_pcsw[3];

/* Variables for handling signals */
static volatile sig_atomic_t sigint_sent = 0;
static volatile sig_atomic_t sigterm_sent = 0;


/* TODO: long options */
int
parse_args(int argc, char **argv, option_t *opt)
{
    int optval;

    optind = 1;

    opt->nr_dev       = 0;
    opt->dev_list     = NULL;
    opt->all_devices  = false;
    opt->interval     = 1.0;
    opt->start_delay  = 0.0;
    opt->timeout      = -1.0;   /* negative means no timeout */
    opt->verbose      = false;
    opt->report_cpu   = false;
    opt->report_io    = false;
    opt->report_ctxsw = false;
    opt->output       = stdout;

    while((optval = getopt(argc, argv, "d:Di:s:t:vhCSl:")) != -1) {
        switch(optval) {
        case 'd': // device
            opt->nr_dev ++;
            opt->dev_list = realloc(opt->dev_list, opt->nr_dev * sizeof(char *));
            opt->dev_list[opt->nr_dev - 1] = strdup(optarg);
            opt->report_io = true;
            break;
        case 'D': // show all devices
            opt->report_io = true;
            opt->all_devices = true;
            break;
        case 'i': // interval
            opt->interval = strtod(optarg, NULL);
            break;
        case 's': // start delay
            opt->start_delay = strtod(optarg, NULL);
            break;
        case 't': // timeout
            opt->timeout = strtod(optarg, NULL);
            break;
        case 'v': // verbose
            opt->verbose = true;
            break;
        case 'h': // help
            print_help();
            goto error;
            break;
        case 'C': // show CPU
            opt->report_cpu = true;
            break;
        case 'S': // show context switch per second
            opt->report_ctxsw = true;
            break;
        case 'l': // show context switch per second
            opt->output = fopen(optarg, "w");
            if (opt->output == NULL)
            {
                perror("log file open failed");
                goto error;
            }
            break;
        default:
            print_help();
            goto error;
        }
    }

    if (! (opt->report_io || opt->report_ctxsw))
        opt->report_cpu = true;

    return 0;
error:
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
    printf("Usage: perfmonger [options]\n");
}

/*
 * signal handlers
 */
static void
sigint_handler(int signum, siginfo_t *info, void *handler)
{
    sigint_sent = 1;
}

static void
sigterm_handler(int signum, siginfo_t *info, void *handler)
{
    sigterm_sent = 1;
}


void
init_subsystem(option_t *opt)
{
    int i;
    struct io_dlist *st_dev_list_i;
    struct sigaction sigint_act;
    struct sigaction sigterm_act;

    get_HZ();
    salloc_dev_list(opt->nr_dev);
    io_sys_init();

    if (! opt->all_devices)
    {
        for (i = 0; i < opt->nr_dev; i++) {
            update_dev_list(&dlist_idx, opt->dev_list[i]);
            st_dev_list_i = st_dev_list + i;
            st_dev_list_i->disp_part = TRUE;
        }
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
    // init st_pcsw
    read_stat_pcsw(&st_pcsw[0]);
    // init st_iodev
    read_diskstats_stat(0);

    /* Save the first stats collected. Will be used to compute the average */
    mp_tstamp[2] = mp_tstamp[0];
    uptime[2] = uptime[0];
    uptime0[2] = uptime0[0];
    st_pcsw[2] = st_pcsw[0];
    memcpy(st_cpu[2], st_cpu[0], STATS_CPU_SIZE * (cpu_nr + 1));
    memcpy(st_irq[2], st_irq[0], STATS_IRQ_SIZE * (cpu_nr + 1));
    memcpy(st_irqcpu[2], st_irqcpu[0], STATS_IRQCPU_SIZE * (cpu_nr + 1) * irqcpu_nr);
    if (DISPLAY_SOFTIRQS(actflags)) {
        memcpy(st_softirqcpu[2], st_softirqcpu[0],
               STATS_IRQCPU_SIZE * (cpu_nr + 1) * softirqcpu_nr);
    }

    /* setup signal handlers */

    bzero(&sigint_act, sizeof(struct sigaction));
    sigint_act.sa_sigaction = sigint_handler;
    sigint_act.sa_flags = SA_SIGINFO | SA_RESTART;
    if (sigaction(SIGINT, &sigint_act, NULL) != 0) {
        perror("failed to set SIGINT handler");
        exit(EXIT_FAILURE);
    }

    bzero(&sigterm_act, sizeof(struct sigaction));
    sigterm_act.sa_sigaction = sigterm_handler;
    sigterm_act.sa_flags = SA_SIGINFO | SA_RESTART;
    if (sigaction(SIGTERM, &sigterm_act, NULL) != 0) {
        perror("failed to set SIGTERM handler");
        exit(EXIT_FAILURE);
    }
}

void
destroy_subsystem(option_t *opt)
{
    io_sys_free();
    sfree_dev_list();
}

void
collector_loop(option_t *opt)
{
    int curr;
    struct timeval tv;
    long wait_until;
    long wait_interval;
    long timeout_when;
    bool running;

    if (opt->start_delay > 0.0) {
        usleep(opt->start_delay * 1000000L);
    }

    curr = 1;
    setbuf(stdout, NULL);

    gettimeofday(&tv, NULL);
    wait_until = tv.tv_sec * 1000000L + tv.tv_usec;

    if (opt->timeout > 0) {
        timeout_when = wait_until + opt->timeout * 1000000L; /* in usec */
    } else {
        timeout_when = LONG_MAX;
    }


    running = true;
    while(running) {
        if (sigint_sent || sigterm_sent) {
            /* Do not break loop here. For capturing execution time
             * accurate as possible, it is necessary to outputing 1
             * line just after SIGINT was handled */
            running = false;
        }

        if (wait_until >= timeout_when) {
            running = false;
        }

        wait_until += opt->interval * 1000000L;

        uptime0[curr] = 0;
        read_uptime(&(uptime0[curr]));

        if (opt->report_cpu)
            read_stat_cpu(st_cpu[curr], cpu_nr + 1, &(uptime[curr]), &(uptime0[curr]));
        if (opt->report_io)
            read_diskstats_stat(curr);
        if (opt->report_ctxsw)
            read_stat_pcsw(&st_pcsw[curr]);

        output_stat(opt, curr);

        if (! running) break;

        if (wait_until > timeout_when) {
            wait_until = timeout_when;
        }

        curr ^= 1;
        gettimeofday(&tv, NULL);

        wait_interval = wait_until - (tv.tv_sec * 1000000L + tv.tv_usec);

        if (wait_interval < 0){
            if (opt->verbose)
                fprintf(stderr, "panic!: %ld\n", wait_interval);
        } else {
            usleep(wait_interval);
        }
    }

    fflush(opt->output);
    fclose(opt->output);
}


void
output_stat(option_t *opt, int curr)
{
    struct timeval tv;
    strbuf_t *output;

    output = strbuf_new();

    gettimeofday(&tv, NULL);
    strbuf_append(output,
                  "{\"time\": %.4lf", tv.tv_sec + ((double) tv.tv_usec) / 1000000.0);

    if (opt->report_cpu)   output_cpu_stat(output, curr);
    if (opt->report_io)    output_io_stat(output, curr);
    if (opt->report_ctxsw) output_ctxsw_stat(output, curr);

    strbuf_append(output, "}");
    fprintf(opt->output, "%s\n",  output->buffer);
    strbuf_free(output);
}

static strbuf_t *
strbuf_new(void)
{
    strbuf_t *strbuf;

    strbuf = malloc(sizeof(strbuf_t));
    if (strbuf == NULL)
    {
        return NULL;
    }

#define INIT_STRBUF_SIZE 1024
    strbuf->buffer = malloc(sizeof(char) * INIT_STRBUF_SIZE);
    bzero(strbuf->buffer, sizeof(char) * INIT_STRBUF_SIZE);
    strbuf->cursor = strbuf->buffer;
    strbuf->size = INIT_STRBUF_SIZE;
    strbuf->len = 0;

    return strbuf;
}

static void
strbuf_free(strbuf_t *strbuf)
{
    free(strbuf->buffer);
    free(strbuf);
}

static int
strbuf_append(strbuf_t *strbuf, const char *format, ...)
{
    va_list ap;
    int n;
    size_t size;

    for (;;)
    {
        va_start(ap, format);

        size = strbuf->size - strbuf->len;
        n = vsnprintf(strbuf->cursor, size, format, ap);
        if (n < 0)
        {
            return n; // error
        }
        else if (n < size)
        {
            strbuf->cursor += n;
            strbuf->len += n;

            va_end(ap);
            break;
        }
        else
        {
            int cursor_ofst = strbuf->cursor - strbuf->buffer;

            strbuf->size *= 2;
            strbuf->buffer = realloc(strbuf->buffer, strbuf->size);
            strbuf->cursor = strbuf->buffer + cursor_ofst;
        }

        va_end(ap);
    }

    return n;
}

static void
output_io_stat (strbuf_t *output, int curr)
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
    double r_sectors, w_sectors;

    double reqsz;

    interval = get_interval(uptime[!curr], uptime[curr]);
    gettimeofday(&tv, NULL);

    strbuf_append(output, ", \"ioinfo\": {\"devices\": [");

    int nr_dev_printed = 0;
    for (i = 0, shi = st_hdr_iodev; i < iodev_nr; i++, shi++) {
        if (! shi->used) {
            continue;
        }
        if (dlist_idx) {
            for (dev_idx = 0; dev_idx < dlist_idx; dev_idx++) {
                if (! strcmp(shi->name, st_dev_list[dev_idx].dev_name)) {
                    break;
                }
            }
            if (dev_idx == dlist_idx) {
                continue;
            }
        }

        ioi = st_iodev[curr] + i;
        if (!ioi->rd_ios && !ioi->wr_ios) continue;

        if (nr_dev_printed > 0) {
            strbuf_append(output, ", ");
        }
        strbuf_append(output, "\"%s\"", shi->name);

        nr_dev_printed++;
    }
    strbuf_append(output, "], ");


    interval = get_interval(uptime0[!curr], uptime0[curr]);

    r_await = 0;
    w_await = 0;
    r_iops = 0;
    w_iops = 0;
    r_sectors = 0;
    w_sectors = 0;
    reqsz = 0;
    nr_dev = 0;

    for (i = 0, shi = st_hdr_iodev; i < iodev_nr; i++, shi++) {
        if (! shi->used) {
            continue;
        }

        if (dlist_idx) {
            for (dev_idx = 0; dev_idx < dlist_idx; dev_idx++) {
                if (! strcmp(shi->name, st_dev_list[dev_idx].dev_name)) {
                    break;
                }
            }
            if (dev_idx == dlist_idx) {
                continue;
            }
        }

        ioi = st_iodev[curr] + i;
        ioj = st_iodev[!curr] + i;

        ioi = st_iodev[curr] + i;
        if (!ioi->rd_ios && !ioi->wr_ios) continue;

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
        r_sectors += ll_s_value(ioj->rd_sectors, ioi->rd_sectors, interval);
        w_sectors += ll_s_value(ioj->wr_sectors, ioi->wr_sectors, interval);

        reqsz += xds.arqsz;
        nr_dev ++;

        strbuf_append(output,
                      "\"%s\": {\"riops\": %.4lf, \"wiops\": %.4lf, "
                      "\"rsecps\": %.4lf, \"wsecps\": %.4lf, "
                      "\"r_await\": %.4lf, \"w_await\": %.4lf, "
                      "\"avgrq-sz\": %.4lf, \"avgqu-sz\": %.4lf}, ",
                      shi->name,
                      S_VALUE(ioj->rd_ios, ioi->rd_ios, interval),
                      S_VALUE(ioj->wr_ios, ioi->wr_ios, interval),
                      ll_s_value(ioj->rd_sectors, ioi->rd_sectors, interval),
                      ll_s_value(ioj->wr_sectors, ioi->wr_sectors, interval),
                      (ioi->rd_ios - ioj->rd_ios) ?
                      (ioi->rd_ticks - ioj->rd_ticks) /
                      ((double) (ioi->rd_ios - ioj->rd_ios)) : 0.0,
                      (ioi->wr_ios - ioj->wr_ios) ?
                      (ioi->wr_ticks - ioj->wr_ticks) /
                      ((double) (ioi->wr_ios - ioj->wr_ios)) : 0.0,
                      (double) xds.arqsz,
                      (double) S_VALUE(ioj->rq_ticks, ioi->rq_ticks, interval) / 1000.0
            );
    }
    r_await /= nr_dev;
    w_await /= nr_dev;
    reqsz /= nr_dev;

    strbuf_append(output,
                  "\"total\": {\"riops\": %.4lf, \"wiops\": %.4lf, "
                  "\"rsecps\": %.4lf, \"wsecps\": %.4lf, "
                  "\"r_await\": %.4lf, \"w_await\": %.4lf}}",
                  r_iops, w_iops,
                  r_sectors, w_sectors,
                  r_await, w_await
        );
}

static void
output_cpu_stat(strbuf_t *output, int curr)
{
    struct stats_cpu *scc, *scp;
    unsigned long long pc_itv, g_itv;
    int cpu;
    int nr_cpu_printed = 0;

    g_itv = get_interval(uptime[!curr], uptime[curr]);

    strbuf_append(output, ", \"cpuinfo\": {\"nr_cpu\": %d", cpu_nr);
    strbuf_append(output, ", \"all\": {\"usr\": %.2f, \"nice\": %.2f, "
                  "\"sys\": %.2f, \"iowait\": %.2f, "
                  "\"irq\": %.2f, \"soft\": %.2f, "
                  "\"steal\": %.2f, \"guest\": %.2f, "
                  "\"idle\": %.2f}",
                  (st_cpu[curr]->cpu_user - st_cpu[curr]->cpu_guest) <
                  (st_cpu[!curr]->cpu_user - st_cpu[!curr]->cpu_guest) ?
                  0.0 :
                  ll_sp_value(st_cpu[!curr]->cpu_user - st_cpu[!curr]->cpu_guest,
                              st_cpu[curr]->cpu_user - st_cpu[curr]->cpu_guest,
                              g_itv),
                  ll_sp_value(st_cpu[!curr]->cpu_nice,
                              st_cpu[curr]->cpu_nice,
                              g_itv),
                  ll_sp_value(st_cpu[!curr]->cpu_sys,
                              st_cpu[curr]->cpu_sys,
                              g_itv),
                  ll_sp_value(st_cpu[!curr]->cpu_iowait,
                              st_cpu[curr]->cpu_iowait,
                              g_itv),
                  ll_sp_value(st_cpu[!curr]->cpu_hardirq,
                              st_cpu[curr]->cpu_hardirq,
                              g_itv),
                  ll_sp_value(st_cpu[!curr]->cpu_softirq,
                              st_cpu[curr]->cpu_softirq,
                              g_itv),
                  ll_sp_value(st_cpu[!curr]->cpu_steal,
                              st_cpu[curr]->cpu_steal,
                              g_itv),
                  ll_sp_value(st_cpu[!curr]->cpu_guest,
                              st_cpu[curr]->cpu_guest,
                              g_itv),
                  (st_cpu[curr]->cpu_idle < st_cpu[!curr]->cpu_idle) ?
                  0.0 :
                  ll_sp_value(st_cpu[!curr]->cpu_idle,
                              st_cpu[curr]->cpu_idle,
                              g_itv));

    strbuf_append(output, ", \"cpus\": [");
    for (cpu = 1; cpu <= cpu_nr; cpu++) {
        scc = st_cpu[curr] + cpu;
        scp = st_cpu[!curr] + cpu;

        if (!(*(cpu_bitmap + (cpu >> 3)) & (1 << (cpu & 0x07))))
            continue;

        strbuf_append(output, (nr_cpu_printed > 0 ? ", " : ""));

        nr_cpu_printed++;

        /* Recalculate itv for current proc */
        pc_itv = get_per_cpu_interval(scc, scp);
        if (!pc_itv) {
            /*
             * If the CPU is tickless then there is no change in CPU values
             * but the sum of values is not zero.
             */
            strbuf_append(output, "{\"usr\": %.2f, \"nice\": %.2f, "
                          "\"sys\": %.2f, \"iowait\": %.2f, "
                          "\"irq\": %.2f, \"soft\": %.2f, "
                          "\"steal\": %.2f, \"guest\": %.2f, "
                          "\"idle\": %.2f}",
                          0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 100.0);
        }
        else
        {
            strbuf_append(output, "{\"usr\": %.2f, \"nice\": %.2f, "
                          "\"sys\": %.2f, \"iowait\": %.2f, "
                          "\"irq\": %.2f, \"soft\": %.2f, "
                          "\"steal\": %.2f, \"guest\": %.2f, "
                          "\"idle\": %.2f}",
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
    strbuf_append(output, "]}");
}

static void
output_ctxsw_stat(strbuf_t *output, int curr)
{
    unsigned long long itv;
    itv = get_interval(uptime0[!curr], uptime0[curr]);
    strbuf_append(output, ", \"ctxsw\": %.2f",
                  ll_s_value(st_pcsw[!curr].context_switch, st_pcsw[curr].context_switch, itv));
}
